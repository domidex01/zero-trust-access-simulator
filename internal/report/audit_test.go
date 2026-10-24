package report_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"zt-sim/internal/model"
	"zt-sim/internal/report"
)

var fixedClock = func() time.Time { return time.Date(2026, 1, 15, 3, 14, 0, 0, time.UTC) }

var (
	auditSignal = model.Signal{Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), User: "attacker@x"}
	auditD      = model.Decision{Action: model.Block,
		Reasons: []string{"P01: no second factor presented — verify explicitly"},
		Pillars: []string{"identity"}, Matched: []string{"P01"}}
)

// wantLine is auditSignal + auditDecision through the writer with the fixed
// clock — §7's field order, generated once and pinned (no re-typing).
const wantLine = `{"ts":"2026-01-15T03:14:00Z","cmd":"decide","source":"attacker.json","signal":{"timestamp":"2026-01-15T10:00:00Z","user":"attacker@x","role":"","app":"","dataSensitivity":"","location":"","unfamiliarLocation":false,"deviceManaged":false,"deviceCompliant":false,"localHour":0,"mfaPresent":false,"riskScore":0},"action":"block","scope":"","reasons":["P01: no second factor presented — verify explicitly"],"pillars":["identity"],"matched":["P01"]}`

func auditPath(t *testing.T) string {
	t.Helper()
	return filepath.Join(t.TempDir(), "audit.jsonl")
}

func readAuditFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	return string(b)
}

func TestReadAudit(t *testing.T) {
	bLine := strings.ReplaceAll(wantLine, `"user":"attacker@x"`, `"user":"b@x"`)
	tests := []struct {
		name        string
		give        string
		wantRecords int
		wantErrs    []string // substrings errs must contain, in order; nil = no errs
	}{
		{"valid log decodes fully", wantLine + "\n" + bLine, 2, nil},
		{"corrupt JSON reported with line number", wantLine + "\n{oops", 1, []string{"line 2"}},
		{"unknown field is corrupt", strings.ReplaceAll(wantLine, `"source":"attacker.json"`, `"source":"attacker.json","foo":1`), 0, []string{"line 1: json: unknown field \"foo\""}},
		{"unknown action text is corrupt", strings.ReplaceAll(wantLine, `"action":"block"`, `"action":"detonate"`), 0, []string{"line 1: unknown action \"detonate\""}},
		{"bad ts is corrupt", strings.ReplaceAll(wantLine, `"ts":"2026-01-15T03:14:00Z"`, `"ts":"yesterday"`), 0, []string{"line 1"}},
		{"blank line is corrupt", wantLine + "\n\n" + bLine, 2, []string{"line 2"}},
		{"truncated last line still seen", wantLine + "\n{" + `"ts":"2026-`, 1, []string{"line 2"}},
		{"empty reader", "", 0, nil},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			recs, errs := report.ReadAudit(strings.NewReader(tc.give))
			if len(recs) != tc.wantRecords {
				t.Errorf("records = %d, want %d (errs: %v)", len(recs), tc.wantRecords, errs)
			}
			if tc.wantErrs == nil {
				if len(errs) != 0 {
					t.Errorf("errs = %v, want none", errs)
				}
				return
			}
			if len(errs) != len(tc.wantErrs) {
				t.Fatalf("errs = %v, want %d", errs, len(tc.wantErrs))
			}
			for i, want := range tc.wantErrs {
				if !strings.Contains(errs[i].Error(), want) {
					t.Errorf("errs[%d] = %q, want %q", i, errs[i], want)
				}
			}
		})
	}
	t.Run("records round-trip all fields", func(t *testing.T) {
		recs, errs := report.ReadAudit(strings.NewReader(wantLine + "\n"))
		if len(errs) != 0 || len(recs) != 1 {
			t.Fatalf("recs = %d, errs = %v", len(recs), errs)
		}
		r := recs[0]
		if r.TS != fixedClock() || r.Cmd != "decide" || r.Source != "attacker.json" {
			t.Errorf("header fields = %v %q %q", r.TS, r.Cmd, r.Source)
		}
		if r.Signal != auditSignal || r.Action != model.Block || len(r.Matched) != 1 || r.Matched[0] != "P01" {
			t.Errorf("record did not round-trip: %+v", r)
		}
	})
}

func TestAuditSummary(t *testing.T) {
	rec := func(a model.Action) report.AuditRecord {
		return report.AuditRecord{Decision: model.Decision{Action: a}}
	}
	tests := []struct {
		name    string
		give    []report.AuditRecord
		wantOut string
	}{
		{"counts in §2 severity order", []report.AuditRecord{rec(model.Allow), rec(model.Limited), rec(model.StepUpMFA), rec(model.Block), rec(model.Block)}, "counts: allow=1 limited=1 stepUpMFA=1 block=2\n"},
		{"empty log", nil, "counts: allow=0 limited=0 stepUpMFA=0 block=0\n"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var b strings.Builder
			if err := report.AuditSummary(&b, tc.give); err != nil {
				t.Fatalf("AuditSummary: %v", err)
			}
			if b.String() != tc.wantOut {
				t.Errorf("out = %q, want %q", b.String(), tc.wantOut)
			}
		})
	}
}

func TestAuditWriter(t *testing.T) {
	t.Run("record is one JSON line in §7 field order", func(t *testing.T) {
		path := auditPath(t)
		w, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("NewAuditWriter: %v", err)
		}
		if err := w.Append("decide", "attacker.json", auditSignal, auditD); err != nil {
			t.Fatalf("Append: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if got := readAuditFile(t, path); got != wantLine+"\n" {
			t.Errorf("file content:\n got %s\nwant %s", got, wantLine+"\n")
		}
	})

	t.Run("ts is the injected clock, not marshal time", func(t *testing.T) {
		path := auditPath(t)
		w, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("NewAuditWriter: %v", err)
		}
		if err := w.Append("decide", "attacker.json", auditSignal, auditD); err != nil {
			t.Fatalf("Append: %v", err)
		}
		w.Close()
		if got := readAuditFile(t, path); !strings.Contains(got, `"ts":"2026-01-15T03:14:00Z"`) {
			t.Errorf("ts not the injected clock: %s", got)
		}
	})

	t.Run("O_CREATE creates the missing file", func(t *testing.T) {
		path := auditPath(t) // dir exists, file does not
		w, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("NewAuditWriter: %v", err)
		}
		w.Close()
		if _, err := os.Stat(path); err != nil {
			t.Errorf("audit file not created: %v", err)
		}
	})

	t.Run("limited scope flattens top-level", func(t *testing.T) {
		path := auditPath(t)
		w, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("NewAuditWriter: %v", err)
		}
		d := model.Decision{Action: model.Limited, Scope: "read-only",
			Reasons: []string{"P04: unmanaged device — least-privilege grant"},
			Pillars: []string{"devices"}, Matched: []string{"P04"}}
		if err := w.Append("run", "scenarios/byod.json", auditSignal, d); err != nil {
			t.Fatalf("Append: %v", err)
		}
		w.Close()
		got := readAuditFile(t, path)
		if !strings.Contains(got, `"action":"limited","scope":"read-only"`) {
			t.Errorf("limited decision not flattened: %s", got)
		}
		if strings.Index(got, `"scope":"read-only"`) > strings.Index(got, `"reasons"`) {
			t.Errorf("scope must precede reasons (§7 field order): %s", got)
		}
	})

	t.Run("reopen appends, never truncates: line 1 byte-identical", func(t *testing.T) {
		path := auditPath(t)
		w1, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("NewAuditWriter: %v", err)
		}
		if err := w1.Append("decide", "attacker.json", auditSignal, auditD); err != nil {
			t.Fatalf("Append: %v", err)
		}
		if err := w1.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		first := readAuditFile(t, path)

		w2, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("reopen: %v", err)
		}
		other := auditSignal
		other.User = "someone-else@x"
		if err := w2.Append("decide", "attacker.json", other, auditD); err != nil {
			t.Fatalf("Append: %v", err)
		}
		if err := w2.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		second := readAuditFile(t, path)

		lines := strings.Split(strings.TrimSuffix(second, "\n"), "\n")
		if len(lines) != 2 {
			t.Fatalf("want 2 lines after reopen, got %d: %q", len(lines), second)
		}
		if lines[0] != strings.TrimSuffix(first, "\n") {
			t.Errorf("line 1 mutated by reopen:\n got %s\nwant %s", lines[0], first)
		}
		if !strings.Contains(lines[1], "someone-else@x") {
			t.Errorf("second append missing: %s", lines[1])
		}
	})

	t.Run("append after Close fails", func(t *testing.T) {
		path := auditPath(t)
		w, err := report.NewAuditWriter(path, fixedClock)
		if err != nil {
			t.Fatalf("NewAuditWriter: %v", err)
		}
		if err := w.Append("decide", "attacker.json", auditSignal, auditD); err != nil {
			t.Fatalf("Append: %v", err)
		}
		if err := w.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
		if err := w.Append("decide", "attacker.json", auditSignal, auditD); err == nil {
			t.Error("Append after Close must error, got nil")
		}
	})
}
