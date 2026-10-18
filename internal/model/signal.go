// Package model defines the zero-trust domain model: one access Signal,
// the four Actions, and the resulting Decision.
package model

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

// Signal is one access request — the "verify explicitly" input.
type Signal struct {
	Timestamp       time.Time `json:"timestamp"`       // RFC 3339, required
	User            string    `json:"user"`            // e.g. "alice@contoso.com"
	Role            string    `json:"role"`            // free-form
	App             string    `json:"app"`             // e.g. "payroll-app"
	DataSensitivity string    `json:"dataSensitivity"` // public|internal|confidential|high
	Location        string    `json:"location"`        // ISO 3166-1 alpha-2, "" = unknown
	UnfamiliarLoc   bool      `json:"unfamiliarLocation"`
	DeviceManaged   bool      `json:"deviceManaged"`   // enrolled in device management
	DeviceCompliant bool      `json:"deviceCompliant"` // patched, encrypted, healthy
	LocalHour       int       `json:"localHour"`       // 0–23, user's local wall-clock hour
	MFAPresent      bool      `json:"mfaPresent"`      // a second factor was already used
	RiskScore       int       `json:"riskScore"`       // 0–100, composite heuristic
}

var sensitivityRanks = map[string]int{
	"public":       0,
	"internal":     1,
	"confidential": 2,
	"high":         3,
}

// SensitivityRank maps a dataSensitivity value to its place in the total
// order public < internal < confidential < high ("high" ≙ highly confidential).
// Matching is case-sensitive; unknown values are an error.
func SensitivityRank(s string) (int, error) {
	rank, ok := sensitivityRanks[s]
	if !ok {
		return 0, fmt.Errorf("dataSensitivity %q not one of public|internal|confidential|high", s)
	}
	return rank, nil
}

// Validate enforces every §2 rule and returns the first violation, checking
// in spec-table order. RFC 3339 format is enforced by time.Time unmarshalling;
// Validate only rejects the zero timestamp.
func (s Signal) Validate() error {
	if s.User == "" {
		return errors.New("user must not be empty")
	}
	if s.Role == "" {
		return errors.New("role must not be empty")
	}
	if s.App == "" {
		return errors.New("app must not be empty")
	}
	if _, err := SensitivityRank(s.DataSensitivity); err != nil {
		return err
	}
	if s.RiskScore < 0 || s.RiskScore > 100 {
		return fmt.Errorf("riskScore %d out of range [0,100]", s.RiskScore)
	}
	if s.LocalHour < 0 || s.LocalHour > 23 {
		return fmt.Errorf("localHour %d out of range [0,23]", s.LocalHour)
	}
	if !validLocation(s.Location) {
		return fmt.Errorf("location %q must be empty or ISO 3166-1 alpha-2", s.Location)
	}
	if s.DeviceCompliant && !s.DeviceManaged {
		return errors.New("deviceCompliant true requires deviceManaged")
	}
	if s.Timestamp.IsZero() {
		return errors.New("timestamp must not be zero")
	}
	return nil
}

// validLocation: "" or exactly two ASCII uppercase letters.
func validLocation(loc string) bool {
	if loc == "" {
		return true
	}
	if len(loc) != 2 {
		return false
	}
	for _, c := range loc {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// DecodeSignal decodes one Signal strictly: unknown fields are rejected
// (misspelled keys fail loudly instead of zero-valueing) and the result
// must pass Validate.
func DecodeSignal(r io.Reader) (Signal, error) {
	var s Signal
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return Signal{}, err
	}
	if err := s.Validate(); err != nil {
		return Signal{}, err
	}
	return s, nil
}
