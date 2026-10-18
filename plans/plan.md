# Plan — zero-trust-access-simulator (Phase 3 — Policy pack)
> Goal: `internal/policies` complete and tested — the v1 pack P01–P11 exactly as §4:
> every matcher, priority, pillar, scope and verbatim audit note, a catch-all that fires
> only when nothing else does, and the six §6 scenario shapes derivable from the pack
> as an executable test (before Phase 4 authors the JSON packs).
> Source: spec.md §10 Phase 3 (T3.1–T3.4), §4 (starter policy pack), §6 (scenario table).
> Update run: Phases 2/1/0 complete, ids stable, archived below. Task ids match spec §10
> verbatim (T3.1–T3.4).

## Status
Phase 3:
- [x] T3.1 P01–P10 matchers in `Pack()`; Notes verbatim (structurally pinned by T3.4)
- [x] T3.2 P11 catch-all: matches iff no P01–P10 matched
- [x] T3.3 Sweep: boundaries (risk 70/71/40/39, hour 4/5/0, confidential/internal) + the six §6 scenario shapes against the real pack
- [x] T3.4 Sweep: pack sanity — §4 metadata verbatim, IDs unique, catch-all exactly once, pillars valid

Phase 2 (complete): T2.1 ✅ · T2.2 ✅ · T2.3 ✅ · T2.4 ✅
Phase 1 (complete): T1.1 ✅ · T1.2 ✅ · T1.3 ✅ · T1.4 ✅ · T1.5 ✅
Phase 0 (complete): T1 ✅ · T2 ✅ · T3 ✅

## Run log
- T1 ✅ 2026-10-19 — no test (scaffolding) · go build ./... exit 0 · git repo initialized (repo-local identity, global untouched)
- T2 ✅ 2026-10-19 — red: compile error (undefined: run) · green: 1 attempt · 7 cases
- T3 ✅ 2026-10-19 — no test (build wiring) · make build/vet/test/fmt/run-demo all pass · +1-line .gitignore for the built binary
- T1.1 ✅ 2026-10-20 — red: compile error (model.Signal undefined) · green: 1 attempt · 2 cases
- T1.2 ✅ 2026-10-20 — red: compile error (Action.String undefined) · green: 1 attempt · 10 cases · test-plan fix: out-of-range MarshalText row now asserts an error (was vacuous)
- T1.3 ✅ 2026-10-20 — red: compile error (model.SensitivityRank undefined) · green: 1 attempt · 8 cases
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
(appended by execute-plan)

## Not building (YAGNI, Phase 3)
- Policy pack as JSON/YAML config — policies live in code per §4; trigger: rule set
  outgrows code (post-v1 stretch §11)
- `networks`/`infrastructure` policies — v1 has none (§4); trigger: §11 stretch goal
- Exported per-policy vars — `Pack()` is the single surface; the Phase 5 `policies`
  subcommand iterates `Pack()`
- Rewiring engine tests from `miniPack()` to the real pack — the synthetic fixture keeps
  the engine suite independent of pack content
- Engine changes — Phase 2 contract is frozen; if a pack need surfaces one, that's a
  Phase 2 bugfix commit, separate from any T3.x
- P06 invalid-sensitivity policy (err → block?) — unreachable: `DecodeSignal` validates
  at the trust boundary; matcher just no-matches on error
- Benchmarks/allocation tests — eleven one-line matchers
- A cached/singleton `Engine` over the pack — Phase 5 decides where the Engine lives

## File map (Phase 3 creates all of these; nothing else changes)
- `internal/policies/policies.go` — `Pack()`, the eleven policies, matchers inline
- `internal/policies/policies_test.go` — package `policies_test`, black-box via `Pack()`
- Untouched: `internal/model`, `internal/engine`, `main.go`, `Makefile`, `scenarios/`

## Tasks

### T3.1 — P01–P10 matchers, `Pack()` surface
- Does: `internal/policies/policies.go` — package doc, `func Pack() []engine.Policy`
  returning the ten core policies as composite literals, one-line closures for `When`
  (formulas pinned in test-plan). `P06` uses `model.SensitivityRank` with an
  `err == nil &&` guard plus `const rankConfidential = 2` (the error path is unreachable
  post-`DecodeSignal`; the const avoids discarding the error return). Notes copied
  verbatim from §4 incl. em dashes — their verbatim-ness is asserted structurally by
  T3.4's metadata table, so this task's test stays behavioral. `Pack()` returns the core
  ten only; P11 arrives in T3.2. Test helper: `policyByID(t, id) engine.Policy` scanning
  `Pack()` — matchers are tested through the exported pack, no white-box file needed
- Files: `internal/policies/policies.go` (new), `internal/policies/policies_test.go` (new)
- Depends: —
- Acceptance: `go test ./internal/policies/ -run TestPack_FireSilence -count=1` exits 0
- Est: M

### T3.2 — P11 catch-all: matches iff nothing else did
- Does: extend `Pack()` to append P11 (Allow, priority 90, pillar identity, note
  `all signals clean — explicit allow`) whose `When` loops over the ten core policies and
  returns false on the first match — §4's "catch-all: no P01–P10 matched", literally.
  Why exclusive rather than always-true: with strictest-wins the *decision* would be
  identical either way, but `Matched`/`Reasons` must never carry a false "all clear"
  alongside a real finding (§6 expects P11 absent whenever others match). Red test is
  behavioral: `policyByID(t, "P11")` Fatalfs while the pack holds only ten
- Files: `internal/policies/policies.go` (edit), `internal/policies/policies_test.go`
  (edit — reuses T3.1's shared `triggers` table)
- Depends: T3.1
- Acceptance: `go test ./internal/policies/ -run TestPack_CatchAllExclusive -count=1` exits 0
- Est: S

### T3.3 — Sweep: boundaries + the six §6 scenario shapes
- Does: `TestPack_Boundaries` — rows of (signal, checks map[policyID]bool) covering
  risk 70/71/40/39, intern hour 4/5/0 and the role guard, confidential/high/internal ×
  device posture for P06, P04/P05 disjointness across the three reachable device states,
  P02-without-P03 (unfamiliar + MFA, the "P02 is not dead weight" note), P03's `&&`
  guard, P10's role guard. `TestPack_ScenarioShapes` — the six §6 rows (happy,
  noteScenario, stolenCred, insider, byod, attacker) as full signals through
  `engine.New(policies.Pack()).Evaluate`, asserting Action, Scope, Matched, Pillars
  (exact slices). This is the executable form of the phase done-when "every scenario row
  in §6 can be derived by hand from the pack". Gap-closing sweep per the T1.5/T2.4
  precedent: 0 red expected; a red is a real pack bug — fix the pack, not the test
- Files: `internal/policies/policies_test.go` (edit only)
- Depends: T3.2
- Acceptance: `go test ./internal/policies/ -run 'TestPack_Boundaries|TestPack_ScenarioShapes' -count=1`
  exits 0
- Est: M

### T3.4 — Sweep: pack sanity (§4 metadata verbatim)
- Does: `TestPack_Metadata` — pinned §4 table (ID, Priority, Pillar, Then, Scope, Note
  verbatim; `When` is compared behaviorally elsewhere, and `Policy` has a func field so
  it isn't `==`-comparable anyway) plus: `len(Pack()) == 11`, IDs unique, exactly one
  `Allow` and it is P11 ("catch-all present exactly once"), Scope `"read-only"` iff
  `Then == Limited` else `""`, every Pillar ∈ {identity, devices, applications, data,
  networks, infrastructure}. Then audit the §9 policies bullets against named tests.
  0 red expected; a red is a transcription bug — the table is §4 verbatim, so fix the
  pack; if the table itself diverges from §4, fix both and say so in the run log
- Files: `internal/policies/policies_test.go` (edit only)
- Depends: T3.2
- Acceptance: `go test ./internal/policies/ -count=1 && go test ./... -count=1 && go vet ./...`
  all exit 0 AND `gofmt -l .` prints nothing AND every §4 row + §9 policies bullet maps
  to a named test (audit recorded in the run log)
- Est: S

## Done when (Phase 3 exit criteria)
```
go test ./internal/policies/ -count=1         # exit 0 (matchers, catch-all, boundaries, §6 shapes)
go test ./... -count=1 && go vet ./...        # exit 0 (model + engine + main suites still green)
gofmt -l .                                    # prints nothing
```
Plus: `TestPack_ScenarioShapes` green = the six §6 rows derive from the pack.

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
