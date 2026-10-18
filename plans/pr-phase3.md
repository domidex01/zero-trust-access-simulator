# Phase 3: policy pack — P01–P11 verbatim from §4, catch-all exclusive, §6 derivable

<!-- PR body. Title = the H1 above. Base: main ← head: phase-3-policy-pack -->

## Summary

Lands Phase 3 (spec §10 T3.1–T3.4): `internal/policies` complete and tested —
the v1 pack P01–P11 exactly as §4: every matcher, priority, pillar, scope and
verbatim audit note, a catch-all that matches iff nothing else did, and the six
§6 scenario shapes derivable from the pack as an executable test (the
executable form of the phase done-when, before Phase 4 authors the JSON packs).
87 lines of production code, 291 of tests, zero new dependencies; nothing
wired into the CLI yet (the `policies` subcommand is Phase 5).

## What's in

- **`internal/policies/policies.go`** — `Pack() []engine.Policy` is the single
  export surface (no per-policy vars); slice order is the §4 table order,
  already (Priority asc, ID asc). Ten core policies as composite literals with
  one-line closures for `When`, formulas pinned exactly: `!MFAPresent` (P01),
  `UnfamiliarLoc` (P02), `UnfamiliarLoc && !MFAPresent` (P03), `!DeviceManaged`
  (P04), `DeviceManaged && !DeviceCompliant` (P05), P06 via
  `model.SensitivityRank` with an `err == nil &&` guard plus `const
  rankConfidential = 2` (error path unreachable post-`DecodeSignal`, but the
  error return is not discarded), `RiskScore > 70` (P07), `40 ≤ RiskScore ≤ 70`
  (P08), `LocalHour < 5 && Role == "intern"` (P09), `Role != "finance" &&
  App == "payroll-app"` (P10). Notes copied verbatim from §4 including em
  dashes — they are the audit-line contract. **P11** (Allow, priority 90)
  loops over the ten core `When`s and returns false on the first match:
  with strictest-wins the decision would be identical either way, but
  `Matched`/`Reasons` must never carry a false "all clear" alongside a real
  finding (§6 expects P11 absent whenever others match).
- **`internal/policies/policies_test.go`** — black-box (`package
  policies_test`) through the exported `Pack()`; no white-box file needed.
  Shared fixtures: `clean()` (the §6 happy-path signal — no P01–P10 matcher
  fires on it), `policyByID(t, id)` scanning the pack, and the `triggers`
  mutation table (mutations set both device fields together so the §2
  validation invariant `!DeviceManaged ⇒ !DeviceCompliant` always holds).
  `TestPack_FireSilence` — each P01–P10 fires on its trigger, silent on clean;
  `TestPack_CatchAllExclusive` — P11 true on clean, false on **every**
  trigger row (the iff-direction that matters: P11 must not co-sign a
  finding); `TestPack_Boundaries` — 18 rows pinning risk 70/71/40/39,
  intern hour 4/5/0 + role guard, confidential/high/internal × device posture
  for P06, P04/P05 disjointness across the three reachable device states,
  P02-without-P03 ("not dead weight"), P03's `&&` guard, P10's role guard;
  `TestPack_ScenarioShapes` — the six §6 rows (happy, noteScenario,
  stolenCred, insider, byod, attacker) as full signals through
  `engine.New(policies.Pack()).Evaluate`, asserting Action, Scope, Matched and
  Pillars as exact slices; `TestPack_Metadata` — positional §4-verbatim table
  (ID, Priority, Pillar, Then, Scope, Note; `When` behavioral elsewhere —
  `Policy` has a func field, not `==`-comparable) plus: 11 policies, unique
  IDs, exactly one Allow and it is P11, `Scope == "read-only"` iff
  `Then == Limited`, every pillar ∈ the six valid values.

6 test functions / 45 subtests total.

## Evidence (all exit 0, re-verified on this branch)

    go test ./internal/policies/ -count=1   # ok — 100.0% statement coverage
    go test ./... -count=1 -cover           # zt-sim 91.7% · model 100.0% · engine 100.0% · policies 100.0%
    go test ./... -count=1 -race            # clean
    go vet ./...                            # clean
    gofmt -l .                              # (no output)

## TDD trail (run log in plans/plan.md)

Every task red before green; every green in 1 attempt:

- T3.1 — red: compile error (`no non-test Go files` — `Pack` undefined)
- T3.2 — red: **behavioral** — `policyByID(t, "P11")` Fatalfs while the pack
  holds only ten (`policy P11 not in pack`)
- T3.3 — 0 red: gap-closing sweep (T1.5/T2.4 precedent); no pack bugs found
- T3.4 — 0 red: transcription sweep; pack + pinned table both §4-verbatim
  (spec §4 diff-checked before T3.1)

## Not built (YAGNI, from the plan)

pack as JSON/YAML config (policies live in code per §4) · `networks`/
`infrastructure` policies (v1 has none) · exported per-policy vars ·
rewiring engine tests from `miniPack()` to the real pack (keeps the engine
suite independent of pack content) · engine changes (Phase 2 contract frozen)
· P06 invalid-sensitivity policy (unreachable: `DecodeSignal` validates at the
trust boundary) · benchmarks · cached/singleton Engine (Phase 5 decides).

## §9 audit

| §9 requirement | Covered by |
|---|---|
| each P01–P11 fires on trigger, silent on clean | `TestPack_FireSilence` + `TestPack_CatchAllExclusive` |
| boundary tests (risk 70/71, localHour 4/5, sensitivity confidential vs internal) | `TestPack_Boundaries` |
| unique IDs, priorities match §4, catch-all exactly once, pillar values valid | `TestPack_Metadata` |
| every §6 scenario row derives from the pack | `TestPack_ScenarioShapes` |
