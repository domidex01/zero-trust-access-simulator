# Plan — zero-trust-access-simulator (Phase 6 — Audit log)
> Goal: `--audit <file>` appends one JSONL record per decision from `decide` and `run`
> (§7 — existing lines never rewritten), and `zt-sim audit <file>` re-prints the
> decisions with action counts, reporting corrupt lines and exiting 65.
> Source: spec.md §10 Phase 6 (T6.1–T6.4), §7 (record schema + append-only invariant),
> §5 (--audit flag, audit subcommand, exit codes, session footer line), §8 (audit
> writer/reader lives in `internal/report`), §9 (report: write-twice byte-compare +
> corrupt-line detection).
> Update run: Phases 5/4/3/2/1/0 complete, ids stable, archived below. Task ids match
> spec §10 verbatim (T6.1–T6.4).
>
> **Decisions pinned up front** (spec leaves these open; execution needs no judgment):
> 1. `report.AuditRecord{TS, Cmd, Source, Signal}` with `model.Decision` **embedded** —
>    its json tags flatten to top level in §7's exact field order (ts, cmd, source,
>    signal, action, scope, reasons, pillars, matched; struct field order = JSON order).
> 2. `ts` comes from an injected clock — `NewAuditWriter(path, now func() time.Time)`;
>    main passes `time.Now`, tests a fixed clock (project rule: no `time.Now` inline in
>    code under test). §7's "wall-clock write time" is the default wiring, not a hardcode.
> 3. `source`: decide = the path exactly as given, or `"stdin"` under `--stdin`; run =
>    the pack's file path (`filepath.Join(dir, entry)`) — §7's example only pins decide.
> 4. Audit footer (§5 session line, printed once per invocation, after all output):
>    `audit: %d policies matched, decision appended to %s`, %d = Σ len(Matched) over
>    appended decisions — matches §5 verbatim for the one-decision demo.
>    `// shortcut: "decision" stays singular for N>1 appended records; reword when a
>    consumer of the footer cares`. Printed by **main**, not report — `report.go`'s §5
>    tables stay frozen this phase.
> 5. Audit read is decode-only strict: `DisallowUnknownFields` (like every decoder here);
>    a line is valid iff it decodes into `AuditRecord` — bad `ts` (RFC 3339) and unknown
>    action text fail decode for free. §7's bar is "validates every line parses", not
>    re-validation of the embedded signal (the writer already validated it).
> 6. Ordering & precedence: input is decoded/validated **before** the audit file is
>    opened (a bad input must not create an empty audit file); audit open/append failure
>    exits 65 and beats decide's strictest 0–3 and run's 0/1.

## Status
Phase 6:
- [x] T6.1 `internal/report`: `AuditRecord` + `AuditWriter` — `O_APPEND|O_CREATE|O_WRONLY`, one JSON line per decision, `ts` = injected write clock
- [x] T6.2 Wire `--audit` into `decide` and `run` (append per decision, footer, 65 on audit failure)
- [x] T6.3 `audit` subcommand: `ReadAudit` + counts; re-print decisions, corrupt lines reported → exit 65
- [ ] T6.4 Append-only + corruption sweep at CLI level; demo session with `--audit`; spec §10 ticks

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
(appended by execute-plan)

## Not building (YAGNI, Phase 6)
- Log rotation / size caps — a simulator's log is demo-scale; trigger: a long-running
  deployment writes unbounded JSONL
- `--audit` on `policies` — §5 pins the flag on decide/run only; trigger: spec change
- Filters on `audit` output (`--user`, `--since`, action args) — trigger: logs outgrow
  one screen
- `audit --format json` — §5 defines no machine payload for audit; trigger: a consumer
  of machine-readable audit summaries
- fsync/durability beyond `O_APPEND` — process-crash atomicity is out of scope; trigger:
  a crash-safety requirement
- Re-validating embedded signals on audit read (`Signal.Validate` in the reader) — §7's
  bar is "every line parses" and the writer already validated; trigger: audit logs from
  writers we don't own
- Buffered audit writes — one `Write` per record is the durability story here; trigger:
  throughput on huge decide arrays

## File map (Phase 6 touches only these)
- `internal/report/audit.go` (new) + `internal/report/audit_test.go` (new) —
  `AuditRecord`, `NewAuditWriter`/`Append`/`Close` (T6.1); `ReadAudit`, `AuditSummary`
  (T6.3). Separate file: `report.go`'s §5 tables stay byte-frozen this phase
- `main.go` (edit) + `main_test.go` (edit) — usage strings gain `--audit`
  (usage const, `decideUsage`, `runUsage`); `cmdDecide`/`cmdRun` flag wiring + footer
  (T6.2); `audit` dispatch → `cmdAudit` + `auditUsage` (T6.3); append-only/truncated
  sweep tests (T6.4)
- `.gitignore` (edit) — `audit.jsonl` demo artifact (T6.4)
- Untouched: `internal/report/report.go`, `internal/{model,engine,policies,scenario}`,
  `scenarios/`, `examples/`, `Makefile` (T7.4 owns run-demo)

## Tasks

### T6.1 — report: AuditRecord + AuditWriter
- Does: `internal/report/audit.go`: `AuditRecord` per pinned decision 1 (Decision
  embedded — flatten, §7 field order); `NewAuditWriter(path string, now func()
  time.Time) (*AuditWriter, error)` — `os.OpenFile(path, os.O_APPEND|os.O_CREATE|
  os.O_WRONLY, 0o644)` (§7: append-only, existing lines never rewritten, missing file
  created); `(*AuditWriter) Append(cmd, source string, s model.Signal, d
  model.Decision) error` — `AuditRecord{TS: now(), Cmd: cmd, Source: source, Signal:
  s, Decision: d}`, `json.Marshal` + one `Write` of `line + "\n"` (no buffering, no
  validation — the CLI validated upstream); `(*AuditWriter) Close() error`. cmd/source
  ride on `Append` (run's source varies per pack; no per-record writer churn)
- Files: `internal/report/audit.go` (new), `internal/report/audit_test.go` (new)
- Depends: —
- Acceptance: `go test ./internal/report/ -run TestAuditWriter -count=1` exits 0
- Est: M

### T6.2 — `--audit` on decide + run
- Does: `cmdDecide`: `audit := fs.String("audit", "", "append decisions to this JSONL audit log")`; after input decode (never before — pinned decision 6), if set:
  `report.NewAuditWriter(*audit, time.Now)` → err 65; after evaluation, `Append("decide",
  source, s, d)` per decision (source = path as given, or `"stdin"`), `Close` → err 65
  (beats strictest); after rendering, footer per pinned decision 4 on stdout. `cmdRun`:
  gains its own `flag.NewFlagSet("run", flag.ContinueOnError)` (output → stderr;
  `--audit` same semantics), `fs.Args()` ≠ 1 → 64; per record `Append("run",
  filepath.Join(dir, entry.Name()), c.Signal, d)`; append error → 65 (beats 0/1);
  footer after `PackSummary`. Usage strings updated: usage const's flag block becomes
  "flags on decide/run:" + `--audit <file>` line; `decideUsage`/`runUsage` gain
  `[--audit <file>]`. Footer lives in main — report untouched
- Files: `main.go` (edit), `main_test.go` (edit)
- Depends: T6.1
- Acceptance: `go test . -run 'TestRun_Decide|TestRun_Run' -count=1` exits 0
- Est: M

### T6.3 — `audit` subcommand: reader + counts
- Does: `internal/report/audit.go`: `ReadAudit(r io.Reader) (records []AuditRecord,
  errs []error)` — `bufio.Scanner`, per line strict decode (`json.NewDecoder` over the
  line, `DisallowUnknownFields`) into `AuditRecord`; corrupt line → `fmt.Errorf("line
  %d: %v", n, err)` into errs while valid records still decode (§7: reported, not
  silently skipped); blank line is corrupt; `scanner.Err()` surfaces as an error.
  `AuditSummary(w io.Writer, records []AuditRecord) error` — one line
  `counts: allow=N limited=N stepUpMFA=N block=N` (§2 severity order, `Action.String()`
  names, all four always). `main.go`: `auditUsage`; `cmdAudit(args, stdout, stderr)
  int` — ≠1 arg → 64; `os.Open` err → 65; `ReadAudit`; each err → stderr
  `audit: <path>: <err>`; valid records re-printed via existing `report.Decisions`
  (rows `{Signal: r.Signal, Decision: r.Decision}` — §7 "re-prints decisions"; skip
  the table when 0 records, avoiding an empty pillars footer), then `AuditSummary`;
  any corrupt line → exit 65 (valid records still printed). `run`'s audit case
  replaces the 70 stub; `TestRun_Dispatch`'s "audit is stubbed" row deleted (replaced
  by `TestRun_Audit`); `TestRun_ExitCodes`' audit-70 row → 0/64/65 rows
- Files: `internal/report/audit.go` (edit), `internal/report/audit_test.go` (edit),
  `main.go` (edit), `main_test.go` (edit)
- Depends: T6.1 (record type; natural order after T6.2 — footer/flag wiring unchanged)
- Acceptance: `go test ./internal/report/ -run 'TestReadAudit|TestAuditSummary' -count=1 && go test . -run 'TestRun_Audit|TestRun_ExitCodes' -count=1` exits 0
- Est: M

### T6.4 — append-only + corruption sweep, demo session
- Does: `TestRun_AuditAppendOnly` — two `run()` decide invocations sharing one
  `--audit` temp file (two opens = two processes' worth); after run 2 the file has 2
  records and line 1 is byte-identical to after run 1 (§10 T6.4 + Phase 6 done-when
  "re-running decide never mutates old lines"). `TestRun_AuditTruncated` — one valid
  record (written via decide `--audit`) + a trailing truncated half-line, no `\n` →
  `audit` reports `line 2` on stderr, exit 65. Sweep per the T1.5/T2.4/T3.3/T4.3/T5.5
  precedent: 0 red expected; a red is a wiring bug — fix the root cause, never the
  test. Then the demo session: `make build`; `decide examples/attacker-signals.json
  --audit audit.jsonl` → §5 table + `audit: 7 policies matched, decision appended to
  audit.jsonl` footer, exit 3; same command again → 2 lines in the log, line 1
  unchanged; `audit audit.jsonl` → re-printed table + counts, exit 0;
  `printf 'garbage\n' >> audit.jsonl` + `audit audit.jsonl` → line 3 reported, exit
  65. `.gitignore` += `audit.jsonl`; tick spec §10 Phase 6 boxes; record the session
  in the run log
- Files: `main_test.go` (edit), `.gitignore` (edit), `plans/plan.md` (run log),
  `spec.md` (ticks)
- Depends: T6.2, T6.3
- Acceptance: `go test . -run 'TestRun_Audit' -count=1 && go test ./... -count=1 && go vet ./...` exit 0, `gofmt -l .` prints nothing, and the demo session above behaves as written (footer line verbatim, exits 3/3/0/65)
- Est: S

## Done when (Phase 6 exit criteria)
```
go test ./... -count=1      # exit 0 — audit writer/reader, decide/run wiring, audit cmd green
go vet ./...                # exit 0
gofmt -l .                  # prints nothing
make build
./zt-sim decide examples/attacker-signals.json --audit audit.jsonl   # §5 table + audit footer, exit 3
./zt-sim decide examples/attacker-signals.json --audit audit.jsonl   # exit 3 again; audit.jsonl now 2 lines, line 1 byte-identical
./zt-sim audit audit.jsonl                                           # decisions re-printed + counts line, exit 0
printf 'garbage\n' >> audit.jsonl && ./zt-sim audit audit.jsonl      # corrupt line 3 reported, exit 65
rm -f audit.jsonl
```
Plus: spec §10 Phase 6 "Done when" — re-running decide never mutates old lines;
tests green.

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
