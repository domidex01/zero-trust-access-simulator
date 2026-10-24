package scenario_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"zt-sim/internal/engine"
	"zt-sim/internal/model"
	"zt-sim/internal/policies"
	"zt-sim/internal/scenario"
)

// signalJSON is the happy-path signal as JSON — same values as the policies
// suite's clean() fixture.
const signalJSON = `{"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance","app":"intranet-wiki","dataSensitivity":"internal","location":"US","unfamiliarLocation":false,"deviceManaged":true,"deviceCompliant":true,"localHour":10,"mfaPresent":true,"riskScore":15}`

const baseExpect = `{"action":"block","reasonsContain":["P01","P04"]}`

func writePack(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "pack.json")
	if err := os.WriteFile(path, []byte(content), 0o666); err != nil {
		t.Fatal(err)
	}
	return path
}

func caseJSON(name, signal, expect string) string {
	return `{"name":` + quote(name) + `,"signal":` + signal + `,"expect":` + expect + `}`
}

func quote(s string) string {
	return `"` + s + `"`
}

func basePack(caseBody string) string {
	return `{"name":"note","description":"test pack","cases":[` + caseBody + `]}`
}

func TestLoad_Pack(t *testing.T) {
	p, err := scenario.Load(writePack(t, basePack(caseJSON("c1", signalJSON, baseExpect))))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if p.Name != "note" {
		t.Errorf("Name = %q, want %q", p.Name, "note")
	}
	if p.Description != "test pack" {
		t.Errorf("Description = %q, want %q", p.Description, "test pack")
	}
	if len(p.Cases) != 1 {
		t.Fatalf("len(Cases) = %d, want 1", len(p.Cases))
	}
	c := p.Cases[0]
	if c.Name != "c1" {
		t.Errorf("case Name = %q, want %q", c.Name, "c1")
	}
	if c.Signal.User != "alice@contoso.com" || c.Signal.RiskScore != 15 || !c.Signal.MFAPresent {
		t.Errorf("signal decoded wrong: %+v", c.Signal)
	}
	if c.Signal.DataSensitivity != "internal" {
		t.Errorf("DataSensitivity = %q, want internal", c.Signal.DataSensitivity)
	}
	if c.Expect.Action == nil || *c.Expect.Action != model.Block {
		t.Errorf("Expect.Action = %v, want block", c.Expect.Action)
	}
	if len(c.Expect.ReasonsContain) != 2 || c.Expect.ReasonsContain[0] != "P01" || c.Expect.ReasonsContain[1] != "P04" {
		t.Errorf("ReasonsContain = %v, want [P01 P04]", c.Expect.ReasonsContain)
	}
}

func TestLoad_SixPacks(t *testing.T) {
	for _, name := range []string{"happy", "note-scenario", "stolen-credential", "insider", "byod", "attacker"} {
		t.Run(name, func(t *testing.T) {
			p, err := scenario.Load(filepath.Join("..", "..", "scenarios", name+".json"))
			if err != nil {
				t.Fatalf("load: %v", err)
			}
			if p.Name != name {
				t.Errorf("Name = %q, want %q", p.Name, name)
			}
			if len(p.Cases) == 0 {
				t.Error("no cases")
			}
		})
	}
}

func TestLoad_Invalid(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "unknown top-level field",
			json: `{"name":"t","cases":[],"extra":1}`,
			want: `unknown field "extra"`,
		},
		{
			name: "unknown case field",
			json: basePack(`{"name":"c1","signal":` + signalJSON + `,"expect":` + baseExpect + `,"expct":{}}`),
			want: `unknown field "expct"`,
		},
		{
			name: "unknown signal field",
			json: basePack(caseJSON("c1", strings.Replace(signalJSON, `"riskScore":15}`, `"riskScore":15,"risk":9}`, 1), baseExpect)),
			want: `unknown field "risk"`,
		},
		{
			name: "missing pack name",
			json: `{"description":"d","cases":[]}`,
			want: `name must not be empty`,
		},
		{
			name: "empty cases",
			json: `{"name":"t","cases":[]}`,
			want: `cases must not be empty`,
		},
		{
			name: "case name empty",
			json: basePack(caseJSON("", signalJSON, baseExpect)),
			want: `case 0`,
		},
		{
			name: "signal fails validation",
			json: basePack(caseJSON("c1", strings.Replace(signalJSON, `"alice@contoso.com"`, `""`, 1), baseExpect)),
			want: `user must not be empty`,
		},
		{
			name: "expect.action missing",
			json: basePack(caseJSON("c1", signalJSON, `{}`)),
			want: `expect.action required`,
		},
		{
			name: "expect null",
			json: basePack(caseJSON("c1", signalJSON, `null`)),
			want: `expect.action required`,
		},
		{
			name: "expect.action not an enum",
			json: basePack(caseJSON("c1", signalJSON, `{"action":"maybe"}`)),
			want: `unknown action "maybe"`,
		},
		{
			name: "scope on non-limited",
			json: basePack(caseJSON("c1", signalJSON, `{"action":"block","scope":"read-only"}`)),
			want: `expect.scope set but expect.action is block`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := scenario.Load(writePack(t, tc.json))
			if err == nil {
				t.Fatalf("Load succeeded, want error containing %q", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Load error = %q, want it to contain %q", err, tc.want)
			}
		})
	}
	t.Run("file missing", func(t *testing.T) {
		_, err := scenario.Load(filepath.Join(t.TempDir(), "nope.json"))
		if err == nil || !strings.Contains(err.Error(), "no such file") {
			t.Fatalf("Load error = %v, want no such file", err)
		}
	})
}

func loadPacks(t *testing.T) []scenario.Pack {
	t.Helper()
	files, err := filepath.Glob(filepath.Join("..", "..", "scenarios", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 6 {
		t.Fatalf("glob found %d packs, want ≥ 6", len(files))
	}
	have := make(map[string]bool, len(files))
	for _, f := range files {
		have[filepath.Base(f)] = true
	}
	for _, want := range []string{"happy.json", "note-scenario.json", "stolen-credential.json", "insider.json", "byod.json", "attacker.json"} {
		if !have[want] {
			t.Fatalf("missing scenarios/%s", want)
		}
	}
	packs := make([]scenario.Pack, 0, len(files))
	for _, f := range files {
		p, err := scenario.Load(f)
		if err != nil {
			t.Fatalf("load %s: %v", f, err)
		}
		packs = append(packs, p)
	}
	return packs
}

func TestPacks_Expectations(t *testing.T) {
	eng := engine.New(policies.Pack())
	for _, p := range loadPacks(t) {
		for _, c := range p.Cases {
			t.Run(p.Name+"/"+c.Name, func(t *testing.T) {
				d := eng.Evaluate(c.Signal)
				if err := c.Check(d); err != nil {
					t.Fatalf("check: %v\ngot decision: %+v", err, d)
				}
			})
		}
	}
}

func TestPacks_ActionCoverage(t *testing.T) {
	eng := engine.New(policies.Pack())
	seen := make(map[model.Action]bool)
	for _, p := range loadPacks(t) {
		for _, c := range p.Cases {
			seen[eng.Evaluate(c.Signal).Action] = true
		}
	}
	for _, a := range []model.Action{model.Allow, model.Limited, model.StepUpMFA, model.Block} {
		if !seen[a] {
			t.Errorf("action %s not exercised by any scenario", a)
		}
	}
}

func TestCase_Check(t *testing.T) {
	block := model.Action(model.Block)
	allow := model.Action(model.Allow)
	limited := model.Action(model.Limited)

	blockD := model.Decision{
		Action:  model.Block,
		Reasons: []string{"P01: …", "P04: …"},
		Matched: []string{"P01", "P04"},
	}
	limitedD := model.Decision{Action: model.Limited, Scope: "read-only", Matched: []string{"P04"}}
	bareD := model.Decision{Action: model.Limited, Scope: "", Matched: []string{"P04"}}

	tests := []struct {
		name   string
		expect scenario.Expectation
		give   model.Decision
		want   string // "" = want nil error
	}{
		{
			name:   "exact match",
			expect: scenario.Expectation{Action: &block, ReasonsContain: []string{"P01"}},
			give:   blockD,
		},
		{
			name:   "action mismatch",
			expect: scenario.Expectation{Action: &allow},
			give:   blockD,
			want:   "want action allow, got block",
		},
		{
			name:   "missing policy in reasons",
			expect: scenario.Expectation{Action: &block, ReasonsContain: []string{"P01", "P07"}},
			give:   blockD,
			want:   "reasons missing P07",
		},
		{
			name:   "scope matches",
			expect: scenario.Expectation{Action: &limited, Scope: "read-only"},
			give:   limitedD,
		},
		{
			name:   "scope mismatch",
			expect: scenario.Expectation{Action: &limited, Scope: "read-only"},
			give:   bareD,
			want:   `want scope "read-only", got ""`,
		},
		{
			name:   "scope absent not checked",
			expect: scenario.Expectation{Action: &limited},
			give:   bareD,
		},
		{
			name:   "empty reasonsContain",
			expect: scenario.Expectation{Action: &block},
			give:   blockD,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := scenario.Case{Name: "c", Expect: tc.expect}
			err := c.Check(tc.give)
			if tc.want == "" {
				if err != nil {
					t.Fatalf("Check = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Check error = %v, want it to contain %q", err, tc.want)
			}
		})
	}
}
