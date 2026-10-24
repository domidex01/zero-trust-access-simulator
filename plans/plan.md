# Plan — zero-trust-access-simulator (Phase 5 — CLI)
> Goal: `zt-sim` works end-to-end per §5 — `decide` (file array or `--stdin`, table or
> JSON, exit 0–3 by strictest action), `run` (per-case results + PACK summary table,
> exit 0/1), `policies` (ID/priority/pillar/action/note table), with the §5 exit-code
> map locked by tests and the demo session producing exit 3.
> Source: spec.md §10 Phase 5 (T5.1–T5.5), §5 (CLI design + exit codes), §8 (layout:
> rendering lives in `internal/report`, wiring in `main.go`).
> Update run: Phases 4/3/2/1/0 complete, ids stable, archived below. Task ids match
> spec §10 verbatim (T5.1–T5.5).
>
> **Spec divergence resolved up front** — §5's example session runs
> `zt-sim decide scenarios/attacker.json`, but §5's own input rule says decide accepts
> only JSON signal arrays and *rejects* scenario packs ("use run"). The normative rule
> wins: the example filename is illustrative, not the pack file. Phase 5 therefore adds
> `examples/attacker-signals.json` — the attacker pack's signal as a decide-legal
> array — so the demo session and the Phase 5 done-when are satisfiable. Feeding the
> *pack* to decide correctly exits 65; T5.5 pins that too. (spec §8 layout grows an
> `examples/` dir; `scenarios/` can't host it — the T4.3 walker Loads every
> `scenarios/*.json` and a signals array would fail it.)

## Status
Phase 5:
- [x] T5.1 `internal/report`: decisions table (USER / ACTION / REASONS + pillars footer) + `--format json`
- [x] T5.2 `decide`: `model.DecodeSignals` (strict array) + subcommand — file or `--stdin`, exit 65 on bad input
- [x] T5.3 `run`: walk dir, per-case PASS/FAIL lines + PACK / CASES / PASS / FAIL / STRICTEST summary, exit 1 on failure
- [x] T5.4 `policies`: ID / PRIORITY / PILLAR / ACTION / NOTE table
- [x] T5.5 Exit-code map locked per §5 (sweep) + `examples/attacker-signals.json` demo fixture

Phase 4 (complete): T4.1 ✅ · T4.2 ✅ · T4.3 ✅ · T4.4 ✅
Phase 3 (complete): T3.1 ✅ · T3.2 ✅ · T3.3 ✅ · T3.4 ✅
Phase 2 (complete): T2.1 ✅ · T2.2 ✅ · T2.3 ✅ · T2.4 ✅
Phase 1 (complete): T1.1 ✅ · T1.2 ✅ · T1.3 ✅ · T1.4 ✅ · T1.5 ✅
Phase 0 (complete): T1 ✅ · T2 ✅ · T3 ✅

## Run log
- T1 ✅ 2026-10-19 — no test (scaffolding) · go build ./... exit 0 · git repo initialized (repo-local identity, global untouched)
- T2 ✅ 2026-10-19 — red: compile error (undefined: run) · green: 1 attempt · 7 cases
- T3 ✅ 2026-10-19 — no test (build wiring) · make build/vet/test/fmt/run-demo all pass · +1-line .gitignore for the built binary
- T1.1 ✅ 2026-10-20 — red: compile error (model.Signal undefined) · green: 1 attempt · 2 cases
- T1.2 ✅ 2026-10-20 — red: compile error (Action.String undefined) · green: 1 attempt · 10 cases · test-plan fix: out-of-range MarshalText row now asserts an error (was vacuous)
- T1.3 ✅ 2026-10-20 — red: compile error (SensitivityRank undefined) · green: 1 attempt · 8 cases
- T1.4 ✅ 2026-10-20 — red: compile error (Validate/DecodeSignal undefined) · green: 1 attempt · 21 cases · test-plan fix: "limits accepted" split into lower/upper rows (LocalHour can't be 0 and 23 at once)
- T1.5 ✅ 2026-10-20 — audit: all §9 bullets already mapped; +1 regression test (UnmarshalText no-state-change, passed immediately) · model 100% stmt coverage · no model bugs found
- review #2 fix ✅ 2026-10-20 — thermonuclear review: JSONText test re-tabled with concrete types (deleted any-dispatch/assertDecision/anyErr), single actionNames table (deleted knownAction) · net −50 lines · 43 subtests + coverage identical
- T2.1 ✅ 2026-10-20 — red: compile error (Policy/New undefined) · green: 1 attempt · 4 cases (3 sort + 1 clone)
- T2.2 ✅ 2026-10-20 — red: compile error (Engine.Evaluate undefined) · green: 1 attempt · 6 cases
- T2.3 ✅ 2026-10-20 — red: behavioral (zero-match returned Allow) · green: 1 attempt · 3 cases
- T2.4 ✅ 2026-10-20 — sweep: 0 red (gap-closing, T1.5 precedent; no engine bugs) · 5 scenario shapes + 6 strictest-wins + 5+3 no-silent-pass + 5 determinism · engine 100% stmt coverage · §9 audit: all bullets + 4 properties mapped to named tests
- review #3 fix ✅ 2026-10-20 — thermonuclear review: 3 findings, all minor, all fixed (cmp.Or two-key comparator + builtin max in Evaluate; fail-closed table deduped into shared failClosedCases, killing the parallel names slice + duplicated F01 literal) · net −9 lines · 36 subtests + coverage identical
- T3.1 ✅ 2026-10-20 — red: build failure (no non-test Go files in internal/policies — Pack undefined) · green: 1 attempt · 10 cases
- T3.2 ✅ 2026-10-20 — red: behavioral (policyByID("P11") Fatalf: policy P11 not in pack) · green: 1 attempt · 11 subtests
- T3.3 ✅ 2026-10-20 — sweep: 0 red (gap-closing, T1.5/T2.4 precedent; no pack bugs) · 18 boundary rows + 6 §6 scenario shapes
- T3.4 ✅ 2026-10-20 — sweep: 0 red (no transcription bugs; pack + table both §4-verbatim, spec diff-checked before T3.1) · policies 100% stmt coverage · §9 audit: fire/silent → FireSilence+CatchAllExclusive; boundaries → Boundaries; IDs/priorities/catch-all-once/pillars → Metadata; §6 rows → ScenarioShapes — all four bullets + all eleven §4 rows mapped
- review #4 fix ✅ 2026-10-20 — thermonuclear review: 1 minor finding fixed (P11 hand-rolled any-match loop → slices.ContainsFunc) + 1 nit (validPillars moved above its use) · net −3 lines · 45 subtests + coverage identical
- T4.1 ✅ 2026-10-24 — red: build failure (no non-test Go files in internal/scenario — scenario.Load undefined, T3.1 precedent) · green: 1 attempt (after fixing 2 test-side compile bugs: strings.Replace count arg; unknown-case-field row appended outside the case object → test-plan row corrected) · 1 + 13 + 7 subtests · scenario 100% stmt coverage
- T4.2 ✅ 2026-10-24 — red: behavioral (all six Load: no such file or directory) · green: 1 attempt · 6 subtests · note-scenario signal = §6 example verbatim, others = TestPack_ScenarioShapes pins
- T4.3 ✅ 2026-10-24 — sweep: 0 red (gap-closing, T1.5/T2.4/T3.3 precedent; no divergence) · 6 expectation subtests + coverage · glob guard: ≥6 files + six pinned names (empty glob fails, never passes vacuously)
- T4.4 ✅ 2026-10-24 — hand-derivation vs §4 (test-plan table), all six agree, 0 divergences: happy→P11 only→allow · note-scenario→P01,P02,P03,P04,P06,P08 fire (P05 not-enrolled, P07 65≤70, P09 finance, P10 finance silent)→block · stolen-credential→P01,P04,P08 (P02/P03 familiar, P06 internal, P10 non-payroll silent)→stepUpMFA · insider→P10 only (P06 internal∧compliant silent)→block · byod→P04 only (P06 internal, P01 mfa-on silent)→limited read-only · attacker→P01,P02,P03,P04,P06,P07,P10 (P05 unmanaged, P08 85>70, P09 guest silent)→block, 7 matched = §5 example footer · spec §10 Phase 4 boxes ticked
- review #5 fix ✅ 2026-10-24 — thermonuclear review (APPROVE): 3 nits fixed (no-op `model.Action` casts → plain consts; ReasonsContain field comment reasons→decision's Matched; happy/insider expect blocks normalized to the multi-line pack format) · net +6 lines (JSON cosmetics only) · 45 subtests + coverage identical
- T5.1 ✅ 2026-10-24 — red: build failure (no non-test Go files in internal/report — T3.1/T4.1 precedent) · green: 1 attempt · 5 + 3 subtests · test-plan fix ×2: continuation-line indices off by one in attacker-row + stack rows (pre-red and first-green; production output was correct both times, fixture misindexed lines[2]/[3] as lines[1]/[2])
- T5.2 ✅ 2026-10-24 — red: compile error (undefined: model.DecodeSignals) + behavioral (decide still exit-70 stub) · green: 1 attempt · 6 + 14 subtests + DecideFormatJSON · run gains stdin io.Reader param (main passes os.Stdin) · mid-green fix: signal.go edit accidentally replaced DecodeSignal, restored immediately
- T5.3 ✅ 2026-10-24 — red: compile error (undefined: report.PackResult) + behavioral (run stub 70) · green: 1 attempt · 1 + 7 subtests · test-plan fix: PASS/FAIL assertions matched the summary header cells (Count("PASS ")==7, Contains("FAIL")) → per-line HasPrefix counting, header cells excluded
- T5.4 ✅ 2026-10-24 — red: compile error (undefined: report.Policies) + behavioral (policies stub 70) · green: 1 attempt · 1 + 2 subtests · report imports engine (pure pkg) for the Policies table
- T5.5 ✅ 2026-10-24 — sweep: 0 red (T1.5/T2.4/T3.3/T4.3 precedent; no wiring bugs) · 16 matrix rows, all §5 codes incl. audit-stub 70 · examples/attacker-signals.json = pack signal verbatim (python assert, no re-typing) · demo session run: decide → 7 reasons, pillars identity/devices/data/applications, exit 3; run scenarios → 6 PASS + summary, exit 0 · coverage: model/engine/policies/scenario 100%, report 96.6%, main 93.2% · spec §10 Phase 5 boxes ticked
- Phase 5 complete — §5 example session satisfied under the documented divergence resolution (examples/ fixture for the illustrative filename; decide scenarios/attacker.json → 65 pinned as the normative rule)
- review #6 fix ✅ 2026-10-24 — thermonuclear review (APPROVE): 4 findings, all minor, all fixed — cmdDecide's 4× usage pair → one guard switch; strictest int-tracked → model.Action (cast once at return, §2 comment); cmdRun's double-meaning e → eng; +TestExamples_AttackerMatchesPack pinning the demo fixture to its pack (drift guard for T7.3) · rejected with reasons: generic table writer (stringly rows), Row{User string} (less self-describing), DecodeSignal/DecodeSignals unification (different wire shapes), exit-code consts (spec's own vocabulary) · stray working-tree revert of test-plan.md restored from HEAD (uncommitted, no history impact)
(appended by execute-plan)

## Not building (YAGNI, Phase 5)
- Box-drawing borders + cell truncation from the §5 sketch — §5 says "exact table
  rendering is up to the report package; the columns and content are the contract";
  `text/tabwriter` gives aligned columns for free — trigger: README demo wants the
  boxed look (Phase 7 T7.3)
- `run --format json` — §5 lists `--format` on run but never defines its payload;
  trigger: a consumer of machine-readable run results
- `--audit` flag on decide/run — inert until the AuditWriter exists; lands with the
  Phase 6 T6.2 wiring (spec T5.x never mentions it)
- SCOPE column in the decide table — §5 pins USER / ACTION / REASONS; scope rides in
  `--format json` — trigger: table-only consumers need it
- Machine markers / colors in run output — plain `PASS`/`FAIL` lines carry it; exit
  code is the machine interface — trigger: CI tooling wants parseable run output
  (then build `run --format json`, not colors)
- Trailing-content rejection after the signals array / pack value — single-Decode
  precedent (T4.1 not-building) — trigger: multi-document input
- An `internal/cli` package split — spec §8 pins dispatch + flags + exit codes in
  `main.go`; four subcommands ≈ 300 lines is fine — trigger: main.go hurts to navigate
- More `examples/` fixtures beyond attacker-signals.json — one demo file; trigger:
  README (T7.3) needs more transcripts

## File map (Phase 5 touches only these)
- `internal/report/report.go` (new) + `report_test.go` (new) — `Row`, `Decisions`,
  `DecisionsJSON` (T5.1); `PackResult`, `PackSummary` (T5.3); `Policies` (T5.4)
- `internal/model/signal.go` (edit) + `model_test.go` (edit) — `DecodeSignals` (T5.2)
- `main.go` (edit) + `main_test.go` (edit) — `run` gains an `stdin io.Reader` param,
  `cmdDecide` (T5.2), `cmdRun` (T5.3), `cmdPolicies` (T5.4); `audit` stays stubbed
- `examples/attacker-signals.json` (new) — T5.5 demo fixture
- Untouched: `internal/{engine,policies,scenario}`, `scenarios/`, `Makefile`

## Tasks

### T5.1 — report: decisions table + JSON
- Does: `internal/report/report.go` — `Row{Signal model.Signal; Decision
  model.Decision}` (the USER column comes from the signal, everything else from the
  decision), `Decisions(w io.Writer, rows []Row) error`: `text/tabwriter`
  (minwidth 0, tabwidth 0, padding 2, `' '`) writing header `USER\tACTION\tREASONS`,
  one line per reason (first on the row line, continuations as `\t\t<reason>` so
  columns align), action as `strings.ToUpper(d.Action.String())`, then Flush and a
  final `pillars: <unique pillars across all rows, comma-space, first-appearance
  order>` footer line (§5: columns + content are the contract, not the box art);
  `DecisionsJSON(w, decisions []model.Decision) error`: `json.NewEncoder` +
  `SetIndent("", "  ")` — §5 "`--format json` emits a machine-readable `[]Decision`".
  Empty rows never occur (decide rejects empty arrays in T5.2) — no special case
- Files: `internal/report/report.go` (new), `internal/report/report_test.go` (new)
- Depends: —
- Acceptance: `go test ./internal/report/ -count=1` exits 0
- Est: M

### T5.2 — `decide`: strict array/stdin decode + wiring
- Does: `model.DecodeSignals(r io.Reader) ([]Signal, error)` next to `DecodeSignal`
  — same strict decoder (`DisallowUnknownFields`), decodes `[]Signal`, validates each,
  rejects the empty array (`empty signals array` — nothing to decide is an input
  error, never a vacuous exit 0; a lone object or a §6 pack fails as `cannot
  unmarshal object into []model.Signal`, which *is* §5's "packs are rejected there"
  rule). `main.go`: `run` gains `stdin io.Reader` (main passes `os.Stdin`) so
  `--stdin` is testable; `cmdDecide(args, stdin, stdout, stderr) int` with
  `flag.NewFlagSet("decide", flag.ContinueOnError)` (output → stderr; **never**
  ExitOnError — the CLI owns exit codes), flags `--stdin` and `--format
  table|json` (default table; anything else → 64 after parse). Exactly one input
  source (`--stdin` xor one positional path; both or neither → 64). File path:
  `os.Open` → `DecodeSignals`; `--stdin`: `model.DecodeSignal(stdin)`. One
  `engine.New(policies.Pack())`, evaluate all, print via `report.Decisions` /
  `report.DecisionsJSON`, exit = strictest action as int (`max` over `int(d.Action)`
  — §2: exit code = severity). Input errors → stderr (path-prefixed) + 65; usage →
  64. `TestRun_Dispatch`'s "decide is stubbed" row is deleted (replaced by
  `TestRun_Decide`; the run/policies/audit stub rows stay until their tasks)
- Files: `internal/model/signal.go` (edit), `internal/model/model_test.go` (edit),
  `main.go` (edit), `main_test.go` (edit)
- Depends: T5.1
- Acceptance: `go test ./internal/model/ -run TestDecodeSignals -count=1 && go test . -run 'TestRun_Decide|TestRun_Dispatch' -count=1` exit 0
- Est: M

### T5.3 — `run`: pack walker + summary table
- Does: `report.PackResult{Name string; Cases, Pass, Fail int; Strictest
  model.Action}` + `report.PackSummary(w, results []PackResult) error` — same
  tabwriter style, header `PACK\tCASES\tPASS\tFAIL\tSTRICTEST`, strictest action
  uppercased. `cmdRun(args, stdout, stderr) int`: exactly one dir arg (else 64);
  `os.ReadDir` (already sorted), keep top-level `*.json` only; zero packs found →
  65 (`no scenario packs in <dir>` — an empty suite must fail loudly); per file
  `scenario.Load` — first error aborts → 65 (stderr names the file); per case
  `Evaluate` + `Check`, print `PASS <pack>/<case>` or `FAIL <pack>/<case>:
  <check error>` per line, collect `PackResult` (Strictest = max action across the
  pack's decisions); `report.PackSummary` last; exit 0 all pass, 1 otherwise.
  `TestRun_Dispatch`'s "run is stubbed" row deleted (replaced by `TestRun_Run`)
- Files: `internal/report/report.go` (edit), `internal/report/report_test.go` (edit),
  `main.go` (edit), `main_test.go` (edit)
- Depends: T5.1
- Acceptance: `go test ./internal/report/ -run TestReport_PackSummary -count=1 && go test . -run 'TestRun_Run|TestRun_Dispatch' -count=1` exit 0
- Est: M

### T5.4 — `policies`: pack listing
- Does: `report.Policies(w io.Writer, policies []engine.Policy) error` (report
  imports engine — it's pure; a mapping struct would be ceremony) — tabwriter table,
  header `ID\tPRIORITY\tPILLAR\tACTION\tNOTE`, action uppercased, the five §5
  columns and no scope column. `cmdPolicies(args, stdout, stderr) int`: no extra
  args (else 64), print `report.Policies(w, policies.Pack())` (slice is already
  (Priority, ID)-ordered per its doc), exit 0. `TestRun_Dispatch`'s "policies is
  stubbed" row deleted (replaced by `TestRun_Policies`)
- Files: `internal/report/report.go` (edit), `internal/report/report_test.go` (edit),
  `main.go` (edit), `main_test.go` (edit)
- Depends: T5.1
- Acceptance: `go test ./internal/report/ -run TestReport_Policies -count=1 && go test . -run 'TestRun_Policies|TestRun_Dispatch' -count=1` exit 0
- Est: S

### T5.5 — exit-code map sweep + demo fixture
- Does: `TestRun_ExitCodes` in `main_test.go` locking the §5 table as one matrix —
  decide 0/1/2/3 via four temp fixture files + 64 (usage ×2) + 65 (bad input ×2,
  incl. `decide scenarios/attacker.json` → 65, pinning the §5 pack-rejection rule);
  run 0 (`scenarios/`) / 1 (failing temp pack) / 64 / 65; policies 0 / 64; `audit`
  still stubbed 70 (Phase 6 replaces it — the row keeps the map complete and makes
  the Phase 6 diff visible); `--version` 0. Sweep per the T1.5/T2.4/T3.3/T4.3
  precedent: 0 red expected; a red is a real wiring bug — fix the root cause, never
  the matrix. Then create `examples/attacker-signals.json` = the `signal` object of
  `scenarios/attacker.json`'s case verbatim, wrapped in `[ ]` (no re-typing — copy
  from the pack), tick the spec §10 Phase 5 boxes, and record the demo session in
  the run log
- Files: `main_test.go` (edit), `examples/attacker-signals.json` (new),
  `plans/plan.md` (run log), `spec.md` (ticks)
- Depends: T5.2, T5.3, T5.4
- Acceptance: `go test . -run TestRun_ExitCodes -count=1 && go test ./... -count=1 && go vet ./...` exit 0, `gofmt -l .` prints nothing, and `make build && ./zt-sim decide examples/attacker-signals.json; echo $?` prints the table (7 reasons, `pillars: identity, devices, data, applications`) and `3`
- Est: S

## Done when (Phase 5 exit criteria)
```
go test ./... -count=1      # exit 0 — report, model.DecodeSignals, main wiring all green
go vet ./...                # exit 0
gofmt -l .                  # prints nothing
make build
./zt-sim decide examples/attacker-signals.json   # USER/ACTION/REASONS table, 7 reasons, 4-pillar footer
echo $?                                           # 3  (§5 session, mirrored — see divergence note above)
./zt-sim run scenarios/                           # exit 0, six PASS lines + PACK summary
```
Plus: spec §10 Phase 5 "Done when" satisfied under the documented divergence
resolution (examples/ fixture stands in for the illustrative pack filename).

---

# Phase 4 — Scenarios & acceptance (complete, archived)

> Goal: the six §6 scenario packs exist on disk and run as acceptance tests —
> `internal/scenario` loads + validates packs and checks expectations, and
> `go test ./...` walks `scenarios/*.json` green.
> All tasks ✅ 2026-10-24 (run log above); full task detail in git history /
> `plans/pr-phase4.md`.

- T4.1 `internal/scenario` (engine-free, imports only model): `Pack`/`Case`/
  `Expectation` (`Action *model.Action` — pointer so "required" beats the Allow zero
  value), `Load` = ReadFile → strict decode → `validate()` (name → ≥1 case → per
  case: name → `Signal.Validate` → `expect.action` non-nil → scope ⇒ limited),
  errors path-prefixed; `Case.Check(d)`: action → every `reasonsContain` ID ∈
  `d.Matched` → scope only when set. scenario at 100% stmt coverage
- T4.2 the six packs authored from §6 (signals = TestPack_ScenarioShapes pins,
  note-scenario = §6 example verbatim); `Name` == file stem
- T4.3 `TestPacks_Expectations` walker (glob guard: ≥6 files + six pinned names, so
  an empty glob fails) + `TestPacks_ActionCoverage` — wired into `go test ./...`
- T4.4 hand-derivation of all six vs §4 matchers, 0 divergences (table in the
  Phase 4 test-plan revision, git history)

Files: `internal/scenario/scenario.go`, `internal/scenario/scenario_test.go`,
`scenarios/{happy,note-scenario,stolen-credential,insider,byod,attacker}.json`.

---

# Phase 3 — Policy pack (complete, archived)

> Goal: `internal/policies` complete and tested — the v1 pack P01–P11 exactly as §4.
> All tasks ✅ 2026-10-20 (run log above); full task detail in git history /
> `plans/pr-phase3.md`.

- T3.1 `Pack()` single export surface; P01–P10 matchers as one-line closures,
  formulas pinned in test-plan, Notes verbatim from §4 (incl. em dashes)
- T3.2 P11 catch-all (Allow, priority 90): `slices.ContainsFunc` over the ten core
  `When`s — matches iff none did, so `Matched`/`Reasons` never carry a false
  all-clear next to a real finding
- T3.3 sweep: 18 boundary rows (risk 70/71/40/39, intern hour 4/5/0 + role guard,
  sensitivity × device posture, P04/P05 disjointness, P02-without-P03) + the six
  §6 scenario shapes through the real pack (exact Action/Scope/Matched/Pillars)
- T3.4 sweep: §4 metadata pinned positionally (ID/Priority/Pillar/Then/Scope/Note),
  11 policies, unique IDs, exactly one Allow = P11, scope iff Limited, pillars
  valid; policies at 100% statement coverage

Files: `internal/policies/policies.go`, `internal/policies/policies_test.go`.

---

# Phase 2 — Engine (complete, archived)

> Goal: `internal/engine` complete and tested — a pure evaluator where sorted policies in
> yield a deterministic Decision out, strictest match wins, and every path fails closed.
> All tasks ✅ 2026-10-20 (run log above); full task detail in git history /
> `plans/pr-phase2.md`.

- T2.1 `Policy` verbatim from §2 + `Engine`/`New` cloning and pre-sorting by
  (Priority, ID) via `slices.SortFunc` + `cmp.Or` (`engine.go`, white-box sort tests)
- T2.2 `Evaluate`: every policy evaluated (no short-circuit), `"<ID>: <Note>"` reasons,
  unique pillars in evaluation order, numeric-max severity, first-Limited scope kept
  only when Limited wins
- T2.3 Fail-closed: zero matches / empty pack → `Block` with the pinned §3.5 engine
  reasons (em dash included)
- T2.4 §9 sweep: five scenario shapes on a synthetic `miniPack()`, strictest-wins,
  no-silent-pass, determinism (double-eval + reordered-`New`); engine 100% stmt coverage

Files: `internal/engine/engine.go`, `internal/engine/engine_test.go` (black-box),
`internal/engine/engine_internal_test.go` (white-box `New` sort).

---

# Phase 1 — Domain model (complete, archived)

> Goal: `internal/model` complete and tested — every §2 rule enforced at load, JSON tags
> exact, all four actions marshal as spec text. All tasks ✅ 2026-10-20 (run log above);
> full task detail in git history / `plans/pr-phase1.md`.

- T1.1 `Signal` + `Decision` structs with exact §2 JSON tags + bare `Action` type
  (`signal.go`, `decision.go`, `action.go`)
- T1.2 `Action` behavior: `String()`, `MarshalText`/`UnmarshalText`, severity = iota
  order (asserted numerically, no extra API)
- T1.3 `SensitivityRank`: public < internal < confidential < high, case-sensitive,
  error names the value
- T1.4 `Signal.Validate()` (first error, spec-table order) + `DecodeSignal`
  (`DisallowUnknownFields` + Validate)
- T1.5 §9 completeness sweep — model at 100% statement coverage; review #2 re-tabled
  tests with concrete types (net −50 lines, 43 subtests)

Files: `internal/model/{signal,action,decision}.go`, `internal/model/model_test.go`.

---

# Phase 0 — Scaffold (complete, archived)

> Goal: a building, vet-clean skeleton — `zt-sim --version` prints, bad usage exits 64,
> `go build ./...` / `go vet ./...` / `make test` all pass.
> Task mapping: T1 ↔ spec T0.1, T2 ↔ spec T0.2, T3 ↔ spec T0.3.

## Tasks

### T1 — Init module, scaffold directories, git repo
- Did: `go mod init zt-sim` (go 1.27, zero deps); created `internal/{model,engine,policies,scenario,report}`, `scenarios/`; `git init` (no repo existed up the tree)
- Files: `go.mod`, scaffold dirs
- Test: none — scaffolding only

### T2 — main.go: testable dispatch with `--version` and usage exits
- Did: `run(args []string, stdout, stderr io.Writer) int` — `--version` prints
  `zt-sim version 0.1.0` (exit 0); no args / unknown subcommand → usage on stderr
  (exit 64); four subcommands wired to `not implemented yet` stubs (exit 70, EX_SOFTWARE)
- Files: `main.go`, `main_test.go`
- Test: `TestRun_Dispatch`, 7 cases (table-driven)

### T3 — Makefile: build / test / fmt / vet / run-demo
- Did: five phony targets, stdlib tooling only; `run-demo` is a stub (`./zt-sim --version`)
  that Phase 7 (T7.4) expands
- Files: `Makefile`, `.gitignore` (built binary)
- Test: none — covered by T2's suite via `make test`
