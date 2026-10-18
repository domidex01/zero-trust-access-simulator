# Plan — zero-trust-access-simulator (Phase 2 — Engine)
> Goal: `internal/engine` complete and tested — a pure evaluator where sorted policies in
> yield a deterministic Decision out, strictest match wins, and every path fails closed.
> Source: spec.md §10 Phase 2 (T2.1–T2.4), §3 (evaluation semantics), §9 (engine suite).
> Update run: Phase 1 (T1.1–T1.5) and Phase 0 (T1–T3) complete, ids stable, archived below.
> Task ids match spec §10 verbatim (T2.1–T2.4).

## Status
Phase 2:
- [x] T2.1 `Policy` struct + `Engine` with `New` pre-sorting by `(Priority, ID)`
- [ ] T2.2 `Evaluate(Signal) Decision`: evaluate all, strictest wins, scope, reason format
- [ ] T2.3 Fail-closed paths: zero matches and empty pack → `Block` with engine reason
- [ ] T2.4 Test sweep: §9 scenario table + four global properties, incl. determinism

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
- T2.1 ✅ 2026-10-20 — red: compile error (Policy/New undefined) · green: 1 attempt · 4 cases (3 sort + 1 clone)


## Not building (YAGNI, Phase 2)
- nil-`When` guard in Evaluate — the pack is authored in code (Phase 3), not external
  data; trigger: policy packs loaded from config (post-v1 stretch)
- Signal re-validation inside Evaluate — the trust boundary is load
  (`model.DecodeSignal`/`Validate`, Phase 1); engine input is already trusted
- Batch `EvaluateAll([]Signal)` — trigger: Phase 5 `decide` loops over its own slice
- Engine error return — a fail-closed `Decision` *is* the failure mode (§3.5); there is
  no error path to return
- Exported order accessor (`Policies()`/`Len()`) — reason order in `Decision` is the
  observable contract; the Phase 5 `policies` subcommand reads the pack from
  `internal/policies` directly
- Scope validation (`Then == Limited` with empty `Scope`) — pack-authoring concern,
  checked by the Phase 3 sanity test
- Concurrency machinery — `Engine` is read-only after `New`; nothing shares it across
  goroutines in v1
- Purity lint (banning I/O imports) — proved behaviorally by the determinism test; the
  package stays three files small

## File map (Phase 2 creates all of these; nothing else changes)
- `internal/engine/engine.go` — `Policy`, `Engine`, `New`, `Evaluate`
- `internal/engine/engine_internal_test.go` — white-box `New` tests (package `engine`;
  the sort is observable only through the unexported slice — precedent: `main_test.go`
  tests unexported `run`)
- `internal/engine/engine_test.go` — the black-box suite (package `engine_test`)
- Untouched: `internal/model`, `main.go`, `Makefile`, `scenarios/`

## Tasks

### T2.1 — `Policy` struct + `Engine` constructor pre-sorted
- Does: `engine.Policy` verbatim from §2 (ID, Priority, Pillar, `When
  func(model.Signal) bool`, Then, Scope, Note — no JSON tags, policies live in code);
  `type Engine struct` with an unexported `policies []Policy` kept sorted by
  (Priority asc, ID asc); `func New(policies []Policy) Engine` — clones the input
  (caller's slice order untouched — sorting in place would be a side effect) and sorts
  via `slices.SortFunc` + `cmp.Compare`/`strings.Compare`. Zero-value `Engine{}` = the
  empty pack (its fail-closed behavior arrives in T2.3)
- Files: `internal/engine/engine.go` (new), `internal/engine/engine_internal_test.go` (new)
- Depends: —
- Acceptance: `go test ./internal/engine/ -run TestNew_ -count=1` exits 0
- Est: S

### T2.2 — `Evaluate`: every policy, strictest wins
- Does: `func (e Engine) Evaluate(s model.Signal) model.Decision` — walk the sorted
  policies, evaluate **every** one (no short-circuit); collect `Reasons` as
  `"<ID>: <Note>"`, `Matched` IDs, and `Pillars` (unique, evaluation order); final
  Action = numeric max among matched actions (severity *is* the iota order — plain `>`
  comparison, per Phase 1's not-building); `Scope` = the first matching `Limited`
  policy's Scope when the final action is `Limited`, `""` otherwise (§3.3 says
  "strictest matching Limited policy"; all Limited matches tie in severity, and the
  total `(Priority, ID)` sort makes "first in evaluation order" the deterministic read).
  Zero-match packs are not handled yet — that's T2.3's red test
- Files: `internal/engine/engine.go` (edit), `internal/engine/engine_test.go` (new —
  shared `base()` signal fixture defined here)
- Depends: T2.1
- Acceptance: `go test ./internal/engine/ -run TestEvaluate_Semantics -count=1` exits 0
- Est: M

### T2.3 — Fail-closed paths
- Does: zero matched policies → `Decision{Action: Block, Reasons: ["engine: no policy
  matched — fail closed (assume breach)"]}` with empty Pillars/Matched; empty pack
  (`New(nil)` or zero-value `Engine{}`) → `Block` with `engine: no policies loaded —
  fail closed`. The engine never defaults to a grant (§3.5). Red test is behavioral:
  T2.2's zero-match path returns the zero-value Decision, i.e. `Allow`
- Files: `internal/engine/engine.go` (edit), `internal/engine/engine_test.go` (edit)
- Depends: T2.2
- Acceptance: `go test ./internal/engine/ -run TestEvaluate_FailClosed -count=1` exits 0
- Est: S

### T2.4 — Test sweep: §9 scenario table + global properties
- Does: add `TestEvaluate_ScenarioShapes` (the five §9 shapes — noteScenario → Block,
  happy → Allow, stolenCred → StepUpMFA, byod → Limited, insider → Block — against a
  synthetic fixture pack `miniPack()`, **not** the real P01–P11, which is Phase 3);
  `TestEvaluate_StrictestWins` (subset table: one always-true policy per action);
  `TestEvaluate_NoSilentPass` (every decision has ≥ 1 reason; Reasons count == Matched
  count when any matched; an `Allow` always cites ≥ 1 matched policy);
  `TestEvaluate_Deterministic` (double-eval → deep-equal decision; `New` over a
  reordered pack → identical decision). Then audit the §9 engine bullets and four
  must-pass properties against named tests. Regression tests may pass immediately — if
  one goes red, that's a real engine bug: fix the engine, not the test. No new
  production code expected
- Files: `internal/engine/engine_test.go` (edit only — `miniPack()` fixture added here)
- Depends: T2.2, T2.3
- Acceptance: `go test ./internal/engine/ -count=1 && go test ./... -count=1 && go vet ./...`
  all exit 0 AND `gofmt -l .` prints nothing AND every §9 engine bullet + property maps
  to a named test in engine_test.go
- Est: M

## Done when (Phase 2 exit criteria)
```
go test ./internal/engine/ -count=1         # exit 0 (incl. property tests)
go test ./... -count=1 && go vet ./...      # exit 0 (model + main suites still green)
gofmt -l .                                  # prints nothing
```

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
