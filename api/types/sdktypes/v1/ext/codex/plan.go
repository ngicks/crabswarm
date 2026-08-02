package codex

import "encoding/json"

// PlanStatus is the status of a single update_plan step.
//
// Source: codex-rs/core/src/tools/handlers/plan_spec.rs
type PlanStatus string

const (
	// PlanStatusPending marks a step not yet started.
	PlanStatusPending PlanStatus = "pending"
	// PlanStatusInProgress marks the step currently being worked on. At most
	// one step is in progress at a time.
	PlanStatusInProgress PlanStatus = "in_progress"
	// PlanStatusCompleted marks a finished step.
	PlanStatusCompleted PlanStatus = "completed"
)

// PlanItem is one step in an update_plan tool_input.
//
// Source: codex-rs/core/src/tools/handlers/plan_spec.rs
type PlanItem struct {
	// Step is the task step text.
	Step string `json:"step"`
	// Status is the step status.
	Status PlanStatus `json:"status"`
}

// UpdatePlanInput is a decoded Codex update_plan tool_input:
// {"explanation"?: string, "plan": [{"step": ..., "status": ...}]}.
//
// The decoded fields expose the plan; the original object round-trips
// byte-for-byte via the preserved raw bytes.
//
// Source: codex-rs/core/src/tools/handlers/plan_spec.rs
type UpdatePlanInput struct {
	// Explanation is the optional explanation for this plan update, nil when
	// absent.
	Explanation *string `json:"-"`
	// Plan is the ordered list of plan steps.
	Plan []PlanItem `json:"-"`

	// raw preserves the original tool_input bytes for a byte-exact re-marshal.
	raw json.RawMessage
}

// RawJSON returns a copy of the original tool_input bytes, or nil for a value
// that wasn't decoded from JSON.
func (i UpdatePlanInput) RawJSON() json.RawMessage { return cloneRaw(i.raw) }

// MarshalJSON re-emits the original tool_input bytes when present, otherwise
// builds the {"explanation"?, "plan"} object from the decoded fields.
func (i UpdatePlanInput) MarshalJSON() ([]byte, error) {
	if len(i.raw) > 0 {
		return cloneRaw(i.raw), nil
	}
	return json.Marshal(struct {
		Explanation *string    `json:"explanation,omitzero"`
		Plan        []PlanItem `json:"plan"`
	}{Explanation: i.Explanation, Plan: i.Plan})
}

// UnmarshalJSON decodes the update_plan object, preserves the raw bytes, and
// records the decoded fields.
func (i *UpdatePlanInput) UnmarshalJSON(data []byte) error {
	var v struct {
		Explanation *string    `json:"explanation"`
		Plan        []PlanItem `json:"plan"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	i.raw = cloneRaw(data)
	i.Explanation = v.Explanation
	i.Plan = v.Plan
	return nil
}
