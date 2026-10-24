# Phase 4: scenarios & acceptance — the six §6 packs on disk, green under `go test ./...`

<!-- PR body. Title = the H1 above. Base: main ← head: phase-4-scenarios -->

## Summary

Lands Phase 4 (spec §10 T4.1–T4.4): `internal/scenario` loads and validates
§6 scenario packs and checks each case's `expect` block against a Decision,
and the six §6 packs (`happy`, `note-scenario`, `stolen-credential`,
`insider`, `byod`, `attacker`) are authored on disk and run as acceptance
tests wired into `go test ./...` — so every action is exercised and every
§6 expectation is asserted against the real P01–P11 pack on every build.
108 lines of production code, 303 of tests, six JSON packs, zero new
dependencies; nothing wired into the CLI yet (the `run` subcommand is
Phase 5).

## What's in

- **`internal/scenario/scenario.go`** — engine-free by design (imports only
  `model`): `Pack{Name, Description, Cases}` / `Case{Name, Signal, Expect}` /
  `Expectation{Action *model.Action, ReasonsContain, Scope}` with §6 JSON
  tags. `Action` is a pointer because the enum's zero value is `Allow`, so a
  value field can't tell "absent" from `"allow"` — the pointer makes §6's
  "action (required)" enforceable, and `*Action` still decodes through the
  existing `UnmarshalText` (unknown text errors). `Load(path)`:
  ReadFile → strict decode (`DisallowUnknownFields`, the `DecodeSignal`
  precedent) → unexported `validate()` in pinned order (pack name → ≥1 case
  → per case: name → `Signal.Validate()` → `expect.action` non-nil →
  scope ⇒ limited), with decode/validate errors prefixed by the path and
  case errors wrapped `case %d %q:` so Phase 5's `run` failures are
  debuggable. `Case.Check(d)` compares action equality → every
  `ReasonsContain` ID ∈ `d.Matched` → scope only when set (§6: scope is
  optional, checked when the action is limited).
- **`scenarios/*.json` (six packs)** — signals are the
  `TestPack_ScenarioShapes` pins verbatim (note-scenario's signal = the §6
  example JSON verbatim, timestamp 03:14Z included), `reasonsContain` = the
  §6 matched-policies column in full (stronger than the two-ID example
  expect), descriptions = the §6 Story column. byod carries
  `"scope": "read-only"` — the only Limited pack, and the loader's
  scope⇒limited rule's only on-disk user.
- **`internal/scenario/scenario_test.go`** — black-box (`package
  scenario_test`). `TestLoad_Pack` (happy path, full field assertions);
  `TestLoad_Invalid` — 13 rows: unknown fields at all three nesting levels,
  missing pack/case names, empty cases, signal-validation failure (loader
  must call `Signal.Validate`), `expect` absent/null, non-enum action,
  scope-on-non-limited, missing file; `TestCase_Check` — 7 rows: exact
  match, action mismatch, missing reasons ID, scope match/mismatch/absent,
  empty reasonsContain; `TestLoad_SixPacks` — the six files load, `Name`
  == file stem, ≥1 case; `TestPacks_Expectations` — globs
  `../../scenarios/*.json` with a vacuity guard (≥6 files AND all six
  pinned basenames present — an empty glob fails the test instead of
  passing), then `engine.New(policies.Pack()).Evaluate` → `Check` per case;
  `TestPacks_ActionCoverage` — §6's "every action is exercised by at least
  one scenario" claim asserted (all four actions seen).

6 test functions / 31 subtests; `internal/scenario` at 100% statement coverage.

## T4.4 — hand-verification against §4 (recorded in plans/plan.md run log)

All six derived by hand from the §4 matchers before ticking; zero
divergences. Highlights: note-scenario fires P01,P02,P03,P04,P06,P08 (P07
silent at 65 ≤ 70, P09/P10 silent — finance role); attacker fires 7 (P08
silent at 85 > 70, P09 silent — guest), matching the §5 example footer
"7 policies matched"; stolen-credential's strictest is StepUpMFA (P01/P08
outrank P04's Limited), the severity-order fix from revision note 1 doing
its job.

## Evidence (all exit 0, re-verified on this branch)

    go test ./internal/scenario/ -count=1          # ok — 100.0% statement coverage
    go test ./... -count=1 -cover                  # zt-sim 91.7% · model/engine/policies/scenario 100.0%
    go test ./... -count=1 -race                   # clean
    go vet ./...                                   # clean
    gofmt -l .                                     # (no output)

## TDD trail (run log in plans/plan.md)

- T4.1 — red: compile error (`no non-test Go files in …/internal/scenario`,
  the T3.1 precedent) · green: 1 attempt, after fixing two test-side bugs
  (`strings.Replace` count arg; the unknown-case-field row appended its
  JSON *after* the case object's closing brace — invalid JSON, not an
  unknown field; row corrected in test-plan too)
- T4.2 — red: **behavioral** — all six `Load`s fail `no such file or
  directory` · green: 1 attempt (authoring only)
- T4.3 — 0 red: gap-closing sweep (T1.5/T2.4/T3.3 precedent); no divergence
  between packs, §6 tables, and the real pack
- T4.4 — hand-derivation sweep; zero divergences; spec §10 Phase 4 ticked

## Not built (YAGNI, from the plan)

the `run` subcommand + dir walking + summary table (Phase 5 T5.3 — `Load`
takes one path, the walker lives in the test) · engine import inside
`internal/scenario` (Check takes the Decision; Phase 5 wires the CLI) ·
`expect.pillars` / exact matched-set equality (§6 pins action +
reasonsContain ⊇ + scope-on-limited; exact sets already pinned in
`TestPack_ScenarioShapes`) · required-description validation (§6 marks only
`expect.action` required) · trailing-content rejection after the pack value
(matches `DecodeSignal` precedent) · schema/version field, multi-pack files.

## §9 audit

| §9 requirement | Covered by |
|---|---|
| loads every `scenarios/*.json` | `TestLoad_SixPacks` + the `loadPacks` glob (future packs auto-run) |
| validates | `TestLoad_Invalid` (13 rows incl. all three unknown-field levels) |
| asserts all expectations | `TestPacks_Expectations` (6 packs × cases through the real engine+pack) |
| wired into `go test ./...` | walker lives in `internal/scenario` → runs with `./...` |
| §6 "every action exercised" | `TestPacks_ActionCoverage` |
