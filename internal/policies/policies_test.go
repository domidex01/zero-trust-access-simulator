package policies_test

import (
	"slices"
	"testing"
	"time"

	"zt-sim/internal/engine"
	"zt-sim/internal/model"
	"zt-sim/internal/policies"
)

// clean is the §6 happy-path signal: no P01–P10 matcher fires on it.
func clean() model.Signal {
	return model.Signal{
		Timestamp:       time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC),
		User:            "alice@contoso.com",
		Role:            "finance",
		App:             "intranet-wiki",
		DataSensitivity: "internal",
		Location:        "US",
		UnfamiliarLoc:   false,
		DeviceManaged:   true,
		DeviceCompliant: true,
		LocalHour:       10,
		MFAPresent:      true,
		RiskScore:       15,
	}
}

func policyByID(t *testing.T, id string) engine.Policy {
	t.Helper()
	for _, p := range policies.Pack() {
		if p.ID == id {
			return p
		}
	}
	t.Fatalf("policy %s not in pack", id)
	return engine.Policy{}
}

// triggers maps each core policy to a mutation of clean() that fires it.
// Mutations set both device fields together so the validation invariant
// !DeviceManaged ⇒ !DeviceCompliant always holds.
var triggers = []struct {
	id     string
	mutate func(model.Signal) model.Signal
}{
	{"P01", func(s model.Signal) model.Signal { s.MFAPresent = false; return s }},
	{"P02", func(s model.Signal) model.Signal { s.UnfamiliarLoc = true; return s }},
	{"P03", func(s model.Signal) model.Signal { s.UnfamiliarLoc, s.MFAPresent = true, false; return s }},
	{"P04", func(s model.Signal) model.Signal { s.DeviceManaged, s.DeviceCompliant = false, false; return s }},
	{"P05", func(s model.Signal) model.Signal { s.DeviceManaged, s.DeviceCompliant = true, false; return s }},
	{"P06", func(s model.Signal) model.Signal {
		s.DeviceCompliant = false
		s.DataSensitivity = "confidential"
		return s
	}},
	{"P07", func(s model.Signal) model.Signal { s.RiskScore = 71; return s }},
	{"P08", func(s model.Signal) model.Signal { s.RiskScore = 70; return s }},
	{"P09", func(s model.Signal) model.Signal { s.Role, s.LocalHour = "intern", 4; return s }},
	{"P10", func(s model.Signal) model.Signal { s.Role, s.App = "engineering", "payroll-app"; return s }},
}

func TestPack_CatchAllExclusive(t *testing.T) {
	t.Run("clean signal allows", func(t *testing.T) {
		if !policyByID(t, "P11").When(clean()) {
			t.Error("P11.When(clean) = false, want true")
		}
	})
	for _, tc := range triggers {
		t.Run(tc.id+"'s trigger blocks the catch-all", func(t *testing.T) {
			if policyByID(t, "P11").When(tc.mutate(clean())) {
				t.Errorf("P11.When(%s trigger) = true, want false", tc.id)
			}
		})
	}
}

func TestPack_FireSilence(t *testing.T) {
	for _, tc := range triggers {
		t.Run(tc.id+" fires on trigger, silent on clean", func(t *testing.T) {
			p := policyByID(t, tc.id)
			if !p.When(tc.mutate(clean())) {
				t.Errorf("%s.When(trigger) = false, want true", tc.id)
			}
			if p.When(clean()) {
				t.Errorf("%s.When(clean) = true, want false", tc.id)
			}
		})
	}
}

func TestPack_Boundaries(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(model.Signal) model.Signal
		checks map[string]bool
	}{
		{"risk 70 still elevated", func(s model.Signal) model.Signal { s.RiskScore = 70; return s },
			map[string]bool{"P07": false, "P08": true}},
		{"risk 71 is hard threshold", func(s model.Signal) model.Signal { s.RiskScore = 71; return s },
			map[string]bool{"P07": true, "P08": false}},
		{"risk 40 enters elevated", func(s model.Signal) model.Signal { s.RiskScore = 40; return s },
			map[string]bool{"P07": false, "P08": true}},
		{"risk 39 below elevated", func(s model.Signal) model.Signal { s.RiskScore = 39; return s },
			map[string]bool{"P07": false, "P08": false}},
		{"intern at 04:59 fires", func(s model.Signal) model.Signal { s.Role, s.LocalHour = "intern", 4; return s },
			map[string]bool{"P09": true}},
		{"intern at 05:00 silent (half-open)", func(s model.Signal) model.Signal { s.Role, s.LocalHour = "intern", 5; return s },
			map[string]bool{"P09": false}},
		{"intern at 00:00 fires", func(s model.Signal) model.Signal { s.Role, s.LocalHour = "intern", 0; return s },
			map[string]bool{"P09": true}},
		{"finance at 03:00 silent (role guard)", func(s model.Signal) model.Signal { s.LocalHour = 3; return s },
			map[string]bool{"P09": false}},
		{"confidential on non-compliant", func(s model.Signal) model.Signal {
			s.DataSensitivity, s.DeviceCompliant = "confidential", false
			return s
		}, map[string]bool{"P06": true}},
		{"high on non-compliant", func(s model.Signal) model.Signal {
			s.DataSensitivity, s.DeviceCompliant = "high", false
			return s
		}, map[string]bool{"P06": true}},
		{"internal on non-compliant", func(s model.Signal) model.Signal { s.DeviceCompliant = false; return s },
			map[string]bool{"P06": false}},
		{"confidential on compliant (device guard)", func(s model.Signal) model.Signal { s.DataSensitivity = "confidential"; return s },
			map[string]bool{"P06": false}},
		{"unmanaged device", func(s model.Signal) model.Signal { s.DeviceManaged, s.DeviceCompliant = false, false; return s },
			map[string]bool{"P04": true, "P05": false}},
		{"enrolled not compliant", func(s model.Signal) model.Signal { s.DeviceCompliant = false; return s },
			map[string]bool{"P04": false, "P05": true}},
		{"fully healthy device", func(s model.Signal) model.Signal { return s },
			map[string]bool{"P04": false, "P05": false}},
		{"unfamiliar with MFA (P02 not dead weight)", func(s model.Signal) model.Signal { s.UnfamiliarLoc = true; return s },
			map[string]bool{"P02": true, "P03": false}},
		{"familiar without MFA (P03 && guard)", func(s model.Signal) model.Signal { s.MFAPresent = false; return s },
			map[string]bool{"P03": false}},
		{"finance on payroll-app (role guard)", func(s model.Signal) model.Signal { s.App = "payroll-app"; return s },
			map[string]bool{"P10": false}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sig := tc.mutate(clean())
			for id, want := range tc.checks {
				if got := policyByID(t, id).When(sig); got != want {
					t.Errorf("%s.When(%s) = %t, want %t", id, tc.name, got, want)
				}
			}
		})
	}
}

func TestPack_ScenarioShapes(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(model.Signal) model.Signal
		action  model.Action
		scope   string
		matched []string
		pillars []string
	}{
		{"happy allows via P11 alone", func(s model.Signal) model.Signal { return s },
			model.Allow, "", []string{"P11"}, []string{"identity"}},
		{"noteScenario blocks — flagship 03:14", func(s model.Signal) model.Signal {
			s.App, s.DataSensitivity, s.Location = "payroll-app", "high", "BR"
			s.UnfamiliarLoc, s.MFAPresent = true, false
			s.DeviceManaged, s.DeviceCompliant = false, false
			s.LocalHour, s.RiskScore = 3, 65
			return s
		}, model.Block, "", []string{"P01", "P02", "P03", "P04", "P06", "P08"},
			[]string{"identity", "devices", "data"}},
		{"stolenCred steps up MFA", func(s model.Signal) model.Signal {
			s.MFAPresent, s.DeviceManaged, s.DeviceCompliant, s.RiskScore = false, false, false, 45
			return s
		}, model.StepUpMFA, "", []string{"P01", "P04", "P08"},
			[]string{"identity", "devices"}},
		{"insider blocks on least privilege", func(s model.Signal) model.Signal {
			s.User, s.Role, s.App = "bob@contoso.com", "engineering", "payroll-app"
			return s
		}, model.Block, "", []string{"P10"}, []string{"applications"}},
		{"byod limited to read-only", func(s model.Signal) model.Signal {
			s.DeviceManaged, s.DeviceCompliant = false, false
			return s
		}, model.Limited, "read-only", []string{"P04"}, []string{"devices"}},
		{"attacker blocks on everything at once", func(s model.Signal) model.Signal {
			s.User, s.Role, s.App = "unknown@throwaway.example", "guest", "payroll-app"
			s.DataSensitivity, s.Location = "high", "NG"
			s.UnfamiliarLoc, s.MFAPresent = true, false
			s.DeviceManaged, s.DeviceCompliant = false, false
			s.LocalHour, s.RiskScore = 2, 85
			return s
		}, model.Block, "", []string{"P01", "P02", "P03", "P04", "P06", "P07", "P10"},
			[]string{"identity", "devices", "data", "applications"}},
	}
	e := engine.New(policies.Pack())
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			d := e.Evaluate(tc.mutate(clean()))
			if d.Action != tc.action {
				t.Errorf("Action = %s, want %s", d.Action, tc.action)
			}
			if d.Scope != tc.scope {
				t.Errorf("Scope = %q, want %q", d.Scope, tc.scope)
			}
			if !slices.Equal(d.Matched, tc.matched) {
				t.Errorf("Matched = %v, want %v", d.Matched, tc.matched)
			}
			if !slices.Equal(d.Pillars, tc.pillars) {
				t.Errorf("Pillars = %v, want %v", d.Pillars, tc.pillars)
			}
		})
	}
}

var validPillars = []string{"identity", "devices", "applications", "data", "networks", "infrastructure"}

func TestPack_Metadata(t *testing.T) {
	// Pinned §4 table, positional; Notes verbatim (em dashes included).
	// When is tested behaviorally in the other tests — Policy has a func
	// field, so it isn't ==-comparable anyway.
	want := []struct {
		id       string
		priority int
		pillar   string
		then     model.Action
		scope    string
		note     string
	}{
		{"P01", 10, "identity", model.StepUpMFA, "", "no second factor presented — verify explicitly"},
		{"P02", 20, "identity", model.StepUpMFA, "", "unfamiliar sign-in location — re-verify"},
		{"P03", 20, "identity", model.Block, "", "unfamiliar location and no MFA — assume breach"},
		{"P04", 30, "devices", model.Limited, "read-only", "unmanaged device — least-privilege grant"},
		{"P05", 30, "devices", model.Limited, "read-only", "enrolled but non-compliant device — least-privilege grant"},
		{"P06", 40, "data", model.Block, "", "confidential/high data on non-compliant device"},
		{"P07", 50, "identity", model.Block, "", "risk score above hard threshold"},
		{"P08", 60, "identity", model.StepUpMFA, "", "elevated risk — verify explicitly"},
		{"P09", 70, "identity", model.StepUpMFA, "", "odd-hours intern activity"},
		{"P10", 80, "applications", model.Block, "", "app outside requester's role — least privilege"},
		{"P11", 90, "identity", model.Allow, "", "all signals clean — explicit allow"},
	}
	pack := policies.Pack()
	if len(pack) != len(want) {
		t.Fatalf("len(Pack()) = %d, want %d", len(pack), len(want))
	}
	seen := map[string]bool{}
	allowed := 0
	for i, w := range want {
		p := pack[i]
		if p.ID != w.id {
			t.Errorf("pack[%d].ID = %q, want %q", i, p.ID, w.id)
		}
		if p.Priority != w.priority {
			t.Errorf("%s.Priority = %d, want %d", w.id, p.Priority, w.priority)
		}
		if p.Pillar != w.pillar {
			t.Errorf("%s.Pillar = %q, want %q", w.id, p.Pillar, w.pillar)
		}
		if p.Then != w.then {
			t.Errorf("%s.Then = %s, want %s", w.id, p.Then, w.then)
		}
		if p.Scope != w.scope {
			t.Errorf("%s.Scope = %q, want %q", w.id, p.Scope, w.scope)
		}
		if p.Note != w.note {
			t.Errorf("%s.Note = %q, want %q", w.id, p.Note, w.note)
		}
		if seen[p.ID] {
			t.Errorf("duplicate policy ID %q", p.ID)
		}
		seen[p.ID] = true
		if p.Then == model.Allow {
			allowed++
			if p.ID != "P11" {
				t.Errorf("Allow policy %q is not the catch-all P11", p.ID)
			}
		}
		wantScope := ""
		if p.Then == model.Limited {
			wantScope = "read-only"
		}
		if p.Scope != wantScope {
			t.Errorf("%s.Scope = %q, want %q for Then=%s", w.id, p.Scope, wantScope, p.Then)
		}
		if !slices.Contains(validPillars, p.Pillar) {
			t.Errorf("%s.Pillar %q not a valid pillar", w.id, p.Pillar)
		}
	}
	if allowed != 1 {
		t.Errorf("pack has %d Allow policies, want exactly 1", allowed)
	}
}
