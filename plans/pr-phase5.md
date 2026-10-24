# Phase 5: CLI — decide / run / policies end-to-end, §5 exit codes locked

<!-- PR body. Title = the H1 above. Base: main ← head: phase-5-cli -->

## Summary

Lands Phase 5 (spec §10 T5.1–T5.5): `internal/report` renders the §5 tables
(decisions USER/ACTION/REASONS + pillars footer, run summary
PACK/CASES/PASS/FAIL/STRICTEST, policies ID/PRIORITY/PILLAR/ACTION/NOTE —
`text/tabwriter`, columns are the contract) and the `--format json` payload;
`main.go` grows thin `cmdDecide` / `cmdRun` / `cmdPolicies` wiring (parse →
strict decode → evaluate → render) on a testable `run(args, stdin, stdout,
stderr) int`; `model.DecodeSignals` owns the signals-array trust boundary
(strict decode, per-element validation, empty array rejected, §6 packs
rejected in decide per §5). The §5 exit-code map is locked by
`TestRun_ExitCodes` (16 rows, incl. the still-stubbed `audit` → 70 so the
Phase 6 diff is visible). 214 lines of production code, 662 of tests, one
demo fixture, zero new dependencies.

## What's in

- **`internal/report/report.go`** — `Row{Signal, Decision}` (USER comes from
  the signal, the rest from the decision); `Decisions` (one line per reason,
  action uppercased, pillars footer = unique across rows in first-appearance
  order); `DecisionsJSON` (indented `[]Decision`); `PackResult` +
  `PackSummary`; `Policies` (the five §5 columns, no scope). Three concrete
  table funcs — a generic `[][]string` writer was considered and rejected
  (stringly rows lose the type checking); `report` imports `engine` (pure
  pkg) for the Policies table rather than adding a mapping struct in main.
- **`internal/model/signal.go`** — `DecodeSignals` next to `DecodeSignal`:
  `DisallowUnknownFields` + per-element `Validate` + `empty signals array`
  (nothing to decide is an input error, never a vacuous exit 0). A lone
  object or a §6 pack fails as `cannot unmarshal object into []model.Signal`
  — §5's "packs are rejected in decide" rule enforced at the decoder.
- **`main.go`** — `run` gains `stdin io.Reader` (main passes `os.Stdin`) so
  `decide --stdin` is testable. `cmdDecide`: `flag.ContinueOnError` (the CLI
  owns exit codes), `--stdin` xor one positional, `--format table|json`,
  every usage rule in one guard switch, exit = strictest action (§2: the
  severity doubles as the exit code; tracked as `model.Action`, cast once at
  return). `cmdRun`: `os.ReadDir` (sorted, top-level `*.json` only),
  first `Load` error aborts → 65, zero packs → 65, per-case `PASS/FAIL`
  lines + summary, exit 1 on any expectation failure. `cmdPolicies`: the
  §5 listing. `audit` remains the exit-70 stub for Phase 6.
- **`examples/attacker-signals.json`** — resolves the §5 divergence: the
  spec's example session names `scenarios/attacker.json`, but §5's own rule
  rejects packs in decide. The normative rule wins; this fixture is the
  pack's signal as a decide-legal array (extracted programmatically, pinned
  equal to the pack by `TestExamples_AttackerMatchesPack`) so the §5 session
  and the Phase 7 transcript work. `decide scenarios/attacker.json` → 65 is
  pinned as *correct* behavior.
- **`main_test.go`** — `TestRun_Dispatch` sheds each subcommand's stub row
  as it lands; `TestRun_Decide` (14 rows: all four actions, mixed-file
  strictest, stdin, missing file, pack rejection, bad flag/format, usage
  combos), `TestRun_DecideFormatJSON` (round-trips to `[]Decision`),
  `TestRun_Run` (7 rows: real `scenarios/` all-pass, failing pack, corrupt
  pack aborts, missing dir, empty dir, usage), `TestRun_Policies`,
  `TestRun_ExitCodes` (the §5 matrix), `TestExamples_AttackerMatchesPack`.
- **`internal/report/report_test.go`** — hand-built decisions (renderer
  tests, no engine): table structure (line counts, stacked reasons, pillar
  union, uppercase actions, scope stays out of the table), JSON round-trips,
  PackSummary cells, Policies columns.

Review #6 (thermonuclear, APPROVE): 4 minor findings fixed — one usage
guard instead of 4 repeated pairs, `model.Action`-typed strictest tracking,
`cmdRun`'s double-meaning `e` renamed, fixture-parity test added; rejected
with reasons: generic table writer, `Row{User string}`, decoder
unification, exit-code consts (logged in plans/plan.md).

## Evidence (all exit 0, on this branch)

    go test ./... -count=1 -cover    # model/engine/policies/scenario 100% · report 96.6% · zt-sim 92.9%
    go vet ./...                     # clean
    gofmt -l .                       # (no output)
    make build && ./zt-sim decide examples/attacker-signals.json   # 7 reasons, 4-pillar footer
    echo $?                          # 3  (the §5 done-when, via the examples/ fixture)
    ./zt-sim run scenarios/          # six PASS lines + PACK summary, exit 0

## TDD trail (run log in plans/plan.md)

- T5.1 — red: build failure (`no non-test Go files in …/internal/report`,
  the T3.1/T4.1 precedent) · green: 1 attempt · 2 test-side fixture index
  fixes (production output was correct both times)
- T5.2 — red: compile error (`model.DecodeSignals` undefined) + behavioral
  (exit-70 stub) · green: 1 attempt
- T5.3 — red: compile error (`report.PackResult` undefined) + behavioral ·
  green: 1 attempt · test-side fix: PASS/FAIL counting matched the summary
  header cells → per-line prefix counting
- T5.4 — red: compile error (`report.Policies` undefined) + behavioral ·
  green: 1 attempt
- T5.5 — 0 red: sweep (T1.5/T2.4/T3.3/T4.3 precedent); 16 matrix rows; demo
  session verified; spec §10 Phase 5 ticked

## Not built (YAGNI, from the plan)

box-drawing borders + truncation (§5: columns are the contract) ·
`run --format json` (payload undefined in §5) · `--audit` flags (Phase 6
T6.2) · SCOPE column in the decide table (rides in `--format json`) ·
machine markers/colors in run output (exit code is the machine interface) ·
trailing-content rejection (single-Decode precedent) · `internal/cli`
package split (spec §8 pins wiring in main.go; trigger: main.go hurts to
navigate) · more `examples/` fixtures (trigger: Phase 7 README needs them).

## §5 audit

| §5 requirement | Covered by |
|---|---|
| `decide <signals.json>` array input | `TestRun_Decide` file rows + `TestDecodeSignals` |
| `decide --stdin` single strict object | stdin rows (good + unknown-field) |
| packs rejected in decide | `TestDecodeSignals` "scenario pack rejected" + exit-code row (65) |
| `--format table\|json` (default table) | `TestRun_DecideFormatJSON` + bad-value usage row |
| decide exit 0–3 = strictest action | 4 action rows + mixed-file row |
| `run` summary + expectation results, exit 1 on failure | `TestRun_Run` |
| `policies` listing | `TestRun_Policies` |
| usage 64 / input 65 across commands | `TestRun_ExitCodes` matrix (audit → 70 until Phase 6) |
| example session | demo fixture + `echo $?` → 3 (evidence above) |
