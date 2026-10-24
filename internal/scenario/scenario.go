// Package scenario loads §6 scenario packs from disk and checks a case's
// expectations against a Decision. It is engine-free by design: Check takes
// the Decision, so the package imports only the domain model — Phase 5 wires
// engine + policies + scenario together in the CLI.
package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"

	"zt-sim/internal/model"
)

// Expectation is the "expect" block of a §6 case. Action is a pointer so
// "required" is distinguishable from the zero value (Allow).
type Expectation struct {
	Action         *model.Action `json:"action"`         // required
	ReasonsContain []string      `json:"reasonsContain"` // policy IDs that must appear in the decision's Matched
	Scope          string        `json:"scope"`          // checked when Action is limited
}

// Case is one named signal plus its expectation.
type Case struct {
	Name   string       `json:"name"`
	Signal model.Signal `json:"signal"`
	Expect Expectation  `json:"expect"`
}

// Pack is one scenario file.
type Pack struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Cases       []Case `json:"cases"`
}

// Load reads one pack file, strictly decodes it (unknown fields are
// rejected, the DecodeSignal precedent) and validates it. Errors name the
// file so run failures are debuggable.
func Load(path string) (Pack, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Pack{}, err
	}
	var p Pack
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return Pack{}, fmt.Errorf("%s: %w", path, err)
	}
	if err := p.validate(); err != nil {
		return Pack{}, fmt.Errorf("%s: %w", path, err)
	}
	return p, nil
}

// validate enforces the §6 schema at the file trust boundary, first error wins.
func (p Pack) validate() error {
	if p.Name == "" {
		return errors.New("name must not be empty")
	}
	if len(p.Cases) == 0 {
		return errors.New("cases must not be empty")
	}
	for i, c := range p.Cases {
		if err := c.validate(); err != nil {
			return fmt.Errorf("case %d %q: %w", i, c.Name, err)
		}
	}
	return nil
}

func (c Case) validate() error {
	if c.Name == "" {
		return errors.New("name must not be empty")
	}
	if err := c.Signal.Validate(); err != nil {
		return fmt.Errorf("signal: %w", err)
	}
	if c.Expect.Action == nil {
		return errors.New("expect.action required")
	}
	if c.Expect.Scope != "" && *c.Expect.Action != model.Limited {
		return fmt.Errorf("expect.scope set but expect.action is %s", c.Expect.Action)
	}
	return nil
}

// Check compares d against the case's expectations: the action must match,
// every ReasonsContain ID must appear in d.Matched, and Scope is compared
// only when set (§6: scope is optional, checked when the action is limited).
func (c Case) Check(d model.Decision) error {
	if d.Action != *c.Expect.Action {
		return fmt.Errorf("want action %s, got %s", c.Expect.Action, d.Action)
	}
	for _, id := range c.Expect.ReasonsContain {
		if !slices.Contains(d.Matched, id) {
			return fmt.Errorf("reasons missing %s", id)
		}
	}
	if c.Expect.Scope != "" && d.Scope != c.Expect.Scope {
		return fmt.Errorf("want scope %q, got %q", c.Expect.Scope, d.Scope)
	}
	return nil
}
