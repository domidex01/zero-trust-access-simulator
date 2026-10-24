# Test plan — zero-trust-access-simulator (Phase 4 — Scenarios & acceptance)

Test file:
- `internal/scenario/scenario_test.go` — **package `scenario_test`** (black-box; imports
  `zt-sim/internal/model` for T4.1, plus `zt-sim/internal/{engine,policies}` from T4.3
  on). No white-box file: everything is observable through `Load` and `Check`.

Names follow `TestFunc_Scenario`; subtests read as sentences in failure output.

## Code under test (pin these so execution needs no decisions)

```go
// internal/scenario/scenario.go
package scenario // imports zt-sim/internal/model + stdlib only — no engine:
                 // Check takes the Decision, so the package stays decoupled
                 // (Phase 5 wires engine + policies + scenario in the CLI)

type Expectation struct {
	Action         *model.Action `json:"action"`         // required; nil = absent
	ReasonsContain []string      `json:"reasonsContain"` // policy IDs that must appear in reasons
	Scope          string        `json:"scope"`          // checked when Action is limited
}

type Case struct {
	Name   string       `json:"name"`
	Signal model.Signal `json:"signal"`
	Expect Expectation  `json:"expect"`
}

type Pack struct {
	Name        string `json:"name"`
	Description string `json:"description"` // optional metadata (§6 has no rule for it)
	Cases       []Case `json:"cases"`
}

func Load(path string) (Pack, error)
// os.ReadFile → json.Decoder with DisallowUnknownFields (DecodeSignal precedent,
// single Decode) → validate(); decode and validate errors prefixed with the path.

func (p Pack) validate() error // unexported; order pinned:
// name non-empty → ≥1 case → per case (wrapped "case %d %q: %w"):
//   name non-empty → Signal.Validate() (wrapped "signal: %w") →
//   expect.action non-nil ("expect.action required") →
//   scope non-empty ⇒ *Action == Limited
//     ("expect.scope set but expect.action is %s")

func (c Case) Check(d model.Decision) error
// action equality ("want action %s, got %s") →
// every ReasonsContain ID ∈ d.Matched ("reasons missing %s") →
// scope equality only when Expect.Scope != "" ("want scope %q, got %q")
```

Why `*model.Action`: the zero value of `Action` is `Allow`, so a value field can't
tell "absent" from `"allow"` — the pointer makes §6's "action (required)" enforceable.
`*Action` still decodes through the existing `UnmarshalText` (unknown text errors).

## Shared fixtures (T4.1)

`signalJSON` — the happy-path signal as a JSON literal (same values as the policies
suite's `clean()`):

```json
{"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance",
 "app":"intranet-wiki","dataSensitivity":"internal","location":"US",
 "unfamiliarLocation":false,"deviceManaged":true,"deviceCompliant":true,
 "localHour":10,"mfaPresent":true,"riskScore":15}
```

Helpers in the test file (no production counterparts):

```go
func writePack(t *testing.T, content string) string // t.TempDir + os.WriteFile "pack.json", returns path
func caseJSON(name, signal, expect string) string   // fmt.Sprintf case object with those three parts
```

## T4.1 — loader + validator + Check

### TestLoad_Pack (happy path — exactly one)
- Target: `internal/scenario`, `TestLoad_Pack`
- Shape: write the base pack (name `"note"`, description, one case `c1` with
  `signalJSON`, expect `{"action":"block","reasonsContain":["P01","P04"]}`) to a temp
  file; `Load`; assert fields.
- Red because: `internal/scenario` has no non-test Go files → build fails (`no
  non-test Go files in …/internal/scenario` — the T3.1 precedent red)
- Run: `go test ./internal/scenario/ -run TestLoad_Pack -count=1 -v`

| field | want |
|---|---|
| `p.Name` / `p.Description` | `"note"` / the description string |
| `len(p.Cases)` | 1 |
| `p.Cases[0].Name` | `"c1"` |
| signal spot checks | `User` alice@contoso.com · `RiskScore` 15 · `MFAPresent` true · `DataSensitivity` internal |
| `*p.Cases[0].Expect.Action` | `model.Block` |
| `Expect.ReasonsContain` | `["P01","P04"]` (exact) |

### TestLoad_Invalid (every load-time failure)
- Target: same file, `TestLoad_Invalid`
- Shape: table-driven — each row writes a mutated pack JSON, `Load` must error with a
  message containing the `want` substring. Signal-level enum/bounds rules are NOT
  re-tested here (model owns them, §9 audit there) — one representative row proves the
  loader calls `Signal.Validate`.
- Run: `go test ./internal/scenario/ -run TestLoad_Invalid -count=1 -v`

| name | give | want err contains |
|---|---|---|
| unknown top-level field | base + `,"extra":1` | `unknown field "extra"` |
| unknown case field (typo demo) | case gains `,"expct":{}` | `unknown field "expct"` |
| unknown signal field | signal gains `,"risk":9` | `unknown field "risk"` |
| missing pack name | name key dropped | `name must not be empty` |
| empty cases | `"cases":[]` | `cases must not be empty` |
| case name empty | `caseJSON("", …)` | `case 0` |
| signal fails validation | signal user `""` | `user must not be empty` |
| expect.action missing | expect `{}` | `expect.action required` |
| expect null | expect `null` | `expect.action required` |
| expect.action not an enum | expect action `"maybe"` | `unknown action "maybe"` |
| scope on non-limited | expect `{"action":"block","scope":"read-only"}` | `expect.scope set but expect.action is block` |
| file missing | Load of a nonexistent path (row handled separately, not via writePack) | `no such file` |

### TestCase_Check (expectation → decision comparison)
- Target: same file, `TestCase_Check`
- Shape: table of (Case built by hand with an explicit `&action`, Decision) → want nil
  or error substring. Cases always come pre-validated from `Load`; hand-built rows set
  `Expect.Action` explicitly so `Check`'s deref is safe.
- Run: `go test ./internal/scenario/ -run TestCase_Check -count=1 -v`

Fixed decisions used by rows (hand-built, not engine output — `Check` is engine-free):

```go
blockD   := model.Decision{Action: model.Block, Scope: "",
             Reasons: []string{"P01: …", "P04: …"}, Matched: []string{"P01", "P04"}}
limitedD := model.Decision{Action: model.Limited, Scope: "read-only", Matched: []string{"P04"}}
bareD    := limitedD with Scope ""   // for the scope-mismatch row
```

| name | give (expect → decision) | want |
|---|---|---|
| exact match | block, [P01] → blockD | nil |
| action mismatch | allow → blockD | `want action allow, got block` |
| missing policy in reasons | block, [P01, P07] → blockD | `reasons missing P07` |
| scope matches | limited, scope read-only → limitedD | nil |
| scope mismatch | limited, scope read-only → bareD | `want scope "read-only", got ""` |
| scope absent not checked | limited, no scope → bareD | nil |
| empty reasonsContain | block, [] → blockD | nil |

## T4.2 — author the six packs (`TestLoad_SixPacks` + pinned file contents)
- Target: `internal/scenario`, `TestLoad_SixPacks`
- Shape: table over the six filenames — `scenario.Load("../../scenarios/<name>.json")`
  succeeds, `p.Name == name` (file stem), `len(p.Cases) ≥ 1`.
- Red because: behavioral — the files don't exist yet, every `Load` fails with
  `no such file or directory`
- Run: `go test ./internal/scenario/ -run TestLoad_SixPacks -count=1 -v`

### Pinned pack contents (= file authoring spec; signals are the
`TestPack_ScenarioShapes` pins verbatim, note-scenario's signal = §6 example verbatim;
`reasonsContain` = §6 matched-policies column in full; descriptions = §6 Story column)

| pack / case name | timestamp | user | role | app | sens | loc | unfam | mgd | comp | hour | mfa | risk | expect |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| happy / alice-normal-day | 10:00Z | alice@contoso.com | finance | intranet-wiki | internal | US | f | t | t | 10 | t | 15 | allow · [P11] |
| note-scenario / alice-travel-personal-device | 03:14Z | alice@contoso.com | finance | payroll-app | high | BR | t | f | f | 3 | f | 65 | block · [P01,P02,P03,P04,P06,P08] |
| stolen-credential / attacker-with-alices-password | 10:00Z | alice@contoso.com | finance | intranet-wiki | internal | US | f | f | f | 10 | f | 45 | stepUpMFA · [P01,P04,P08] |
| insider / engineering-requests-payroll | 10:00Z | bob@contoso.com | engineering | payroll-app | internal | US | f | t | t | 10 | t | 15 | block · [P10] |
| byod / personal-phone-internal-data | 10:00Z | alice@contoso.com | finance | intranet-wiki | internal | US | f | f | f | 10 | t | 15 | limited · scope read-only · [P04] |
| attacker / everything-wrong-at-once | 10:00Z | unknown@throwaway.example | guest | payroll-app | high | NG | t | f | f | 2 | f | 85 | block · [P01,P02,P03,P04,P06,P07,P10] |

All timestamps are `2026-01-15T…Z` (note-scenario: `03:14:00Z`, §6 verbatim; the rest
`10:00:00Z`). Only byod carries `expect.scope` (`"read-only"`). Descriptions:
note-scenario's is §6-verbatim ("Flagship demo from the SC-900 note: Alice travelling,
personal laptop, payroll app, 03:14 local."); the others restate the §6 Story column.

## T4.3 — acceptance walker + action coverage
- Target: `internal/scenario`, `TestPacks_Expectations` + `TestPacks_ActionCoverage`
- Shape: shared helper `loadPacks(t) []scenario.Pack` — glob
  `../../scenarios/*.json`; **guard**: ≥6 files found AND all six pinned basenames
  present (an empty glob fails the test instead of passing vacuously); Load each
  (Fatal on error). `TestPacks_Expectations`: one engine
  (`engine.New(policies.Pack())`), per pack per case `t.Run("<pack>/<case>")` →
  `eng.Evaluate(c.Signal)` → `c.Check(d)` nil (fatalf prints the Check error **and**
  the full decision). `TestPacks_ActionCoverage`: evaluate every case, assert each of
  Allow/Limited/StepUpMFA/Block was seen (§6's "every action is exercised" claim).
- Red because: nothing — gap-closing sweep (T1.5/T2.4/T3.3 precedent); 0 red
  expected, a red is a real divergence → fix the pack file, or the §6 table + spec if
  *it* is wrong (say so in the run log)
- Run: `go test ./internal/scenario/ -run 'TestPacks_Expectations|TestPacks_ActionCoverage' -count=1 -v`

## T4.4 — hand-derivation table (verify each pack against §4, recorded in run log)

| pack | fires (why) | silent (why) | strictest |
|---|---|---|---|
| happy | P11 (nothing else matched) | P01–P10 all guarded off by clean signals | allow |
| note-scenario | P01 (!mfa) P02 (unfam) P03 (unfam∧!mfa) P04 (!mgd) P06 (high∧!comp) P08 (40≤65≤70) | P05 (not enrolled) P07 (65≤70) P09 (finance) P10 (finance role) | block |
| stolen-credential | P01 (!mfa) P04 (!mgd) P08 (45) | P02/P03 (familiar) P06 (internal) P10 (intranet-wiki) | stepUpMFA |
| insider | P10 (engineering∧payroll-app) | P06 (internal∧compliant) P01/P02 (mfa, familiar) | block |
| byod | P04 (!mgd) | P06 (internal) P01 (mfa on) P10 (finance) | limited · read-only |
| attacker | P01 P02 P03 P04 P06 (high∧!comp) P07 (85>70) P10 (guest∧payroll) | P05 (not enrolled) P08 (85>70) P09 (guest) | block |

### §9 audit mapping (scenario bullet → named tests)

| §9 requirement | Covered by |
|---|---|
| loads every `scenarios/*.json` | `TestLoad_SixPacks` (the six) + `loadPacks` glob guard (any future pack too) |
| validates | `TestLoad_Invalid` (12 rows) |
| asserts all expectations | `TestPacks_Expectations` |
| wired into `go test ./...` | walker lives in `internal/scenario` → runs with `./...` |
| §6 "every action exercised" | `TestPacks_ActionCoverage` |

---

# Phase 3 — Policy pack (complete, archived)

Test file: `internal/policies/policies_test.go`, package `policies_test` (black-box
through `Pack()`). Named tests: `TestPack_FireSilence`, `TestPack_CatchAllExclusive`,
`TestPack_Boundaries`, `TestPack_ScenarioShapes` (the six §6 rows — the exact signal
pins Phase 4's JSON files reuse), `TestPack_Metadata` (§4 verbatim). Matcher formulas,
Notes, and case tables: git history / `plans/pr-phase3.md`.

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
