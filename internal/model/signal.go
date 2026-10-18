// Package model defines the zero-trust domain model: one access Signal,
// the four Actions, and the resulting Decision.
package model

import "time"

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
