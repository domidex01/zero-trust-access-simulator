// Package policies holds the v1 starter policy pack from §4, authored in
// code (typed Go funcs, not config). Pack is the single export surface.
package policies

import (
	"slices"

	"zt-sim/internal/engine"
	"zt-sim/internal/model"
)

// rankConfidential mirrors model.SensitivityRank("confidential"); the enum
// is frozen, and the matcher must not discard the error return.
const rankConfidential = 2

// Pack returns the eleven v1 policies; slice order is the §4 table order,
// which is already (Priority asc, ID asc).
func Pack() []engine.Policy {
	core := []engine.Policy{
		{
			ID: "P01", Priority: 10, Pillar: "identity", Then: model.StepUpMFA,
			Note: "no second factor presented — verify explicitly",
			When: func(s model.Signal) bool { return !s.MFAPresent },
		},
		{
			ID: "P02", Priority: 20, Pillar: "identity", Then: model.StepUpMFA,
			Note: "unfamiliar sign-in location — re-verify",
			When: func(s model.Signal) bool { return s.UnfamiliarLoc },
		},
		{
			ID: "P03", Priority: 20, Pillar: "identity", Then: model.Block,
			Note: "unfamiliar location and no MFA — assume breach",
			When: func(s model.Signal) bool { return s.UnfamiliarLoc && !s.MFAPresent },
		},
		{
			ID: "P04", Priority: 30, Pillar: "devices", Then: model.Limited, Scope: "read-only",
			Note: "unmanaged device — least-privilege grant",
			When: func(s model.Signal) bool { return !s.DeviceManaged },
		},
		{
			ID: "P05", Priority: 30, Pillar: "devices", Then: model.Limited, Scope: "read-only",
			Note: "enrolled but non-compliant device — least-privilege grant",
			When: func(s model.Signal) bool { return s.DeviceManaged && !s.DeviceCompliant },
		},
		{
			ID: "P06", Priority: 40, Pillar: "data", Then: model.Block,
			Note: "confidential/high data on non-compliant device",
			When: func(s model.Signal) bool {
				r, err := model.SensitivityRank(s.DataSensitivity)
				return err == nil && r >= rankConfidential && !s.DeviceCompliant
			},
		},
		{
			ID: "P07", Priority: 50, Pillar: "identity", Then: model.Block,
			Note: "risk score above hard threshold",
			When: func(s model.Signal) bool { return s.RiskScore > 70 },
		},
		{
			ID: "P08", Priority: 60, Pillar: "identity", Then: model.StepUpMFA,
			Note: "elevated risk — verify explicitly",
			When: func(s model.Signal) bool { return s.RiskScore >= 40 && s.RiskScore <= 70 },
		},
		{
			ID: "P09", Priority: 70, Pillar: "identity", Then: model.StepUpMFA,
			Note: "odd-hours intern activity",
			When: func(s model.Signal) bool { return s.LocalHour < 5 && s.Role == "intern" },
		},
		{
			ID: "P10", Priority: 80, Pillar: "applications", Then: model.Block,
			Note: "app outside requester's role — least privilege",
			When: func(s model.Signal) bool { return s.Role != "finance" && s.App == "payroll-app" },
		},
	}
	// P11 matches iff no P01–P10 matched, so Matched/Reasons never carry a
	// false "all clear" alongside a real finding.
	catch := engine.Policy{
		ID: "P11", Priority: 90, Pillar: "identity", Then: model.Allow,
		Note: "all signals clean — explicit allow",
		When: func(s model.Signal) bool {
			return !slices.ContainsFunc(core, func(p engine.Policy) bool { return p.When(s) })
		},
	}
	return append(core, catch)
}
