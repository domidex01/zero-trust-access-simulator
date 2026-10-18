# Test plan — zero-trust-access-simulator (Phase 2 — Engine)

Test files:
- `internal/engine/engine_test.go` — **package `engine_test`** (black-box; imports
  `zt-sim/internal/engine` + `zt-sim/internal/model`), everything except `New`'s sort
- `internal/engine/engine_internal_test.go` — **package `engine`** (white-box; `New`'s
  sort is observable only through the unexported slice — precedent: `main_test.go` tests
  unexported `run`)

Names follow `TestFunc_Scenario`.

## Code under test (pin these so execution needs no decisions)

```go
// internal/engine/engine.go
package engine // imports zt-sim/internal/model — pure: no I/O, no clock, no randomness

type Policy struct {
	ID       string
	Priority int // lower runs first; ties broken by ID ascending
	Pillar   string // identity|devices|applications|data|networks|infrastructure
	When     func(model.Signal) bool
	Then     model.Action
	Scope    string // populated when Then == Limited, e.g. "read-only"
	Note     string // audit line; Reasons entries read "<ID>: <Note>"
}

type Engine struct{ /* unexported policies []Policy, pre-sorted */ }

func New(policies []Policy) Engine // clone + sort by (Priority asc, ID asc); caller's slice untouched
func (e Engine) Evaluate(s model.Signal) model.Decision // value receiver — Engine is immutable after New
```

Pinned engine reason strings (verbatim §3.5 — note the em dash):

| Path | Exact reason |
|---|---|
| zero matches | `engine: no policy matched — fail closed (assume breach)` |
| empty pack | `engine: no policies loaded — fail closed` |

Semantics pins (§3):
1. Sort `(Priority asc, ID asc)` at construction; Reasons/Matched/Pillars follow that order.
2. Every policy evaluated, no short-circuit — one Reasons entry per matched policy.
3. Final action = numeric max among matched actions (severity is the iota order).
   Scope = first matching Limited policy's Scope when final action is Limited, else "".
4. Pillars = unique pillars of matched policies, evaluation order.
5. Fail closed — never a silent Allow (see reason strings above).

## Shared fixtures (black-box file)

`base()` — the alice note-scenario signal (§6), defined in T2.2, reused by T2.3/T2.4:

```go
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
```

`miniPack()` — synthetic fixture pack (added in T2.4; **not** the real P01–P11, which are
Phase 3). Mirrors the pack's shape: one policy per §9 scenario shape, all four actions
reachable, F01+F05 share the `identity` pillar so dedupe is exercised:

```go
func miniPack() []engine.Policy {
	return []engine.Policy{
		{ID: "F01", Priority: 10, Pillar: "identity", When: func(s model.Signal) bool { return !s.MFAPresent }, Then: model.StepUpMFA, Note: "no second factor presented"},
		{ID: "F02", Priority: 30, Pillar: "devices", When: func(s model.Signal) bool { return !s.DeviceManaged }, Then: model.Limited, Scope: "read-only", Note: "unmanaged device"},
		{ID: "F03", Priority: 40, Pillar: "data", When: func(s model.Signal) bool { return s.RiskScore > 60 }, Then: model.Block, Note: "risk score above fixture threshold"},
		{ID: "F04", Priority: 70, Pillar: "applications", When: func(s model.Signal) bool { return s.Role != "finance" && s.App == "payroll-app" }, Then: model.Block, Note: "app outside requester role"},
		{ID: "F05", Priority: 99, Pillar: "identity", When: func(s model.Signal) bool { return true }, Then: model.Allow, Note: "all signals clean"},
	}
}
```

## T2.1 — `New` sorts by (Priority, ID) and clones the input
- Target: `internal/engine`, `TestNew_SortsByPriorityThenID`, `TestNew_ClonesCallerSlice`
  (engine_internal_test.go, package `engine`)
- Shape: table-driven (sort) + one case (clone)
- Red because: `engine.New` undefined → compile error
- Run: `go test ./internal/engine/ -run TestNew_ -count=1 -v`

TestNew_SortsByPriorityThenID — table field `give` = (Priority, ID) pairs, `want` =
expected `e.policies` ID sequence:

| name | give | want |
|---|---|---|
| priority ascending | (30,C),(10,A),(20,B) | `A,B,C` |
| tie broken by ascending ID | (10,B),(10,A),(10,C) | `A,B,C` |
| mixed priorities and ties | (30,C),(10,B),(10,A),(20,D) | `A,B,D,C` |

TestNew_ClonesCallerSlice:

| name | give | want |
|---|---|---|
| caller slice untouched | caller slice `[(30,C),(10,A)]`, call `New` | caller slice still `[(30,C),(10,A)]` — New sorted a copy (no side effects) |

## T2.2 — `Evaluate` core semantics
- Target: `internal/engine`, `TestEvaluate_Semantics` (engine_test.go, package `engine_test`)
- Shape: table-driven; each row builds an inline pack + uses `base()` (field deltas per row)
- Red because: `Engine.Evaluate` undefined → compile error
- Run: `go test ./internal/engine/ -run TestEvaluate_Semantics -count=1 -v`

| name | give (pack: ID/prio/Then/Note, all `When: true` unless noted) | want Decision |
|---|---|---|
| single match allows with one reason | F99/99/Allow/"all signals clean" | Allow, Scope "", Reasons `["F99: all signals clean"]`, Pillars `["identity"]`, Matched `["F99"]` |
| strictest match wins across actions | F10/StepUp/"no second factor", F20/Limited(read-only)/"unmanaged device", F30/Block/"risk too high" | Block, Scope "", Reasons in order `F10,F20,F30`, Pillars `["identity","devices","data"]`, Matched `["F10","F20","F30"]` |
| reasons follow priority order, not input order | same three policies handed to `New` as F30,F10,F20 | Reasons `F10,F20,F30` (sort normalizes input) |
| priority tie breaks by ascending ID | FA/10/StepUp/"a note", FB/10/Allow/"b note" | StepUpMFA, Reasons `["FA: a note","FB: b note"]` |
| scope from first matching limited policy | F20/Limited("read-only")/"limited one", F25/Limited("time-boxed")/"limited two", F99/Allow/"clean" | Limited, Scope `"read-only"`, Reasons `["F20: limited one","F25: limited two","F99: clean"]` |
| curveball: same pillar listed once | F10/identity/StepUp, `When: !MFAPresent`/"no mfa" + F99/identity/Allow/"clean", signal = base() | StepUpMFA, Pillars `["identity"]` exactly once, Reasons `["F10: no mfa","F99: clean"]` |

Give each row's policies distinct pillars unless the row says otherwise (pillar text is
free-form in the engine; the pack-level enum check is Phase 3's sanity test).

## T2.3 — Fail-closed paths
- Target: `internal/engine`, `TestEvaluate_FailClosed` (engine_test.go)
- Shape: table-driven
- Red because: behavioral — T2.2's zero-match path returns the zero-value Decision
  (`Allow`), so the first row fails on Action
- Run: `go test ./internal/engine/ -run TestEvaluate_FailClosed -count=1 -v`

| name | give | want |
|---|---|---|
| no policy matched | pack = one policy with `When: s.MFAPresent` (false for base()) | Block, Scope "", Reasons `["engine: no policy matched — fail closed (assume breach)"]`, len(Pillars)==0, len(Matched)==0 |
| empty pack via New(nil) | `engine.New(nil)`, base() | Block, Reasons `["engine: no policies loaded — fail closed"]`, len(Pillars)==0, len(Matched)==0 |
| zero-value engine | `var e engine.Engine`, base() | same as empty pack |

Assert `len(Pillars)==0` / `len(Matched)==0`, not nil-vs-empty — Phase 1 already decided
nil-slice normalization is not observable in v1.

## T2.4 — Scenario table + the four §9 properties
- Target: `internal/engine`, `TestEvaluate_ScenarioShapes`, `TestEvaluate_StrictestWins`,
  `TestEvaluate_NoSilentPass`, `TestEvaluate_Deterministic` (engine_test.go)
- Shape: one big table (scenario shapes) + three property tests
- Red because: nothing — gap-closing sweep (T1.5 precedent); a red test here is an engine
  bug: fix the engine, not the test
- Run: `go test ./internal/engine/ -count=1 -v`

### TestEvaluate_ScenarioShapes — `miniPack()`, signal = base() + deltas, full Decisions pinned

| name | delta from base() | want |
|---|---|---|
| note scenario | — | Block, Scope "", Reasons `["F01: no second factor presented","F02: unmanaged device","F03: risk score above fixture threshold","F05: all signals clean"]`, Pillars `["identity","devices","data"]`, Matched `["F01","F02","F03","F05"]` |
| happy | MFAPresent=true, UnfamiliarLoc=false, DeviceManaged=true, DeviceCompliant=true, RiskScore=15, DataSensitivity="internal", Location="US" | Allow, Reasons `["F05: all signals clean"]`, Pillars `["identity"]`, Matched `["F05"]` |
| stolen credential | UnfamiliarLoc=false, DeviceManaged=true, DeviceCompliant=true, RiskScore=45, DataSensitivity="internal", Location="US" (MFA stays false) | StepUpMFA, Reasons `["F01: no second factor presented","F05: all signals clean"]`, Pillars `["identity"]` (deduped), Matched `["F01","F05"]` |
| byod | MFAPresent=true, UnfamiliarLoc=false, RiskScore=15, DataSensitivity="internal", Location="US" (device stays unmanaged) | Limited, Scope `"read-only"`, Reasons `["F02: unmanaged device","F05: all signals clean"]`, Pillars `["devices","identity"]`, Matched `["F02","F05"]` |
| insider | happy delta + Role="engineering", RiskScore=10 | Block, Reasons `["F04: app outside requester role","F05: all signals clean"]`, Pillars `["applications","identity"]`, Matched `["F04","F05"]` |

Every action is exercised by ≥ 1 row; every non-happy row also proves strictest-wins
against the always-matching F05 Allow.

### TestEvaluate_StrictestWins — one always-true policy per listed action, distinct pillars

| name | actions in pack | want action | want scope |
|---|---|---|---|
| allow alone | [Allow] | Allow | "" |
| limited beats allow | [Allow, Limited] | Limited | "read-only" |
| step-up beats allow and limited | [Allow, Limited, StepUpMFA] | StepUpMFA | "" |
| block beats everything | [Allow, Limited, StepUpMFA, Block] | Block | "" |
| step-up beats limited | [Limited, StepUpMFA] | StepUpMFA | "" |
| block beats step-up | [StepUpMFA, Block] | Block | "" |

### TestEvaluate_NoSilentPass — loops over the five scenario rows + the three fail-closed cases
- every decision: `len(Reasons) >= 1`
- when `len(Matched) > 0`: `len(Reasons) == len(Matched)` (one reason per matched policy, §3.2)
- `Action == Allow` ⇒ `len(Matched) >= 1` (an Allow always cites its allow policy)
- fail-closed: exactly 1 reason starting with `engine:`

### TestEvaluate_Deterministic
- double eval: for each scenario row, `d1, d2 := e.Evaluate(s), e.Evaluate(s)` →
  `reflect.DeepEqual(d1, d2)` (§9 property 4)
- construction order irrelevant: `New(miniPack())` vs `New(reversed miniPack())` →
  identical Decision for the same signal (New's sort normalizes input order)

### §9 audit mapping (every bullet + property → named test)

| §9 requirement | Covered by |
|---|---|
| scenario table (noteScenario→Block, happy→Allow, stolenCred→StepUpMFA, byod→Limited, insider→Block) | `TestEvaluate_ScenarioShapes` |
| property 1 — fail closed (zero matches / empty pack → Block, never Allow) | `TestEvaluate_FailClosed` (T2.3) |
| property 2 — strictest wins | `TestEvaluate_StrictestWins` + every multi-match scenario row |
| property 3 — no silent passes (≥ 1 reason, incl. the Allow all-clear) | `TestEvaluate_NoSilentPass` |
| property 4 — determinism (same input → same output/order, two Evaluate calls) | `TestEvaluate_Deterministic` + `TestEvaluate_Semantics` order/tie rows |

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
