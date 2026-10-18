# Phase 1: domain model — Signal, Action, Decision with strict validation

<!-- PR body. Title = the H1 above. Base: main ← head: phase-1-domain-model -->

## Summary

Lands Phase 1 (spec §10 T1.1–T1.5): `internal/model` complete and tested —
every §2 validation rule enforced at load, JSON tags byte-exact against the
spec, all four actions marshal as spec text. Pure domain types, zero new
dependencies; nothing wired into the CLI yet (engine lands in Phase 2).

## What's in

- **`internal/model/signal.go`** — `Signal` verbatim from §2 (12 fields, exact
  tags); `SensitivityRank` exposing the total order
  `public < internal < confidential < high`; `Signal.Validate()` returning the
  **first** violation in spec-table order (user/role/app → sensitivity →
  riskScore → localHour → location → unmanaged⇒non-compliant invariant →
  timestamp); `DecodeSignal` = `json.Decoder` + `DisallowUnknownFields` +
  `Validate`, so misspelled keys fail loudly instead of zero-valueing.
- **`internal/model/action.go`** — bare `type Action int` with the iota
  constants (Allow=0 … Block=3; numeric order *is* the severity order, reused
  as the CLI exit code); `String()` → `allow|limited|stepUpMFA|block` with
  `Action(%d)` fallback out of range; `MarshalText`/`UnmarshalText` (pointer
  receiver) so `Decision` JSON carries `"action":"block"` style text — unknown
  text errors, out-of-range values refuse to marshal, failed unmarshals leave
  state unchanged.
- **`internal/model/decision.go`** — `Decision` with `action scope reasons
  pillars matched` tags (audit trail in evaluation order).
- **`internal/model/model_test.go`** — black-box (`model_test`), table-driven,
  8 test functions / 43 subtests: golden JSON round-trips (byte-identical),
  action text both directions, severity order, sensitivity ranks + total order,
  17 validation rows including the device invariant and a first-error-wins
  curveball, strict decode (unknown field, validation-at-load, bad RFC 3339).

## Evidence (all exit 0, re-verified on this branch)

    go test ./... -count=1 -cover   # ok zt-sim 91.7% · ok internal/model 100.0%
    go vet ./...                    # clean
    gofmt -l .                      # (no output)

Every §9 model-suite bullet maps to a named test (audited in T1.5); one
regression test added during the sweep (`UnmarshalText` no-state-change).

## Process notes

- Strict red→green, one commit per task: T1.1–T1.5, each red was the planned
  compile error, each green on attempt 1. Run log in `plans/plan.md`.
- Two test-plan fixes, both because the row was unsatisfiable as written, not
  to weaken coverage: the T1.2 out-of-range MarshalText row now asserts an
  error; the T1.4 "limits accepted" row split into lower/upper bounds (LocalHour
  can't be 0 and 23 in one struct). `plans/test-plan.md` updated to match.

## Not in scope (YAGNI, per plan)

- `Policy` struct — Phase 2 (T2.1)
- Array decoder `DecodeSignals` — Phase 5 trigger
- Severity API beyond the iota order (`Severity()`, `Less()`) — plain `>`
  in Phase 2
- Error-collecting `Validate` (all violations at once) — CLI UX trigger
- Trailing-data rejection after the JSON value — Phase 5 `--stdin` if needed
- Real ISO-3166 country list — spec pins "2 uppercase letters", so `"ZZ"` passes
