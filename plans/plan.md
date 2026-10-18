# Plan — zero-trust-access-simulator (Phase 1 — Domain model)
> Goal: `internal/model` complete and tested — every §2 rule enforced at load, JSON tags
> exact, all four actions marshal as spec text.
> Source: spec.md §10 Phase 1 (T1.1–T1.5), §2 (domain model), §9 (model test suite).
> Update run: Phase 0 (T1–T3) complete, ids stable, archived below. Task ids match spec
> §10 verbatim (T1.1–T1.5).

## Status
Phase 1:
- [x] T1.1 `Signal` + `Decision` structs, exact §2 JSON tags (incl. bare `Action` type)
- [x] T1.2 `Action` behavior: `String()`, text marshalling, severity order
- [x] T1.3 `SensitivityRank(string) (int, error)`
- [x] T1.4 `Signal.Validate()` (every §2 rule) + strict decoder helper
- [x] T1.5 Test completeness sweep against the §9 model-suite list

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
(appended by execute-plan)

## Not building (YAGNI, Phase 1)
- `Policy` struct — spec §2 defines it but Phase 2 (T2.1) owns it
- Array decoder `DecodeSignals([]Signal)` — trigger: Phase 5 `decide` file mode
- Severity API beyond the iota order (no `Severity()`, `Less()`, `Rank()`) — the numeric
  value *is* the severity (§2: exit code = severity); plain `>` comparison in Phase 2
- Error-collecting `Validate` (all violations at once) — trigger: CLI UX asks for it
- Trailing-data rejection after the JSON value — trigger: Phase 5 `--stdin` if needed
- Nil-slice normalization for `Decision.Reasons/Pillars/Matched` — engine guarantees
  ≥ 1 reason (§9 property 3); `null` vs `[]` is never observable in v1
- Full ISO-3166 country-list check — spec pins `""` or 2 uppercase letters, nothing more
- Scenario-pack types — Phase 4

## File map (Phase 1 creates all of these; nothing else changes)
- `internal/model/signal.go` — Signal, Validate, SensitivityRank, DecodeSignal
- `internal/model/action.go` — Action type, constants, text behavior
- `internal/model/decision.go` — Decision
- `internal/model/model_test.go` — the whole suite (package `model_test`)

## Tasks

### T1.1 — Signal + Decision structs with exact §2 JSON tags
- Does: define `Signal` and `Decision` verbatim from §2 (tags: `timestamp user role app
  dataSensitivity location unfamiliarLocation deviceManaged deviceCompliant localHour
  mfaPresent riskScore` / `action scope reasons pillars matched`); define the bare
  `type Action int` with the four iota constants (Allow=0, Limited=1, StepUpMFA=2,
  Block=3) — `Decision` cannot compile without it; behavior arrives in T1.2
- Files: `internal/model/signal.go` (new), `internal/model/decision.go` (new),
  `internal/model/action.go` (new, type + constants only), `internal/model/model_test.go` (new)
- Depends: —
- Acceptance: `go test ./internal/model/ -run TestSignal_JSONRoundTrip -count=1` exits 0
- Est: S

### T1.2 — Action behavior: String, text marshalling, severity order
- Does: `String()` → `allow|limited|stepUpMFA|block` (fallback `Action(%d)` out of
  range); `MarshalText`/`UnmarshalText` (pointer receiver) so `Decision` JSON carries
  `"action":"block"` style text; unknown text → error naming the value and valid set.
  Severity = numeric order, asserted directly (`Block > StepUpMFA > Limited > Allow`) —
  no new API (see Not building)
- Files: `internal/model/action.go` (edit), `internal/model/model_test.go` (edit)
- Depends: T1.1
- Acceptance: `go test ./internal/model/ -run 'TestAction_' -count=1` exits 0
- Est: S

### T1.3 — SensitivityRank
- Does: `func SensitivityRank(s string) (int, error)` with
  `public=0 < internal=1 < confidential=2 < high=3`; unknown (incl. wrong case, empty)
  → error naming the value. Exported — `policies` P06 needs it in Phase 3
- Files: `internal/model/signal.go` (edit), `internal/model/model_test.go` (edit)
- Depends: T1.1
- Acceptance: `go test ./internal/model/ -run TestSensitivityRank -count=1` exits 0
- Est: S

### T1.4 — Signal.Validate + strict decoder helper
- Does: `func (s Signal) Validate() error` enforcing every §2 rule, checks in spec-table
  order (user/role/app → dataSensitivity → riskScore → localHour → location → device
  invariant → timestamp), returning the **first** error; `func DecodeSignal(r io.Reader)
  (Signal, error)` = `json.NewDecoder` + `DisallowUnknownFields` + `Validate`. RFC 3339
  for `timestamp` is enforced by `time.Time` unmarshalling itself; Validate only checks
  non-zero
- Files: `internal/model/signal.go` (edit), `internal/model/model_test.go` (edit)
- Depends: T1.1, T1.3 (Validate reuses SensitivityRank for the enum check)
- Acceptance: `go test ./internal/model/ -run 'TestSignal_Validate|TestDecodeSignal' -count=1` exits 0
- Est: M

### T1.5 — Test completeness sweep (§9 model suite)
- Does: audit T1.1–T1.4 tests against the §9 list — JSON round-trip (tags!), action text
  marshalling, sensitivity ordering, validation rules incl. the unmanaged⇒non-compliant
  invariant, unknown-field rejection — and add any missing cases (regression tests; they
  may pass immediately — if one goes red, that's a real model bug: fix the model, not the
  test). No new production code expected
- Files: `internal/model/model_test.go` (edit only)
- Depends: T1.1, T1.2, T1.3, T1.4
- Acceptance: `go test ./... -count=1` exits 0 AND `gofmt -l .` prints nothing AND
  `go vet ./...` exits 0 AND every §9 model bullet maps to a named test in model_test.go
- Est: S

## Done when (Phase 1 exit criteria)
```
go test ./internal/model/ -count=1        # exit 0
go test ./... -count=1 && go vet ./...    # exit 0 (main suite still green)
gofmt -l .                                # prints nothing
```

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
