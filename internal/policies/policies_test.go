package policies_test

import (
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
