package engine_test

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"zt-sim/internal/engine"
	"zt-sim/internal/model"
)

// base is the alice note-scenario signal (§6), shared by the engine suite.
func base() model.Signal {
	return model.Signal{
		Timestamp:       time.Date(2026, 1, 15, 3, 14, 0, 0, time.UTC),
		User:            "alice@contoso.com",
		Role:            "finance",
		App:             "payroll-app",
		DataSensitivity: "high",
		Location:        "BR",
		UnfamiliarLoc:   true,
		DeviceManaged:   false,
		DeviceCompliant: false,
		LocalHour:       3,
		MFAPresent:      false,
		RiskScore:       65,
	}
}

func always(model.Signal) bool { return true }

func TestEvaluate_Semantics(t *testing.T) {
	tests := []struct {
		name     string
		policies []engine.Policy
		signal   model.Signal
		want     model.Decision
	}{
		{
			name: "single match allows with one reason",
			policies: []engine.Policy{
				{ID: "F99", Priority: 99, Pillar: "identity", When: always, Then: model.Allow, Note: "all signals clean"},
			},
			signal: base(),
			want: model.Decision{
				Action:  model.Allow,
				Reasons: []string{"F99: all signals clean"},
				Pillars: []string{"identity"},
				Matched: []string{"F99"},
			},
		},
		{
			name: "strictest match wins across actions",
			policies: []engine.Policy{
				{ID: "F10", Priority: 10, Pillar: "identity", When: always, Then: model.StepUpMFA, Note: "no second factor"},
				{ID: "F20", Priority: 20, Pillar: "devices", When: always, Then: model.Limited, Scope: "read-only", Note: "unmanaged device"},
				{ID: "F30", Priority: 30, Pillar: "data", When: always, Then: model.Block, Note: "risk too high"},
			},
			signal: base(),
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"F10: no second factor", "F20: unmanaged device", "F30: risk too high"},
				Pillars: []string{"identity", "devices", "data"},
				Matched: []string{"F10", "F20", "F30"},
			},
		},
		{
			name: "reasons follow priority order, not input order",
			policies: []engine.Policy{
				{ID: "F30", Priority: 30, Pillar: "data", When: always, Then: model.Block, Note: "risk too high"},
				{ID: "F10", Priority: 10, Pillar: "identity", When: always, Then: model.StepUpMFA, Note: "no second factor"},
				{ID: "F20", Priority: 20, Pillar: "devices", When: always, Then: model.Limited, Scope: "read-only", Note: "unmanaged device"},
			},
			signal: base(),
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"F10: no second factor", "F20: unmanaged device", "F30: risk too high"},
				Pillars: []string{"identity", "devices", "data"},
				Matched: []string{"F10", "F20", "F30"},
			},
		},
		{
			name: "priority tie breaks by ascending ID",
			policies: []engine.Policy{
				{ID: "FA", Priority: 10, Pillar: "identity", When: always, Then: model.StepUpMFA, Note: "a note"},
				{ID: "FB", Priority: 10, Pillar: "applications", When: always, Then: model.Allow, Note: "b note"},
			},
			signal: base(),
			want: model.Decision{
				Action:  model.StepUpMFA,
				Reasons: []string{"FA: a note", "FB: b note"},
				Pillars: []string{"identity", "applications"},
				Matched: []string{"FA", "FB"},
			},
		},
		{
			name: "scope from first matching limited policy",
			policies: []engine.Policy{
				{ID: "F20", Priority: 20, Pillar: "identity", When: always, Then: model.Limited, Scope: "read-only", Note: "limited one"},
				{ID: "F25", Priority: 25, Pillar: "devices", When: always, Then: model.Limited, Scope: "time-boxed", Note: "limited two"},
				{ID: "F99", Priority: 99, Pillar: "applications", When: always, Then: model.Allow, Note: "clean"},
			},
			signal: base(),
			want: model.Decision{
				Action:  model.Limited,
				Scope:   "read-only",
				Reasons: []string{"F20: limited one", "F25: limited two", "F99: clean"},
				Pillars: []string{"identity", "devices", "applications"},
				Matched: []string{"F20", "F25", "F99"},
			},
		},
		{
			name: "curveball: same pillar listed once",
			policies: []engine.Policy{
				{ID: "F10", Priority: 10, Pillar: "identity", When: func(s model.Signal) bool { return !s.MFAPresent }, Then: model.StepUpMFA, Note: "no mfa"},
				{ID: "F99", Priority: 99, Pillar: "identity", When: always, Then: model.Allow, Note: "clean"},
			},
			signal: base(),
			want: model.Decision{
				Action:  model.StepUpMFA,
				Reasons: []string{"F10: no mfa", "F99: clean"},
				Pillars: []string{"identity"},
				Matched: []string{"F10", "F99"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			e := engine.New(tc.policies)
			got := e.Evaluate(tc.signal)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Evaluate = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// failClosedCases: the three §3.5 paths as named engines with their pinned
// decisions. FailClosed asserts the full want; NoSilentPass re-loops them
// for the property check.
func failClosedCases() []struct {
	name string
	eng  engine.Engine
	want model.Decision
} {
	return []struct {
		name string
		eng  engine.Engine
		want model.Decision
	}{
		{
			name: "no policy matched",
			eng: engine.New([]engine.Policy{
				{ID: "F01", Priority: 10, Pillar: "identity", When: func(s model.Signal) bool { return s.MFAPresent }, Then: model.Allow, Note: "has second factor"},
			}),
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"engine: no policy matched — fail closed (assume breach)"},
			},
		},
		{
			name: "empty pack via New(nil)",
			eng:  engine.New(nil),
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"engine: no policies loaded — fail closed"},
			},
		},
		{
			name: "zero-value engine",
			eng:  engine.Engine{},
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"engine: no policies loaded — fail closed"},
			},
		},
	}
}

func TestEvaluate_FailClosed(t *testing.T) {
	for _, tc := range failClosedCases() {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.eng.Evaluate(base())
			if got.Action != tc.want.Action {
				t.Errorf("Action = %v, want %v", got.Action, tc.want.Action)
			}
			if got.Scope != tc.want.Scope {
				t.Errorf("Scope = %q, want %q", got.Scope, tc.want.Scope)
			}
			if !reflect.DeepEqual(got.Reasons, tc.want.Reasons) {
				t.Errorf("Reasons = %v, want %v", got.Reasons, tc.want.Reasons)
			}
			if len(got.Pillars) != 0 {
				t.Errorf("Pillars = %v, want empty", got.Pillars)
			}
			if len(got.Matched) != 0 {
				t.Errorf("Matched = %v, want empty", got.Matched)
			}
		})
	}
}

// miniPack is a synthetic fixture pack — one policy per §9 scenario shape,
// all four actions reachable, F01+F05 sharing the identity pillar so pillar
// dedupe is exercised. Not the real P01–P11 (that pack is Phase 3).
func miniPack() []engine.Policy {
	return []engine.Policy{
		{ID: "F01", Priority: 10, Pillar: "identity", When: func(s model.Signal) bool { return !s.MFAPresent }, Then: model.StepUpMFA, Note: "no second factor presented"},
		{ID: "F02", Priority: 30, Pillar: "devices", When: func(s model.Signal) bool { return !s.DeviceManaged }, Then: model.Limited, Scope: "read-only", Note: "unmanaged device"},
		{ID: "F03", Priority: 40, Pillar: "data", When: func(s model.Signal) bool { return s.RiskScore > 60 }, Then: model.Block, Note: "risk score above fixture threshold"},
		{ID: "F04", Priority: 70, Pillar: "applications", When: func(s model.Signal) bool { return s.Role != "finance" && s.App == "payroll-app" }, Then: model.Block, Note: "app outside requester role"},
		{ID: "F05", Priority: 99, Pillar: "identity", When: func(s model.Signal) bool { return true }, Then: model.Allow, Note: "all signals clean"},
	}
}

// scenarioShapes is the shared §9 scenario table: miniPack against base()
// deltas, with the full expected Decision. ScenarioShapes pins it; the
// NoSilentPass and Deterministic properties loop over it too.
func scenarioShapes() []struct {
	name   string
	signal model.Signal
	want   model.Decision
} {
	note := base()

	happy := base()
	happy.MFAPresent = true
	happy.UnfamiliarLoc = false
	happy.DeviceManaged = true
	happy.DeviceCompliant = true
	happy.RiskScore = 15
	happy.DataSensitivity = "internal"
	happy.Location = "US"

	stolen := happy
	stolen.MFAPresent = false
	stolen.RiskScore = 45

	byod := base()
	byod.MFAPresent = true
	byod.UnfamiliarLoc = false
	byod.RiskScore = 15
	byod.DataSensitivity = "internal"
	byod.Location = "US"

	insider := happy
	insider.Role = "engineering"
	insider.RiskScore = 10

	return []struct {
		name   string
		signal model.Signal
		want   model.Decision
	}{
		{
			name:   "note scenario",
			signal: note,
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"F01: no second factor presented", "F02: unmanaged device", "F03: risk score above fixture threshold", "F05: all signals clean"},
				Pillars: []string{"identity", "devices", "data"},
				Matched: []string{"F01", "F02", "F03", "F05"},
			},
		},
		{
			name:   "happy",
			signal: happy,
			want: model.Decision{
				Action:  model.Allow,
				Reasons: []string{"F05: all signals clean"},
				Pillars: []string{"identity"},
				Matched: []string{"F05"},
			},
		},
		{
			name:   "stolen credential",
			signal: stolen,
			want: model.Decision{
				Action:  model.StepUpMFA,
				Reasons: []string{"F01: no second factor presented", "F05: all signals clean"},
				Pillars: []string{"identity"},
				Matched: []string{"F01", "F05"},
			},
		},
		{
			name:   "byod",
			signal: byod,
			want: model.Decision{
				Action:  model.Limited,
				Scope:   "read-only",
				Reasons: []string{"F02: unmanaged device", "F05: all signals clean"},
				Pillars: []string{"devices", "identity"},
				Matched: []string{"F02", "F05"},
			},
		},
		{
			name:   "insider",
			signal: insider,
			want: model.Decision{
				Action:  model.Block,
				Reasons: []string{"F04: app outside requester role", "F05: all signals clean"},
				Pillars: []string{"applications", "identity"},
				Matched: []string{"F04", "F05"},
			},
		},
	}
}

func TestEvaluate_ScenarioShapes(t *testing.T) {
	for _, tc := range scenarioShapes() {
		t.Run(tc.name, func(t *testing.T) {
			got := engine.New(miniPack()).Evaluate(tc.signal)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("Evaluate = %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestEvaluate_StrictestWins(t *testing.T) {
	pillars := []string{"identity", "devices", "applications", "data"}
	tests := []struct {
		name    string
		actions []model.Action
		want    model.Action
		wantScp string
	}{
		{name: "allow alone", actions: []model.Action{model.Allow}, want: model.Allow, wantScp: ""},
		{name: "limited beats allow", actions: []model.Action{model.Allow, model.Limited}, want: model.Limited, wantScp: "read-only"},
		{name: "step-up beats allow and limited", actions: []model.Action{model.Allow, model.Limited, model.StepUpMFA}, want: model.StepUpMFA, wantScp: ""},
		{name: "block beats everything", actions: []model.Action{model.Allow, model.Limited, model.StepUpMFA, model.Block}, want: model.Block, wantScp: ""},
		{name: "step-up beats limited", actions: []model.Action{model.Limited, model.StepUpMFA}, want: model.StepUpMFA, wantScp: ""},
		{name: "block beats step-up", actions: []model.Action{model.StepUpMFA, model.Block}, want: model.Block, wantScp: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pack := make([]engine.Policy, len(tc.actions))
			for i, a := range tc.actions {
				pack[i] = engine.Policy{
					ID:       fmt.Sprintf("S%02d", i+1),
					Priority: 10 * (i + 1),
					Pillar:   pillars[i],
					When:     always,
					Then:     a,
					Scope:    "read-only",
					Note:     "always true",
				}
			}
			got := engine.New(pack).Evaluate(base())
			if got.Action != tc.want {
				t.Errorf("Action = %v, want %v", got.Action, tc.want)
			}
			if got.Scope != tc.wantScp {
				t.Errorf("Scope = %q, want %q", got.Scope, tc.wantScp)
			}
		})
	}
}

func TestEvaluate_NoSilentPass(t *testing.T) {
	for _, tc := range scenarioShapes() {
		t.Run(tc.name, func(t *testing.T) {
			d := engine.New(miniPack()).Evaluate(tc.signal)
			if len(d.Reasons) < 1 {
				t.Errorf("decision has no reasons: %+v", d)
			}
			if len(d.Matched) > 0 && len(d.Reasons) != len(d.Matched) {
				t.Errorf("Reasons (%d) != Matched (%d): %+v", len(d.Reasons), len(d.Matched), d)
			}
			if d.Action == model.Allow && len(d.Matched) < 1 {
				t.Errorf("Allow without a matched policy: %+v", d)
			}
		})
	}

	for _, tc := range failClosedCases() {
		t.Run(tc.name, func(t *testing.T) {
			d := tc.eng.Evaluate(base())
			if len(d.Reasons) != 1 || !strings.HasPrefix(d.Reasons[0], "engine:") {
				t.Errorf("fail-closed decision must carry exactly one engine: reason, got %+v", d)
			}
		})
	}
}

func TestEvaluate_Deterministic(t *testing.T) {
	for _, tc := range scenarioShapes() {
		t.Run(tc.name, func(t *testing.T) {
			e := engine.New(miniPack())
			d1, d2 := e.Evaluate(tc.signal), e.Evaluate(tc.signal)
			if !reflect.DeepEqual(d1, d2) {
				t.Errorf("double eval differs: %+v vs %+v", d1, d2)
			}

			reversed := miniPack()
			slices.Reverse(reversed)
			dRev := engine.New(reversed).Evaluate(tc.signal)
			if !reflect.DeepEqual(d1, dRev) {
				t.Errorf("construction order changed the decision: %+v vs %+v", d1, dRev)
			}
		})
	}
}
