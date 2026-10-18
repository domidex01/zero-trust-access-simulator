# Zero Trust Access Simulator — Spec

A small, dependency-free Go CLI that acts as a **Zero Trust policy decision engine**. It
reads sign-in requests (with their signals), evaluates them against an ordered set of
conditional-access policies, and outputs a decision — **Allow / StepUpMFA / Limited /
Block** — with a full audit trail explaining *why*.

Built as a hands-on companion to SC-900 (Microsoft *Security, Compliance, and Identity
Fundamentals*) study notes on Zero Trust, conditional access, and least privilege.

---

## 1. Purpose & learning goals

- Practice the Zero Trust guiding principles: **verify explicitly**, **least privilege**,
  **assume breach**
- Turn those concepts into executable logic:
  - Multi-signal evaluation (identity, device, location, data sensitivity, risk)
  - Just-enough-access (limited grants) and step-up authentication
  - Assume breach → fail closed, always log, never silently pass
- Learn Go fundamentals along the way: structs + JSON tags, pure packages, table-driven
  tests, subcommand CLIs with `stdlib flag`

### Non-goals (v1)

- No real authentication, network, or crypto — this is a *simulator*
- No server/UI — CLI only (web dashboard is a stretch goal)
- No persistence beyond JSON/JSONL files on disk
- No external dependencies — stdlib only, so `go build` works offline

---

## 2. Core domain model

### Signal — one access request, the "verify explicitly" input

```go
type Signal struct {
	Timestamp       time.Time `json:"timestamp"`        // RFC 3339, required (engine never uses wall clock)
	User            string    `json:"user"`             // e.g. "alice@contoso.com"
	Role            string    `json:"role"`             // free-form, e.g. "finance", "intern", "engineering"
	App             string    `json:"app"`              // e.g. "payroll-app", "intranet-wiki"
	DataSensitivity string    `json:"dataSensitivity"`  // "public" | "internal" | "confidential" | "high"
	Location        string    `json:"location"`         // ISO 3166-1 alpha-2, e.g. "US", "NG"; "" = unknown
	UnfamiliarLoc   bool      `json:"unfamiliarLocation"`
	DeviceManaged   bool      `json:"deviceManaged"`    // enrolled in device management (Intune-style)
	DeviceCompliant bool      `json:"deviceCompliant"`  // patched, encrypted, healthy
	LocalHour       int       `json:"localHour"`        // 0–23, user's local wall-clock hour at sign-in
	MFAPresent      bool      `json:"mfaPresent"`       // a second factor was already used
	RiskScore       int       `json:"riskScore"`        // 0–100, composite heuristic
}
```

Design notes:

- `LocalHour` replaces the earlier `TimeOfDay time.Time`. The sign-in moment is already
  carried by `Timestamp`; policies only care about *the user's local clock hour*
  (e.g. 03:14 → `3`). This removes timezone parsing pain and makes JSON authoring trivial.
- Sensitivity has a total order used by policies: `public < internal < confidential < high`
  ("high" ≙ *highly confidential*). Expose it as `model.SensitivityRank(s) int`.

**Validation rules** (enforced at load; violation → input rejected, exit 65):

| Rule | Rationale |
|---|---|
| `user`, `role`, `app` non-empty | can't evaluate an anonymous request |
| `dataSensitivity` ∈ enum | typo protection |
| `0 ≤ riskScore ≤ 100`, `0 ≤ localHour ≤ 23` | bounds |
| `location` is `""` or 2 uppercase letters | ISO alpha-2 |
| `!DeviceManaged ⇒ !DeviceCompliant` | compliance presupposes management — an unmanaged device cannot be *verified* compliant |
| `timestamp` non-zero, RFC 3339 | engine determinism: no wall-clock reads |

JSON decoding uses `json.Decoder` with `DisallowUnknownFields`, so misspelled keys fail
loudly instead of silently zero-valueing a signal.

### Action — the four outcomes

```go
type Action int

const (
	Allow Action = iota // 0
	Limited             // 1
	StepUpMFA           // 2
	Block               // 3
)
```

`Action` also implements `String()` / `MarshalText()` as `"allow"`, `"limited"`,
`"stepUpMFA"`, `"block"`.

**Severity order (strictest wins): `Block > StepUpMFA > Limited > Allow`.**

Rationale — a `Limited` grant hands out access *now*, merely narrowed; a `StepUpMFA`
outcome withholds access entirely until a second factor is presented. Strictness = how
much verification still stands between the requester and the resource. This ordering also
gives the right story for the stolen-credential scenario: an attacker holding only a
password gets stopped by the MFA challenge, not gifted read-only access.

| Action | Meaning | Exit code (= severity) |
|---|---|---|
| `Allow` | all signals clean, low risk | 0 |
| `Limited` | identity checks out, but device posture / least privilege caps the grant (read-only, time-boxed JIT) | 1 |
| `StepUpMFA` | some doubt (no second factor yet, unfamiliar location, medium risk) → require MFA before any grant | 2 |
| `Block` | high risk or hard-fail conditions | 3 |

### Decision — the output, with reasons as an audit trail

```go
type Decision struct {
	Action  Action    `json:"action"`
	Scope   string    `json:"scope"`   // set for Limited grants, e.g. "read-only"; "" otherwise
	Reasons []string  `json:"reasons"` // "<ID>: <Note>" per matched policy, evaluation order
	Pillars []string  `json:"pillars"` // unique pillars of matched policies, evaluation order
	Matched []string  `json:"matched"` // policy IDs that matched, evaluation order
}
```

### Policy — a named, ordered rule

```go
type Policy struct {
	ID       string
	Priority int               // lower runs first; ties broken by ID (ascending)
	Pillar   string            // identity | devices | applications | data | networks | infrastructure
	When     func(Signal) bool // signal matcher
	Then     Action
	Scope    string // populated when Then == Limited, e.g. "read-only"
	Note     string // human-readable rule description; becomes the audit line
}
```

---

## 3. Evaluation semantics

1. Sort policies by `(Priority asc, ID asc)` — the tie-break makes reason order deterministic.
2. Evaluate **every** policy (no short-circuit) — all matches land in `Reasons`, in order.
3. Final action = strictest match by severity (`Block > StepUpMFA > Limited > Allow`).
   `Scope` is taken from the strictest matching `Limited` policy (empty if the final
   action isn't `Limited`).
4. `Pillars` = unique pillars of matched policies, in evaluation order.
5. **Fail closed**: zero matched policies → `Block` with reason
   `engine: no policy matched — fail closed (assume breach)`. An empty policy pack →
   `Block` with `engine: no policies loaded — fail closed`. The engine never defaults to
   a grant.

The engine package is **pure**: no I/O, no clock, no randomness — same input, same
output, same reason order. That is what makes table-driven testing trivial.

---

## 4. Starter policy pack (v1)

Policies live in code (`internal/policies`), not config — typed Go funcs beat hand-rolled
config parsing at this size. Every matcher reads fields that actually exist on `Signal`.

| ID | Priority | Pillar | When | Then | Note (audit line) |
|----|----------|--------|------|------|--------------------|
| P01 | 10 | identity | `!MFAPresent` | StepUpMFA | no second factor presented — verify explicitly |
| P02 | 20 | identity | `UnfamiliarLoc` | StepUpMFA | unfamiliar sign-in location — re-verify |
| P03 | 20 | identity | `UnfamiliarLoc && !MFAPresent` | Block | unfamiliar location **and** no MFA — assume breach |
| P04 | 30 | devices | `!DeviceManaged` | Limited (read-only) | unmanaged device — least-privilege grant |
| P05 | 30 | devices | `DeviceManaged && !DeviceCompliant` | Limited (read-only) | enrolled but non-compliant device — least-privilege grant |
| P06 | 40 | data | `SensitivityRank ≥ confidential && !DeviceCompliant` | Block | confidential/high data on non-compliant device |
| P07 | 50 | identity | `RiskScore > 70` | Block | risk score above hard threshold |
| P08 | 60 | identity | `40 ≤ RiskScore ≤ 70` | StepUpMFA | elevated risk — verify explicitly |
| P09 | 70 | identity | `LocalHour < 5 && Role == "intern"` | StepUpMFA | odd-hours intern activity |
| P10 | 80 | applications | `Role != "finance" && App == "payroll-app"` | Block | app outside requester's role — least privilege |
| P11 | 90 | identity | **catch-all**: no P01–P10 matched | Allow | all signals clean — explicit allow |

Notes on the pack:

- **P11 is the only source of `Allow`** and matches iff no other policy in the pack
  matched. Combined with rule 5 above, the pack can never silently grant.
- **P02 is not dead weight** even though P03 dominates its weak half: a user *with* MFA
  from an unfamiliar location still gets `StepUpMFA` via P02 alone (re-verification).
- **P04/P05 are disjoint** given the validation invariant `!DeviceManaged ⇒ !DeviceCompliant`
  (unmanaged ⇒ P04; enrolled-but-unhealthy ⇒ P05).
- Risk thresholds are exact: `riskScore = 70` → P08 fires (StepUpMFA); `71` → P07 (Block).
- `LocalHour < 5` = the half-open window `[00:00, 05:00)` local time — no boundary ambiguity.
- Pillar coverage in v1: identity, devices, data, applications. `networks` and
  `infrastructure` are valid pillar values but have no v1 policies — location/time
  telemetry is attributed to identity for now; adding a networks rule is a stretch goal.

---

## 5. CLI design

Binary name: `zt-sim`. Stdlib `flag` with one `flag.NewFlagSet` per subcommand — no cobra
in v1 (zero deps). Usage errors exit 64, input/data errors exit 65.

```
zt-sim decide <signals.json>   # evaluate a JSON array of signals, print decisions
zt-sim decide --stdin          # read ONE signal as a single JSON object from stdin
zt-sim run <scenarios-dir>     # run scenario packs, print summary + expectation results
zt-sim policies                # list loaded policies: ID, priority, pillar, action, note
zt-sim audit <audit.jsonl>     # print + validate an audit log (counts by action)

Flags on decide/run: --audit <file>   append decisions to an append-only JSONL audit log
                    --format table|json (default table)
Global: --version
```

Exit codes:

| Command | Codes |
|---|---|
| `decide` | 0 Allow, 1 Limited, 2 StepUpMFA, 3 Block (strictest decision across all inputs), 64 usage, 65 bad input |
| `run` | 0 all expectations pass, 1 ≥1 expectation failed, 64 usage, 65 bad pack |
| `policies`, `audit` | 0 ok, 64 usage, 65 unreadable/corrupt log |

`decide` input files must be a JSON **array** of signal objects (scenario packs with
`expect` fields are rejected there — use `run`). `--stdin` accepts exactly one JSON
object, also strict (`DisallowUnknownFields`).

Example session:

```
$ zt-sim decide scenarios/attacker.json --audit audit.jsonl
┌────────────────────────┬────────┬──────────────────────────────────────────────┐
│ USER                   │ ACTION │ REASONS                                      │
├────────────────────────┼────────┼──────────────────────────────────────────────┤
│ unknown@throwaway.ex…  │ BLOCK  │ P01: no second factor presented — verify …   │
│                        │        │ P02: unfamiliar sign-in location — re-verify │
│                        │        │ P03: unfamiliar location and no MFA — assum… │
│                        │        │ P04: unmanaged device — least-privilege gra… │
│                        │        │ P06: confidential/high data on non-complian… │
│                        │        │ P07: risk score above hard threshold         │
│                        │        │ P10: app outside requester's role — least p… │
└────────────────────────┴────────┴──────────────────────────────────────────────┘
pillars: identity, devices, data, applications
audit: 7 policies matched, decision appended to audit.jsonl
$ echo $?
3
```

(Exact table rendering is up to the `report` package; the columns and content are the
contract. `--format json` emits a machine-readable `[]Decision`.)

---

## 6. Scenario packs

Scenario files live in `scenarios/*.json`. Schema:

```json
{
  "name": "note-scenario",
  "description": "Flagship demo from the SC-900 note: Alice travelling, personal laptop, payroll app, 03:14 local.",
  "cases": [
    {
      "name": "alice-travel-personal-device",
      "signal": {
        "timestamp": "2026-01-15T03:14:00Z",
        "user": "alice@contoso.com",
        "role": "finance",
        "app": "payroll-app",
        "dataSensitivity": "high",
        "location": "BR",
        "unfamiliarLocation": true,
        "deviceManaged": false,
        "deviceCompliant": false,
        "localHour": 3,
        "mfaPresent": false,
        "riskScore": 65
      },
      "expect": { "action": "block", "reasonsContain": ["P03", "P06"] }
    }
  ]
}
```

`expect` fields: `action` (required, one of the four action strings), `reasonsContain`
(optional, list of policy IDs that must appear in `reasons`), `scope` (optional, checked
when action is `limited`).

Minimum packs (each doubles as an acceptance test via `run` and `go test`):

| File | Story | Expected | Matched policies |
|---|---|---|---|
| `happy.json` | normal day: MFA on, familiar location, managed + compliant device, low risk | **Allow** | P11 |
| `note-scenario.json` | the flagship: unfamiliar country, personal unmanaged laptop, payroll app, high-sensitivity data, 03:14 local, no MFA, risk 65 | **Block** | P01, P02, P03, P04, P06, P08 |
| `stolen-credential.json` | attacker has Alice's password: familiar location, but unmanaged device and no MFA, risk 45 | **StepUpMFA** | P01, P04, P08 — the MFA challenge is the layer that stops a password-only attacker |
| `insider.json` | valid identity, compliant device, MFA — but engineering role requests `payroll-app` | **Block** | P10 (least privilege) |
| `byod.json` | personal phone, MFA done, familiar location, internal data, risk 15 | **Limited** (scope `read-only`) | P04 — the just-enough-access demo |
| `attacker.json` | everything wrong at once: no MFA, unfamiliar country, unmanaged device, high-risk app, risk 85 | **Block** | P01, P02, P03, P04, P06, P07, P10 — cites identity, devices, data, applications |

With these six packs, **every action is exercised by at least one scenario** — the
acceptance suite can't go green by accident.

---

## 7. Audit log

`--audit audit.jsonl` appends one JSON object per decision, opened with
`O_APPEND|O_CREATE|O_WRONLY` (append-only; existing lines are never rewritten). One
record per line, `json.Marshal` + `\n`:

```json
{"ts":"2026-01-15T03:14:00Z","cmd":"decide","source":"scenarios/attacker.json","signal":{"user":"unknown@throwaway.example","...":"..."},"action":"block","scope":"","reasons":["P01: ..."],"pillars":["identity","devices","data","applications"],"matched":["P01","P02","P03","P04","P06","P07","P10"]}
```

- `ts` is wall-clock write time (I/O layer — the engine itself stays pure and reads only
  `signal.timestamp`).
- `zt-sim audit audit.jsonl` re-prints decisions and validates every line parses;
  corrupt lines are reported, not silently skipped.

---

## 8. Project layout

```
zero-trust-access-simulator/
├── spec.md
├── go.mod                 # module zt-sim; go 1.27; zero dependencies
├── main.go                # CLI entry: subcommand dispatch, flags, exit codes
├── Makefile               # build / test / fmt / vet / run-demo
├── internal/
│   ├── model/             # Signal, Decision, Action, sensitivity rank, validation
│   │   └── model_test.go
│   ├── engine/            # Evaluate: sorted policies → decision (pure, no I/O)
│   │   └── engine_test.go # table-driven tests + global properties
│   ├── policies/          # the v1 policy pack (P01–P11)
│   │   └── policies_test.go
│   ├── scenario/          # scenario pack loading + expectation checking
│   │   └── scenario_test.go  # runs ../../scenarios packs as acceptance tests
│   └── report/            # table/JSON printer, audit JSONL writer/reader
│       └── report_test.go
└── scenarios/             # the six JSON packs from §6
```

---

## 9. Testing strategy

Table-driven tests live inside each package. Minimum suites:

- `model`: JSON round-trip (tags!), action text marshalling, sensitivity ordering,
  validation rules incl. the unmanaged⇒non-compliant invariant, unknown-field rejection.
- `engine`: one row per scenario shape (`noteScenario → Block`, `happy → Allow`,
  `stolenCred → StepUpMFA`, `byod → Limited`, `insider → Block`), plus the global
  properties below.
- `policies`: each P01–P11 fires on its trigger signal and stays silent on a clean one;
  pack sanity — unique IDs, priorities match §4, catch-all present exactly once.
- `scenario`: loads every `scenarios/*.json`, validates, and asserts all expectations —
  wired into `go test ./...` so a regression fails the build.
- `report`: audit append-only test — write twice, byte-compare the first line unchanged;
  corrupt-line detection.

Must-pass properties (asserted globally in engine tests):

1. **Fail closed**: empty signal set / zero matches / empty pack → `Block`, never `Allow`.
2. **Strictest wins**: any `Block` match → final `Block`; likewise down the severity ladder.
3. **No silent passes**: every decision has ≥ 1 reason (an `Allow` gets the P11 all-clear).
4. **Determinism**: same input → same action, same reasons, same order — enforced by the
   `(Priority, ID)` sort; assert identical output across two `Evaluate` calls.

---

## 10. Phases & tasks

Work top-to-bottom; each phase builds on the previous one. Commit per task.

### Phase 0 — Scaffold
> Goal: a building, vet-clean skeleton.

- [ ] **T0.1** `go mod init zt-sim`; create `internal/{model,engine,policies,scenario,report}` and `scenarios/`
- [ ] **T0.2** `main.go`: subcommand dispatch (`decide|run|policies|audit`), `--version`, usage errors → exit 64; unknown subcommand → exit 64
- [ ] **T0.3** `Makefile`: `build`, `test`, `fmt`, `vet`, `run-demo`

**Done when**: `go build ./...` and `go vet ./...` pass; `./zt-sim --version` prints.

### Phase 1 — Domain model
> Goal: `internal/model` complete and tested.

- [ ] **T1.1** `Signal`, `Decision` structs with the exact JSON tags from §2
- [ ] **T1.2** `Action` enum: `String()`, `MarshalText()`/`UnmarshalText()`, severity constants
- [ ] **T1.3** `SensitivityRank(string) int` (`public=0 < internal=1 < confidential=2 < high=3`), error on unknown
- [ ] **T1.4** `Signal.Validate()` implementing every §2 rule; strict decoder helper (`DisallowUnknownFields`)
- [ ] **T1.5** `model_test.go`: round-trip, marshalling, all validation rules

**Done when**: `go test ./internal/model/` green.

### Phase 2 — Engine
> Goal: pure evaluator with the §3 semantics.

- [ ] **T2.1** `Policy` struct; `Engine` holds policies pre-sorted by `(Priority, ID)`
- [ ] **T2.2** `Evaluate(Signal) Decision`: evaluate all, strictest-wins, scope from the strictest Limited match, reason format `"<ID>: <Note>"`
- [ ] **T2.3** Fail-closed paths: zero matches and empty pack → `Block` with engine reason
- [ ] **T2.4** `engine_test.go`: scenario table + the four global properties (§9), incl. double-eval determinism

**Done when**: `go test ./internal/engine/` green with property tests.

### Phase 3 — Policy pack
> Goal: P01–P11 exactly as §4.

- [ ] **T3.1** Implement P01–P10 matchers; each `Note` copied verbatim from §4 (audit lines)
- [ ] **T3.2** P11 catch-all: matches iff no other pack policy matched
- [ ] **T3.3** `policies_test.go`: per-policy fire/silence tests; boundary tests (risk 70/71, `localHour` 4/5, sensitivity confidential vs internal)
- [ ] **T3.4** Pack sanity test: IDs unique & ordered as table, catch-all present once, pillar values valid

**Done when**: `go test ./internal/policies/` green; every scenario row in §6 can be derived by hand from the pack.

### Phase 4 — Scenarios & acceptance
> Goal: the six packs run as tests.

- [ ] **T4.1** `internal/scenario`: pack loader + validator (schema from §6), `Check(case) error` comparing action / `reasonsContain` / `scope`
- [ ] **T4.2** Author the six packs from §6 with `expect` blocks (use the §6 tables as the source of truth)
- [ ] **T4.3** `scenario_test.go` walks `../../scenarios/*.json` and asserts every expectation
- [ ] **T4.4** Verify the outcome of each pack by hand against §4 once — they must agree before moving on

**Done when**: `go test ./...` runs all packs green.

### Phase 5 — CLI
> Goal: `zt-sim` end-to-end per §5.

- [ ] **T5.1** `internal/report`: decision table renderer (USER / ACTION / REASONS, pillars footer) + `--format json`
- [ ] **T5.2** `decide` subcommand: file array or `--stdin` single object; strict decode + `Signal.Validate` → exit 65 on bad input
- [ ] **T5.3** `run` subcommand: walk dir, per-pack summary table (PACK / CASES / PASS / FAIL / STRICTEST), exit 1 on expectation failure
- [ ] **T5.4** `policies` subcommand: ID / priority / pillar / action / note table
- [ ] **T5.5** Exit-code map wired exactly as §5 (decide 0–3 by strictest, run 0/1, 64/65 errors)

**Done when**: the §5 example session runs verbatim against `scenarios/attacker.json` and `echo $?` prints 3.

### Phase 6 — Audit log
> Goal: append-only JSONL per §7.

- [ ] **T6.1** `internal/report`: `AuditWriter` — `O_APPEND|O_CREATE|O_WRONLY`, one JSON line per decision, `ts` = write time
- [ ] **T6.2** Wire `--audit` into `decide` and `run`
- [ ] **T6.3** `audit` subcommand: read, validate, print action counts; corrupt lines reported → exit 65
- [ ] **T6.4** Test: append twice → first line byte-identical; truncated/corrupt line detected

**Done when**: re-running `decide` never mutates old lines; tests green.

### Phase 7 — Polish
> Goal: shippable v1.

- [ ] **T7.1** `gofmt -l .` empty; `go vet ./...` clean
- [ ] **T7.2** README: what/why, install (`go build`), usage, exit codes, policy table, scenario table
- [ ] **T7.3** Demo transcript in README: the §5 session + `run scenarios/` summary
- [ ] **T7.4** `make run-demo` reproduces the transcript from a clean build

**v1 definition of done**: fresh clone → `go build ./... && go test ./...` green →
`./zt-sim run scenarios/` all packs pass → README demo reproduces.

Rough effort: Phases 0–3 ≈ one weekend; 4–5 ≈ an evening each; 6–7 ≈ one evening together.

---

## 11. Stretch goals (post-v1)

- **Networks/infrastructure policies**: e.g. sign-in from blocklisted country, or
  "impossible travel" (last sign-in geo vs. this one) → gives the two unused pillars rules
- **Shared-responsibility sorter** subcommand: `zt-sim sortresp` — interactive quiz mapping
  responsibilities (OS patching, data classification, physical datacenter) to
  On-prem/IaaS/PaaS/SaaS, scored
- **Attack replay mode**: feed a scripted attack chain (phish → credential → lateral
  move), show which layer caught it — defense-in-depth visualization
- **Web dashboard**: serve decisions + audit log on localhost:8080
- **Policy pack as JSON/YAML config** once the rule set outgrows code
- **CIA tagging**: each scenario tagged with the triad leg it threatens
  (steal = confidentiality, tamper = integrity, DDoS = availability)
- **Step-up resolution**: `decide --step-up` mode that takes a StepUpMFA decision plus
  simulated MFA success/failure and re-evaluates

---

## 12. Source alignment (SC-900 concepts ↔ project)

| Note concept | Where it shows up |
|---|---|
| Verify explicitly | multi-signal `Signal`; P01/P02/P08 demand a second factor on doubt |
| Least privilege | `Limited` action + `Scope` ("read-only" JIT-style grant); P04/P05 cap grants; P10 role check |
| Assume breach | fail-closed defaults (§3.5), append-only audit log, every decision carries reasons |
| Zero Trust pillars (six: identity, devices/endpoints, applications, data, infrastructure, networks — plus cross-cutting visibility, automation & orchestration) | `Policy.Pillar` + `Decision.Pillars`; the audit log plays the visibility role; v1 policies cover four pillars (§4) |
| Conditional access (signals → decision) | the whole engine: signals in, Allow/StepUp/Limited/Block out |
| The flagship scenario | `scenarios/note-scenario.json` |
| Shared responsibility (IaaS/PaaS/SaaS) | stretch-goal sorter subcommand |

---

## 13. Revision notes (what changed in this revision of the spec)

1. **Severity order fixed** to `Block > StepUpMFA > Limited > Allow` (was `Block >
   Limited > StepUpMFA`, which granted an attacker with a stolen password read-only
   access and left the stolen-credential scenario ambiguous "StepUpMFA/Limited").
2. **P06 rewritten** — it matched on an "app sensitivity" field that doesn't exist on
   `Signal`; it now uses `SensitivityRank ≥ confidential && !DeviceCompliant`, and the
   duplicate old P07 was merged into it. Pack renumbered to P01–P11.
3. **`TimeOfDay time.Time` → `LocalHour int`** — removed duplication with `Timestamp`,
   killed timezone ambiguity; odd-hours window pinned to half-open `hour < 5`.
4. **Deterministic tie-breaking** `(Priority, ID)` added — required by the determinism
   test, previously unspecified.
5. **Scenario JSON schema formalized** (`name`/`cases`/`signal`/`expect` with
   `action`, `reasonsContain`, `scope`) — `expect` was mentioned but never defined.
6. **`byod.json` added** — previously no scenario produced `Limited`, so a quarter of the
   action space was untested.
7. **Exit codes pinned** per subcommand (and matched to severity for `decide`);
   `report` (undefined semantics) replaced by `--audit` flag + `zt-sim audit`.
8. **Validation rules added** incl. the `!DeviceManaged ⇒ !DeviceCompliant` invariant and
   `DisallowUnknownFields` decoding, so typos can't silently weaken signals.
9. **Milestones replaced by phases with concrete tasks** (§10), each with acceptance
   criteria; engine tests moved into `internal/engine` (a root-level `engine_test.go`
   can't live next to the package it tests).
10. Course-note reference generalized to SC-900 (the previously cited local note path
    doesn't exist on this machine); pillar wording aligned with Microsoft guidance
    (six pillars + visibility/automation & orchestration as the cross-cutting capability).
