package model_test

import (
	"encoding/json"
	"testing"
	"time"

	"zt-sim/internal/model"
)

// goldenSignal is Alice from spec §6; field order = declaration order, so
// json.Marshal output is deterministic and byte-comparable.
const goldenSignal = `{"timestamp":"2026-01-15T03:14:00Z","user":"alice@contoso.com","role":"finance","app":"payroll-app","dataSensitivity":"high","location":"BR","unfamiliarLocation":true,"deviceManaged":false,"deviceCompliant":false,"localHour":3,"mfaPresent":false,"riskScore":65}`

func TestSignal_JSONRoundTrip(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		check func(t *testing.T, got model.Signal)
	}{
		{
			name: "full alice signal",
			json: goldenSignal,
			check: func(t *testing.T, got model.Signal) {
				want := model.Signal{
					Timestamp:       time.Date(2026, 1, 15, 3, 14, 0, 0, time.UTC),
					User:            "alice@contoso.com",
					Role:            "finance",
					App:             "payroll-app",
					DataSensitivity: "high",
					Location:        "BR",
					UnfamiliarLoc:   true,
					LocalHour:       3,
					RiskScore:       65,
				}
				if got != want {
					t.Errorf("unmarshal got %+v, want %+v", got, want)
				}
				remarshal, err := json.Marshal(got)
				if err != nil {
					t.Fatalf("marshal: %v", err)
				}
				if string(remarshal) != goldenSignal {
					t.Errorf("marshal got %s, want %s", remarshal, goldenSignal)
				}
			},
		},
		{
			name: "minimal signal",
			json: `{"user":"a@b.c","role":"eng","app":"wiki","dataSensitivity":"public","timestamp":"2026-01-15T03:14:00Z"}`,
			check: func(t *testing.T, got model.Signal) {
				if got.Location != "" || got.UnfamiliarLoc || got.DeviceManaged ||
					got.DeviceCompliant || got.LocalHour != 0 || got.MFAPresent || got.RiskScore != 0 {
					t.Errorf("absent keys must stay zero-valued, got %+v", got)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got model.Signal
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			tc.check(t, got)
		})
	}
}
