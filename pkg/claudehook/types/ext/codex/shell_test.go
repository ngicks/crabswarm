package codex

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestShellOutput_RoundTrip(t *testing.T) {
	// Decoded values re-emit their preserved raw bytes verbatim.
	const raw = `"total 0\ndrwxr-xr-x  2 user  staff   64 Jun  8 10:00 .\n"`

	var out ShellOutput
	assert.NilError(t, json.Unmarshal([]byte(raw), &out))
	assert.Equal(t, out.Output, "total 0\ndrwxr-xr-x  2 user  staff   64 Jun  8 10:00 .\n")

	got, err := json.Marshal(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}

func TestShellOutput_HTMLCharsSemantic(t *testing.T) {
	// json.Marshal re-escapes '<', '>', '&'; bytes differ but the value
	// round-trips.
	const raw = `"if x < y && y > z then\n"`

	var out ShellOutput
	assert.NilError(t, json.Unmarshal([]byte(raw), &out))
	got, err := json.Marshal(out)
	assert.NilError(t, err)

	var back ShellOutput
	assert.NilError(t, json.Unmarshal(got, &back))
	assert.Equal(t, back.Output, out.Output)
}

func TestShellOutput_MarshalFromFields(t *testing.T) {
	out := ShellOutput{Output: "ok"}
	got, err := json.Marshal(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got), `"ok"`)
}

func TestShellOutput_NonStringTolerated(t *testing.T) {
	const raw = `{"output":"x"}`
	var out ShellOutput
	assert.NilError(t, json.Unmarshal([]byte(raw), &out))
	assert.Equal(t, out.Output, "")
	got, err := json.Marshal(out)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}
