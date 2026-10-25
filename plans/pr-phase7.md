# Phase 7: Polish — README + `make run-demo`, shippable v1 (§10 Phase 7)

<!-- PR body. Title = the H1 above. Base: main ← head: phase-7-polish -->

## Summary

Lands Phase 7 (spec §10 T7.1–T7.4) — the v1 "shippable" pass. **Zero `.go`
files change**: README.md (what/why, Install, Usage, Exit codes, the §4
policy table, the §6 scenario table, a demo transcript), the Makefile's
`run-demo` (Phase 0 stub → the real demo), and the spec's Phase 7 ticks.
T7.1 records the clean baseline: `gofmt -l .` empty, `go vet ./...` clean,
`go test ./... -count=1` green across all six packages — re-verified after
every task, since the phase's product is exactly that claim.

The demo transcript is not typed by hand: the session is captured from the
built binary into temp files and assembled into `## Demo` with `cat`, then
`diff`ed back against the live output — byte-exact (the project's own T5.1
indent-drift bug is the precedent this avoids). `make run-demo` then
reproduces that same block from a clean build: every recipe line
`@`-silenced so stdout is pure command output, `; echo $$?` after decide so
Block's exit 3 is shown without failing make, `rm -f audit.jsonl` first so
the target is idempotent. The fresh-clone check (build → test → run-demo
from a `git clone` of this branch) reproduces the transcript byte for byte —
the spec's v1 definition of done, met.

One plan bug found and fixed mid-execution (T7.4): the pinned run-demo
recipe omitted `; echo $$?` after `run` while the pinned transcript ends
with `0`, making byte-exactness unsatisfiable. The Makefile gained the echo
(run's exit 0 is worth showing, symmetric with decide's 3); the acceptance
was not weakened. Also `build`'s recipe line is now `@`-silenced — make's
own echo of `go build -o zt-sim .` was leaking into run-demo's stdout as
line 1. Both fixes are recorded in the run log and folded back into
plans/plan.md decision 4.

## What's in

- **`README.md`** (new) — H1 + one-paragraph what/why (SC-900 companion;
  verify explicitly / least privilege / assume breach → mechanics), then six
  H2 sections: `Install` (`go build`, zero deps, offline), `Usage` (usage
  strings copied from main.go's own consts, not the spec sketch; the four
  actions with meanings and exit codes), `Exit codes` (§5's tables:
  decide 0–3 strictest, run 0/1, 64/65), `Policies` (§4's 11 rows verbatim,
  em dashes intact, plus the P11-only-Allow / exact-threshold / half-open-
  window notes), `Scenarios` (§6's 6 rows verbatim), `Demo` (transcript +
  the two pinned divergence notes).
- **Divergence notes in `## Demo`** — a fresh-clone user can copy-paste
  every command: (a) "flags come before the positional argument" (stdlib
  `flag`, Phase 6 pin); (b) decide takes signal *arrays*, so the demo uses
  `examples/attacker-signals.json` (kept in lockstep with the attacker pack
  by `TestExamples_AttackerMatchesPack`) while packs belong to `run`
  (Phase 5 pin).
- **`Makefile`** — `build:` recipe `@`-silenced; `run-demo` = `rm -f
  audit.jsonl` → decide with `--audit` + `; echo $$?` → run + `; echo $$?`.
  Output = the README Demo block minus `$`-prompt lines, byte for byte;
  idempotent across back-to-back runs (`cmp`-verified); audit.jsonl left at
  exactly 1 line.
- **`spec.md`** — §10 Phase 7 boxes T7.1–T7.4 ticked.
- **Post-phase addendum** (same PR, follow-up request): `## How it works` section with
  two Mermaid diagrams (decision pipeline, strictness ladder), an attacker-walkthrough
  Mermaid in `## Demo`, and four terminal screenshots (`docs/*.svg`) generated from
  live output by `docs/term2svg.py` + `make screenshots`. README H2 count 6 → 7
  (documented in the run log); no `.go` changes.

## Evidence (all on this branch)

    gofmt -l .                       # empty
    go vet ./...                     # exit 0
    go test ./... -count=1           # 6 packages ok
    make run-demo | head -1          # USER … REASONS — no make noise
    make run-demo > a && make run-demo > b && cmp a b   # identical
    # fresh clone of this branch:
    git clone . /tmp/zt && cd /tmp/zt && go build ./... && go test ./... -count=1 && make run-demo
                                     # exit 0, transcript byte-exact vs README Demo block

## TDD trail (run log in plans/plan.md)

- T7.1 — 0 red: sweep (T1.5/T2.4/T3.3/T4.3/T5.5/T6.4 precedent); baseline
  green at plan time, re-verified
- T7.2 — red: acceptance exit 1 (`test -f README.md`, file absent) · green:
  1 attempt · 6 H2s / 11 policy rows / 6 scenario rows grep-verified
- T7.3 — red: anchors 0 (Demo header-only) · green: 1 attempt + 1 README
  wording fix (lowercase "flags come before" to match the pinned anchor —
  README side, test untouched) · transcript assembled from live captures,
  diff-verified byte-exact
- T7.4 — red: stub anchors 0 · green: 1 attempt + 2 fixes (build recipe
  silenced; run line echo — plan bug, folded back into plan.md) · make exit
  0 despite decide's 3 · idempotent · byte-exact vs README · fresh-clone
  check green

## Not built (YAGNI, from the plan)

CI/badges/coverage shields · `make check` aggregate target · main_test.go
split (zero tests added; 752 lines unchanged) · README drift-guard Go test
(`TestExamples_AttackerMatchesPack` already pins the demo data) ·
`policies`/`audit` steps in the transcript (§10 pins "§5 session + run
summary") · `go test -race` (zero goroutines) · version bump / tag /
CHANGELOG / LICENSE · all §11 stretch goals.
