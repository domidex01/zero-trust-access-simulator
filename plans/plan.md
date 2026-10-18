# Plan — zero-trust-access-simulator (Phase 0 — Scaffold)
> Goal: a building, vet-clean skeleton — `zt-sim --version` prints, bad usage exits 64,
> `go build ./...` / `go vet ./...` / `make test` all pass.
> Source: spec.md §10 Phase 0 (T0.1–T0.3), §5 (CLI contract), §8 (layout).
> Task mapping: T1 ↔ spec T0.1, T2 ↔ spec T0.2, T3 ↔ spec T0.3.

## Status
- [x] T1 Init module, scaffold directories, git repo
- [x] T2 main.go — testable dispatch: `--version`, usage/unknown → 64, subcommand stubs
- [x] T3 Makefile — build / test / fmt / vet / run-demo

## Run log
- T1 ✅ 2026-10-19 — no test (scaffolding) · go build ./... exit 0 · git repo initialized (repo-local identity, global untouched)
- T2 ✅ 2026-10-19 — red: compile error (undefined: run) · green: 1 attempt · 7 cases
- T3 ✅ 2026-10-19 — no test (build wiring) · make build/vet/test/fmt/run-demo all pass · +1-line .gitignore for the built binary
(appended by execute-plan)

## Not building (YAGNI, Phase 0)
- Subcommand bodies (decide/run/policies/audit logic) — trigger: Phases 5–6
- Per-subcommand flag parsing (`--stdin`, `--audit`, `--format`) — trigger: Phase 5
- README — trigger: Phase 7 (T7.2)
- CI / lint config / goreleaser — trigger: publishing or sharing the repo
- `.gitkeep` files in empty dirs — git won't track them; each dir gets real files in Phases 1–4
- Full `run-demo` transcript — trigger: Phase 7 (T7.4); Phase 0 demos `--version` only
- Strict `--version` arg checking (rejecting `--version extra`) — unspecified in spec

## Tasks

### T1 — Init module, scaffold directories, git repo
- Does: `go mod init zt-sim` (go 1.27, zero deps); create `internal/{model,engine,policies,scenario,report}`, `scenarios/`; `git init` so commit-per-task works (no repo exists up the tree)
- Files: `go.mod` (new), `internal/model/` `internal/engine/` `internal/policies/` `internal/scenario/` `internal/report/` `scenarios/` (new, empty)
- Depends: —
- Acceptance: `go build ./...` exits 0 (a "matched no packages" warning is fine — main.go lands in T2; note `go vet ./...` exits 1 on an empty module, so vet moves to T2's acceptance)
- Est: S
- Test: none — scaffolding only

### T2 — main.go: testable dispatch with `--version` and usage exits
- Does: `run(args []string, stdout, stderr io.Writer) int` holding a switch dispatch — `--version` prints `zt-sim version 0.1.0` (exit 0); no args or unknown subcommand print usage to stderr (exit 64); the four spec subcommands (`decide|run|policies|audit`) are wired to stubs printing `not implemented yet` to stderr (exit 70 — EX_SOFTWARE: not a usage (64) or input (65) error, and it can't collide with decide's 0–3 / run's 0/1)
- Files: `main.go` (new), `main_test.go` (new)
- Depends: T1
- Acceptance: `go test . -count=1` exits 0 AND `go vet ./...` exits 0
- Est: S

### T3 — Makefile: build / test / fmt / vet / run-demo
- Does: five phony targets, stdlib tooling only; `run-demo` is a Phase-0 stub (`./zt-sim --version`) that Phase 7 (T7.4) expands into the full §5 transcript
- Files: `Makefile` (new)
- Depends: T2 (`test` target needs a suite; `build` needs main.go)
- Acceptance: `make build && ./zt-sim --version && make vet && make test` exits 0 and prints the version line; afterwards `make fmt && git diff --exit-code Makefile main.go main_test.go` exits 0 (fmt is a no-op on already-clean code)
- Est: S
- Test: none — build wiring; covered by T2's suite via `make test`

## Done when (Phase 0 exit criteria)
```
go build ./... && go vet ./... && go test ./... -count=1   # all exit 0
gofmt -l .                                                 # prints nothing
./zt-sim --version                                         # prints "zt-sim version 0.1.0"
./zt-sim frobnicate; echo $?                               # usage on stderr, prints 64
```
