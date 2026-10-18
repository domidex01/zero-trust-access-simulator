# Test plan — zero-trust-access-simulator (Phase 3 — Policy pack)

Test file:
- `internal/policies/policies_test.go` — **package `policies_test`** (black-box; imports
  `zt-sim/internal/engine` + `zt-sim/internal/model`). No white-box file: matchers are
  observable through the exported `Pack()`, so no `main_test.go`-style exception applies.

Names follow `TestFunc_Scenario`; subtests read as sentences in failure output.

## Code under test (pin these so execution needs no decisions)

```go
// internal/policies/policies.go
package policies // imports zt-sim/internal/{engine,model}; pure data + funcs, no I/O

// rankConfidential mirrors model.SensitivityRank("confidential") — the enum is
// frozen, and the matcher must not discard the error return.
const rankConfidential = 2

// Pack returns the eleven v1 policies; slice order is the §4 table order,
// which is already (Priority asc, ID asc).
func Pack() []engine.Policy
```

Matcher pins (§4 formula → exact Go predicate, `s` is the `model.Signal`):

| ID | Pri | Pillar | `When` | Then | Scope |
|---|---|---|---|---|---|
| P01 | 10 | identity | `!s.MFAPresent` | StepUpMFA | |
| P02 | 20 | identity | `s.UnfamiliarLoc` | StepUpMFA | |
| P03 | 20 | identity | `s.UnfamiliarLoc && !s.MFAPresent` | Block | |
| P04 | 30 | devices | `!s.DeviceManaged` | Limited | `read-only` |
| P05 | 30 | devices | `s.DeviceManaged && !s.DeviceCompliant` | Limited | `read-only` |
| P06 | 40 | data | `r, err := model.SensitivityRank(s.DataSensitivity); err == nil && r >= rankConfidential && !s.DeviceCompliant` | Block | |
| P07 | 50 | identity | `s.RiskScore > 70` | Block | |
| P08 | 60 | identity | `s.RiskScore >= 40 && s.RiskScore <= 70` | StepUpMFA | |
| P09 | 70 | identity | `s.LocalHour < 5 && s.Role == "intern"` | StepUpMFA | |
| P10 | 80 | applications | `s.Role != "finance" && s.App == "payroll-app"` | Block | |
| P11 | 90 | identity | loop over P01–P10, below | Allow | |

P11 shape (tail of `Pack` — T3.2):

```go
core := []engine.Policy{ /* P01–P10 literals from the table above */ }
catch := engine.Policy{
	ID: "P11", Priority: 90, Pillar: "identity", Then: model.Allow,
	Note: "all signals clean — explicit allow",
	When: func(s model.Signal) bool {
		for _, p := range core {
			if p.When(s) {
				return false
			}
		}
		return true
	},
}
return append(core, catch)
```

Notes verbatim from §4 (em dashes are part of the audit-line contract; `**and**` in the
spec table is markdown bold → plain `and`):

| ID | Note |
|---|---|
| P01 | `no second factor presented — verify explicitly` |
| P02 | `unfamiliar sign-in location — re-verify` |
| P03 | `unfamiliar location and no MFA — assume breach` |
| P04 | `unmanaged device — least-privilege grant` |
| P05 | `enrolled but non-compliant device — least-privilege grant` |
| P06 | `confidential/high data on non-compliant device` |
| P07 | `risk score above hard threshold` |
| P08 | `elevated risk — verify explicitly` |
| P09 | `odd-hours intern activity` |
| P10 | `app outside requester's role — least privilege` |
| P11 | `all signals clean — explicit allow` |

## Shared fixtures (defined once, in T3.1's file)

`clean()` — the §6 happy-path signal; **no P01–P10 matcher fires on it** (MFA on,
familiar, managed + compliant, internal data, daytime, finance, non-payroll app, risk 15):

```go
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
```

`policyByID(t, id) engine.Policy` — `t.Helper()`; scans `policies.Pack()`, returns the
policy with that ID, `t.Fatalf("policy %s not in pack", id)` otherwise (that Fatalf is
T3.2's red).

`triggers` — the shared mutation table used by both T3.1 and T3.2 (same slice, two
readings). Mutations set **both** device fields together so the validation invariant
`!DeviceManaged ⇒ !DeviceCompliant` always holds:

| id | mutation of `clean()` | also fires (expected, not asserted) |
|---|---|---|
| P01 | `s.MFAPresent = false` | — |
| P02 | `s.UnfamiliarLoc = true` | — (P03 silent: MFA still on) |
| P03 | `s.UnfamiliarLoc, s.MFAPresent = true, false` | P01, P02 |
| P04 | `s.DeviceManaged, s.DeviceCompliant = false, false` | — (P06 silent: internal) |
| P05 | `s.DeviceManaged, s.DeviceCompliant = true, false` | — |
| P06 | `s.DeviceCompliant = false; s.DataSensitivity = "confidential"` | P05 |
| P07 | `s.RiskScore = 71` | — (P08 silent: 71 > 70) |
| P08 | `s.RiskScore = 70` | — (P07 silent: 70 not > 70) |
| P09 | `s.Role, s.LocalHour = "intern", 4` | — |
| P10 | `s.Role, s.App = "engineering", "payroll-app"` | — |

## T3.1 — P01–P10 fire on trigger, silent on clean
- Target: `internal/policies`, `TestPack_FireSilence` (package `policies_test`)
- Shape: table-driven over `triggers` — two assertions per row:
  `policyByID(t, tc.id).When(tc.mutate(clean()))` is true, and
  `policyByID(t, tc.id).When(clean())` is false
- Red because: importing the empty `internal/policies` package fails the build
  (`no Go files in …/internal/policies` — `Pack` undefined)
- Run: `go test ./internal/policies/ -run TestPack_FireSilence -count=1 -v`

| name | give | want |
|---|---|---|
| P01 no second factor | triggers.P01 on clean | P01.When true; on clean false |
| P02 unfamiliar location | triggers.P02 | P02.When true; clean false |
| P03 unfamiliar and no MFA | triggers.P03 | P03.When true; clean false |
| P04 unmanaged device | triggers.P04 | P04.When true; clean false |
| P05 enrolled not compliant | triggers.P05 | P05.When true; clean false |
| P06 confidential on non-compliant | triggers.P06 | P06.When true; clean false |
| P07 risk above threshold | triggers.P07 | P07.When true; clean false |
| P08 elevated risk | triggers.P08 | P08.When true; clean false |
| P09 odd-hours intern | triggers.P09 | P09.When true; clean false |
| P10 app outside role | triggers.P10 | P10.When true; clean false |

## T3.2 — P11 catch-all is exclusive
- Target: `internal/policies`, `TestPack_CatchAllExclusive` (same file/package)
- Shape: two parts — (a) `policyByID(t, "P11").When(clean())` is true;
  (b) table over the same `triggers`: `policyByID(t, "P11").When(tc.mutate(clean()))`
  is false for **every** row. §4: "matches iff no P01–P10 matched" — the table reading
  proves the iff-direction that matters (P11 must not co-sign a finding)
- Red because: behavioral — `Pack()` still returns only ten policies, so
  `policyByID(t, "P11")` Fatalfs (`policy P11 not in pack`)
- Run: `go test ./internal/policies/ -run TestPack_CatchAllExclusive -count=1 -v`

| name | give | want |
|---|---|---|
| clean signal | `clean()` | P11.When true |
| P01's trigger | triggers.P01 on clean | P11.When false |
| … | …(one subtest per `triggers` row, ids P01–P10)… | P11.When false |

## T3.3 — Sweep: boundaries + §6 scenario shapes
- Target: `internal/policies`, `TestPack_Boundaries` + `TestPack_ScenarioShapes`
- Shape: `TestPack_Boundaries` — rows of (name, mutation, `checks map[string]bool`);
  for each `id, want` in checks: `policyByID(t, id).When(sig) == want`. Only the listed
  ids are asserted (multi-fire is normal — exact match sets are the scenario test's job).
  `TestPack_ScenarioShapes` — rows of (name, overrides on `clean()`, want Decision);
  `d := engine.New(policies.Pack()).Evaluate(sig)`; assert `d.Action`, `d.Scope`,
  `d.Matched`, `d.Pillars` as exact slices. Reasons are pinned structurally by T3.4's
  Note table + T2.2's `"<ID>: <Note>"` format — not re-asserted here
- Red because: nothing — gap-closing sweep (T1.5/T2.4 precedent); 0 red expected, a red
  is a real pack bug → fix the pack, not the test
- Run: `go test ./internal/policies/ -run 'TestPack_Boundaries|TestPack_ScenarioShapes' -count=1 -v`

### Boundaries table

| name | mutation on clean() | checks (id → When) |
|---|---|---|
| risk 70 still elevated | `RiskScore = 70` | P07:false, P08:true |
| risk 71 is hard threshold | `RiskScore = 71` | P07:true, P08:false |
| risk 40 enters elevated | `RiskScore = 40` | P07:false, P08:true |
| risk 39 below elevated | `RiskScore = 39` | P07:false, P08:false |
| intern at 04:59 fires | `Role="intern"; LocalHour = 4` | P09:true |
| intern at 05:00 silent (half-open) | `Role="intern"; LocalHour = 5` | P09:false |
| intern at 00:00 fires | `Role="intern"; LocalHour = 0` | P09:true |
| finance at 03:00 silent (role guard) | `LocalHour = 3` | P09:false |
| confidential on non-compliant | `DataSensitivity="confidential"; DeviceCompliant=false` | P06:true |
| high on non-compliant | `DataSensitivity="high"; DeviceCompliant=false` | P06:true |
| internal on non-compliant | `DeviceCompliant=false` | P06:false |
| confidential on compliant (device guard) | `DataSensitivity="confidential"` | P06:false |
| unmanaged device | `DeviceManaged, DeviceCompliant = false, false` | P04:true, P05:false |
| enrolled not compliant | `DeviceCompliant=false` | P04:false, P05:true |
| fully healthy device | — | P04:false, P05:false |
| unfamiliar with MFA (P02 not dead weight) | `UnfamiliarLoc = true` | P02:true, P03:false |
| familiar without MFA (P03 && guard) | `MFAPresent = false` | P03:false |
| finance on payroll-app (role guard) | `App = "payroll-app"` | P10:false |

(The 4th device state — unmanaged **and** compliant — violates the §2 validation
invariant and is unreachable post-`DecodeSignal`; no row.)

### Scenario shapes table (§6, overrides on `clean()`; Timestamp untouched)

| name | overrides | want action | want matched | want scope | want pillars |
|---|---|---|---|---|---|
| happy | — | allow | P11 | "" | identity |
| noteScenario | `App="payroll-app"; DataSensitivity="high"; Location="BR"; UnfamiliarLoc=true; DeviceManaged=false; DeviceCompliant=false; LocalHour=3; MFAPresent=false; RiskScore=65` | block | P01,P02,P03,P04,P06,P08 | "" | identity,devices,data |
| stolenCred | `MFAPresent=false; DeviceManaged=false; DeviceCompliant=false; RiskScore=45` | stepUpMFA | P01,P04,P08 | "" | identity,devices |
| insider | `User="bob@contoso.com"; Role="engineering"; App="payroll-app"` | block | P10 | "" | applications |
| byod | `DeviceManaged=false; DeviceCompliant=false` | limited | P04 | read-only | devices |
| attacker | `User="unknown@throwaway.example"; Role="guest"; App="payroll-app"; DataSensitivity="high"; Location="NG"; UnfamiliarLoc=true; DeviceManaged=false; DeviceCompliant=false; LocalHour=2; MFAPresent=false; RiskScore=85` | block | P01,P02,P03,P04,P06,P07,P10 | "" | identity,devices,data,applications |

Hand-derivation (verified against §6 while planning): noteScenario — P09 silent (finance
not intern), P07 silent (65 ≤ 70), P10 silent (finance); attacker — P08 silent (85 > 70),
P09 silent (guest); stolenCred/byod keep sensitivity internal so P06 stays silent.

## T3.4 — Sweep: pack sanity, §4 metadata verbatim
- Target: `internal/policies`, `TestPack_Metadata` (same file/package)
- Shape: iterate `policies.Pack()` positionally against the pinned table below — fields
  compared individually (`Policy` has a func field, so no `==`; `When` is tested
  behaviorally in T3.1–T3.3). Then the structural assertions:
  `len(Pack()) == 11` (= table length); IDs unique; exactly one policy with
  `Then == model.Allow` and its ID is `"P11"`; `Scope == "read-only"` iff
  `Then == model.Limited`, else `Scope == ""`; every Pillar ∈ {identity, devices,
  applications, data, networks, infrastructure}. Priorities are **not** asserted unique —
  P02/P03 share 20 and P04/P05 share 30 by design; the table match pins them instead
- Red because: nothing — transcription sweep; a red means the pack (or, if the table was
  mistranscribed from §4, both) gets fixed, noted in the run log
- Run: `go test ./internal/policies/ -run TestPack_Metadata -count=1 -v`

Pinned metadata table (= §4 verbatim, Notes as in the table above):

| # | ID | Priority | Pillar | Then | Scope |
|---|---|---|---|---|---|
| 0 | P01 | 10 | identity | StepUpMFA | "" |
| 1 | P02 | 20 | identity | StepUpMFA | "" |
| 2 | P03 | 20 | identity | Block | "" |
| 3 | P04 | 30 | devices | Limited | read-only |
| 4 | P05 | 30 | devices | Limited | read-only |
| 5 | P06 | 40 | data | Block | "" |
| 6 | P07 | 50 | identity | Block | "" |
| 7 | P08 | 60 | identity | StepUpMFA | "" |
| 8 | P09 | 70 | identity | StepUpMFA | "" |
| 9 | P10 | 80 | applications | Block | "" |
| 10 | P11 | 90 | identity | Allow | "" |

### §9 audit mapping (policies bullets → named tests)

| §9 requirement | Covered by |
|---|---|
| each P01–P11 fires on its trigger signal and stays silent on a clean one | `TestPack_FireSilence` + `TestPack_CatchAllExclusive` |
| boundary tests (risk 70/71, localHour 4/5, sensitivity confidential vs internal) | `TestPack_Boundaries` |
| unique IDs, priorities match §4, catch-all present exactly once, pillar values valid | `TestPack_Metadata` |
| every §6 scenario row derives from the pack | `TestPack_ScenarioShapes` |

---

# Phase 2 — Engine (complete, archived)

Test files: `internal/engine/engine_test.go` (black-box, package `engine_test`),
`internal/engine/engine_internal_test.go` (white-box sort). Key pins still in force:
`New` clones + sorts by (Priority, ID); `Evaluate` evaluates all, numeric-max severity,
first-Limited scope kept only when Limited wins, fail-closed reasons verbatim
(`engine: no policy matched — fail closed (assume breach)` /
`engine: no policies loaded — fail closed`). Named tests: `TestNew_Sort`,
`TestNew_DoesNotMutateCallerSlice`, `TestEvaluate_Semantics`, `TestEvaluate_FailClosed`,
`TestEvaluate_ScenarioShapes` (miniPack), `TestEvaluate_StrictestWins`,
`TestEvaluate_NoSilentPass`, `TestEvaluate_Deterministic`. Full case tables: git
history / `plans/pr-phase2.md`.

---

# Phase 1 — Domain model (complete, archived)

All in `internal/model/model_test.go`, package `model_test`. Test names:
`TestSignal_JSONRoundTrip`, `TestAction_String`, `TestAction_JSONText`,
`TestAction_SeverityOrder`, `TestSensitivityRank`, `TestSignal_Validate`,
`TestDecodeSignal` (+ T1.5 regression: `TestAction_UnmarshalTextNoStateChange`).
Run: `go test ./internal/model/ -count=1`. Full case tables: git history /
`plans/pr-phase1.md`.

---

# Phase 0 — Scaffold (complete, archived)

## T1 — Module scaffold
- Test: none. Scaffolding has no behavior to assert. Acceptance: `go build ./...` exits 0.

## T2 — main dispatch
- `TestRun_Dispatch` (package `main`, white-box — `run` is unexported): version → 0 +
  `zt-sim version 0.1.0\n`; no args / unknown subcommand → 64 + usage on stderr; four
  subcommand stubs → 70 + `not implemented yet`. Exit codes are the CLI's real API —
  asserted exactly.

## T3 — Makefile
- Test: none. `make test` re-runs T2's suite. Acceptance: `make build &&
  ./zt-sim --version && make vet && make test`, then `make fmt` changes nothing.
