package model

import "fmt"

// Action is one of the four zero-trust outcomes. The numeric order is the
// severity order (Block strictest); the CLI reuses it as its exit code.
type Action int

const (
	Allow     Action = iota // 0
	Limited                 // 1
	StepUpMFA               // 2
	Block                   // 3
)

// actionNames is the single source of truth for the wire text, indexed by
// Action; its order must match the const block (pinned by
// TestAction_SeverityOrder).
var actionNames = [...]string{"allow", "limited", "stepUpMFA", "block"}

func (a Action) String() string {
	if a < 0 || int(a) >= len(actionNames) {
		return fmt.Sprintf("Action(%d)", a)
	}
	return actionNames[a]
}

// MarshalText implements encoding.TextMarshaler so Decision JSON carries
// "action":"block" style text; out-of-range actions refuse to marshal
// rather than emit garbage.
func (a Action) MarshalText() ([]byte, error) {
	if a < 0 || int(a) >= len(actionNames) {
		return nil, fmt.Errorf("unknown action %d", a)
	}
	return []byte(actionNames[a]), nil
}

// UnmarshalText implements encoding.TextUnmarshaler; unknown text is an
// error and leaves the receiver unchanged.
func (a *Action) UnmarshalText(text []byte) error {
	for i, name := range actionNames {
		if string(text) == name {
			*a = Action(i)
			return nil
		}
	}
	return fmt.Errorf("unknown action %q", string(text))
}
