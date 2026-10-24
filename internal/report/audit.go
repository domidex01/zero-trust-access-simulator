// Audit JSONL writer (§7): one record per decision, append-only.
// report.go's §5 tables are untouched; audit lives here.
package report

import (
	"encoding/json"
	"os"
	"time"

	"zt-sim/internal/model"
)

// AuditRecord is one JSONL line of the audit log. Decision is embedded
// so action/scope/reasons/pillars/matched flatten to top level — struct
// field order is §7's field order.
type AuditRecord struct {
	TS             time.Time    `json:"ts"`     // write time — injected clock
	Cmd            string       `json:"cmd"`    // "decide" | "run"
	Source         string       `json:"source"` // path as given / "stdin" / pack path
	Signal         model.Signal `json:"signal"`
	model.Decision              // embedded: action, scope, reasons, pillars, matched
}

// AuditWriter appends records to an append-only JSONL file — existing
// lines are never rewritten (§7).
type AuditWriter struct {
	f   *os.File
	now func() time.Time
}

// NewAuditWriter opens path O_APPEND|O_CREATE|O_WRONLY. now is the ts
// clock; callers inject it so tests stay deterministic.
func NewAuditWriter(path string, now func() time.Time) (*AuditWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &AuditWriter{f: f, now: now}, nil
}

// Append writes one record: json.Marshal + "\n", one Write call, no
// buffering and no validation — the CLI validated the signal upstream.
func (a *AuditWriter) Append(cmd, source string, s model.Signal, d model.Decision) error {
	b, err := json.Marshal(AuditRecord{TS: a.now(), Cmd: cmd, Source: source, Signal: s, Decision: d})
	if err != nil {
		return err
	}
	_, err = a.f.Write(append(b, '\n'))
	return err
}

// Close releases the underlying file.
func (a *AuditWriter) Close() error {
	return a.f.Close()
}
