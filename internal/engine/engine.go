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

// Engine failure reasons, pinned verbatim from §3.5 — the engine never
// defaults to a grant.
const (
	reasonNoMatch    = "engine: no policy matched — fail closed (assume breach)"
	reasonNoPolicies = "engine: no policies loaded — fail closed"
)

// Evaluate runs every policy (no short-circuit — each match is audited) and
// returns the strictest matched action. Severity is the numeric order, so a
// plain comparison picks the winner. Scope comes from the first matching
// Limited policy, kept only when Limited is the final action. Zero matches
// or an empty pack block — never a silent Allow.
func (e Engine) Evaluate(s model.Signal) model.Decision {
	if len(e.policies) == 0 {
		return model.Decision{Action: model.Block, Reasons: []string{reasonNoPolicies}}
	}
	var d model.Decision
	var firstLimited string
	var limitedSeen bool
	for _, p := range e.policies {
		if !p.When(s) {
			continue
		}
		d.Matched = append(d.Matched, p.ID)
		d.Reasons = append(d.Reasons, p.ID+": "+p.Note)
		if !slices.Contains(d.Pillars, p.Pillar) {
			d.Pillars = append(d.Pillars, p.Pillar)
		}
		if p.Then > d.Action {
			d.Action = p.Then
		}
		if p.Then == model.Limited && !limitedSeen {
			limitedSeen = true
			firstLimited = p.Scope
		}
	}
	if len(d.Matched) == 0 {
		return model.Decision{Action: model.Block, Reasons: []string{reasonNoMatch}}
	}
	if d.Action == model.Limited {
		d.Scope = firstLimited
	}
	return d
}
