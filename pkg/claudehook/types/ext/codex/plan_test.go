package codex

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestUpdatePlanInput_Parse(t *testing.T) {
	const raw = `{"explanation":"do it","plan":[` +
		`{"step":"write code","status":"in_progress"},` +
		`{"step":"run tests","status":"pending"}]}`

	var in UpdatePlanInput
	assert.NilError(t, json.Unmarshal([]byte(raw), &in))
	assert.Assert(t, in.Explanation != nil)
	assert.Equal(t, *in.Explanation, "do it")
	assert.DeepEqual(t, in.Plan, []PlanItem{
		{Step: "write code", Status: PlanStatusInProgress},
		{Step: "run tests", Status: PlanStatusPending},
	})

	got, err := json.Marshal(in)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}

func TestUpdatePlanInput_MarshalFromFields(t *testing.T) {
	in := UpdatePlanInput{Plan: []PlanItem{{Step: "a", Status: PlanStatusCompleted}}}
	got, err := json.Marshal(in)
	assert.NilError(t, err)
	assert.Equal(t, string(got), `{"plan":[{"step":"a","status":"completed"}]}`)
}
