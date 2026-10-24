# Plan — zero-trust-access-simulator (Phase 4 — Scenarios & acceptance)
> Goal: the six §6 scenario packs exist on disk and run as acceptance tests —
> `internal/scenario` loads + validates packs and checks expectations, and
> `go test ./...` walks `scenarios/*.json` green.
> Source: spec.md §10 Phase 4 (T4.1–T4.4), §6 (pack schema + the six packs),
> §4 (hand-derivation source for T4.4).
> Update run: Phases 3/2/1/0 complete, ids stable, archived below. Task ids
> match spec §10 verbatim (T4.1–T4.4).

## Status
Phase 4:
- [x] T4.1 `internal/scenario`: pack loader + validator (§6 schema) + `Case.Check`
- [x] T4.2 Author the six packs from §6 with `expect` blocks (§6 tables = source of truth)
- [x] T4.3 `scenario_test.go` walks `../../scenarios/*.json`, asserts every expectation
- [x] T4.4 Hand-verify each pack's outcome against §4 once (derivation in run log)

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
(appended by execute-plan)

## Not building (YAGNI, Phase 4)
- The `run` subcommand — dir walking, per-pack summary table, exit codes are Phase 5
  T5.3; `Load` takes one path, and the T4.3 walker lives in the test until then
- An engine dependency inside `internal/scenario` — `Case.Check` takes the
  `model.Decision`, so the package imports only model; Phase 5 wires engine +
  policies + scenario together in the CLI
- `expect.pillars` / exact `matched`-set equality — §6 pins action +
  `reasonsContain` (⊇ semantics) + scope-on-limited only; exact matched sets are
  already pinned in `TestPack_ScenarioShapes`
- Required `description` validation — §6 marks only `expect.action` required;
  name/cases non-empty is loader sanity, description stays optional metadata
- Rejecting trailing content after the pack value — matches the `DecodeSignal`
  precedent (single Decode); trigger: the CLI grows multi-document input
- Schema/version field, multi-pack files — one pack per file per §6

## File map (Phase 4 creates all of these; nothing else changes)
- `internal/scenario/scenario.go` — `Pack`/`Case`/`Expectation`, `Load`,
  unexported `validate`, `Case.Check`
- `internal/scenario/scenario_test.go` — package `scenario_test`, black-box:
  loader/check unit tests + the acceptance walker
- `scenarios/{happy,note-scenario,stolen-credential,insider,byod,attacker}.json` —
  the six packs, content pinned in test-plan
- Untouched: `internal/{model,engine,policies}`, `main.go`, `Makefile`

## Tasks

### T4.1 — loader + validator + `Case.Check`
- Does: `internal/scenario/scenario.go` — the §6 schema as
  `Pack{Name, Description, Cases []Case}` / `Case{Name, Signal, Expect}` /
  `Expectation{Action *model.Action, ReasonsContain []string, Scope string}`
  (Action is a pointer so "required" is distinguishable from the zero value Allow).
  `Load(path)`: ReadFile → strict decode (`DisallowUnknownFields`, the `DecodeSignal`
  precedent) → unexported `validate()` — name non-empty → ≥1 case → per case: name
  non-empty, `Signal.Validate()`, `expect.action` non-nil, scope ⇒ action limited;
  errors prefixed with the path so `run` failures are debuggable. `Case.Check(d)
  error`: action equality → every `reasonsContain` ID present in `d.Matched` →
  scope equality only when set (§6: scope is optional, checked when action is
  limited)
- Files: `internal/scenario/scenario.go` (new), `internal/scenario/scenario_test.go` (new)
- Depends: —
- Acceptance: `go test ./internal/scenario/ -run 'TestLoad_Pack|TestLoad_Invalid|TestCase_Check' -count=1` exits 0
- Est: M

### T4.2 — author the six packs
- Does: write `scenarios/*.json` exactly per the test-plan content table — signals
  are the `TestPack_ScenarioShapes` pins verbatim (note-scenario's signal = the §6
  example JSON verbatim), `reasonsContain` = the §6 matched-policies column in full,
  descriptions = the §6 Story column. Test: `TestLoad_SixPacks` — each of the six
  filenames Loads cleanly, `Name` == file stem, ≥1 case
- Files: the six `scenarios/*.json` (new), `internal/scenario/scenario_test.go` (edit)
- Depends: T4.1
- Acceptance: `go test ./internal/scenario/ -run TestLoad_SixPacks -count=1` exits 0
- Est: S

### T4.3 — acceptance walker
- Does: `TestPacks_Expectations` — glob `../../scenarios/*.json`; guard ≥6 files
  found AND all six pinned names present (an empty glob must fail the test, not pass
  vacuously); per pack, per case (`t.Run "pack/case"`):
  `engine.New(policies.Pack()).Evaluate(sig)` → `Check` returns nil.
  `TestPacks_ActionCoverage` — all four actions appear across the packs (§6's
  "every action is exercised by at least one scenario" claim). Sweep per the
  T1.5/T2.4/T3.3 precedent: 0 red expected; a red is a real divergence — fix the
  pack file, or, if the §6 table itself is wrong, fix spec.md + file and say so in
  the run log
- Files: `internal/scenario/scenario_test.go` (edit only)
- Depends: T4.2
- Acceptance: `go test ./internal/scenario/ -count=1 && go test ./... -count=1` exit 0
- Est: S

### T4.4 — hand-verify each pack against §4
- Does: re-derive all six outcomes from the §4 matchers by hand (no code) — which
  policies fire and why each silent one stays silent (derivation table is in
  test-plan); compare with each pack's `expect`; record the result in the run log.
  Any disagreement → fix per the T4.3 rule before ticking. Then tick the spec §10
  Phase 4 checkboxes
- Files: `plans/plan.md` (run log), `spec.md` (ticks) — no production change
- Depends: T4.3
- Acceptance: derivation recorded in run log; `go test ./... -count=1 && go vet ./...` exit 0 and `gofmt -l .` prints nothing
- Est: S

## Done when (Phase 4 exit criteria)
```
go test ./... -count=1      # exit 0 — model, engine, policies, main + the six packs as acceptance tests
go vet ./...                # exit 0
gofmt -l .                  # prints nothing
```
Plus: spec §10 "Done when: `go test ./...` runs all packs green" satisfied.

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
