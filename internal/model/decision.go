package model

// Decision is the evaluation output; Reasons, Pillars, and Matched form the
// audit trail in evaluation order.
type Decision struct {
	Action  Action   `json:"action"`
	Scope   string   `json:"scope"`   // set for Limited grants, "" otherwise
	Reasons []string `json:"reasons"` // "<ID>: <Note>" per matched policy
	Pillars []string `json:"pillars"` // unique pillars of matched policies
	Matched []string `json:"matched"` // policy IDs that matched
}
