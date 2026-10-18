// Package engine is the pure policy evaluator: sorted policies in, a
// deterministic Decision out. No I/O, no clock, no randomness.
package engine

import (
	"cmp"
	"slices"
	"strings"

	"zt-sim/internal/model"
)

// Policy is one authored rule. Policies live in code (internal/policies),
// so there are no JSON tags.
type Policy struct {
	ID       string
	Priority int    // lower runs first; ties broken by ID ascending
	Pillar   string // identity|devices|applications|data|networks|infrastructure
	When     func(model.Signal) bool
	Then     model.Action
	Scope    string // populated when Then == Limited, e.g. "read-only"
	Note     string // audit line; Reasons entries read "<ID>: <Note>"
}

// Engine holds the pack pre-sorted by (Priority asc, ID asc) and is
// immutable after New; the zero value is the empty pack (fails closed).
type Engine struct {
	policies []Policy
}

// New clones policies (the caller's slice order is left untouched) and
// returns an Engine holding them sorted.
func New(policies []Policy) Engine {
	sorted := slices.Clone(policies)
	slices.SortFunc(sorted, func(a, b Policy) int {
		if c := cmp.Compare(a.Priority, b.Priority); c != 0 {
			return c
		}
		return strings.Compare(a.ID, b.ID)
	})
	return Engine{policies: sorted}
}
