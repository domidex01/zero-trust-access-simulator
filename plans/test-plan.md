# Test plan — zero-trust-access-simulator (Phase 7 — Polish)

Phase 7 changes **zero `.go` files** (plan decision 1): docs + Makefile + verification.
There are no red/green cycles and no new test files. Verification is **command-shaped**
— every check below is a shell command with an exact expected result, runnable from the
project root. Precedents: T0.3 (build wiring, no test), T4.2 (authoring, loader tests
only), T1.5/T2.4/T3.3/T4.3/T5.5/T6.4 (0-red sweeps).

Test files: **none touched.** `main_test.go` stays 752 lines (split deferred —
Not-building list). The existing suite is the regression net: `go test ./... -count=1`
must stay green after every task.

## T7.1 — fmt/vet/test clean-baseline sweep
- Target: repo root, all packages
- Shape: three commands, run as written
- Red because: nothing — 0-red sweep (verified green at plan time, 2026-10-25);
  a red here is a bug in code, never a reason to touch a test
- Run:
  ```
  gofmt -l .                 # prints NOTHING (empty output)
  go vet ./...               # exit 0
  go test ./... -count=1     # exit 0, all 6 packages ok
  ```

| name | give | want |
|---|---|---|
| formatted | `gofmt -l .` | no output |
| vetted | `go vet ./...` | exit 0 |
| green | `go test ./... -count=1` | exit 0, 6 × `ok` |

## T7.2 — README structure and content
- Target: `README.md` (new file)
- Shape: grep-count acceptance — section headers and table rows are countable anchors
- Red because: `README.md` does not exist → `test -f` fails
- Run (one line, all must pass):
  ```
  test -f README.md \
    && [ "$(grep -c '^## ' README.md)" -eq 6 ] \
    && [ "$(grep -c '^| P[0-9][0-9] ' README.md)" -eq 11 ] \
    && [ "$(grep -c '\.json` |' README.md)" -eq 6 ] \
    && grep -q 'go build' README.md && grep -q '64' README.md && grep -q '65' README.md
  ```

| name | give | want |
|---|---|---|
| file exists | `ls README.md` | exists |
| six H2 sections | `grep -c '^## '` | 6 — Install, Usage, Exit codes, Policies, Scenarios, Demo, in that order (order checked by eye) |
| what/why intro | text between H1 and `## Install` | one paragraph naming SC-900 + the three Zero Trust principles |
| policy table complete | `grep -c '^| P[0-9][0-9] '` | 11 (P01–P11, one row each) |
| policy notes verbatim | `grep -c 'unfamiliar location and no MFA — assume breach'` | ≥ 1 (em dash, §4 wording spot-check) |
| scenario table complete | rows whose first cell is a backticked `.json` filename (`grep -c '\.json\` |'`) | 6 (happy, note-scenario, stolen-credential, insider, byod, attacker) |
| install uses go build | `grep -c 'go build'` | ≥ 1 |
| exit codes present | grep for `64` / `65` | both ≥ 1 (decide 0–3, run 0/1, 64 usage, 65 bad input) |
| usage covers subcommands | `grep -c 'zt-sim decide\|zt-sim run\|zt-sim policies\|zt-sim audit'` | ≥ 4 distinct commands |
| flags documented | grep `--stdin`, `--format`, `--audit` | each present |
| Demo stub | `grep -c '^## Demo$'` | 1, nothing under it yet (T7.3 fills) |

## T7.3 — Demo transcript
- Target: `README.md` `## Demo` section
- Shape: byte-anchor greps against the verbatim capture (plan pins the full transcript)
- Red because: `## Demo` is header-only after T7.2 → anchors count 0
- Run:
  ```
  [ "$(grep -c 'audit: 7 policies matched, decision appended to audit.jsonl' README.md)" -eq 1 ] \
    && [ "$(grep -cx 3 README.md)" -eq 1 ] \
    && [ "$(grep -c '^PASS ' README.md)" -eq 6 ] \
    && grep -q 'flags precede\|flags come before\|flags-first' README.md \
    && grep -q 'examples/attacker-signals.json' README.md
  ```

| name | give | want |
|---|---|---|
| audit footer verbatim | `grep -c 'audit: 7 policies matched, decision appended to audit.jsonl'` | exactly 1 |
| exit 3 line | `grep -cx 3` | exactly 1 (the `$ echo $?` → `3` step; bare digit on its own line) |
| six PASS lines | `grep -c '^PASS '` | 6 |
| run summary rows | `grep -c '^stolen-credential '` | 1 (last table row, STEPUPMFA — §6's every-action-covered) |
| table is real capture | reason lines vs §5 example | tabwriter spacing (2+ spaces), not §5's box-drawing sketch — spec §5: rendering is `report`'s, content is the contract |
| divergence note: flag order | grep `flags` | ≥ 1 sentence noting flags precede the positional (stdlib `flag`) |
| divergence note: fixture | grep `examples/attacker-signals.json` | ≥ 1 sentence noting decide takes signal arrays; packs belong to `run` |
| fresh paste, not stale | re-run the capture before committing | transcript bytes == live output (re-capture if anything could have shifted them) |

## T7.4 — make run-demo reproduces the transcript
- Target: `Makefile` `run-demo` target (replaces Phase 0 stub)
- Shape: behavioral checks of the target — idempotency, exit-3 handling, byte-exactness,
  fresh clone
- Red because: stub prints only `zt-sim version 0.1.0` → footer/PASS anchors count 0
- Run (from repo root):
  ```
  make run-demo > /tmp/demo.out; echo $?      # exit 0 despite decide's exit 3
  make run-demo > /tmp/demo2.out && cmp /tmp/demo.out /tmp/demo2.out   # idempotent
  ```

| name | give | want |
|---|---|---|
| make survives Block | `make run-demo; echo $?` | make exit 0 (decide's 3 is echoed as `3`, swallowed by `; echo $$?`) |
| transcript reproduced | footer/PASS/`3` anchors in `/tmp/demo.out` | footer count 1, `^PASS ` count 6, `grep -cx 3` count 1 |
| no recipe noise | `/tmp/demo.out` first line | `USER …` — every recipe line `@`-silenced, no `rm -f`/command echo |
| idempotent | run twice, `cmp` | byte-identical (`rm -f audit.jsonl` resets audit state) |
| byte-exact vs README | README Demo block minus `$`-prompt lines vs `/tmp/demo.out` | identical (diff once by hand during execution; note in run log) |
| audit log left at 1 line | `wc -l < audit.jsonl` after run-demo | 1 |
| clean up | `rm -f audit.jsonl` | repo root tidy (gitignored anyway) |

Fresh-clone v1 check — **run after committing all Phase 7 changes** (plan decision 6:
`git clone .` sees committed state only):
```
d=$(mktemp -d) && git clone -q . "$d/zt" && cd "$d/zt" \
  && go build ./... && go test ./... -count=1 && make run-demo
```

| name | give | want |
|---|---|---|
| builds from nothing | fresh clone, `go build ./...` | exit 0, no binary carried over |
| suite green | `go test ./... -count=1` | exit 0 (6 packages) |
| demo reproduces | `make run-demo` | the README transcript, exit 0 |

## Sweep policy for Phase 7
Every task ends with `gofmt -l .` empty and `go vet ./... && go test ./... -count=1`
exit 0 — trivially true since no `.go` file changes, but run it anyway: the phase's
product is the claim "shippable v1", and the claim is only cheap to make if it keeps
being re-verified. Final state = spec §10's v1 definition of done (plan's Done-when
block), at which point spec §10 Phase 7 boxes get ticked.
