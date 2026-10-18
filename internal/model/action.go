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

func (a Action) String() string {
	switch a {
	case Allow:
		return "allow"
	case Limited:
		return "limited"
	case StepUpMFA:
		return "stepUpMFA"
	case Block:
		return "block"
	default:
		return fmt.Sprintf("Action(%d)", a)
	}
}

// MarshalText implements encoding.TextMarshaler so Decision JSON carries
// "action":"block" style text; out-of-range actions refuse to marshal
// rather than emit garbage.
func (a Action) MarshalText() ([]byte, error) {
	if !knownAction(a) {
		return nil, fmt.Errorf("unknown action %d", a)
	}
	return []byte(a.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler; unknown text is an
// error and leaves the receiver unchanged.
func (a *Action) UnmarshalText(text []byte) error {
	switch string(text) {
	case "allow":
		*a = Allow
	case "limited":
		*a = Limited
	case "stepUpMFA":
		*a = StepUpMFA
	case "block":
		*a = Block
	default:
		return fmt.Errorf("unknown action %q", string(text))
	}
	return nil
}

func knownAction(a Action) bool {
	switch a {
	case Allow, Limited, StepUpMFA, Block:
		return true
	default:
		return false
	}
}
