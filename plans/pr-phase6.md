# Phase 6: Audit log — --audit JSONL append + `zt-sim audit` (§7)

<!-- PR body. Title = the H1 above. Base: main ← head: phase-6-audit -->

## Summary

Lands Phase 6 (spec §10 T6.1–T6.4): `internal/report/audit.go` owns the §7
record and both directions of the audit log — `AuditWriter`
(`O_APPEND|O_CREATE|O_WRONLY`, one `json.Marshal` + `\n` per decision, `ts`
from an injected clock so tests stay deterministic) and `ReadAudit` (strict
per-line decode; corrupt lines come back as errors carrying their 1-based
line number while valid records still decode — §7's "reported, not silently
skipped"). `main.go` wires `--audit <file>` into `decide` and `run` (input
validated **before** the file is opened — bad input never creates an audit
file; append/open failure exits 65, beating decide's strictest 0–3 and run's
0/1) and adds `cmdAudit`: re-print the log's decisions through the existing
§5 table, then one `counts: allow=N limited=N stepUpMFA=N block=N` line.
224 lines of production code, 603 of tests, zero new dependencies. The
`audit` exit-70 stub rows are retired; the §5 exit map now covers audit
0/64/65.

Two divergences pinned up front in the plan (execution needed no judgment):

- `source` for run records = the pack file path (§7's example only pins
  decide's); decide records the path as given, or `"stdin"`.
- §5's sketch writes `decide <file> --audit <log>` (flag after positional);
  §5 also pins stdlib `flag`, which parses flags-first only. The demo and
  tests run flags-first — §5's session ordering is illustrative, same
  precedent as the Phase 5 pack-filename resolution.

## What's in

- **`internal/report/audit.go`** (new) — `AuditRecord{TS, Cmd, Source,
  Signal}` with `model.Decision` **embedded**: its json tags flatten to top
  level in §7's exact field order (ts, cmd, source, signal, action, scope,
  reasons, pillars, matched) — struct field order is the wire order, pinned
  byte-for-byte by a test literal generated from a scratch marshal.
  `NewAuditWriter(path, now func() time.Time)`: main passes `time.Now`,
  tests a fixed clock. `Append(cmd, source, signal, decision)`: one Write
  per record, no buffering, no validation (the CLI validated upstream).
  `ReadAudit(r) (records, errs)`: `bufio.Scanner` + per-line
  `DisallowUnknownFields` decode — bad `ts` (RFC 3339) and unknown action
  text fail decode for free; blank line is corrupt. `AuditSummary`: the
  counts line, `[4]int` indexed by `Action` (iota 0–3, decode refuses
  anything else). `report.go`'s §5 tables are untouched — audit is its own
  file.
- **`main.go`** — `cmdDecide`/`cmdRun` gain `--audit` (cmdRun moves to its
  own `flag.NewFlagSet`, ContinueOnError, output → stderr; its usage rows
  and exit codes are unchanged). Footer after the tables:
  `audit: %d policies matched, decision appended to %s` (§5 session line
  verbatim for the one-decision demo; shortcut comment marks the singular
  "decision" for N>1). `auditFooter` is the one shared helper — the
  open/append/close flows genuinely differ per command (one source vs
  per-pack sources), a shared abstraction would be ceremony.
  `cmdAudit`: ≠1 arg → 64, `os.Open` error → 65, per-error stderr
  `audit: <path>: <err>`, records → existing `report.Decisions` (no table
  when 0 records — no empty pillars footer), always `AuditSummary`, any
  corrupt line → 65 with valid records still printed. Usage strings grow
  the `--audit` line.
- **`internal/report/audit_test.go`** (new) — `TestAuditWriter` (6: exact
  §7-order line byte-compare, injected clock, O_CREATE, limited-scope
  flatten, reopen-never-truncates with line 1 byte-identical, append after
  Close errors), `TestReadAudit` (9: round-trip, corrupt-JSON/unknown-field/
  unknown-action/bad-ts/blank/truncated-no-\n/empty), `TestAuditSummary` (2:
  §2 severity order, zero log).
- **`main_test.go`** — `TestRun_DecideAudit` (6: record per signal + exit
  still strictest, stdin source, §5 footer, no-flag no-file, unwritable →
  65, bad input never creates the file), `TestRun_RunAudit` (3: per-case
  records with pack-path source, footer after summary, append failure →
  65), `TestRun_Audit` (5: re-print + counts, empty log valid, corrupt
  reported + valid printed, missing file, usage), `TestRun_ExitCodes`
  audit rows 0/64/65/65, `TestRun_AuditAppendOnly` (two run() opens —
  §10's "re-running decide never mutates old lines" done-when),
  `TestRun_AuditTruncated` (no-\n half-line → 65, valid record still
  printed and counted).
- **`.gitignore`** — `audit.jsonl` (demo artifact; Phase 7's transcript
  writes it in cwd).

## Evidence (all on this branch)

    go test ./... -count=1 -cover    # model/engine/policies/scenario 100% · report 93.1% · zt-sim 87.2%
    go vet ./...                     # clean
    gofmt -l .                       # (no output)
    make build
    ./zt-sim decide --audit audit.jsonl examples/attacker-signals.json   # §5 table + audit footer, exit 3
    ./zt-sim decide --audit audit.jsonl examples/attacker-signals.json   # exit 3; audit.jsonl now 2 lines, line 1 byte-identical
    ./zt-sim audit audit.jsonl                                           # table + pillars + counts: allow=0 … block=2, exit 0
    printf 'garbage\n' >> audit.jsonl && ./zt-sim audit audit.jsonl      # line 3 reported, exit 65

## TDD trail (run log in plans/plan.md)

- T6.1 — red: compile error (`report.NewAuditWriter` undefined) · green: 1
  attempt · test-side fix: the pinned literal's tail mistyped, corrected to
  the scratch-marshaled bytes (production output was right)
- T6.2 — red: behavioral (`flag provided but not defined: -audit`, 8/9
  subtests) · green: 1 attempt
- T6.3 — red: compile error (`ReadAudit`/`AuditSummary` undefined) +
  behavioral (70 stub) · green: 1 attempt · test-side fix ×2: rows listing
  two substrings for one error → single substring pinning line + token
- T6.4 — 0 red: sweep (T1.5/T2.4/T3.3/T4.3/T5.5 precedent); demo session
  verified; spec §10 Phase 6 ticked

## Not built (YAGNI, from the plan)

log rotation / size caps · `--audit` on policies (§5 pins decide/run) ·
audit filters (`--user`/`--since`) · `audit --format json` · fsync beyond
O_APPEND · `Signal.Validate` on read (decode-only per §7 "parses") ·
buffered writes · trailing-flag arg parsing (stdlib flag pinned by §5).
