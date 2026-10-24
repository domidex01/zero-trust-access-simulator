# Test plan — zero-trust-access-simulator (Phase 5 — CLI)

Test files:
- `internal/report/report_test.go` — **package `report_test`** (black-box; imports
  `zt-sim/internal/model`, and `zt-sim/internal/engine` from T5.4 on for `Policies`)
- `internal/model/model_test.go` (edit) — `TestDecodeSignals`, package `model_test`
- `main_test.go` (edit) — package `main`, white-box (`run`/`cmd*` are unexported);
  `TestRun_Dispatch` rows for an implemented subcommand are deleted when that
  subcommand lands (replaced by its own `Test*`), the `audit` stub row survives
  until Phase 6

Names follow `TestFunc_Scenario`; subtests read as sentences in failure output.

## Code under test (pin these so execution needs no decisions)

```go
// internal/report/report.go
package report // imports model (+ engine from T5.4) and stdlib only

// Row pairs the request with its outcome: the table's USER column comes
// from the signal, everything else from the decision.
type Row struct {
	Signal   model.Signal
	Decision model.Decision
}

// Decisions writes the USER / ACTION / REASONS table and a pillars footer
// (§5: columns + content are the contract; rendering beyond that is ours —
// tabwriter, padding 2). One line per reason; action uppercased; footer
// pillars unique across rows in first-appearance order.
func Decisions(w io.Writer, rows []Row) error

// DecisionsJSON writes decisions as an indented JSON array (§5 --format json).
func DecisionsJSON(w io.Writer, decisions []model.Decision) error

// PackResult is one row of the run summary table (T5.3).
type PackResult struct {
	Name      string
	Cases     int
	Pass, Fail int
	Strictest model.Action // max action across the pack's decisions
}

// PackSummary writes the PACK / CASES / PASS / FAIL / STRICTEST table (T5.3).
func PackSummary(w io.Writer, results []PackResult) error

// Policies writes the ID / PRIORITY / PILLAR / ACTION / NOTE table (T5.4).
func Policies(w io.Writer, policies []engine.Policy) error
```

```go
// internal/model/signal.go (T5.2, next to DecodeSignal)
// DecodeSignals strictly decodes the decide file format — a JSON array of
// signals — validating every element. Empty array (or null) is an input
// error: nothing to decide must not exit 0.
func DecodeSignals(r io.Reader) ([]Signal, error)
// json.Decoder + DisallowUnknownFields → Decode(&[]Signal) →
// len 0 → "empty signals array" → Validate each (first failure wins).
```

```go
// main.go
func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int
// T5.2 adds the stdin param (main passes os.Stdin) so decide --stdin is testable.

func cmdDecide(args []string, stdin io.Reader, stdout, stderr io.Writer) int
// flag.NewFlagSet("decide", flag.ContinueOnError), fs.SetOutput(stderr)
// flags: --stdin bool · --format string (default "table")
// parse error, format ∉ {table,json}, (no positional && !stdin), (positional && stdin),
// >1 positional → usage + 64.
// input: os.Open(path)+DecodeSignals | DecodeSignal(stdin); error → stderr + 65.
// engine.New(policies.Pack()) once; evaluate all (input order);
// report.Decisions | report.DecisionsJSON (error → 65);
// return max int(d.Action) over decisions.

func cmdRun(args []string, stdout, stderr io.Writer) int
// ≠1 arg → 64. os.ReadDir(dir) error → 65. Keep top-level *.json (ReadDir is
// sorted). 0 packs → 65 "no scenario packs in <dir>". scenario.Load error →
// abort → 65. Per case: Evaluate + Check; print "PASS <pack>/<case>" or
// "FAIL <pack>/<case>: <err>"; PackResult{Name: pack.Name, Strictest: max};
// report.PackSummary last; return 1 if any fail else 0.

func cmdPolicies(args []string, stdout, stderr io.Writer) int
// ≠0 extra args → 64. report.Policies(stdout, policies.Pack()); error → 65; else 0.
```

## Shared fixtures

Hand-built decisions/rows (report is engine-free for T5.1 — no evaluation involved):

```go
attackerD := model.Decision{Action: model.Block,
	Reasons: []string{"P01: no second factor presented — verify explicitly",
		"P03: unfamiliar location and no MFA — assume breach",
		"P07: risk score above hard threshold"},
	Pillars: []string{"identity", "devices", "data", "applications"},
	Matched: []string{"P01", "P03", "P07"}}
attackerRow := report.Row{Signal: model.Signal{User: "unknown@throwaway.example"}, Decision: attackerD}

happyD := model.Decision{Action: model.Allow,
	Reasons: []string{"P11: all signals clean — explicit allow"},
	Pillars: []string{"identity"}, Matched: []string{"P11"}}
happyRow := report.Row{Signal: model.Signal{User: "alice@contoso.com"}, Decision: happyD}

byodD := model.Decision{Action: model.Limited, Scope: "read-only",
	Reasons: []string{"P04: unmanaged device — least-privilege grant"},
	Pillars: []string{"devices"}, Matched: []string{"P04"}}
```

Signal JSON for the main tests (the T4.1 `signalJSON` + the
`TestPack_ScenarioShapes` pins as JSON — copy verbatim):

```json
happy    = {"timestamp":"2026-01-15T10:00:00Z","user":"alice@contoso.com","role":"finance","app":"intranet-wiki","dataSensitivity":"internal","location":"US","unfamiliarLocation":false,"deviceManaged":true,"deviceCompliant":true,"localHour":10,"mfaPresent":true,"riskScore":15}
byod     = happy with "deviceManaged":false,"deviceCompliant":false
stolen   = happy with "deviceManaged":false,"deviceCompliant":false,"mfaPresent":false,"riskScore":45
attacker = {"timestamp":"2026-01-15T10:00:00Z","user":"unknown@throwaway.example","role":"guest","app":"payroll-app","dataSensitivity":"high","location":"NG","unfamiliarLocation":true,"deviceManaged":false,"deviceCompliant":false,"localHour":2,"mfaPresent":false,"riskScore":85}
```

Main-test helpers (test file only): `writeFile(t, content string) string`
(t.TempDir + one file, returns path), `runCLI(t, args, stdin string) (exit int,
stdout, stderr string)` wrapping `run` with bytes buffers.

## T5.1 — report: decisions table + JSON

### TestReport_DecisionsTable (happy path + structure)
- Target: `internal/report`, `TestReport_DecisionsTable`
- Shape: table-driven over rows; `want` = substrings + structural line counts
  (assert on `strings.Split(out, "\n")` — trimmed lines, so tabwriter widths
  stay free). Red because: no non-test Go files in `internal/report` → build
  fails (`no non-test Go files in …/internal/report` — the T3.1/T4.1 precedent red)
- Run: `go test ./internal/report/ -run TestReport_DecisionsTable -count=1 -v`

| name | give rows | want |
|---|---|---|
| attacker row | [attackerRow] | header line `USER ACTION REASONS` fields present; `unknown@throwaway.example` + `BLOCK` on the row line; each of the 3 full reasons on its own line; footer line `pillars: identity, devices, data, applications`; total lines = 5 (header + 3 reasons + footer) |
| allow row | [happyRow] | `alice@contoso.com` + `ALLOW`; `P11: all signals clean — explicit allow`; footer `pillars: identity`; 3 lines |
| two rows, pillar union | [happyRow, attackerRow] | alice line before unknown line (input order); footer exactly `pillars: identity, devices, data, applications` (each pillar once) |
| reasons stack, not joined | [attackerRow] | trimmed line 2 == reason[1], trimmed line 3 == reason[2] (no `; `-joined one-liner regression) |
| action text is uppercased | [byodRow] | `LIMITED` (not `limited`); note: NO scope column — `read-only` absent from table (scope rides in JSON) |

### TestReport_DecisionsJSON
- Target: same file, `TestReport_DecisionsJSON`
- Shape: table-driven; round-trip via `json.Unmarshal` into `[]model.Decision`
  plus one text substring each (indented encoder output)
- Run: `go test ./internal/report/ -run TestReport_DecisionsJSON -count=1 -v`

| name | give | want |
|---|---|---|
| single decision | [attackerD] | round-trips equal (len 1, all fields); text contains `"action": "block"` |
| limited keeps scope | [byodD] | round-trips equal; text contains `"scope": "read-only"` |
| two decisions in order | [happyD, byodD] | len 2; [0].Action == Allow, [1].Action == Limited |

## T5.2 — `decide`

### TestDecodeSignals (model)
- Target: `internal/model`, `TestDecodeSignals` (appended to `model_test.go`)
- Shape: table-driven over JSON strings; `want` = error substring or decoded
  spot checks. Red because: `model.DecodeSignals` undefined → compile error
- Run: `go test ./internal/model/ -run TestDecodeSignals -count=1 -v`

| name | give | want |
|---|---|---|
| two valid signals | `[happy, attacker]` | len 2; [0].User alice…, [1].RiskScore 85; nil error |
| unknown field in element | happy + `,"risk":9` | `unknown field "risk"` |
| element fails validation | happy with `"user":""` | `user must not be empty` |
| single object, not array | happy alone (no `[ ]`) | `[]model.Signal` (cannot-unmarshal-object) |
| §6 scenario pack rejected | `{"name":"x","cases":[]}` | `[]model.Signal` — §5's "packs are rejected in decide" rule, proven at the decoder |
| empty array | `[]` | `empty signals array` |
| null | `null` | `empty signals array` |

### TestRun_Decide (main)
- Target: `main`, `TestRun_Decide` (package main; `run` now takes stdin)
- Shape: table-driven; fixtures via `writeFile`; stdin fed as a string (ignored
  unless `--stdin`). Red because: behavioral — decide is still the exit-70 stub
- Run: `go test . -run TestRun_Decide -count=1 -v`

| name | args / stdin | wantExit | stdout contains | stderr contains |
|---|---|---|---|---|
| happy file allows | `decide <happy.json>` | 0 | `USER`, `ALLOW`, `P11: all signals clean — explicit allow`, `pillars: identity` | — |
| byod file limits | `decide <byod.json>` | 1 | `LIMITED` | — |
| stolen file steps up | `decide <stolen.json>` | 2 | `STEPUPMFA` | — |
| attacker file blocks | `decide <attacker.json>` | 3 | `BLOCK`, `P07: risk score above hard threshold`, `pillars: identity, devices, data, applications` | — |
| mixed file takes strictest | `decide <[happy,attacker].json>` | 3 | both `alice@contoso.com` and `unknown@throwaway.example`; exactly 1 header line | — |
| stdin single signal | `decide --stdin` / happy | 0 | `ALLOW` | — |
| stdin invalid object | `decide --stdin` / happy + `,"risk":9` | 65 | — | `unknown field "risk"` |
| missing file | `decide /nope/x.json` | 65 | — | `no such file` |
| pack fed to decide | `decide scenarios/attacker.json` | 65 | — | `[]model.Signal` |
| invalid signal in array | file `[happy with riskScore 9000]` | 65 | — | `riskScore` |
| no input source | `decide` | 64 | — | usage |
| stdin AND file | `decide --stdin <happy.json>` | 64 | — | usage |
| unknown flag | `decide --wat x` | 64 | — | usage |
| bad format value | `decide --format yaml <happy.json>` | 64 | — | usage |

Also `TestRun_DecideFormatJSON`: `decide --format json <attacker.json>` → exit 3,
stdout round-trips via `json.Unmarshal` to 1 decision with Action Block, Scope "",
len(Matched) 7. And `TestRun_Dispatch` edits: delete the "decide is stubbed" row;
all remaining rows call `run` with the new stdin arg (pass `strings.NewReader("")`).

## T5.3 — `run`

### TestReport_PackSummary
- Target: `internal/report`, `TestReport_PackSummary`
- Shape: one table-driven case with three results
  (`{happy,1,1,0,Allow}`, `{attacker,1,1,0,Block}`, `{mixed,2,1,1,Limited}`)
- Red because: `report.PackSummary`/`PackResult` undefined → compile error
- Run: `go test ./internal/report/ -run TestReport_PackSummary -count=1 -v`

| want | |
|---|---|
| header fields | `PACK CASES PASS FAIL STRICTEST` present |
| row order | happy row before attacker row before mixed row (input order) |
| cells | `1 1 0 ALLOW` on happy's row; `2 1 1 LIMITED` on mixed's row (trimmed) |

### TestRun_Run (main)
- Target: `main`, `TestRun_Run`
- Shape: table-driven; failing/corrupt packs via `writeFile` in a temp dir;
  the real `scenarios/` dir is used for the all-pass row (tests run at project root)
- Red because: behavioral — run is still the exit-70 stub
- Run: `go test . -run TestRun_Run -count=1 -v`

| name | give | wantExit | output |
|---|---|---|---|
| scenarios dir all pass | `run scenarios` | 0 | six `PASS <name>/` lines (one per pack; `FAIL` absent); summary header `PACK`; rows for all six pack names |
| failing expectation | temp pack: happy signal, `expect.action block` | 1 | `FAIL <pack>/<case>: want action block, got allow`; summary shows FAIL 1 |
| corrupt pack aborts | temp dir: `not json` file | 65 | stderr names the file path |
| missing dir | `run ./nope` | 65 | stderr `no such file` |
| dir with no packs | empty temp dir | 65 | stderr `no scenario packs` |
| no args | `run` | 64 | usage |
| two args | `run a b` | 64 | usage |

`TestRun_Dispatch`: delete the "run is stubbed" row.

## T5.4 — `policies`

### TestReport_Policies
- Target: `internal/report`, `TestReport_Policies`
- Shape: table-driven over a two-policy literal slice
  (P01-shaped `{ID:"P01",Priority:10,Pillar:"identity",Then:StepUpMFA,Note:"no second factor presented — verify explicitly"}`
  + P04-shaped `{ID:"P04",Priority:30,Pillar:"devices",Then:Limited,Scope:"read-only",Note:"unmanaged device — least-privilege grant"}`)
- Red because: `report.Policies` undefined → compile error
- Run: `go test ./internal/report/ -run TestReport_Policies -count=1 -v`

| name | give | want |
|---|---|---|
| two policies | the slice above | header `ID PRIORITY PILLAR ACTION NOTE`; P01 row: `10`, `identity`, `STEPUPMFA`, full note; P04 row before-order after P01 (input order); **no SCOPE column** — `read-only` absent from output |

### TestRun_Policies (main)
- Target: `main`, `TestRun_Policies`
- Shape: table-driven; red because behavioral (exit-70 stub)
- Run: `go test . -run TestRun_Policies -count=1 -v`

| name | args | wantExit | stdout contains |
|---|---|---|---|
| lists the pack | `policies` | 0 | header; all of `P01`…`P11` (11 IDs, count them); `identity`, `devices`, `data`, `applications` pillars; `all signals clean — explicit allow` note |
| extra arg | `policies x` | 64 | usage |

`TestRun_Dispatch`: delete the "policies is stubbed" row.

## T5.5 — exit-code sweep + demo fixture (expected 0 red — T1.5/T2.4/T3.3/T4.3 precedent)

### TestRun_ExitCodes (main)
- Target: `main`, `TestRun_ExitCodes`
- Shape: one table locking §5's exit-code map; decide fixtures built in a temp
  dir (happy/byod/stolen/attacker single-signal files). A red here is a wiring
  bug in T5.2–T5.4 — fix the root cause, never the matrix
- Run: `go test . -run TestRun_ExitCodes -count=1 -v`

| command | wantExit |
|---|---|
| `decide <happy>` / `decide <byod>` / `decide <stolen>` / `decide <attacker>` | 0 / 1 / 2 / 3 |
| `decide` (no input) · `decide --format yaml <happy>` | 64 · 64 |
| `decide scenarios/attacker.json` (§5 pack-rejection rule) · `decide /nope` | 65 · 65 |
| `run scenarios` · `run <temp failing pack>` | 0 · 1 |
| `run` (no args) · `run ./nope` | 64 · 65 |
| `policies` · `policies x` | 0 · 64 |
| `audit` (still the Phase 6 stub — keeps the map complete) | 70 |
| `--version` | 0 |

Then: create `examples/attacker-signals.json` (copy the `signal` object from
`scenarios/attacker.json` verbatim into `[ ]`), tick spec §10 Phase 5 boxes, and
record the demo session in the run log:

```
make build && ./zt-sim decide examples/attacker-signals.json ; echo $?
# USER/ACTION/REASONS table, 7 reasons (P01 P02 P03 P04 P06 P07 P10),
# footer "pillars: identity, devices, data, applications", exit 3
./zt-sim run scenarios/ && echo $?   # six PASS lines + PACK summary, exit 0
```

---

# Phase 4 — Scenarios & acceptance (complete, archived)

Test file: `internal/scenario/scenario_test.go`, package `scenario_test`. Named
tests: `TestLoad_Pack`, `TestLoad_SixPacks`, `TestLoad_Invalid`,
`TestCase_Check`, `TestPacks_Expectations` (the acceptance walker),
`TestPacks_ActionCoverage`. Run: `go test ./internal/scenario/ -count=1`.
Full case tables: git history / `plans/pr-phase4.md`.

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
`TestEvaluate_NoSilentPass`, `TestEvaluate_Determinism`. Full case tables: git
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
  asserted exactly. (Phase 5 replaces the stub rows as each subcommand lands.)

## T3 — Makefile
- Test: none. `make test` re-runs T2's suite. Acceptance: `make build &&
  ./zt-sim --version && make vet && make test`, then `make fmt` changes nothing.
