package codex

import (
	"encoding/json"
	"testing"

	"gotest.tools/v3/assert"
)

func TestCallToolResult_Parse(t *testing.T) {
	const raw = `{"content":[{"type":"text","text":"notes"}],` +
		`"structuredContent":{"bytes":5},"isError":false}`

	r, err := ParseCallToolResult([]byte(raw))
	assert.NilError(t, err)
	assert.Equal(t, len(r.Content), 1)
	assert.Equal(t, string(r.Content[0]), `{"type":"text","text":"notes"}`)
	assert.Equal(t, string(r.StructuredContent), `{"bytes":5}`)
	assert.Assert(t, r.IsError != nil)
	assert.Equal(t, *r.IsError, false)

	got, err := json.Marshal(r)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}

func TestCallToolResult_RoundTripWithMeta(t *testing.T) {
	const raw = `{"content":[],"isError":true,"_meta":{"k":"v"}}`
	r, err := ParseCallToolResult([]byte(raw))
	assert.NilError(t, err)
	assert.Equal(t, string(r.Meta), `{"k":"v"}`)

	got, err := json.Marshal(r)
	assert.NilError(t, err)
	assert.Equal(t, string(got), raw)
}

func TestCallToolResult_MarshalFromFields(t *testing.T) {
	r := CallToolResult{Content: []json.RawMessage{json.RawMessage(`{"type":"text","text":"hi"}`)}}
	got, err := json.Marshal(r)
	assert.NilError(t, err)
	assert.Equal(t, string(got), `{"content":[{"type":"text","text":"hi"}]}`)
}
