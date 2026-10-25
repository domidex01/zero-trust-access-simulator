# Plan — zero-trust-access-simulator (Phase 7 — Polish)
> Goal: shippable v1 — README documents what/why, install, usage, exit codes, the policy
> and scenario tables, and a demo transcript that `make run-demo` reproduces from a clean
> build; fmt/vet/test baseline recorded green. Spec §10 v1 definition of done: fresh
> clone → `go build ./... && go test ./...` green → `./zt-sim run scenarios/` all packs
> pass → README demo reproduces.
> Source: spec.md §10 Phase 7 (T7.1–T7.4), §5 (usage, exit codes, example session),
> §4 (policy table), §6 (scenario table), §8 (layout), §12 (concept mapping for the
> what/why intro).
> Update run: Phases 6/5/4/3/2/1/0 complete, ids stable, archived below. Task ids match
> spec §10 verbatim (T7.1–T7.4).
>
> **Decisions pinned up front** (spec leaves these open; execution needs no judgment):
> 1. **No Go code changes in Phase 7** — no red/green cycles, no new test files. Docs +
>    Makefile + verification only. Precedents: T0.3 (build wiring), T4.2 (authoring),
>    T1.5/T6.4 (0-red sweeps). Every acceptance below is command-shaped instead.
> 2. **README structure pinned**: H1 + what/why intro (SC-900 companion, Zero Trust
>    decision engine, the three principles → mechanics in one paragraph), then exactly
>    six H2 sections in order: `Install`, `Usage`, `Exit codes`, `Policies`,
>    `Scenarios`, `Demo` (T7.2 stubs `Demo` with just the header; T7.3 fills it).
>    Grep-countable: 6 × `^## `, 11 × `^| P`, 6 × scenario rows.
> 3. **Transcript = §5 session + run summary, nothing more**: decide on the attacker
>    fixture with `--audit audit.jsonl` (repo root, already gitignored), `$ echo $?` →
>    `3`, then `run scenarios/`. No `policies`/`audit` read-back in the transcript —
>    §10 T7.3 pins "the §5 session + `run scenarios/` summary". README shows `$` prompts
>    as presentation; the output bytes are the contract.
> 4. **run-demo recipe pinned** (replaces the Phase 0 stub); every line `@`-silenced so
>    run-demo's stdout is *only* command output, and `; echo $$?` after decide both
>    reproduces §5's exit-code step and keeps make green (exit 3 is a successful Block,
>    not a make failure — the recipe line's status becomes echo's 0):
>    ```make
>    run-demo: build
>    	@rm -f audit.jsonl
>    	@./zt-sim decide --audit audit.jsonl examples/attacker-signals.json; echo $$?
>    	@./zt-sim run scenarios/
>    ```
>    `rm -f audit.jsonl` makes run-demo idempotent (audit log reset to the transcript's
>    implied fresh state each run). "Reproduces the transcript" = run-demo's stdout
>    equals the README Demo block minus `$`-prompt lines, byte for byte.
> 5. **Divergences from §5's sketch documented in `## Demo`** (a fresh-clone user must
>    be able to copy-paste every command): (a) flags precede the positional — stdlib
>    `flag` parses flags-first, §5's sketch ordering is illustrative (Phase 6 pin);
>    (b) decide takes signal *arrays*, so the demo uses `examples/attacker-signals.json`
>    (pinned equal to the attacker pack's signal by `TestExamples_AttackerMatchesPack`)
>    while packs like `scenarios/attacker.json` belong to `run` (Phase 5 pin).
> 6. **Fresh-clone check runs after the T7.4 commit** — `git clone .` sees committed
>    state only; the check verifies exactly what a fresh user gets. Version stays
>    `0.1.0`; no tag/release task exists in the spec.

## Status
Phase 7:
- [x] T7.1 fmt/vet/test clean-baseline sweep (0 red expected)
- [ ] T7.2 README.md: what/why, Install, Usage, Exit codes, Policies (§4), Scenarios (§6), Demo stub
- [ ] T7.3 Demo transcript in README: §5 session + `run scenarios/` summary, verbatim capture
- [ ] T7.4 `make run-demo` reproduces the transcript from a clean build; fresh-clone v1 check; spec §10 ticks

Phase 6 (complete): T6.1 ✅ · T6.2 ✅ · T6.3 ✅ · T6.4 ✅
Phase 5 (complete): T5.1 ✅ · T5.2 ✅ · T5.3 ✅ · T5.4 ✅ · T5.5 ✅
Phase 4 (complete): T4.1 ✅ · T4.2 ✅ · T4.3 ✅ · T4.4 ✅
Phase 3 (complete): T3.1 ✅ · T3.2 ✅ · T3.3 ✅ · T3.4 ✅
Phase 2 (complete): T2.1 ✅ · T2.2 ✅ · T2.3 ✅ · T2.4 ✅
Phase 1 (complete): T1.1 ✅ · T1.2 ✅ · T1.3 ✅ · T1.4 ✅ · T1.5 ✅
Phase 0 (complete): T1 ✅ · T2 ✅ · T3 ✅

## Run log
- T1 ✅ 2026-10-19 — no test (scaffolding) · go build ./... exit 0 · git repo initialized (repo-local identity, global untouched)
- T2 ✅ 2026-10-19 — red: compile error (undefined: run) · green: 1 attempt · 7 cases
- T3 ✅ 2026-10-19 — no test (build wiring) · make build/vet/test/fmt/run-demo all pass · +1-line .gitignore for the built binary
- T1.1 ✅ 2026-10-20 — red: compile error (model.Signal undefined) · green: 1 attempt · 2 cases
- T1.2 ✅ 2026-10-20 — red: compile error (Action.String undefined) · green: 1 attempt · 10 cases
- T1.3 ✅ 2026-10-20 — red: compile error (SensitivityRank undefined) · green: 1 attempt · 8 cases
- T1.4 ✅ 2026-10-20 — red: compile error (Validate/DecodeSignal undefined) · green: 1 attempt · 21 cases · test-plan fix: "limits accepted" split into lower/upper rows (LocalHour can't be 0 and 23 at once)
- T1.5 ✅ 2026-10-20 — audit: all §9 bullets already mapped; +1 regression test (UnmarshalText no-state-change, passed immediately) · model 100% stmt coverage · no model bugs found
- review #2 fix ✅ 2026-10-20 — thermonuclear review: JSONText test re-tabled with concrete types (deleted any-dispatch/assertDecision/anyErr), single actionNames table (deleted knownAction) · net −50 lines · 43 subtests + coverage identical
- T2.1 ✅ 2026-10-20 — red: compile error (Policy/New undefined) · green: 1 attempt · 4 cases (3 sort + 1 clone)
- T2.2 ✅ 2026-10-20 — red: compile error (Engine.Evaluate undefined) · green: 1 attempt · 6 cases
- T2.3 ✅ 2026-10-20 — red: behavioral (zero-match returned Allow) · green: 1 attempt · 3 cases
- T2.4 ✅ 2026-10-20 — sweep: 0 red (gap-closing, T1.5 precedent; no engine bugs) · 5 scenario shapes + 6 strictest-wins + 5+3 no-silent-pass + 5 determinism · engine 100% stmt coverage · §9 audit: all bullets + 4 properties mapped to named tests
- review #3 fix ✅ 2026-10-20 — thermonuclear review: 3 findings, all minor, all fixed (cmp.Or two-key comparator + builtin max in Evaluate; fail-closed table deduped into shared failClosedCases, killing the parallel names slice + duplicated F01 literal) · net −9 lines · 36 subtests + coverage identical
- T3.1 ✅ 2026-10-20 — red: build failure (no non-test Go files in internal/policies — Pack undefined) · green: 1 attempt · 10 cases
- T3.2 ✅ 2026-10-20 — red: behavioral (policyByID("P11") Fatalf: policy P11 not in pack) · green: 1 attempt · 11 subtests
- T3.3 ✅ 2026-10-20 — sweep: 0 red (gap-closing, T1.5/T2.4 precedent; no pack bugs) · 18 boundary rows + 6 §6 scenario shapes
- T3.4 ✅ 2026-10-20 — sweep: 0 red (no transcription bugs; pack + table both §4-verbatim, spec diff-checked before T3.1) · policies 100% stmt coverage · §9 audit: fire/silent → FireSilence+CatchAllExclusive; boundaries → Boundaries; IDs/priorities/catch-all-once/pillars → Metadata; §6 rows → ScenarioShapes — all four bullets + all eleven §4 rows mapped
- review #4 fix ✅ 2026-10-20 — thermonuclear review: 1 minor finding fixed (P11 hand-rolled any-match loop → slices.ContainsFunc) + 1 nit (validPillars moved above its use) · net −3 lines · 45 subtests + coverage identical
- T4.1 ✅ 2026-10-24 — red: build failure (no non-test Go files in internal/scenario — scenario.Load undefined, T3.1 precedent) · green: 1 attempt (after fixing 2 test-side compile bugs: strings.Replace count arg; unknown-case-field row appended outside the case object → test-plan row corrected) · 1 + 13 + 7 subtests · scenario 100% stmt coverage
- T4.2 ✅ 2026-10-24 — red: behavioral (all six Load: no such file or directory) · green: 1 attempt · 6 subtests · note-scenario signal = §6 example verbatim, others = TestPack_ScenarioShapes pins
- T4.3 ✅ 2026-10-24 — sweep: 0 red (gap-closing, T1.5/T2.4/T3.3 precedent; no divergence) · 6 expectation subtests + coverage · glob guard: ≥6 files + six pinned names (empty glob fails, never passes vacuously)
- T4.4 ✅ 2026-10-24 — hand-derivation vs §4 (test-plan table), all six agree, 0 divergences: happy→P11 only→allow · note-scenario→P01,P02,P03,P04,P06,P08 fire (P05 not-enrolled, P07 65≤70, P09 finance, P10 finance silent)→block · stolen-credential→P01,P04,P08 (P02/P03 familiar, P06 internal, P10 non-payroll silent)→stepUpMFA · insider→P10 only (P06 internal∧compliant silent)→block · byod→P04 only (P06 internal, P01 mfa-on silent)→limited read-only · attacker→P01,P02,P03,P04,P06,P07,P10 (P05 unmanaged, P08 85>70, P09 guest silent)→block, 7 matched = §5 example footer · spec §10 Phase 4 boxes ticked
- review #5 fix ✅ 2026-10-24 — thermonuclear review (APPROVE): 3 nits fixed (no-op `model.Action` casts → plain consts; ReasonsContain field comment reasons→decision's Matched; happy/insider expect blocks normalized to the multi-line pack format) · net +6 lines (JSON cosmetics only) · 45 subtests + coverage identical
- T5.1 ✅ 2026-10-24 — red: build failure (no non-test Go files in internal/report — T3.1/T4.1 precedent) · green: 1 attempt · 5 + 3 subtests · test-plan fix ×2: continuation-line indices off by one in attacker-row + stack rows (pre-red and first-green; production output was correct both times, fixture misindexed lines[2]/[3] as lines[1]/[2])
- T5.2 ✅ 2026-10-24 — red: compile error (undefined: model.DecodeSignals) + behavioral (decide still exit-70 stub) · green: 1 attempt · 6 + 14 subtests + DecideFormatJSON · run gains stdin io.Reader param (main passes os.Stdin) · mid-green fix: signal.go edit accidentally replaced DecodeSignal, restored immediately
- T5.3 ✅ 2026-10-24 — red: compile error (undefined: report.PackResult) + behavioral (run stub 70) · green: 1 attempt · 1 + 7 subtests · test-plan fix: PASS/FAIL assertions matched the summary header cells (Count("PASS ")==7, Contains("FAIL")) → per-line HasPrefix counting, header cells excluded
- T5.4 ✅ 2026-10-24 — red: compile error (undefined: report.Policies) + behavioral (policies stub 70) · green: 1 attempt · 1 + 2 subtests · report imports engine (pure pkg) for the Policies table
- T5.5 ✅ 2026-10-24 — sweep: 0 red (T1.5/T2.4/T3.3/T4.3 precedent; no wiring bugs) · 16 matrix rows, all §5 codes incl. audit-stub 70 · examples/attacker-signals.json = pack signal verbatim (python assert, no re-typing) · demo session run: decide → 7 reasons, pillars identity/devices/data/applications, exit 3; run scenarios → 6 PASS + summary, exit 0 · coverage: model/engine/policies/scenario 100%, report 96.6%, main 93.2% · spec §10 Phase 5 boxes ticked
- Phase 5 complete — §5 example session satisfied under the documented divergence resolution (examples/ fixture for the illustrative filename; decide scenarios/attacker.json → 65 pinned as the normative rule)
- review #6 fix ✅ 2026-10-24 — thermonuclear review (APPROVE): 4 findings, all minor, all fixed — cmdDecide's 4× usage pair → one guard switch; strictest int-tracked → model.Action (cast once at return, §2 comment); cmdRun's double-meaning e → eng; +TestExamples_AttackerMatchesPack pinning the demo fixture to its pack (drift guard for T7.3) · rejected with reasons: generic table writer (stringly rows), Row{User string} (less self-describing), DecodeSignal/DecodeSignals unification (different wire shapes), exit-code consts (spec's own vocabulary) · stray working-tree revert of test-plan.md restored from HEAD (uncommitted, no history impact)
- T6.1 ✅ 2026-10-24 — red: compile error (undefined: report.NewAuditWriter) · green: 1 attempt · 6 subtests · test-side fix: pinned wantLine constant's tail mistyped ("P01"} + "]}), corrected to the scratch-marshaled literal — production output was right · literal generated via scratch marshal (tmp-gen, deleted)
- T6.2 ✅ 2026-10-24 — red: behavioral (flag provided but not defined: -audit, 8/9 subtests) · green: 1 attempt · 6 + 3 subtests · cmdRun switched to its own FlagSet (usage rows unchanged, exit map preserved) · refactor skipped: audit open/append/close loops differ per command, auditFooter dedupes the shared line
- T6.3 ✅ 2026-10-24 — red: compile error (undefined: report.ReadAudit/AuditSummary) + behavioral (audit 70 stub, 5 + 4 subtests) · green: 1 attempt · 9 + 2 + 5 subtests · test-side fix ×2: unknown-field/unknown-action rows listed two substrings for one error → single substring pinning line number + offending token (runner's one-substring-per-error contract kept) · stub rows retired: TestRun_Dispatch "audit is stubbed" deleted, TestRun_ExitCodes 70-row → 0/64/65/65 rows
- T6.4 ✅ 2026-10-24 — sweep: 0 red (T1.5/T2.4/T3.3/T4.3/T5.5 precedent; no wiring bugs) · TestRun_AuditAppendOnly (two run() opens, line 1 byte-identical) + TestRun_AuditTruncated (no-\n half-line → 65, valid record still printed/counted) · demo session: decide --audit → §5 table + `audit: 7 policies matched, decision appended to audit.jsonl`, exit 3; second run exit 3, 2 lines, line 1 unchanged; audit → table + `counts: allow=0 limited=0 stepUpMFA=0 block=2`, exit 0; garbage line 3 → reported, exit 65 · divergence noted: §5's sketch puts --audit after the positional, stdlib flag (§5-pinned) parses flags-first — demo runs flags-first, §5's session ordering is illustrative (Phase 5 divergence-note precedent) · .gitignore += audit.jsonl · spec §10 Phase 6 boxes ticked
- Phase 6 complete — §7 append-only JSONL + §5 audit subcommand satisfied; §10 Phase 6 done-when "re-running decide never mutates old lines" pinned by TestRun_AuditAppendOnly
- review #7 fix ✅ 2026-10-24 — thermonuclear review: 5 findings, all fixed — F3 ReadAudit blessed trailing content after a valid record ({rec}{rec} silently dropped the second!) → dec.More() check + 2 red/green rows (trust boundary); F1 cmdRun walked with filepath.Join computed twice → hoisted; F2 usage grouped --format under decide/run though it's decide-only → "(decide only)"; F4 main-test records() was a pre-T6.3 lenient shim duplicating ReadAudit → routed through the real reader (CLI tests now writer→reader round-trips); F5 TestRun_DecideAudit's wantOut/wantErr fields dead → deleted · rejected with reasons: shared auditSink type over open/append/close/footer (two call sites, different shapes — wrapper adds a layer to save ~8 lines), append-inside-evaluate-loop (interleaves IO), map over [4]int · noted: main_test.go 766 lines — split per subcommand if Phase 7 grows it · coverage after: report 93.4%, main 87.3%
- T7.1 ✅ 2026-10-25 — sweep: 0 red (plan-time baseline re-verified; T1.5/T2.4/T3.3/T4.3/T5.5/T6.4 precedent) · gofmt -l . empty · vet 0 · 6 packages ok, no .go files touched
(appended by execute-plan)

## Not building (YAGNI, Phase 7)
- CI workflow / badges / coverage shields — trigger: repo published or gains collaborators
- `make check` aggregate target (fmt+vet+test in one) — the three commands already exist; trigger: CI wants a single entry point
- main_test.go split per subcommand (noted in review #7) — Phase 7 adds zero tests; trigger: next task that grows main_test.go past ~800 lines
- README drift-guard Go test (grepping README from `go test`) — brittle; `TestExamples_AttackerMatchesPack` already pins the demo fixture's data; trigger: observed README rot
- `policies` / `audit` read-back steps in the demo transcript — §10 T7.3 pins "§5 session + run summary"; trigger: README readers ask for the audit round-trip
- `go test -race` — zero goroutines in the codebase (`grep 'go func'` empty); trigger: first concurrent code
- Version bump / `git tag v1.0.0` / CHANGELOG / LICENSE — no release task in the spec; trigger: user asks for a release
- §11 stretch goals (networks/infrastructure policies, sortresp, attack replay, web dashboard, config-file policies, CIA tags, step-up mode) — explicitly post-v1

## File map (Phase 7 touches only these)
- `README.md` (new) — T7.2: H1 + what/why intro, `Install`, `Usage`, `Exit codes`, `Policies`, `Scenarios`, `Demo` (header only). T7.3: transcript + divergence notes into `## Demo`
- `Makefile` (edit) — T7.4: replace the Phase 0 `run-demo` stub with the pinned recipe (decision 4)
- `spec.md` (edit) — T7.4: tick §10 Phase 7 boxes (all four at once, phase-complete precedent from T4.4/T5.5/T6.4)
- `plans/plan.md` (edit) — run log entries per task (this file)
- `plans/pr-phase7.md` (new) — PR body at wrap-up (per-phase precedent)
- Untouched: every `.go` file — `main.go`, `main_test.go`, all of `internal/` (no behavior change; fmt/vet/test already green at plan time)

## Tasks

### T7.1 — fmt/vet/test clean-baseline sweep
- Does: record the shippable baseline — `gofmt -l .` prints nothing, `go vet ./...`
  exits 0, `go test ./... -count=1` exits 0. Verified green at plan time (2026-10-25);
  0 red expected (T1.5/T2.4/T3.3/T4.3/T5.5/T6.4 precedent). If anything is red it is a
  bug — fix the root cause in code, never the tests. No file edits expected; this task
  exists so the §10 T7.1 box is earned, not assumed
- Files: none (whatever gofmt/vet names, if anything)
- Depends: —
- Acceptance: `gofmt -l .` prints nothing; `go vet ./... && go test ./... -count=1` exits 0
- Est: S

### T7.2 — README: what/why, install, usage, exit codes, policy + scenario tables
- Does: `README.md` per pinned decision 2. H1 `# zt-sim — Zero Trust access simulator`
  + one-paragraph what/why (hands-on companion to SC-900 study notes; signals in →
  conditional-access evaluation → Allow/Limited/StepUpMFA/Block with an audit trail;
  verify explicitly / least privilege / assume breach as the mapping, §12). Sections:
  `## Install` — `go build` (zero deps, offline), `make build`, Go version; `## Usage`
  — the four subcommands + `--stdin`/`--format`/`--audit`/`--version` from §5, four
  actions with one-line meanings; `## Exit codes` — §5's exit-code tables verbatim
  (decide 0–3 by strictest action; run 0/1; policies/audit 0/64/65); `## Policies` —
  §4's 11-row table verbatim (ID/Priority/Pillar/When/Then/Note, em dashes intact);
  `## Scenarios` — §6's 6-row table verbatim (file/story/expected/matched); `## Demo`
  — header only (T7.3 fills it)
- Files: `README.md` (new)
- Depends: T7.1 (baseline green before docs freeze)
- Acceptance: `test -f README.md && [ "$(grep -c '^## ' README.md)" -eq 6 ] && [ "$(grep -c '^| P[0-9][0-9] ' README.md)" -eq 11 ] && [ "$(grep -c '\.json` |' README.md)" -eq 6 ] && grep -q 'go build' README.md && grep -q '64' README.md && grep -q '65' README.md`
- Est: S

### T7.3 — Demo transcript in README
- Does: capture the real session (verified 2026-10-25, `make build` state) and paste it
  into `## Demo` as one `sh` block with `$` prompts, then the two divergence notes per
  pinned decision 5. Transcript (verbatim — tabwriter spacing and em dashes as
  printed):
  ```sh
  $ ./zt-sim decide --audit audit.jsonl examples/attacker-signals.json
  USER                       ACTION  REASONS
  unknown@throwaway.example  BLOCK   P01: no second factor presented — verify explicitly
                                     P02: unfamiliar sign-in location — re-verify
                                     P03: unfamiliar location and no MFA — assume breach
                                     P04: unmanaged device — least-privilege grant
                                     P06: confidential/high data on non-compliant device
                                     P07: risk score above hard threshold
                                     P10: app outside requester's role — least privilege
  pillars: identity, devices, data, applications
  audit: 7 policies matched, decision appended to audit.jsonl
  $ echo $?
  3
  $ ./zt-sim run scenarios/
  PASS attacker/everything-wrong-at-once
  PASS byod/personal-phone-internal-data
  PASS happy/alice-normal-day
  PASS insider/engineering-requests-payroll
  PASS note-scenario/alice-travel-personal-device
  PASS stolen-credential/attacker-with-alices-password
  PACK               CASES  PASS  FAIL  STRICTEST
  attacker           1      1     0     BLOCK
  byod               1      1     0     LIMITED
  happy              1      1     0     ALLOW
  insider            1      1     0     BLOCK
  note-scenario      1      1     0     BLOCK
  stolen-credential  1      1     0     STEPUPMFA
  $ echo $?
  0
  ```
  Re-capture rather than trust this block if any T7.2/T7.4 change could shift bytes —
  the README must reproduce, not approximate. Clean up `audit.jsonl` after capture
  (`rm -f audit.jsonl`)
- Files: `README.md` (edit)
- Depends: T7.2 (the `## Demo` section exists)
- Acceptance: `[ "$(grep -c 'audit: 7 policies matched, decision appended to audit.jsonl' README.md)" -eq 1 ] && [ "$(grep -cx 3 README.md)" -eq 1 ] && [ "$(grep -c '^PASS ' README.md)" -eq 6 ] && grep -q 'flags precede\|flags come before\|flags-first' README.md && grep -q 'examples/attacker-signals.json' README.md`
- Est: S

### T7.4 — make run-demo reproduces the transcript; fresh-clone v1 check
- Does: replace the Phase 0 `run-demo` stub with the pinned recipe (decision 4).
  Verify from the repo: `make run-demo` twice — identical output both times
  (idempotent via `rm -f audit.jsonl`), make exit 0 despite decide's exit 3, output =
  the README Demo block minus `$`-prompt lines. Verify byte-exactness once by
  diff: `make run-demo > /tmp/demo.out` vs README block with prompt lines stripped;
  note the result in the run log. Then the v1 definition-of-done check **after
  committing all Phase 7 changes** (decision 6):
  `d=$(mktemp -d) && git clone -q . "$d/zt" && cd "$d/zt" && go build ./... && go test ./... -count=1 && make run-demo`
  — exits 0, builds from nothing, prints the transcript. Finally: tick spec §10
  Phase 7 boxes (T7.1–T7.4), `rm -f audit.jsonl`, write `plans/pr-phase7.md`
- Files: `Makefile` (edit), `spec.md` (edit), `plans/plan.md` (edit — run log),
  `plans/pr-phase7.md` (new)
- Depends: T7.3 (the transcript that defines "reproduces")
- Acceptance: `make run-demo > /tmp/demo.out && [ "$(grep -c 'audit: 7 policies matched, decision appended to audit.jsonl' /tmp/demo.out)" -eq 1 ] && [ "$(grep -cx 3 /tmp/demo.out)" -eq 1 ] && make run-demo > /tmp/demo2.out && cmp /tmp/demo.out /tmp/demo2.out && rm -f audit.jsonl` exits 0; plus the fresh-clone command above exits 0 with the transcript printed
- Est: S

## Done when (v1 exit criteria)
```
gofmt -l .                      # prints nothing
go vet ./...                    # exit 0
go test ./... -count=1          # exit 0
make run-demo                   # the README Demo transcript, byte for byte (minus $ prompts); exit 0
# fresh clone (after the Phase 7 commits):
d=$(mktemp -d) && git clone -q . "$d/zt" && cd "$d/zt" \
  && go build ./... && go test ./... -count=1 && make run-demo   # exit 0, transcript printed
rm -f audit.jsonl
```
Plus: spec §10 Phase 7 boxes ticked; README covers what/why, install, usage, exit
codes, §4 policy table, §6 scenario table, demo transcript; v1 definition of done met.

---

# Phase 6 — Audit log (complete, archived)

> Goal: `--audit <file>` appends one JSONL record per decision from `decide` and `run`
> (§7 — existing lines never rewritten), and `zt-sim audit <file>` re-prints the
> decisions with action counts, reporting corrupt lines and exiting 65.
> All tasks ✅ 2026-10-24 (run log above); full task detail in git history /
> `plans/pr-phase6.md`.

- T6.1 `report.AuditRecord` (Decision embedded → §7 field order) + `NewAuditWriter`
  (`O_APPEND|O_CREATE|O_WRONLY`, injected `ts` clock) + `Append` (one Marshal+`\n`,
  one Write) + `Close`
- T6.2 `--audit` on decide + run (input validated before open; append failure → 65
  beats strictest/0-1; footer `audit: %d policies matched, decision appended to %s`;
  cmdRun gained its own FlagSet)
- T6.3 `ReadAudit` (strict per-line decode, corrupt lines → errors with 1-based line
  number, valid records still decode) + `AuditSummary` counts line + `cmdAudit`
  (re-print via `report.Decisions`, corrupt → 65)
- T6.4 sweep: `TestRun_AuditAppendOnly` (line 1 byte-identical across runs) +
  `TestRun_AuditTruncated`; demo session; §5/§7 divergences pinned (flags-first,
  run's `source` = pack path); spec §10 ticks

Files: `internal/report/audit.go` + `audit_test.go`, `main.go` + `main_test.go`,
`.gitignore`.

---

# Phase 5 — CLI (complete, archived)

> Goal: `zt-sim` end-to-end per §5 — decide/run/policies with the exit-code map locked
> by tests and the demo session producing exit 3. `audit` stayed a 70 stub; its rows
> kept the map complete and made the Phase 6 diff visible.
> All tasks ✅ 2026-10-24 (run log above); full task detail in git history /
> `plans/pr-phase5.md`.

- T5.1 `internal/report`: `Row`, `Decisions` tabwriter table (USER/ACTION/REASONS,
  one line per reason) + pillars footer; `DecisionsJSON` (indented `[]Decision`)
- T5.2 `model.DecodeSignals` (strict array, rejects empty/packs) + `cmdDecide`
  (`--stdin`/`--format`, exit = strictest action; `run` gained the stdin param)
- T5.3 `report.PackResult`/`PackSummary` + `cmdRun` walker (PASS/FAIL lines, summary
  table, exit 0/1/65)
- T5.4 `report.Policies` (ID/PRIORITY/PILLAR/ACTION/NOTE) + `cmdPolicies`
- T5.5 `TestRun_ExitCodes` matrix (all §5 codes) + `examples/attacker-signals.json`
  demo fixture; §5 divergence resolved and pinned: decide takes signal arrays only —
  the pack filename in §5's example session is illustrative, feeding it exits 65

Files: `internal/report/report.go` + `report_test.go`, `internal/model/signal.go` +
`model_test.go`, `main.go` + `main_test.go`, `examples/attacker-signals.json`.

---

# Phase 4 — Scenarios & acceptance (complete, archived)

> Goal: the six §6 scenario packs exist on disk and run as acceptance tests —
> `internal/scenario` loads + validates packs and checks expectations, and
> `go test ./...` walks `scenarios/*.json` green.
> All tasks ✅ 2026-10-24 (run log above); full task detail in git history /
> `plans/pr-phase4.md`.

- T4.1 `internal/scenario` (engine-free, imports only model): `Pack`/`Case`/
  `Expectation` (`Action *model.Action` — pointer so "required" beats the Allow zero
  value), `Load` = ReadFile → strict decode → `validate()` (name → ≥1 case → per
  case: name → `Signal.Validate` → `expect.action` non-nil → scope ⇒ limited),
  errors path-prefixed; `Case.Check(d)`: action → every `reasonsContain` ID ∈
  `d.Matched` → scope only when set. scenario at 100% stmt coverage
- T4.2 the six packs authored from §6 (signals = TestPack_ScenarioShapes pins,
  note-scenario = §6 example verbatim); `Name` == file stem
- T4.3 `TestPacks_Expectations` walker (glob guard: ≥6 files + six pinned names, so
  an empty glob fails) + `TestPacks_ActionCoverage` — wired into `go test ./...`
- T4.4 hand-derivation of all six vs §4 matchers, 0 divergences (table in the
  Phase 4 test-plan revision, git history)

Files: `internal/scenario/scenario.go`, `internal/scenario/scenario_test.go`,
`scenarios/{happy,note-scenario,stolen-credential,insider,byod,attacker}.json`.

---

# Phase 3 — Policy pack (complete, archived)

> Goal: `internal/policies` complete and tested — the v1 pack P01–P11 exactly as §4.
> All tasks ✅ 2026-10-20 (run log above); full task detail in git history /
> `plans/pr-phase3.md`.

- T3.1 `Pack()` single export surface; P01–P10 matchers as one-line closures,
  formulas pinned in test-plan, Notes verbatim from §4 (incl. em dashes)
- T3.2 P11 catch-all (Allow, priority 90): `slices.ContainsFunc` over the ten core
  `When`s — matches iff none did, so `Matched`/`Reasons` never carry a false
  all-clear next to a real finding
- T3.3 sweep: 18 boundary rows (risk 70/71/40/39, intern hour 4/5/0 + role guard,
  sensitivity × device posture, P04/P05 disjointness, P02-without-P03) + the six
  §6 scenario shapes through the real pack (exact Action/Scope/Matched/Pillars)
- T3.4 sweep: §4 metadata pinned positionally (ID/Priority/Pillar/Then/Scope/Note),
  11 policies, unique IDs, exactly one Allow = P11, scope iff Limited, pillars
  valid; policies at 100% statement coverage

Files: `internal/policies/policies.go`, `internal/policies/policies_test.go`.

---

# Phase 2 — Engine (complete, archived)

> Goal: `internal/engine` complete and tested — a pure evaluator where sorted policies in
> yield a deterministic Decision out, strictest match wins, and every path fails closed.
> All tasks ✅ 2026-10-20 (run log above); full task detail in git history /
> `plans/pr-phase2.md`.

- T2.1 `Policy` verbatim from §2 + `Engine`/`New` cloning and pre-sorting by
  (Priority, ID) via `slices.SortFunc` + `cmp.Or` (`engine.go`, white-box sort tests)
- T2.2 `Evaluate`: every policy evaluated (no short-circuit), `"<ID>: <Note>"` reasons,
  unique pillars in evaluation order, numeric-max severity, first-Limited scope kept
  only when Limited wins
- T2.3 Fail-closed: zero matches / empty pack → `Block` with the pinned §3.5 engine
  reasons (em dash included)
- T2.4 §9 sweep: five scenario shapes on a synthetic `miniPack()`, strictest-wins,
  no-silent-pass, determinism (double-eval + reordered-`New`); engine 100% stmt coverage

Files: `internal/engine/engine.go`, `internal/engine/engine_test.go` (black-box),
`internal/engine/engine_internal_test.go` (white-box `New` sort).

---

# Phase 1 — Domain model (complete, archived)

> Goal: `internal/model` complete and tested — every §2 rule enforced at load, JSON tags
> exact, all four actions marshal as spec text. All tasks ✅ 2026-10-20 (run log above);
> full task detail in git history / `plans/pr-phase1.md`.

- T1.1 `Signal` + `Decision` structs with exact §2 JSON tags + bare `Action` type
  (`signal.go`, `decision.go`, `action.go`)
- T1.2 `Action` behavior: `String()`, `MarshalText`/`UnmarshalText`, severity = iota
  order (asserted numerically, no extra API)
- T1.3 `SensitivityRank`: public < internal < confidential < high, case-sensitive,
  error names the value
- T1.4 `Signal.Validate()` (first error, spec-table order) + `DecodeSignal`
  (`DisallowUnknownFields` + Validate)
- T1.5 §9 completeness sweep — model at 100% statement coverage; review #2 re-tabled
  tests with concrete types (net −50 lines, 43 subtests)

Files: `internal/model/{signal,action,decision}.go`, `internal/model/model_test.go`.

---

# Phase 0 — Scaffold (complete, archived)

> Goal: a building, vet-clean skeleton — `zt-sim --version` prints, bad usage exits 64,
> `go build ./...` / `go vet ./...` / `make test` all pass.
> Task mapping: T1 ↔ spec T0.1, T2 ↔ spec T0.2, T3 ↔ spec T0.3.

## Tasks

### T1 — Init module, scaffold directories, git repo
- Did: `go mod init zt-sim` (go 1.27, zero deps); created `internal/{model,engine,policies,scenario,report}`, `scenarios/`; `git init` (no repo existed up the tree)
- Files: `go.mod`, scaffold dirs
- Test: none — scaffolding only

### T2 — main.go: testable dispatch with `--version` and usage exits
- Did: `run(args []string, stdout, stderr io.Writer) int` — `--version` prints
  `zt-sim version 0.1.0` (exit 0); no args / unknown subcommand → usage on stderr
  (exit 64); four subcommands wired to `not implemented yet` stubs (exit 70, EX_SOFTWARE)
- Files: `main.go`, `main_test.go`
- Test: `TestRun_Dispatch`, 7 cases (table-driven)

### T3 — Makefile: build / test / fmt / vet / run-demo
- Did: five phony targets, stdlib tooling only; `run-demo` is a stub (`./zt-sim --version`)
  that Phase 7 (T7.4) expands
- Files: `Makefile`, `.gitignore` (built binary)
- Test: none — covered by T2's suite via `make test`
