package engine_test

import (
	"reflect"
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
