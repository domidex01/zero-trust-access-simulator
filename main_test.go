package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zt-sim/internal/model"
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
		{
			name:       "run is stubbed",
			args:       []string{"run"},
			wantExit:   70,
			wantStderr: []string{"run: not implemented yet"},
		},
		{
			name:       "policies is stubbed",
			args:       []string{"policies"},
			wantExit:   70,
			wantStderr: []string{"policies: not implemented yet"},
		},
		{
			name:       "audit is stubbed",
			args:       []string{"audit"},
			wantExit:   70,
			wantStderr: []string{"audit: not implemented yet"},
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
