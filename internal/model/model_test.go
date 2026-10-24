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

// aliceSignal is the valid base signal every Validate row mutates.
func aliceSignal() model.Signal {
	return model.Signal{
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
}

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
				want := aliceSignal()
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
	t.Run("decision marshals action as text", func(t *testing.T) {
		got, err := json.Marshal(model.Decision{
			Action:  model.Limited,
			Scope:   "read-only",
			Reasons: []string{"P04: unmanaged device — least-privilege grant"},
			Pillars: []string{"devices"},
			Matched: []string{"P04"},
		})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if string(got) != goldenDecision {
			t.Errorf("marshal = %s, want %s", got, goldenDecision)
		}
	})

	t.Run("MarshalText out of range", func(t *testing.T) {
		if _, err := json.Marshal(model.Action(9)); err == nil {
			t.Errorf("marshal err = nil, want an error (never emit garbage text)")
		}
	})

	tests := []struct {
		name    string
		give    string
		want    model.Decision
		wantErr string
	}{
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
		{name: "allow round-trips", give: `{"action":"allow"}`, want: model.Decision{Action: model.Allow}},
		{name: "limited round-trips", give: `{"action":"limited"}`, want: model.Decision{Action: model.Limited}},
		{name: "stepUpMFA round-trips", give: `{"action":"stepUpMFA"}`, want: model.Decision{Action: model.StepUpMFA}},
		{name: "block round-trips", give: `{"action":"block"}`, want: model.Decision{Action: model.Block}},
		{name: "unknown action text", give: `{"action":"alow"}`, wantErr: `unknown action "alow"`},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var got model.Decision
			err := json.Unmarshal([]byte(tc.give), &got)
			switch {
			case tc.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("unmarshal err = %v, want containing %q", err, tc.wantErr)
				}
			case err != nil:
				t.Fatalf("unmarshal: %v", err)
			case !equalDecision(got, tc.want):
				t.Errorf("unmarshal = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// equalDecision compares field-wise; Decision is not comparable with ==
// because of its slice fields.
func equalDecision(a, b model.Decision) bool {
	return a.Action == b.Action && a.Scope == b.Scope &&
		slices.Equal(a.Reasons, b.Reasons) &&
		slices.Equal(a.Pillars, b.Pillars) &&
		slices.Equal(a.Matched, b.Matched)
}

func TestAction_SeverityOrder(t *testing.T) {
	if !(model.Allow < model.Limited && model.Limited < model.StepUpMFA && model.StepUpMFA < model.Block) {
		t.Errorf("severity order broken: want Allow < Limited < StepUpMFA < Block")
	}
}

func TestSensitivityRank(t *testing.T) {
	tests := []struct {
		name    string
		give    string
		want    int
		wantErr string
	}{
		{name: "public", give: "public", want: 0},
		{name: "internal", give: "internal", want: 1},
		{name: "confidential", give: "confidential", want: 2},
		{name: "high", give: "high", want: 3},
		{name: "unknown value", give: "ultra", wantErr: `dataSensitivity "ultra"`},
		{name: "wrong case", give: "Public", wantErr: "dataSensitivity"},
		{name: "empty", give: "", wantErr: "dataSensitivity"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := model.SensitivityRank(tc.give)
			switch {
			case tc.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("err = %v, want containing %q", err, tc.wantErr)
				}
			case err != nil:
				t.Errorf("unexpected error: %v", err)
			case got != tc.want:
				t.Errorf("rank = %d, want %d", got, tc.want)
			}
		})
	}

	// total order holds: public < internal < confidential < high
	ranks := []string{"public", "internal", "confidential", "high"}
	prev := -1
	for _, s := range ranks {
		r, err := model.SensitivityRank(s)
		if err != nil {
			t.Fatalf("rank(%q): %v", s, err)
		}
		if r <= prev {
			t.Errorf("total order broken: rank(%q) = %d, not > %d", s, r, prev)
		}
		prev = r
	}
}

func TestSignal_Validate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(s *model.Signal)
		wantErr string // "" → want nil
	}{
		{name: "happy path", wantErr: ""},
		{
			name: "lower limits accepted",
			mutate: func(s *model.Signal) {
				s.RiskScore = 0
				s.LocalHour = 0
				s.Location = ""
			},
		},
		{
			name: "upper limits accepted",
			mutate: func(s *model.Signal) {
				s.RiskScore = 100
				s.LocalHour = 23
			},
		},
		{
			name:    "empty user",
			mutate:  func(s *model.Signal) { s.User = "" },
			wantErr: "user must not be empty",
		},
		{
			name:    "empty role",
			mutate:  func(s *model.Signal) { s.Role = "" },
			wantErr: "role must not be empty",
		},
		{
			name:    "empty app",
			mutate:  func(s *model.Signal) { s.App = "" },
			wantErr: "app must not be empty",
		},
		{
			name:    "sensitivity typo",
			mutate:  func(s *model.Signal) { s.DataSensitivity = "ultra" },
			wantErr: `dataSensitivity "ultra"`,
		},
		{
			name:    "riskScore negative",
			mutate:  func(s *model.Signal) { s.RiskScore = -1 },
			wantErr: "riskScore -1 out of range",
		},
		{
			name:    "riskScore over",
			mutate:  func(s *model.Signal) { s.RiskScore = 101 },
			wantErr: "riskScore 101 out of range",
		},
		{
			name:    "localHour negative",
			mutate:  func(s *model.Signal) { s.LocalHour = -1 },
			wantErr: "localHour -1 out of range",
		},
		{
			name:    "localHour over",
			mutate:  func(s *model.Signal) { s.LocalHour = 24 },
			wantErr: "localHour 24 out of range",
		},
		{
			name:    "location lowercase",
			mutate:  func(s *model.Signal) { s.Location = "us" },
			wantErr: `location "us"`,
		},
		{
			name:    "location three letters",
			mutate:  func(s *model.Signal) { s.Location = "USA" },
			wantErr: `location "USA"`,
		},
		{
			name:    "location letter-digit",
			mutate:  func(s *model.Signal) { s.Location = "U1" },
			wantErr: `location "U1"`,
		},
		{
			name:    "unmanaged but compliant",
			mutate:  func(s *model.Signal) { s.DeviceManaged = false; s.DeviceCompliant = true },
			wantErr: "deviceCompliant true requires deviceManaged",
		},
		{
			name:    "zero timestamp",
			mutate:  func(s *model.Signal) { s.Timestamp = time.Time{} },
			wantErr: "timestamp must not be zero",
		},
		{
			name: "curveball: first error wins",
			mutate: func(s *model.Signal) {
				s.DataSensitivity = "ultra"
				s.DeviceManaged = false
				s.DeviceCompliant = true
			},
			// pins check order: sensitivity before device invariant
			wantErr: `dataSensitivity "ultra"`,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := aliceSignal()
			if tc.mutate != nil {
				tc.mutate(&s)
			}
			err := s.Validate()
			switch {
			case tc.wantErr == "" && err != nil:
				t.Errorf("unexpected error: %v", err)
			case tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)):
				t.Errorf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

func TestDecodeSignal(t *testing.T) {
	tests := []struct {
		name    string
		give    string
		want    model.Signal
		wantErr string
		anyErr  bool
	}{
		{
			name: "strict happy path",
			give: goldenSignal,
			want: aliceSignal(),
		},
		{
			name:    "misspelled key rejected",
			give:    strings.TrimSuffix(goldenSignal, "}") + `,"risk":5}`,
			wantErr: "unknown field", // spec: no silent zero-valueing
		},
		{
			name:    "validation applied at load",
			give:    strings.Replace(goldenSignal, `"alice@contoso.com"`, `""`, 1),
			wantErr: "user must not be empty",
		},
		{
			name:   "bad timestamp format",
			give:   strings.Replace(goldenSignal, `"2026-01-15T03:14:00Z"`, `"yesterday"`, 1),
			anyErr: true, // RFC 3339 enforced by time.Time unmarshalling, not Validate
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := model.DecodeSignal(strings.NewReader(tc.give))
			switch {
			case tc.wantErr != "":
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Errorf("err = %v, want containing %q", err, tc.wantErr)
				}
			case tc.anyErr:
				if err == nil {
					t.Errorf("err = nil, want an error")
				}
			case err != nil:
				t.Errorf("unexpected error: %v", err)
			case got != tc.want:
				t.Errorf("decode = %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestAction_UnmarshalTextUnknownLeavesUnchanged pins the API contract that a
// failed unmarshal must not corrupt a pre-set action (§9: action text marshalling).
func TestAction_UnmarshalTextUnknownLeavesUnchanged(t *testing.T) {
	d := model.Decision{Action: model.Block}
	if err := json.Unmarshal([]byte(`{"action":"alow"}`), &d); err == nil {
		t.Fatalf("err = nil, want unknown-action error")
	}
	if d.Action != model.Block {
		t.Errorf("action = %v, want unchanged Block", d.Action)
	}
}

// TestDecodeSignals pins the decide file format: a strict JSON array of
// signals, every element validated, empty rejected (§5 — packs fail here too).
func TestDecodeSignals(t *testing.T) {
	const happy = `{"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance","app":"intranet-wiki","dataSensitivity":"internal","location":"US","unfamiliarLocation":false,"deviceManaged":true,"deviceCompliant":true,"localHour":10,"mfaPresent":true,"riskScore":15}`
	tests := []struct {
		name    string
		give    string
		wantErr string
	}{
		{name: "unknown field in element", give: "[" + happy + `,{"risk":9}]`, wantErr: `unknown field "risk"`},
		{name: "element fails validation", give: `[{"user":"","role":"finance","app":"wiki","dataSensitivity":"internal","timestamp":"2026-01-15T10:00:00Z"}]`, wantErr: "user must not be empty"},
		{name: "single object not array", give: happy, wantErr: "[]model.Signal"},
		{name: "scenario pack rejected", give: `{"name":"x","cases":[]}`, wantErr: "[]model.Signal"},
		{name: "empty array", give: `[]`, wantErr: "empty signals array"},
		{name: "null", give: `null`, wantErr: "empty signals array"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := model.DecodeSignals(strings.NewReader(tc.give)); err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("err = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
	t.Run("two valid signals decode with fields intact", func(t *testing.T) {
		const attacker = `{"timestamp":"2026-01-15T10:00:00Z","user":"unknown@throwaway.example","role":"guest","app":"payroll-app","dataSensitivity":"high","location":"NG","unfamiliarLocation":true,"deviceManaged":false,"deviceCompliant":false,"localHour":2,"mfaPresent":false,"riskScore":85}`
		got, err := model.DecodeSignals(strings.NewReader("[" + happy + "," + attacker + "]"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if got[0].User != "alice@contoso.com" || got[1].RiskScore != 85 {
			t.Errorf("got[0].User = %q, got[1].RiskScore = %d, want alice…/85", got[0].User, got[1].RiskScore)
		}
	})
}
