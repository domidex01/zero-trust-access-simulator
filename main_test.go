package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zt-sim/internal/model"
	"zt-sim/internal/report"
	"zt-sim/internal/scenario"
)

func TestRun_Dispatch(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantExit   int
		wantStdout string
		wantStderr []string // substrings stderr must contain; empty = stderr must be empty
	}{
		{
			name:       "version flag",
			args:       []string{"--version"},
			wantExit:   0,
			wantStdout: "zt-sim version 0.1.0\n",
		},
		{
			name:       "no arguments",
			args:       nil,
			wantExit:   64,
			wantStderr: []string{"usage: zt-sim <command>"},
		},
		{
			name:       "unknown subcommand",
			args:       []string{"frobnicate"},
			wantExit:   64,
			wantStderr: []string{`unknown subcommand "frobnicate"`, "usage: zt-sim <command>"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			exit := run(tc.args, strings.NewReader(""), &stdout, &stderr)

			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d", exit, tc.wantExit)
			}
			if got := stdout.String(); got != tc.wantStdout {
				t.Errorf("stdout = %q, want %q", got, tc.wantStdout)
			}
			gotErr := stderr.String()
			if len(tc.wantStderr) == 0 {
				if gotErr != "" {
					t.Errorf("stderr = %q, want empty", gotErr)
				}
				return
			}
			for _, want := range tc.wantStderr {
				if !strings.Contains(gotErr, want) {
					t.Errorf("stderr %q does not contain %q", gotErr, want)
				}
			}
		})
	}
}

// Signal fixtures — the T4.1 signalJSON + TestPack_ScenarioShapes pins as JSON.
const (
	happySignal    = `{"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance","app":"intranet-wiki","dataSensitivity":"internal","location":"US","unfamiliarLocation":false,"deviceManaged":true,"deviceCompliant":true,"localHour":10,"mfaPresent":true,"riskScore":15}`
	byodSignal     = `{"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance","app":"intranet-wiki","dataSensitivity":"internal","location":"US","unfamiliarLocation":false,"deviceManaged":false,"deviceCompliant":false,"localHour":10,"mfaPresent":true,"riskScore":15}`
	stolenSignal   = `{"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance","app":"intranet-wiki","dataSensitivity":"internal","location":"US","unfamiliarLocation":false,"deviceManaged":false,"deviceCompliant":false,"localHour":10,"mfaPresent":false,"riskScore":45}`
	attackerSignal = `{"timestamp":"2026-01-15T10:00:00Z","user":"unknown@throwaway.example","role":"guest","app":"payroll-app","dataSensitivity":"high","location":"NG","unfamiliarLocation":true,"deviceManaged":false,"deviceCompliant":false,"localHour":2,"mfaPresent":false,"riskScore":85}`
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "in.json")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// cli runs the command with "@file" args replaced by a fixture holding
// content, returning exit code, stdout, stderr.
func cli(t *testing.T, content string, args []string, stdin string) (int, string, string) {
	t.Helper()
	path := writeFile(t, content)
	for i, a := range args {
		if a == "@file" {
			args[i] = path
		}
	}
	var stdout, stderr bytes.Buffer
	exit := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return exit, stdout.String(), stderr.String()
}

func TestRun_Decide(t *testing.T) {
	tests := []struct {
		name     string
		content  string   // written to the @file fixture
		args     []string // "@file" replaced by the fixture path
		stdin    string
		wantExit int
		wantOut  []string
		wantErr  []string
	}{
		{
			name: "happy file allows", content: "[" + happySignal + "]",
			args: []string{"decide", "@file"}, wantExit: 0,
			wantOut: []string{"USER", "ALLOW", "P11: all signals clean — explicit allow", "pillars: identity"},
		},
		{
			name: "byod file limits", content: "[" + byodSignal + "]",
			args: []string{"decide", "@file"}, wantExit: 1,
			wantOut: []string{"LIMITED"},
		},
		{
			name: "stolen file steps up", content: "[" + stolenSignal + "]",
			args: []string{"decide", "@file"}, wantExit: 2,
			wantOut: []string{"STEPUPMFA"},
		},
		{
			name: "attacker file blocks", content: "[" + attackerSignal + "]",
			args: []string{"decide", "@file"}, wantExit: 3,
			wantOut: []string{"BLOCK", "P07: risk score above hard threshold", "pillars: identity, devices, data, applications"},
		},
		{
			name: "mixed file takes strictest", content: "[" + happySignal + "," + attackerSignal + "]",
			args: []string{"decide", "@file"}, wantExit: 3,
			wantOut: []string{"alice@contoso.com", "unknown@throwaway.example"},
		},
		{
			name: "stdin single signal", args: []string{"decide", "--stdin"},
			stdin: happySignal, wantExit: 0,
			wantOut: []string{"ALLOW"},
		},
		{
			name: "stdin invalid object", args: []string{"decide", "--stdin"},
			stdin: happySignal[:len(happySignal)-1] + `,"risk":9}`, wantExit: 65,
			wantErr: []string{`unknown field "risk"`},
		},
		{
			name: "missing file", args: []string{"decide", "/nope/x.json"},
			wantExit: 65, wantErr: []string{"no such file"},
		},
		{
			name: "pack fed to decide", args: []string{"decide", "scenarios/attacker.json"},
			wantExit: 65, wantErr: []string{"[]model.Signal"},
		},
		{
			name:    "invalid signal in array",
			content: `[{"timestamp":"2026-01-15T10:00:00Z","user":"a@b.c","role":"finance","app":"wiki","dataSensitivity":"internal","riskScore":9000}]`,
			args:    []string{"decide", "@file"}, wantExit: 65,
			wantErr: []string{"riskScore"},
		},
		{
			name: "no input source", args: []string{"decide"},
			wantExit: 64, wantErr: []string{"usage: zt-sim decide"},
		},
		{
			name: "stdin AND file", content: "[" + happySignal + "]",
			args:     []string{"decide", "--stdin", "@file"},
			wantExit: 64, wantErr: []string{"usage: zt-sim decide"},
		},
		{
			name: "unknown flag", content: "[" + happySignal + "]",
			args:     []string{"decide", "--wat", "@file"},
			wantExit: 64, wantErr: []string{"usage: zt-sim decide"},
		},
		{
			name: "bad format value", content: "[" + happySignal + "]",
			args:     []string{"decide", "--format", "yaml", "@file"},
			wantExit: 64, wantErr: []string{"usage: zt-sim decide"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exit, out, errOut := cli(t, tc.content, tc.args, tc.stdin)
			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", exit, tc.wantExit, out, errOut)
			}
			for _, want := range tc.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("stdout missing %q\nstdout:\n%s", want, out)
				}
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(errOut, want) {
					t.Errorf("stderr missing %q\nstderr:\n%s", want, errOut)
				}
			}
			if tc.name == "mixed file takes strictest" && strings.Count(out, "USER") != 1 {
				t.Errorf("header line appears %d times, want 1\nstdout:\n%s", strings.Count(out, "USER"), out)
			}
		})
	}
}

func TestRun_DecideFormatJSON(t *testing.T) {
	exit, out, errOut := cli(t, "["+attackerSignal+"]", []string{"decide", "--format", "json", "@file"}, "")
	if exit != 3 {
		t.Fatalf("exit = %d, want 3\nstdout:\n%s\nstderr:\n%s", exit, out, errOut)
	}
	var got []model.Decision
	if err := json.Unmarshal([]byte(out), &got); err != nil {
		t.Fatalf("stdout is not a decisions array: %v\nstdout:\n%s", err, out)
	}
	if len(got) != 1 || got[0].Action != model.Block || got[0].Scope != "" || len(got[0].Matched) != 7 {
		t.Errorf("got = %+v, want 1 block decision with no scope and 7 matched", got)
	}
}

func TestRun_Run(t *testing.T) {
	tests := []struct {
		name     string
		dir      func(t *testing.T) string // nil = no dir arg
		extraArg bool                      // append a bogus second arg
		wantExit int
		wantOut  []string
		wantErr  []string
	}{
		{
			name: "scenarios dir all pass", dir: func(*testing.T) string { return "scenarios" },
			wantExit: 0,
			wantOut:  []string{"PACK", "happy", "note-scenario", "stolen-credential", "insider", "byod", "attacker"},
		},
		{
			name: "failing expectation", dir: func(t *testing.T) string {
				return writePack(t, `{"name":"failpack","cases":[{"name":"c1","signal":`+happySignal+`,"expect":{"action":"block"}}]}`)
			},
			wantExit: 1,
			wantOut:  []string{"FAIL failpack/c1: want action block, got allow", "failpack", "1", "1"},
		},
		{
			name: "corrupt pack aborts", dir: func(t *testing.T) string {
				return writePack(t, "not json")
			},
			wantExit: 65,
			wantErr:  []string{"pack.json"},
		},
		{
			name: "missing dir", dir: func(*testing.T) string { return "./nope" },
			wantExit: 65, wantErr: []string{"no such file"},
		},
		{
			name: "dir with no packs", dir: func(t *testing.T) string { return t.TempDir() },
			wantExit: 65, wantErr: []string{"no scenario packs"},
		},
		{
			name: "no args", wantExit: 64, wantErr: []string{"usage: zt-sim run"},
		},
		{
			name: "two args", dir: func(*testing.T) string { return "scenarios" }, extraArg: true,
			wantExit: 64, wantErr: []string{"usage: zt-sim run"},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"run"}
			if tc.dir != nil {
				args = append(args, tc.dir(t))
			}
			if tc.extraArg {
				args = append(args, "bogus")
			}
			var stdout, stderr bytes.Buffer
			exit := run(args, strings.NewReader(""), &stdout, &stderr)
			out, errOut := stdout.String(), stderr.String()
			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", exit, tc.wantExit, out, errOut)
			}
			for _, want := range tc.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("stdout missing %q\nstdout:\n%s", want, out)
				}
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(errOut, want) {
					t.Errorf("stderr missing %q\nstderr:\n%s", want, errOut)
				}
			}
			if tc.name == "scenarios dir all pass" {
				pass, fail := 0, 0
				for _, l := range strings.Split(out, "\n") {
					switch {
					case strings.HasPrefix(l, "PASS "):
						pass++
					case strings.HasPrefix(l, "FAIL "):
						fail++
					}
				}
				if pass != 6 || fail != 0 {
					t.Errorf("case lines: %d PASS, %d FAIL, want 6 PASS 0 FAIL (summary header cells don't count)\nstdout:\n%s", pass, fail, out)
				}
			}
		})
	}
}

// writePack writes one pack file into a fresh temp dir, returning the dir.
func writePack(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "pack.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestRun_Policies(t *testing.T) {
	t.Run("lists the pack", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exit := run([]string{"policies"}, strings.NewReader(""), &stdout, &stderr)
		out := stdout.String()
		if exit != 0 {
			t.Fatalf("exit = %d, want 0\nstdout:\n%s\nstderr:\n%s", exit, out, stderr.String())
		}
		if got := strings.Fields(strings.Split(out, "\n")[0]); len(got) != 5 || got[0] != "ID" || got[4] != "NOTE" {
			t.Errorf("header fields = %v, want [ID PRIORITY PILLAR ACTION NOTE]", got)
		}
		for _, id := range []string{"P01", "P02", "P03", "P04", "P05", "P06", "P07", "P08", "P09", "P10", "P11"} {
			if !strings.Contains(out, id+" ") {
				t.Errorf("output missing policy %s\noutput:\n%s", id, out)
			}
		}
		for _, want := range []string{"identity", "devices", "data", "applications", "all signals clean — explicit allow"} {
			if !strings.Contains(out, want) {
				t.Errorf("output missing %q\noutput:\n%s", want, out)
			}
		}
	})
	t.Run("extra arg is a usage error", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exit := run([]string{"policies", "x"}, strings.NewReader(""), &stdout, &stderr)
		if exit != 64 {
			t.Errorf("exit = %d, want 64", exit)
		}
		if !strings.Contains(stderr.String(), "usage: zt-sim policies") {
			t.Errorf("stderr = %q, want usage", stderr.String())
		}
	})
}

// TestRun_ExitCodes locks §5's exit-code map end to end. A red here is a
// wiring bug in T5.2–T5.4, not a matrix problem — fix the cause, not the row.
// auditRun writes content to a fixture file, substitutes @file and @audit
// (a fresh temp audit path), runs, and returns exit, stdout, stderr, auditPath.
func auditRun(t *testing.T, content string, args []string, stdin string) (int, string, string, string) {
	t.Helper()
	path := writeFile(t, content)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
	for i, a := range args {
		switch a {
		case "@file":
			args[i] = path
		case "@audit":
			args[i] = auditPath
		}
	}
	var stdout, stderr bytes.Buffer
	exit := run(args, strings.NewReader(stdin), &stdout, &stderr)
	return exit, stdout.String(), stderr.String(), auditPath
}

// records decodes the audit file's lines into records (T6.2 needs no reader —
// plain unmarshal; ReadAudit lands in T6.3).
func records(t *testing.T, path string) []report.AuditRecord {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	var recs []report.AuditRecord
	for _, line := range strings.Split(strings.TrimSuffix(string(b), "\n"), "\n") {
		var r report.AuditRecord
		if err := json.Unmarshal([]byte(line), &r); err != nil {
			t.Fatalf("decode audit line %q: %v", line, err)
		}
		recs = append(recs, r)
	}
	return recs
}

func TestRun_DecideAudit(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		args     []string
		stdin    string
		wantExit int
		wantOut  []string
		wantErr  []string
		notOut   []string // stdout must NOT contain these
		verify   func(t *testing.T, auditPath, out, errOut string)
	}{
		{
			name:    "one record per signal, exit still strictest",
			content: "[" + byodSignal + "," + attackerSignal + "]",
			args:    []string{"decide", "--audit", "@audit", "@file"}, wantExit: 3,
			verify: func(t *testing.T, auditPath, _, _ string) {
				recs := records(t, auditPath)
				if len(recs) != 2 {
					t.Fatalf("want 2 records, got %d", len(recs))
				}
				if recs[0].Cmd != "decide" || recs[0].Action != model.Limited {
					t.Errorf("record 1 = cmd %q action %v, want decide/limited", recs[0].Cmd, recs[0].Action)
				}
				if recs[0].Source == "" || !strings.HasSuffix(recs[0].Source, "in.json") {
					t.Errorf("record 1 source = %q, want the input path as given", recs[0].Source)
				}
				if recs[1].Action != model.Block {
					t.Errorf("record 2 action = %v, want block", recs[1].Action)
				}
			},
		},
		{
			name: "stdin records source stdin", content: "ignored",
			args: []string{"decide", "--stdin", "--audit", "@audit"}, stdin: happySignal, wantExit: 0,
			verify: func(t *testing.T, auditPath, _, _ string) {
				recs := records(t, auditPath)
				if len(recs) != 1 || recs[0].Source != "stdin" || recs[0].Cmd != "decide" {
					t.Fatalf("record = %+v, want cmd decide source stdin", recs)
				}
			},
		},
		{
			name: "footer line per §5", content: "[" + attackerSignal + "]",
			args: []string{"decide", "--audit", "@audit", "@file"}, wantExit: 3,
			verify: func(t *testing.T, auditPath, out, _ string) {
				want := "audit: 7 policies matched, decision appended to " + auditPath
				if !strings.Contains(out, want) {
					t.Errorf("stdout %q does not contain %q", out, want)
				}
			},
		},
		{
			name: "no --audit means no footer and no file", content: "[" + attackerSignal + "]",
			args: []string{"decide", "@file"}, wantExit: 3,
			notOut: []string{"audit: "},
		},
		{
			name: "unwritable audit path exits 65", content: "[" + happySignal + "]",
			args: []string{"decide", "--audit", filepath.Join(t.TempDir(), "no", "x.jsonl"), "@file"}, wantExit: 65,
			verify: func(t *testing.T, _, _, errOut string) {
				if !strings.Contains(errOut, "x.jsonl") {
					t.Errorf("stderr %q must name the audit path", errOut)
				}
			},
		},
		{
			name: "bad input never creates the audit file", content: "not a signals array",
			args: []string{"decide", "--audit", "@audit", "@file"}, wantExit: 65,
			verify: func(t *testing.T, auditPath, _, _ string) {
				if _, err := os.Stat(auditPath); !os.IsNotExist(err) {
					t.Errorf("audit file must not exist after bad input, stat err = %v", err)
				}
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			exit, out, errOut, auditPath := auditRun(t, tc.content, tc.args, tc.stdin)
			if exit != tc.wantExit {
				t.Errorf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", exit, tc.wantExit, out, errOut)
			}
			for _, want := range tc.wantOut {
				if !strings.Contains(out, want) {
					t.Errorf("stdout %q does not contain %q", out, want)
				}
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(errOut, want) {
					t.Errorf("stderr %q does not contain %q", errOut, want)
				}
			}
			for _, unwanted := range tc.notOut {
				if strings.Contains(out, unwanted) {
					t.Errorf("stdout %q must not contain %q", out, unwanted)
				}
			}
			if tc.verify != nil {
				tc.verify(t, auditPath, out, errOut)
			}
		})
	}
}

func TestRun_RunAudit(t *testing.T) {
	twoCases := writePack(t, `{"name":"twopack","cases":[
		{"name":"ok","signal":`+happySignal+`,"expect":{"action":"allow"}},
		{"name":"byod","signal":`+byodSignal+`,"expect":{"action":"limited","scope":"read-only"}}]}`)
	auditPath := filepath.Join(t.TempDir(), "audit.jsonl")

	t.Run("one record per case, source is the pack path, cmd is run", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exit := run([]string{"run", "--audit", auditPath, twoCases}, strings.NewReader(""), &stdout, &stderr)
		if exit != 0 {
			t.Fatalf("exit = %d, want 0\nstderr:\n%s", exit, stderr.String())
		}
		recs := records(t, auditPath)
		if len(recs) != 2 {
			t.Fatalf("want 2 records, got %d", len(recs))
		}
		wantSrc := filepath.Join(twoCases, "pack.json")
		for i, r := range recs {
			if r.Cmd != "run" || r.Source != wantSrc {
				t.Errorf("record %d = cmd %q source %q, want run/%q", i, r.Cmd, r.Source, wantSrc)
			}
		}
	})

	t.Run("footer after summary", func(t *testing.T) {
		p2 := filepath.Join(t.TempDir(), "audit2.jsonl")
		var stdout, stderr bytes.Buffer
		exit := run([]string{"run", "--audit", p2, twoCases}, strings.NewReader(""), &stdout, &stderr)
		if exit != 0 {
			t.Fatalf("exit = %d, want 0", exit)
		}
		wantFooter := "audit: 2 policies matched, decision appended to " + p2
		if !strings.Contains(stdout.String(), wantFooter) {
			t.Errorf("stdout %q does not contain %q", stdout.String(), wantFooter)
		}
		if strings.Index(stdout.String(), "PACK") > strings.Index(stdout.String(), wantFooter) {
			t.Errorf("footer must come after the PACK summary:\n%s", stdout.String())
		}
	})

	t.Run("append failure exits 65", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		bad := filepath.Join(t.TempDir(), "no", "x.jsonl")
		exit := run([]string{"run", "--audit", bad, twoCases}, strings.NewReader(""), &stdout, &stderr)
		if exit != 65 {
			t.Errorf("exit = %d, want 65\nstderr:\n%s", exit, stderr.String())
		}
		if !strings.Contains(stderr.String(), "x.jsonl") {
			t.Errorf("stderr %q must name the audit path", stderr.String())
		}
	})
}

func TestRun_Audit(t *testing.T) {
	t.Run("reprints decisions and counts", func(t *testing.T) {
		auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
		in := writeFile(t, "["+attackerSignal+"]")
		var pre bytes.Buffer
		if exit := run([]string{"decide", "--audit", auditPath, in}, strings.NewReader(""), &pre, &pre); exit != 3 {
			t.Fatalf("decide exit = %d, want 3", exit)
		}
		var stdout, stderr bytes.Buffer
		exit := run([]string{"audit", auditPath}, strings.NewReader(""), &stdout, &stderr)
		if exit != 0 {
			t.Fatalf("audit exit = %d, want 0\nstderr:\n%s", exit, stderr.String())
		}
		for _, want := range []string{"USER", "BLOCK", "P01: ", "pillars: identity", "counts: allow=0 limited=0 stepUpMFA=0 block=1"} {
			if !strings.Contains(stdout.String(), want) {
				t.Errorf("stdout %q does not contain %q", stdout.String(), want)
			}
		}
	})

	t.Run("empty log is valid", func(t *testing.T) {
		empty := filepath.Join(t.TempDir(), "empty.jsonl")
		if err := os.WriteFile(empty, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		var stdout, stderr bytes.Buffer
		exit := run([]string{"audit", empty}, strings.NewReader(""), &stdout, &stderr)
		if exit != 0 {
			t.Fatalf("exit = %d, want 0\nstderr:\n%s", exit, stderr.String())
		}
		if !strings.Contains(stdout.String(), "counts: allow=0 limited=0 stepUpMFA=0 block=0") {
			t.Errorf("stdout %q must contain the zero counts line", stdout.String())
		}
		if strings.Contains(stdout.String(), "pillars:") {
			t.Errorf("stdout %q must not print an empty decisions table", stdout.String())
		}
	})

	t.Run("corrupt line reported, valid still printed", func(t *testing.T) {
		auditPath := filepath.Join(t.TempDir(), "audit.jsonl")
		in := writeFile(t, "["+attackerSignal+"]")
		run([]string{"decide", "--audit", auditPath, in}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
		f, err := os.OpenFile(auditPath, os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.WriteString("garbage\n"); err != nil {
			t.Fatal(err)
		}
		f.Close()

		var stdout, stderr bytes.Buffer
		exit := run([]string{"audit", auditPath}, strings.NewReader(""), &stdout, &stderr)
		if exit != 65 {
			t.Fatalf("exit = %d, want 65", exit)
		}
		if !strings.Contains(stdout.String(), "BLOCK") {
			t.Errorf("valid record must still print: %q", stdout.String())
		}
		if !strings.Contains(stderr.String(), "line 2") {
			t.Errorf("stderr %q must report line 2", stderr.String())
		}
	})

	t.Run("unreadable file exits 65", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		exit := run([]string{"audit", "/nope/x.jsonl"}, strings.NewReader(""), &stdout, &stderr)
		if exit != 65 {
			t.Fatalf("exit = %d, want 65", exit)
		}
		if !strings.Contains(stderr.String(), "no such file") {
			t.Errorf("stderr %q must say no such file", stderr.String())
		}
	})

	t.Run("usage", func(t *testing.T) {
		for _, args := range [][]string{{"audit"}, {"audit", "a", "b"}} {
			var stdout, stderr bytes.Buffer
			exit := run(args, strings.NewReader(""), &stdout, &stderr)
			if exit != 64 {
				t.Errorf("args %v: exit = %d, want 64", args, exit)
			}
			if !strings.Contains(stderr.String(), "usage: zt-sim audit") {
				t.Errorf("args %v: stderr %q must print auditUsage", args, stderr.String())
			}
		}
	})
}

func TestRun_ExitCodes(t *testing.T) {
	happy := writeFile(t, "["+happySignal+"]")
	byod := writeFile(t, "["+byodSignal+"]")
	stolen := writeFile(t, "["+stolenSignal+"]")
	attacker := writeFile(t, "["+attackerSignal+"]")
	failing := writePack(t, `{"name":"failpack","cases":[{"name":"c1","signal":`+happySignal+`,"expect":{"action":"block"}}]}`)
	validLog := filepath.Join(t.TempDir(), "audit.jsonl")
	run([]string{"decide", "--audit", validLog, attacker}, strings.NewReader(""), &bytes.Buffer{}, &bytes.Buffer{})
	corruptLog := filepath.Join(t.TempDir(), "corrupt.jsonl")
	os.WriteFile(corruptLog, []byte("garbage\n"), 0o644)
	tests := []struct {
		name     string
		args     []string
		wantExit int
	}{
		{"decide allow exits 0", []string{"decide", happy}, 0},
		{"decide limited exits 1", []string{"decide", byod}, 1},
		{"decide stepUpMFA exits 2", []string{"decide", stolen}, 2},
		{"decide block exits 3", []string{"decide", attacker}, 3},
		{"decide no input is usage 64", []string{"decide"}, 64},
		{"decide bad format is usage 64", []string{"decide", "--format", "yaml", happy}, 64},
		{"decide pack rejected 65 (§5 rule)", []string{"decide", "scenarios/attacker.json"}, 65},
		{"decide missing file 65", []string{"decide", "/nope/x.json"}, 65},
		{"run all pass exits 0", []string{"run", "scenarios"}, 0},
		{"run failed expectation exits 1", []string{"run", failing}, 1},
		{"run no args is usage 64", []string{"run"}, 64},
		{"run missing dir 65", []string{"run", "./nope"}, 65},
		{"policies exits 0", []string{"policies"}, 0},
		{"policies extra arg is usage 64", []string{"policies", "x"}, 64},
		{"audit valid log exits 0", []string{"audit", validLog}, 0},
		{"audit no args is usage 64", []string{"audit"}, 64},
		{"audit missing file 65", []string{"audit", "/nope/x.jsonl"}, 65},
		{"audit corrupt log 65", []string{"audit", corruptLog}, 65},
		{"version exits 0", []string{"--version"}, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			if exit := run(tc.args, strings.NewReader(""), &stdout, &stderr); exit != tc.wantExit {
				t.Errorf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", exit, tc.wantExit, stdout.String(), stderr.String())
			}
		})
	}
}

// TestExamples_AttackerMatchesPack pins the demo fixture to the pack it
// mirrors — examples/attacker-signals.json exists so the §5 session works
// under the decide-rejects-packs rule, and silent drift would break the
// Phase 7 README transcript.
func TestExamples_AttackerMatchesPack(t *testing.T) {
	f, err := os.Open("examples/attacker-signals.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	signals, err := model.DecodeSignals(f)
	if err != nil {
		t.Fatalf("examples fixture no longer decodes: %v", err)
	}
	pack, err := scenario.Load("scenarios/attacker.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(signals) != 1 || signals[0] != pack.Cases[0].Signal {
		t.Errorf("examples/attacker-signals.json drifted from scenarios/attacker.json")
	}
}
