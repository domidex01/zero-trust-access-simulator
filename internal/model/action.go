package model

// Action is one of the four zero-trust outcomes. The numeric order is the
// severity order (Block strictest); the CLI reuses it as its exit code.
type Action int

const (
	Allow     Action = iota // 0
	Limited                 // 1
	StepUpMFA               // 2
	Block                   // 3
)
