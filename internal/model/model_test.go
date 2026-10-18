package model_test

import (
	"encoding/json"
	"slices"
	"strings"
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

func TestAction_String(t *testing.T) {
	tests := []struct {
		name string
		give model.Action
		want string
	}{
		{"allow", model.Allow, "allow"},
		{"limited", model.Limited, "limited"},
		{"step-up", model.StepUpMFA, "stepUpMFA"},
		{"block", model.Block, "block"},
		{"out of range", model.Action(9), "Action(9)"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.give.String(); got != tc.want {
				t.Errorf("String() = %q, want %q", got, tc.want)
			}
		})
	}
}

// goldenDecision proves the §2 wire format end-to-end: action as text.
const goldenDecision = `{"action":"limited","scope":"read-only","reasons":["P04: unmanaged device — least-privilege grant"],"pillars":["devices"],"matched":["P04"]}`

func TestAction_JSONText(t *testing.T) {
	tests := []struct {
		name    string
		give    string // JSON to unmarshal, if any
		marshal any    // value to marshal, if any
		want    any    // string (exact JSON), model.Decision, or model.Action
		wantErr string // non-empty → unmarshal must fail containing this
		anyErr  bool   // an error is required, message unpinned
	}{
		{
			name: "decision marshals action as text",
			marshal: model.Decision{
				Action:  model.Limited,
				Scope:   "read-only",
				Reasons: []string{"P04: unmanaged device — least-privilege grant"},
				Pillars: []string{"devices"},
				Matched: []string{"P04"},
			},
			want: goldenDecision,
		},
		{
			name: "decision round-trip",
			give: goldenDecision,
			want: model.Decision{
				Action:  model.Limited,
				Scope:   "read-only",
				Reasons: []string{"P04: unmanaged device — least-privilege grant"},
				Pillars: []string{"devices"},
				Matched: []string{"P04"},
			},
		},
		{name: "allow round-trips", give: `{"action":"allow"}`, want: model.Allow},
		{name: "limited round-trips", give: `{"action":"limited"}`, want: model.Limited},
		{name: "stepUpMFA round-trips", give: `{"action":"stepUpMFA"}`, want: model.StepUpMFA},
		{name: "block round-trips", give: `{"action":"block"}`, want: model.Block},
		{name: "unknown action text", give: `{"action":"alow"}`, wantErr: `unknown action "alow"`},
		{name: "MarshalText out of range", marshal: model.Action(9), anyErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.give != "" {
				var got model.Decision
				err := json.Unmarshal([]byte(tc.give), &got)
				switch {
				case tc.wantErr != "":
					if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
						t.Errorf("unmarshal err = %v, want containing %q", err, tc.wantErr)
					}
				case err != nil:
					t.Fatalf("unmarshal: %v", err)
				case tc.anyErr:
					t.Errorf("unmarshal err = nil, want an error")
				default:
					assertDecision(t, got, tc.want)
				}
				return
			}
			got, err := json.Marshal(tc.marshal)
			switch {
			case tc.anyErr:
				if err == nil {
					t.Errorf("marshal err = nil, want an error (never emit garbage text)")
				}
			case err != nil:
				t.Fatalf("marshal: %v", err)
			default:
				assertDecision(t, got, tc.want) // marshal path: tc.want is the exact JSON string
			}
		})
	}
}

// assertDecision compares tc.want (string JSON, model.Decision, or model.Action)
// against the unmarshaled or marshaled result.
func assertDecision(t *testing.T, got any, want any) {
	t.Helper()
	switch want := want.(type) {
	case string: // marshal direction: want is the exact JSON text
		if s, ok := got.([]byte); !ok || string(s) != want {
			t.Errorf("marshal = %v, want %s", got, want)
		}
	case model.Decision:
		d, ok := got.(model.Decision)
		if !ok {
			t.Fatalf("unexpected type %T", got)
		}
		if d.Action != want.Action || d.Scope != want.Scope ||
			!slices.Equal(d.Reasons, want.Reasons) ||
			!slices.Equal(d.Pillars, want.Pillars) ||
			!slices.Equal(d.Matched, want.Matched) {
			t.Errorf("unmarshal = %+v, want %+v", d, want)
		}
	case model.Action:
		d, ok := got.(model.Decision)
		if !ok {
			t.Fatalf("unexpected type %T", got)
		}
		if d.Action != want {
			t.Errorf("unmarshal action = %v, want %v", d.Action, want)
		}
	default:
		t.Fatalf("bad want type %T", want)
	}
}

func TestAction_SeverityOrder(t *testing.T) {
	if !(model.Allow < model.Limited && model.Limited < model.StepUpMFA && model.StepUpMFA < model.Block) {
		t.Errorf("severity order broken: want Allow < Limited < StepUpMFA < Block")
	}
}
