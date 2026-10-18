# Test plan — zero-trust-access-simulator (Phase 1 — Domain model)

All tests: `internal/model/model_test.go`, **package `model_test`** (black-box — the
contract is the API, not the internals), import `zt-sim/internal/model`. Names follow
`TestFunc_Scenario`.

## Code under test (pin these so execution needs no decisions)

```go
// action.go
type Action int

const (
	Allow Action = iota // 0
	Limited             // 1
	StepUpMFA           // 2
	Block               // 3
)

func (a Action) String() string              // "allow"|"limited"|"stepUpMFA"|"block"; fallback fmt "Action(%d)"
func (a Action) MarshalText() ([]byte, error)
func (a *Action) UnmarshalText(text []byte) error // unknown → error, no state change

// signal.go
type Signal struct { /* verbatim §2, tags: timestamp user role app dataSensitivity
   location unfamiliarLocation deviceManaged deviceCompliant localHour mfaPresent riskScore */ }

func (s Signal) Validate() error
func SensitivityRank(s string) (int, error)
func DecodeSignal(r io.Reader) (Signal, error) // DisallowUnknownFields + Validate

// decision.go
type Decision struct { /* verbatim §2, tags: action scope reasons pillars matched */ }
```

Pinned error substrings (tests assert `strings.Contains`; style: lowercase, field named,
value quoted):

| Rule | Error contains |
|---|---|
| empty user/role/app | `user must not be empty` (resp. `role`, `app`) |
| bad sensitivity | `dataSensitivity "ultra" not one of public|internal|confidential|high` |
| riskScore range | `riskScore 101 out of range [0,100]` |
| localHour range | `localHour 24 out of range [0,23]` |
| bad location | `location "usa" must be empty or ISO 3166-1 alpha-2` |
| device invariant | `deviceCompliant true requires deviceManaged` |
| zero timestamp | `timestamp must not be zero` |
| unknown action text | `unknown action "alow"` |

Validate check order = spec §2 table order (curveball case below pins it).

## T1.1 — Signal JSON round-trip
- Target: `internal/model`, `TestSignal_JSONRoundTrip`
- Shape: table-driven, golden JSON strings
- Red because: `model.Signal` does not exist → compile error
- Run: `go test ./internal/model/ -run TestSignal_JSONRoundTrip -count=1 -v`

Golden signal (Alice from spec §6, compact — field order = declaration order, so
`json.Marshal` output is deterministic):

```json
{"timestamp":"2026-01-15T03:14:00Z","user":"alice@contoso.com","role":"finance","app":"payroll-app","dataSensitivity":"high","location":"BR","unfamiliarLocation":true,"deviceManaged":false,"deviceCompliant":false,"localHour":3,"mfaPresent":false,"riskScore":65}
```

| name | give | want |
|---|---|---|
| full alice signal | unmarshal golden | every field exact (Timestamp 2026-01-15T03:14:00Z, UnfamiliarLoc true, LocalHour 3, RiskScore 65, …); then marshal → byte-identical golden |
| minimal signal | `{"user":"a@b.c","role":"eng","app":"wiki","dataSensitivity":"public","timestamp":"2026-01-15T03:14:00Z"}` | no error; absent keys zero-valued (Location "", bools false, RiskScore 0) |

## T1.2 — Action text behavior and severity order
- Target: `TestAction_String`, `TestAction_JSONText`, `TestAction_SeverityOrder`
- Red because: `String`/`MarshalText`/`UnmarshalText` undefined on `model.Action` → compile error
- Run: `go test ./internal/model/ -run TestAction_ -count=1 -v`

TestAction_String:

| name | give | want |
|---|---|---|
| allow | `model.Allow` | `"allow"` |
| limited | `model.Limited` | `"limited"` |
| step-up | `model.StepUpMFA` | `"stepUpMFA"` |
| block | `model.Block` | `"block"` |
| out of range | `model.Action(9)` | `"Action(9)"` (no panic) |

TestAction_JSONText — via `Decision` (proves the §2 wire format end-to-end):
golden: `{"action":"limited","scope":"read-only","reasons":["P04: unmanaged device — least-privilege grant"],"pillars":["devices"],"matched":["P04"]}`

| name | give | want |
|---|---|---|
| decision marshals action as text | `Decision{Action: Limited, Scope: "read-only", …}` | byte-identical golden |
| decision round-trip | unmarshal golden | equal struct (all five fields) |
| all four actions round-trip | each of allow/limited/stepUpMFA/block in `{"action":"…"}` | unmarshal → same Action back |
| unknown action text | `{"action":"alow"}` | error contains `unknown action "alow"` |
| MarshalText out of range | `model.Action(9)` via `json.Marshal` | error (trust boundary: never emit garbage text) |

TestAction_SeverityOrder:

| name | give | want |
|---|---|---|
| strictest wins ordering | the four constants | `Allow < Limited < StepUpMFA < Block` (numeric comparison — severity *is* the iota order, §2) |

## T1.3 — SensitivityRank
- Target: `TestSensitivityRank`
- Red because: `model.SensitivityRank` undefined → compile error
- Run: `go test ./internal/model/ -run TestSensitivityRank -count=1 -v`

| name | give | want |
|---|---|---|
| public | `"public"` | `0, nil` |
| internal | `"internal"` | `1, nil` |
| confidential | `"confidential"` | `2, nil` |
| high | `"high"` | `3, nil` |
| total order holds | ranks of the four | strictly increasing in the listed order |
| unknown value | `"ultra"` | error contains `dataSensitivity "ultra"` |
| wrong case | `"Public"` | error (case-sensitive) |
| empty | `""` | error |

## T1.4 — Signal.Validate + DecodeSignal
- Target: `TestSignal_Validate`, `TestDecodeSignal`
- Red because: `s.Validate` / `model.DecodeSignal` undefined → compile error
- Run: `go test ./internal/model/ -run 'TestSignal_Validate|TestDecodeSignal' -count=1 -v`

TestSignal_Validate — base signal is the T1.1 alice golden (valid), one delta per row:

| name | delta from base | want error contains |
|---|---|---|
| happy path | — | `nil` |
| limits accepted | riskScore 0→100 sweep: RiskScore 100, LocalHour 0 and 23, Location "" | `nil` (one row, all bounds at once) |
| empty user | `User: ""` | `user must not be empty` |
| empty role | `Role: ""` | `role must not be empty` |
| empty app | `App: ""` | `app must not be empty` |
| sensitivity typo | `DataSensitivity: "ultra"` | `dataSensitivity "ultra"` |
| riskScore negative | `RiskScore: -1` | `riskScore -1 out of range` |
| riskScore over | `RiskScore: 101` | `riskScore 101 out of range` |
| localHour negative | `LocalHour: -1` | `localHour -1 out of range` |
| localHour over | `LocalHour: 24` | `localHour 24 out of range` |
| location lowercase | `Location: "us"` | `location "us"` |
| location three letters | `Location: "USA"` | `location "USA"` |
| location letter-digit | `Location: "U1"` | `location "U1"` |
| unmanaged but compliant | `DeviceManaged: false, DeviceCompliant: true` | `deviceCompliant true requires deviceManaged` |
| zero timestamp | `Timestamp: time.Time{}` | `timestamp must not be zero` |
| curveball: first error wins | `DataSensitivity: "ultra"` **and** unmanaged+compliant | `dataSensitivity "ultra"` — pins check order (sensitivity before device invariant) |

TestDecodeSignal:

| name | give | want |
|---|---|---|
| strict happy path | T1.1 golden via reader | no error, alice fields set |
| misspelled key rejected | golden + `"risk": 5` | error contains `unknown field` (spec: no silent zero-valueing) |
| validation applied at load | golden with `"user": ""` | `user must not be empty` — decode alone isn't enough |
| bad timestamp format | `"timestamp": "yesterday"` | error (time.Time unmarshal fails; RFC 3339 enforced by decode, not Validate) |

## T1.5 — Completeness sweep (no new production code expected)
- Target: `model_test.go` audit against the §9 model-suite list
- Red because: nothing — this is the gap-closing task; a new test going red means a real
  model bug (fix the model, not the test)
- Run: `go test ./... -count=1 && gofmt -l . && go vet ./...`

§9 bullet → must map to a named test (add cases where thin):

| §9 bullet | Covered by |
|---|---|
| JSON round-trip (tags!) | `TestSignal_JSONRoundTrip`, `TestAction_JSONText` |
| action text marshalling | `TestAction_String`, `TestAction_JSONText` |
| sensitivity ordering | `TestSensitivityRank` |
| validation rules incl. unmanaged⇒non-compliant invariant | `TestSignal_Validate` |
| unknown-field rejection | `TestDecodeSignal` |

Likely gaps to look for: absent-key zero-values (T1.1 row 2), out-of-range `String()`
fallback (T1.2), case-sensitivity of sensitivity (T1.3), first-error ordering (T1.4
curveball).

---

# Phase 0 — Scaffold (complete, archived)

## T1 — Module scaffold
- Test: none. Scaffolding has no behavior to assert. Acceptance: `go build ./...` exits 0.

## T2 — main dispatch
- `TestRun_Dispatch` (package `main`, white-box — `run` is unexported): version → 0 +
  `zt-sim version 0.1.0\n`; no args / unknown subcommand → 64 + usage on stderr; four
  subcommand stubs → 70 + `not implemented yet`. Exit codes are the CLI's real API —
  asserted exactly.

## T3 — Makefile
- Test: none. `make test` re-runs T2's suite. Acceptance: `make build &&
  ./zt-sim --version && make vet && make test`, then `make fmt` changes nothing.
