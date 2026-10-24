# Test plan — zero-trust-access-simulator (Phase 6 — Audit log)

Test files:
- `internal/report/audit_test.go` (new) — **package `report_test`** (black-box; imports
  `zt-sim/internal/model`)
- `main_test.go` (edit) — package `main`, white-box (`run`/`cmd*` unexported);
  `TestRun_Dispatch`'s "audit is stubbed" row is deleted when `cmdAudit` lands (T6.3,
  replaced by `TestRun_Audit`), `TestRun_ExitCodes`' audit-70 row → 0/64/65 rows (T6.3)

Names follow `TestFunc_Scenario`; subtests read as sentences in failure output.

## Code under test (pin these so execution needs no decisions)

```go
// internal/report/audit.go — package report; imports model + stdlib only.
// report.go's §5 tables stay frozen; audit lives in its own file.

// AuditRecord is one JSONL line of the audit log (§7). Decision is embedded
// so action/scope/reasons/pillars/matched flatten to top level — struct field
// order is §7's field order: ts, cmd, source, signal, action, scope, reasons,
// pillars, matched.
type AuditRecord struct {
	TS        time.Time    `json:"ts"`     // write time — injected clock
	Cmd       string       `json:"cmd"`    // "decide" | "run"
	Source    string       `json:"source"` // path as given / "stdin" / pack path
	Signal    model.Signal `json:"signal"`
	Decision               // embedded: action, scope, reasons, pillars, matched
}

// NewAuditWriter opens path O_APPEND|O_CREATE|O_WRONLY 0644 — existing lines
// are never rewritten (§7). now is the ts clock; main passes time.Now.
func NewAuditWriter(path string, now func() time.Time) (*AuditWriter, error)

// Append writes one record: json.Marshal + "\n", one Write call, no buffering,
// no validation (the CLI validated upstream).
func (a *AuditWriter) Append(cmd, source string, s model.Signal, d model.Decision) error
func (a *AuditWriter) Close() error

// ReadAudit strictly decodes every line (DisallowUnknownFields); corrupt lines
// come back as errors carrying their 1-based line number while valid records
// still decode (§7: reported, not silently skipped). Blank line = corrupt.
func ReadAudit(r io.Reader) (records []AuditRecord, errs []error)

// AuditSummary writes one line: "counts: allow=N limited=N stepUpMFA=N block=N"
// — §2 severity order, Action.String names, all four always.
func AuditSummary(w io.Writer, records []AuditRecord) error
```

```go
// main.go (T6.2/T6.3 edits)
// cmdDecide: +fs.String("audit", "", …). After input decode (never before —
// bad input must not create the file): if set, NewAuditWriter(*audit, time.Now)
// → err 65. After evaluation: Append("decide", source, s, d) per decision
// (source = path as given | "stdin"), Close → err 65 (beats strictest).
// After rendering: footer "audit: %d policies matched, decision appended to %s"
// (%d = Σ len(Matched)) — printed by main, report untouched.
//   // shortcut: "decision" stays singular for N>1 records; reword when a
//   // consumer of the footer cares
//
// cmdRun: own flag.NewFlagSet("run", ContinueOnError, output→stderr), same
// --audit flag; fs.Args() != 1 → 64. Per record: Append("run",
// filepath.Join(dir, entry.Name()), c.Signal, d); err → 65 (beats 0/1);
// footer after PackSummary.
//
// cmdAudit(args, stdout, stderr) int: != 1 arg → 64 (auditUsage); os.Open
// err → 65; ReadAudit; each err → stderr "audit: <path>: <err>"; if >0 records:
// report.Decisions(stdout, rows built from records) (re-print per §7); always
// report.AuditSummary; any corrupt line → 65 (valid records still printed).
// Usage strings: const's flag block → "flags on decide/run:" + --audit line;
// decideUsage/runUsage gain [--audit <file>]; auditUsage added.
```

## Shared fixtures

```go
fixedClock := func() time.Time { return time.Date(2026, 1, 15, 3, 14, 0, 0, time.UTC) } // §7's ts

auditSignal := model.Signal{Timestamp: time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC), User: "attacker@x"}
auditDecision := model.Decision{Action: model.Block,
	Reasons: []string{"P01: no second factor presented — verify explicitly"},
	Pillars: []string{"identity"}, Matched: []string{"P01"}}
```

The exact-line literal for T6.1 (auditSignal + auditDecision through the writer —
generate once with a scratch marshal and paste; from then on it's pinned):

```
{"ts":"2026-01-15T03:14:00Z","cmd":"decide","source":"attacker.json","signal":{"timestamp":"2026-01-15T10:00:00Z","user":"attacker@x","role":"","app":"","dataSensitivity":"","location":"","unfamiliarLocation":false,"deviceManaged":false,"deviceCompliant":false,"localHour":0,"mfaPresent":false,"riskScore":0},"action":"block","scope":"","reasons":["P01: no second factor presented — verify explicitly"],"pillars":["identity"],"matched":["P01"]}
```

Signal JSON fixtures for main tests: the Phase 5 `happy/byod/stolen/attacker` strings
(copy verbatim from `main_test.go` — no re-typing). Main-test helpers reused:
`writeFile(t, content) string`, `runCLI(t, args, stdin) (exit, stdout, stderr)`.

## T6.1 — report: AuditWriter

### TestAuditWriter
- Target: `internal/report`, `TestAuditWriter` (package `report_test`)
- Shape: subtests (real files in `t.TempDir()` — O_APPEND is a real-fs behavior,
  fakes would prove nothing)
- Red because: `report.NewAuditWriter` undefined → compile error
- Run: `go test ./internal/report/ -run TestAuditWriter -count=1 -v`

| name | give | want |
|---|---|---|
| record is one JSON line in §7 field order | Append("decide", "attacker.json", auditSignal, auditDecision) | file content == exact literal above + `"\n"` (byte-compare) |
| ts is the injected clock, not marshal time | fixedClock | literal's `"ts":"2026-01-15T03:14:00Z"` — no wall-clock leak |
| O_CREATE: missing file is created | path inside empty t.TempDir() | no error; file exists |
| limited scope flattens top-level | Decision{Limited, Scope: "read-only"} | line contains `"action":"limited","scope":"read-only"` before `"reasons"` |
| reopen appends, never truncates: line 1 byte-identical | writer1: Append+Close; read; writer2 (new open, same path): Append(different user)+Close; read | 2 lines; line 1 == first read byte-for-byte; line 2 decodes to the new user |
| append after Close fails | writer.Close(); Append again | error (write on closed file — surfaced, not ignored) |

## T6.2 — `--audit` on decide + run

### TestRun_DecideAudit (main)
- Target: `main`, `TestRun_DecideAudit`
- Shape: table-driven; audit paths via `writeFile`-style temp paths
- Red because: behavioral — `--audit` isn't a defined flag, fs.Parse fails → run
  returns 64 where 0–3 is wanted
- Run: `go test . -run TestRun_DecideAudit -count=1 -v`

| name | args | wantExit | want |
|---|---|---|---|
| one record per signal, exit still strictest | `decide --audit <f> <[byod,attacker].json>` | 3 | 2 lines; line 1: cmd decide, source = the path, action limited; line 2: action block (read via `report.ReadAudit` or a strict unmarshal) |
| stdin records source "stdin" | `decide --stdin --audit <f>` / happy | 0 | 1 record, Source "stdin", Cmd "decide" |
| footer line per §5 | `decide --audit <f> <attacker.json>` | 3 | stdout contains `audit: 7 policies matched, decision appended to <f>` |
| no --audit → no footer, no file | `decide <attacker.json>` | 3 | stdout lacks `audit: `; no file created |
| unwritable audit path → 65 | `--audit <tempdir>/no/x.jsonl` | 65 | stderr names the path |
| bad input never creates the audit file | `decide --audit <f> scenarios/attacker.json` | 65 | `<f>` does not exist after the run |

### TestRun_RunAudit (main)
- Target: `main`, `TestRun_RunAudit`
- Shape: table-driven; one temp dir with a 2-case pack (happy + byod signals,
  expectations per the Phase 5 `writeFile` pack format)
- Red because: behavioral — run has no `--audit` flag → 64
- Run: `go test . -run TestRun_RunAudit -count=1 -v`

| name | args | wantExit | want |
|---|---|---|---|
| one record per case, source = pack path, cmd = run | `run --audit <f> <tmpdir>` | 0 | 2 records; each Source == `filepath.Join(dir, packfile)`, Cmd "run" |
| footer after summary | same | 0 | stdout contains `audit: 2 policies matched, decision appended to <f>` after the PACK table |
| append failure → 65, beats pass/fail | `run --audit <tempdir>/no/x.jsonl <tmpdir>` | 65 | stderr names the path |

## T6.3 — `audit` subcommand: reader + counts

### TestReadAudit (report)
- Target: `internal/report`, `TestReadAudit`
- Shape: table-driven over reader contents (strings.NewReader)
- Red because: `report.ReadAudit` undefined → compile error
- Run: `go test ./internal/report/ -run TestReadAudit -count=1 -v`

| name | give | want |
|---|---|---|
| valid log decodes fully | exact literal line ×2 (second with user b) | 2 records; fields round-trip (TS, Cmd, Source, Signal.User, Action, Matched); 0 errs |
| corrupt JSON reported with line number | valid + `{oops` | 1 record; errs[0] contains `line 2` |
| unknown field is corrupt | valid line + `,"foo":1` inside | errs (DisallowUnknownFields), line 1 |
| unknown action text is corrupt | `"action":"detonate"` | errs — `Action.UnmarshalText` refuses at decode |
| bad ts is corrupt | `"ts":"yesterday"` | errs — RFC 3339 enforced by `time.Time` decode |
| blank line is corrupt | valid + `` + valid | 2 records; errs[0] `line 2` |
| truncated last line (no trailing \n) still seen | valid + `{"ts":"2026-` | 1 record; errs[0] `line 2` |
| trailing content after the record is corrupt | valid + ` junk` (same line) | errs `line 1: trailing content` — a line must be exactly one record (review #7) |
| two records on one line are corrupt | valid + valid, no \n between | errs `line 1: trailing content` (review #7) |
| empty reader | `` | 0 records, 0 errs |

### TestAuditSummary (report)
- Target: `internal/report`, `TestAuditSummary`
- Shape: table-driven over hand-built records
- Red because: `report.AuditSummary` undefined → compile error
- Run: `go test ./internal/report/ -run TestAuditSummary -count=1 -v`

| name | give | want |
|---|---|---|
| counts in §2 severity order | allow + limited + stepUpMFA + block + block | `counts: allow=1 limited=1 stepUpMFA=1 block=2\n` (exact) |
| empty log | none | `counts: allow=0 limited=0 stepUpMFA=0 block=0\n` |

### TestRun_Audit (main)
- Target: `main`, `TestRun_Audit`
- Shape: table-driven; valid logs produced by a decide `--audit` run first
- Red because: behavioral — audit is still the exit-70 stub
- Run: `go test . -run TestRun_Audit -count=1 -v`

| name | give | wantExit | want |
|---|---|---|---|
| reprints decisions + counts | decide --audit (attacker) → `audit <f>` | 0 | stdout: USER/ACTION/REASONS table with BLOCK + `P01: …` line, pillars footer, `counts: allow=0 limited=0 stepUpMFA=0 block=1` |
| empty log is valid | `audit <empty file>` | 0 | counts all-zero; no table (no empty pillars footer) |
| corrupt line reported, valid still printed | 1 valid + `garbage\n` | 65 | valid row printed; stderr contains `line 2` |
| unreadable file | `audit /nope/x.jsonl` | 65 | stderr `no such file` |
| usage | `audit` · `audit a b` | 64 · 64 | auditUsage |

`TestRun_Dispatch`: delete the "audit is stubbed" row.
`TestRun_ExitCodes`: audit-70 row → `audit <valid>` 0 · `audit` 64 · `audit /nope` 65 ·
`audit <corrupt>` 65.

## T6.4 — append-only + corruption sweep (expected 0 red — T1.5/T2.4/T3.3/T4.3/T5.5 precedent)

### TestRun_AuditAppendOnly + TestRun_AuditTruncated (main)
- Target: `main`
- Shape: two focused tests against the finished T6.1–T6.3 wiring; a red is a wiring
  bug — fix the root cause, never the test
- Run: `go test . -run 'TestRun_Audit' -count=1 -v`

| name | give | want |
|---|---|---|
| re-running decide never mutates old lines | runCLI decide happy --audit f; read line1; runCLI decide attacker --audit f; read | exit 0 then 3; 2 lines; line 1 byte-identical across runs (§10 done-when) |
| truncated log detected via CLI | decide --audit f; append `{"ts":"2026-` (no \n) to f; runCLI audit f | exit 65; stderr `line 2` |

Then: `.gitignore` += `audit.jsonl`; demo session recorded in the run log:

```
make build
./zt-sim decide examples/attacker-signals.json --audit audit.jsonl ; echo $?   # 3; footer: audit: 7 policies matched, decision appended to audit.jsonl
./zt-sim decide examples/attacker-signals.json --audit audit.jsonl ; echo $?   # 3; audit.jsonl now 2 lines, line 1 unchanged
./zt-sim audit audit.jsonl ; echo $?                                          # 0; table + pillars + counts: allow=0 … block=2
printf 'garbage\n' >> audit.jsonl && ./zt-sim audit audit.jsonl ; echo $?     # 65; line 3 reported
rm -f audit.jsonl
```

Tick spec §10 Phase 6 boxes.

---

# Phase 5 — CLI (complete, archived)

Test files: `internal/report/report_test.go` (package `report_test`),
`internal/model/model_test.go` (+`TestDecodeSignals`), `main_test.go`. Named tests:
`TestReport_DecisionsTable`, `TestReport_DecisionsJSON`, `TestDecodeSignals`,
`TestRun_Decide`, `TestRun_DecideFormatJSON`, `TestReport_PackSummary`,
`TestRun_Run`, `TestReport_Policies`, `TestRun_Policies`, `TestRun_ExitCodes`
(16-row §5 matrix). Key pins still in force: tabwriter tables assert trimmed lines
not widths; `run` takes stdin as param 2; PASS/FAIL asserted per-line (HasPrefix),
header cells excluded; decide exit = strictest action as int. Full case tables:
git history / `plans/pr-phase5.md`.

---

# Phase 4 — Scenarios & acceptance (complete, archived)

Test file: `internal/scenario/scenario_test.go`, package `scenario_test`. Named
tests: `TestLoad_Pack`, `TestLoad_SixPacks`, `TestLoad_Invalid`,
`TestCase_Check`, `TestPacks_Expectations` (the acceptance walker),
`TestPacks_ActionCoverage`. Run: `go test ./internal/scenario/ -count=1`.
Full case tables: git history / `plans/pr-phase4.md`.

---

# Phase 2 — Engine (complete, archived)

Test files: `internal/engine/engine_test.go` (black-box, package `engine_test`),
`internal/engine/engine_internal_test.go` (white-box sort). Key pins still in force:
`New` clones + sorts by (Priority, ID); `Evaluate` evaluates all, numeric-max severity,
first-Limited scope kept only when Limited wins, fail-closed reasons verbatim
(`engine: no policy matched — fail closed (assume breach)` /
`engine: no policies loaded — fail closed`). Named tests: `TestNew_Sort`,
`TestNew_DoesNotMutateCallerSlice`, `TestEvaluate_Semantics`, `TestEvaluate_FailClosed`,
`TestEvaluate_ScenarioShapes` (miniPack), `TestEvaluate_StrictestWins`,
`TestEvaluate_NoSilentPass`, `TestEvaluate_Determinism`. Full case tables: git
history / `plans/pr-phase2.md`.

---

# Phase 1 — Domain model (complete, archived)

All in `internal/model/model_test.go`, package `model_test`. Test names:
`TestSignal_JSONRoundTrip`, `TestAction_String`, `TestAction_JSONText`,
`TestAction_SeverityOrder`, `TestSensitivityRank`, `TestSignal_Validate`,
`TestDecodeSignal` (+ T1.5 regression: `TestAction_UnmarshalTextNoStateChange`).
Run: `go test ./internal/model/ -count=1`. Full case tables: git history /
`plans/pr-phase1.md`.

---

# Phase 0 — Scaffold (complete, archived)

## T1 — Module scaffold
- Test: none. Scaffolding has no behavior to assert. Acceptance: `go build ./...` exits 0.

## T2 — main dispatch
- `TestRun_Dispatch` (package `main`, white-box — `run` is unexported): version → 0 +
  `zt-sim version 0.1.0\n`; no args / unknown subcommand → 64 + usage on stderr; four
  subcommand stubs → 70 + `not implemented yet`. Exit codes are the CLI's real API —
  asserted exactly. (Phase 5 replaced the stub rows as each subcommand landed; the
  audit row survives until Phase 6 T6.3.)

## T3 — Makefile
- Test: none. `make test` re-runs T2's suite. Acceptance: `make build &&
  ./zt-sim --version && make vet && make test`, then `make fmt` changes nothing.
