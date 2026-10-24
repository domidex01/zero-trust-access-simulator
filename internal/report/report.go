// Package report renders everything the CLI prints for humans: the
// decisions table, the run summary, and the policy listing (§5). The
// columns and content are the contract; text/tabwriter does the aligning.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"zt-sim/internal/model"
)

// Row pairs the request with its outcome: the table's USER column comes
// from the signal, everything else from the decision.
type Row struct {
	Signal   model.Signal
	Decision model.Decision
}

// newTable returns a tabwriter with the package-wide column style.
func newTable(w io.Writer) *tabwriter.Writer {
	return tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
}

// Decisions writes the USER / ACTION / REASONS table — one line per
// reason, action uppercased — followed by a pillars footer: the unique
// pillars across all rows, in first-appearance order (§5). Every engine
// decision carries at least one reason (no silent passes), so each row
// prints at least its first line.
func Decisions(w io.Writer, rows []Row) error {
	tw := newTable(w)
	fmt.Fprintln(tw, "USER\tACTION\tREASONS")
	var pillars []string
	for _, r := range rows {
		for i, reason := range r.Decision.Reasons {
			if i == 0 {
				fmt.Fprintf(tw, "%s\t%s\t%s\n", r.Signal.User, strings.ToUpper(r.Decision.Action.String()), reason)
			} else {
				fmt.Fprintf(tw, "\t\t%s\n", reason)
			}
		}
		for _, p := range r.Decision.Pillars {
			if !slices.Contains(pillars, p) {
				pillars = append(pillars, p)
			}
		}
	}
	if err := tw.Flush(); err != nil {
		return err
	}
	_, err := fmt.Fprintf(w, "pillars: %s\n", strings.Join(pillars, ", "))
	return err
}

// DecisionsJSON writes decisions as an indented JSON array — §5's
// machine-readable --format json payload.
func DecisionsJSON(w io.Writer, decisions []model.Decision) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(decisions)
}

// PackResult is one row of the run summary table — counts per pack and
// the strictest action it produced.
type PackResult struct {
	Name       string
	Cases      int
	Pass, Fail int
	Strictest  model.Action
}

// PackSummary writes the PACK / CASES / PASS / FAIL / STRICTEST table (§5 run).
func PackSummary(w io.Writer, results []PackResult) error {
	tw := newTable(w)
	fmt.Fprintln(tw, "PACK\tCASES\tPASS\tFAIL\tSTRICTEST")
	for _, r := range results {
		fmt.Fprintf(tw, "%s\t%d\t%d\t%d\t%s\n", r.Name, r.Cases, r.Pass, r.Fail, strings.ToUpper(r.Strictest.String()))
	}
	return tw.Flush()
}
