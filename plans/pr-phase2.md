# Phase 2: engine — pure evaluator, strictest match wins, fails closed

<!-- PR body. Title = the H1 above. Base: main ← head: phase-2-engine -->

## Summary

Lands Phase 2 (spec §10 T2.1–T2.4): `internal/engine` complete and tested —
sorted policies in, a deterministic Decision out, strictest match wins, every
path fails closed. A pure evaluator: no I/O, no clock, no randomness (proved
behaviorally by the determinism test). 87 lines of production code, 484 of
tests, zero new dependencies; nothing wired into the CLI yet (the real P01–P11
pack is Phase 3).

## What's in

- **`internal/engine/engine.go`** — `Policy` verbatim from §2 (ID, Priority,
  Pillar, `When func(model.Signal) bool`, Then, Scope, Note — no JSON tags,
  policies live in code); `Engine` with an unexported pre-sorted pack,
  immutable after `New` (value receiver — safe to copy, zero-value = empty
  pack); `New` clones the caller's slice (no side effects) and sorts by
  `(Priority asc, ID asc)` via `slices.SortFunc`; `Evaluate` runs **every**
  policy — no short-circuit, each match is audited — collecting
  `"<ID>: <Note>"` reasons, matched IDs, and unique pillars in evaluation
  order; final action is the numeric max (severity is the iota order, plain
  `>` comparison); scope = first matching Limited policy's scope, kept only
  when Limited is the final action (tracked with a `limitedSeen` flag, so a
  hypothetical empty-scope Limited policy can't shadow a later one); zero
  matches or empty pack → `Block` with the pinned §3.5 engine reason
  (`engine: no policy matched — fail closed (assume breach)` /
  `engine: no policies loaded — fail closed`) — the engine never defaults
  to a grant.
- **`internal/engine/engine_internal_test.go`** — white-box (`package
  engine`): `New`'s sort (3 rows: priority ascending, ID tiebreak, mixed) and
  the caller-slice-untouched clone guarantee. Precedent: `main_test.go` tests
  unexported `run`.
- **`internal/engine/engine_test.go`** — black-box (`package engine_test`):
  `base()` = the alice §6 note-scenario signal; `TestEvaluate_Semantics`
  (6 rows: single-match allow, strictest-wins across actions, reasons follow
  sort not input order, priority tie → ascending ID, scope from first
  matching Limited, pillar dedupe curveball); `TestEvaluate_FailClosed`
  (zero matches / `New(nil)` / zero-value engine); `TestEvaluate_ScenarioShapes`
  — the five §9 shapes against a synthetic `miniPack()` (not the real
  P01–P11), full Decisions pinned: noteScenario→Block, happy→Allow,
  stolenCred→StepUpMFA, byod→Limited, insider→Block; `TestEvaluate_StrictestWins`
  (6 subset rows down the severity ladder); `TestEvaluate_NoSilentPass`
  (≥1 reason everywhere, Reasons==Matched, every Allow cites its policy,
  fail-closed = exactly one `engine:` reason); `TestEvaluate_Deterministic`
  (double-eval deep-equal + construction order irrelevant).

8 test functions / 36 subtests total.

## Evidence (all exit 0, re-verified on this branch)

    go test ./internal/engine/ -count=1   # ok — 100.0% statement coverage
    go test ./... -count=1 -cover         # zt-sim 91.7% · model 100.0% · engine 100.0%
    go vet ./...                          # clean
    gofmt -l .                            # (no output)

## TDD trail (run log in plans/plan.md)

Every task red before green; every green in 1 attempt:

- T2.1 — red: compile error (`Policy`/`New` undefined)
- T2.2 — red: compile error (`Engine.Evaluate` undefined)
- T2.3 — red: **behavioral** — T2.2's zero-match path returned the zero-value
  Decision (`Allow`); fail-closed paths made it Block
- T2.4 — 0 red: gap-closing sweep (T1.5 precedent); no engine bugs found

## Not built (YAGNI, from the plan)

nil-`When` guard (pack is authored in code) · signal re-validation inside
Evaluate (trust boundary is `model.DecodeSignal`) · batch `EvaluateAll` ·
engine error return (a fail-closed Decision *is* the failure mode) · exported
order accessor · scope validation (Phase 3 pack sanity test) · concurrency
machinery (Engine is read-only after New) · purity lint (proved behaviorally).

## §9 audit

| §9 requirement | Covered by |
|---|---|
| scenario table (5 shapes) | `TestEvaluate_ScenarioShapes` |
| property 1 — fail closed | `TestEvaluate_FailClosed` |
| property 2 — strictest wins | `TestEvaluate_StrictestWins` + multi-match rows |
| property 3 — no silent passes | `TestEvaluate_NoSilentPass` |
| property 4 — determinism | `TestEvaluate_Deterministic` + Semantics order/tie rows |
