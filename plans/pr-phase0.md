# Phase 0: project scaffold — CLI dispatch, Makefile, versioned skeleton

<!-- PR body. Title = the H1 above. Base: main ← head: phase-0-scaffold -->

## Summary

Lands Phase 0 of the zero-trust access simulator (spec §10 T0.1–T0.3): a building,
vet-clean Go skeleton with a testable CLI dispatch. No behavior beyond the CLI
contract — subcommand bodies arrive in Phases 5–6.

## What's in

- **go.mod** — module `zt-sim`, Go 1.27, zero dependencies.
- **main.go / main_test.go** — `run(args, stdout, stderr) int` dispatch:
  `--version` → `zt-sim version 0.1.0` (exit 0); no args / unknown subcommand →
  usage on stderr (exit 64, `EX_USAGE`); `decide|run|policies|audit` stubs →
  stderr + exit 70 (`EX_SOFTWARE`; can't collide with decide's 0–3 / run's 0/1).
- **Makefile** — `build / test / fmt / vet / run-demo` (Phase-0 `run-demo` prints
  the version; full §5 transcript lands in T7.4).
- **Layout** — code targets `internal/{model,engine,policies,scenario,report}` +
  `scenarios/`; they exist in the working tree only (git can't track empty dirs, no
  `.gitkeep` by plan decision) and fill with real files in Phases 1–4.
- **plans/** — plan.md + test-plan.md, tasks T1–T3 ticked with run log.

## Evidence (all exit 0, re-verified on this branch)

    go build ./... && go vet ./... && go test ./... -count=1   # ok  zt-sim
    gofmt -l .                                                # (no output)
    ./zt-sim --version                                        # zt-sim version 0.1.0
    ./zt-sim frobnicate; echo $?                              # usage on stderr, 64

7 table-driven cases in main_test.go cover the dispatch contract.

## Not in scope (YAGNI, per plan)

- Subcommand bodies (decide/run/policies/audit) — Phases 5–6
- Per-subcommand flags (`--stdin`, `--audit`, `--format`) — Phase 5
- README / CI / goreleaser — Phase 7 / publishing trigger
