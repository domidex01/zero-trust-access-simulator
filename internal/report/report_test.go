package report_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	"zt-sim/internal/model"
	"zt-sim/internal/report"
)

// Hand-built fixtures — report is a renderer, so decisions come as literals,
// not engine output (the TestPack_ScenarioShapes pins mirrored as values).

var attackerD = model.Decision{Action: model.Block,
	Reasons: []string{
		"P01: no second factor presented — verify explicitly",
		"P03: unfamiliar location and no MFA — assume breach",
		"P07: risk score above hard threshold",
	},
	Pillars: []string{"identity", "devices", "data", "applications"},
	Matched: []string{"P01", "P03", "P07"},
}

var happyD = model.Decision{Action: model.Allow,
	Reasons: []string{"P11: all signals clean — explicit allow"},
	Pillars: []string{"identity"}, Matched: []string{"P11"},
}

var byodD = model.Decision{Action: model.Limited, Scope: "read-only",
	Reasons: []string{"P04: unmanaged device — least-privilege grant"},
	Pillars: []string{"devices"}, Matched: []string{"P04"},
}

func TestReport_DecisionsTable(t *testing.T) {
	tests := []struct {
		name   string
		rows   []report.Row
		want   []string // substrings the output must contain
		bad    []string // substrings the output must NOT contain
		lines  []string // exact trimmed lines, in order (optional spot rows)
		nLines int      // expected line count (excludes the trailing newline split artifact)
	}{
		{
			name: "attacker row",
			rows: []report.Row{{Signal: model.Signal{User: "unknown@throwaway.example"}, Decision: attackerD}},
			want: []string{
				"unknown@throwaway.example", "BLOCK",
				"P01: no second factor presented — verify explicitly",
				"P03: unfamiliar location and no MFA — assume breach",
				"P07: risk score above hard threshold",
				"pillars: identity, devices, data, applications",
			},
			lines:  []string{"", "", "", "", "pillars: identity, devices, data, applications"},
			nLines: 5,
		},
		{
			name:   "allow row",
			rows:   []report.Row{{Signal: model.Signal{User: "alice@contoso.com"}, Decision: happyD}},
			want:   []string{"alice@contoso.com", "ALLOW", "P11: all signals clean — explicit allow", "pillars: identity"},
			nLines: 3,
		},
		{
			name: "two rows keep input order and union pillars once",
			rows: []report.Row{
				{Signal: model.Signal{User: "alice@contoso.com"}, Decision: happyD},
				{Signal: model.Signal{User: "unknown@throwaway.example"}, Decision: attackerD},
			},
			want:   []string{"pillars: identity, devices, data, applications"},
			nLines: 6,
		},
		{
			name: "reasons stack one per line, not joined",
			rows: []report.Row{{Signal: model.Signal{User: "unknown@throwaway.example"}, Decision: attackerD}},
			lines: []string{
				"",
				"",
				"P03: unfamiliar location and no MFA — assume breach",
				"P07: risk score above hard threshold",
				"",
			},
			nLines: 5,
		},
		{
			name:   "action text is uppercased and scope stays out of the table",
			rows:   []report.Row{{Signal: model.Signal{User: "alice@contoso.com"}, Decision: byodD}},
			want:   []string{"LIMITED"},
			bad:    []string{"read-only", "limited"},
			nLines: 3,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := report.Decisions(&buf, tc.rows); err != nil {
				t.Fatalf("Decisions: %v", err)
			}
			out := buf.String()
			for _, want := range tc.want {
				if !strings.Contains(out, want) {
					t.Errorf("output missing %q\noutput:\n%s", want, out)
				}
			}
			for _, bad := range tc.bad {
				if strings.Contains(out, bad) {
					t.Errorf("output must not contain %q\noutput:\n%s", bad, out)
				}
			}
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			if len(lines) != tc.nLines {
				t.Errorf("line count = %d, want %d\noutput:\n%s", len(lines), tc.nLines, out)
			}
			if got := strings.Fields(lines[0]); len(got) != 3 || got[0] != "USER" || got[1] != "ACTION" || got[2] != "REASONS" {
				t.Errorf("header fields = %v, want [USER ACTION REASONS]", got)
			}
			for i, want := range tc.lines {
				if want == "" {
					continue
				}
				if strings.TrimSpace(lines[i]) != want {
					t.Errorf("line %d = %q, want %q", i, strings.TrimSpace(lines[i]), want)
				}
			}
			if tc.name == "two rows keep input order and union pillars once" {
				alice, unknown := -1, -1
				for i, l := range lines {
					if strings.Contains(l, "alice@contoso.com") && alice == -1 {
						alice = i
					}
					if strings.Contains(l, "unknown@throwaway.example") && unknown == -1 {
						unknown = i
					}
				}
				if alice == -1 || unknown == -1 || alice > unknown {
					t.Errorf("alice line (%d) must precede unknown line (%d)", alice, unknown)
				}
			}
		})
	}
}

func TestReport_DecisionsJSON(t *testing.T) {
	tests := []struct {
		name      string
		decisions []model.Decision
		text      string // substring the raw output must contain
	}{
		{name: "single decision round-trips", decisions: []model.Decision{attackerD}, text: `"action": "block"`},
		{name: "limited keeps scope", decisions: []model.Decision{byodD}, text: `"scope": "read-only"`},
		{name: "two decisions in order", decisions: []model.Decision{happyD, byodD}, text: `"action": "allow"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := report.DecisionsJSON(&buf, tc.decisions); err != nil {
				t.Fatalf("DecisionsJSON: %v", err)
			}
			if !strings.Contains(buf.String(), tc.text) {
				t.Errorf("output missing %q\noutput:\n%s", tc.text, buf.String())
			}
			var got []model.Decision
			if err := json.Unmarshal(buf.Bytes(), &got); err != nil {
				t.Fatalf("round-trip unmarshal: %v\noutput:\n%s", err, buf.String())
			}
			if !reflect.DeepEqual(got, tc.decisions) {
				t.Errorf("round-trip = %+v, want %+v", got, tc.decisions)
			}
		})
	}
}

func TestReport_PackSummary(t *testing.T) {
	results := []report.PackResult{
		{Name: "happy", Cases: 1, Pass: 1, Fail: 0, Strictest: model.Allow},
		{Name: "attacker", Cases: 1, Pass: 1, Fail: 0, Strictest: model.Block},
		{Name: "mixed", Cases: 2, Pass: 1, Fail: 1, Strictest: model.Limited},
	}
	var buf bytes.Buffer
	if err := report.PackSummary(&buf, results); err != nil {
		t.Fatalf("PackSummary: %v", err)
	}
	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("line count = %d, want 4 (header + 3 rows)\noutput:\n%s", len(lines), buf.String())
	}
	if got := strings.Fields(lines[0]); len(got) != 5 || got[0] != "PACK" || got[1] != "CASES" || got[2] != "PASS" || got[3] != "FAIL" || got[4] != "STRICTEST" {
		t.Errorf("header fields = %v, want [PACK CASES PASS FAIL STRICTEST]", got)
	}
	wantRows := [][]string{
		{"happy", "1", "1", "0", "ALLOW"},
		{"attacker", "1", "1", "0", "BLOCK"},
		{"mixed", "2", "1", "1", "LIMITED"},
	}
	for i, want := range wantRows {
		if got := strings.Fields(lines[i+1]); !reflect.DeepEqual(got, want) {
			t.Errorf("row %d = %v, want %v", i+1, got, want)
		}
	}
}
