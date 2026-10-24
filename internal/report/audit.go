// Audit JSONL writer (§7): one record per decision, append-only.
// report.go's §5 tables are untouched; audit lives here.
package report

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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

// ReadAudit strictly decodes every JSONL line (DisallowUnknownFields);
// corrupt lines come back as errors carrying their 1-based line number
// while valid records still decode (§7: reported, not silently skipped).
// A blank line fails decode and is therefore corrupt.
func ReadAudit(r io.Reader) (records []AuditRecord, errs []error) {
	sc := bufio.NewScanner(r)
	n := 0
	for sc.Scan() {
		n++
		var rec AuditRecord
		dec := json.NewDecoder(strings.NewReader(sc.Text()))
		dec.DisallowUnknownFields()
		if err := dec.Decode(&rec); err != nil {
			errs = append(errs, fmt.Errorf("line %d: %v", n, err))
			continue
		}
		if dec.More() { // the line must be exactly one record
			errs = append(errs, fmt.Errorf("line %d: trailing content after JSON value", n))
			continue
		}
		records = append(records, rec)
	}
	if err := sc.Err(); err != nil {
		errs = append(errs, fmt.Errorf("read: %v", err))
	}
	return records, errs
}

// AuditSummary writes the counts line — §2 severity order, all four
// actions always, Action.String names.
func AuditSummary(w io.Writer, records []AuditRecord) error {
	var counts [4]int // Action is iota 0..3, decode refuses anything else
	for _, r := range records {
		counts[r.Action]++
	}
	_, err := fmt.Fprintf(w, "counts: allow=%d limited=%d stepUpMFA=%d block=%d\n",
		counts[model.Allow], counts[model.Limited], counts[model.StepUpMFA], counts[model.Block])
	return err
}
