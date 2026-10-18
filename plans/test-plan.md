# Test plan — zero-trust-access-simulator (Phase 0 — Scaffold)

## T1 — Module scaffold
- Test: none. Scaffolding (go.mod, empty dirs, git init) has no behavior to assert.
- Acceptance stands in: `go build ./...` exits 0.

## T2 — main dispatch
- Target: root package `main`, `main_test.go`, `TestRun_Dispatch` (package `main` —
  white-box is required here: `run` is unexported, and `package main` has no external
  test surface)
- Shape: table-driven over `args`, asserting exit code, exact stdout, and a substring of
  stderr (usage text is multi-line; assert its stable first line and key content)
- Red because: `main.go` / `run` do not exist → `go test .` fails to compile
- Run: `go test . -run TestRun_Dispatch -count=1 -v`

Cases (the contract under test):

| name | args | wantExit | wantStdout | wantStderr (contains) |
|---|---|---|---|---|
| version flag | `--version` | 0 | `zt-sim version 0.1.0\n` | — (empty) |
| no arguments | *(none)* | 64 | empty | `usage: zt-sim <command>` |
| unknown subcommand | `frobnicate` | 64 | empty | `unknown subcommand "frobnicate"` and `usage: zt-sim <command>` |
| decide is stubbed | `decide` | 70 | empty | `decide: not implemented yet` |
| run is stubbed | `run` | 70 | empty | `run: not implemented yet` |
| policies is stubbed | `policies` | 70 | empty | `policies: not implemented yet` |
| audit is stubbed | `audit` | 70 | empty | `audit: not implemented yet` |

Design the code against (pin these so execution needs no decisions):

```go
const version = "0.1.0"

const usage = `usage: zt-sim <command> [flags]

commands:
  decide <signals.json>   evaluate a JSON array of signals, print decisions
  run <scenarios-dir>     run scenario packs, print summary
  policies                list loaded policies
  audit <audit.jsonl>     print + validate an audit log

global:
  --version               print version and exit
`

func run(args []string, stdout, stderr io.Writer) int
func main() { os.Exit(run(os.Args[1:], os.Stdout, os.Stderr)) }
```

- Exit codes: 0 success · 64 usage error / unknown subcommand (spec §5) · 70 stub
  (`EX_SOFTWARE` — bodies arrive in Phases 5–6; must not collide with 64, 65, or decide's 0–3)
- Usage goes to **stderr**, version to **stdout**
- Exit codes are the CLI's real API — assert them exactly, not just output

## T3 — Makefile
- Test: none. Build wiring; `make test` re-runs T2's suite, which is the guard.
- Acceptance stands in: `make build && ./zt-sim --version && make vet && make test` exits 0,
  then `make fmt` changes nothing (`git diff --exit-code`).

Makefile content to pin:

```make
.PHONY: build test fmt vet run-demo

build:
	go build -o zt-sim .

test:
	go test ./... -count=1

fmt:
	gofmt -w .

vet:
	go vet ./...

# Phase 0 stub; Phase 7 (T7.4) expands to the full §5 demo transcript.
run-demo: build
	./zt-sim --version
```
