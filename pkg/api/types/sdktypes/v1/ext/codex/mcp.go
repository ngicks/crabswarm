package codex

import "encoding/json"

// CallToolResult is a decoded Codex MCP tool_response — the MCP CallToolResult
// object Codex emits for tool_name values prefixed with [McpToolNamePrefix].
//
// Codex preserves the MCP wire shape with serde rename_all = "camelCase" plus
// an explicit "_meta" rename, so the JSON keys are content, structuredContent,
// isError, and _meta. The decoded fields expose that structure; the original
// object round-trips byte-for-byte via the preserved raw bytes.
//
// Crabswarm decodes MCP tool_response as the SDK ToolOutputUnknown variant
// (which preserves the raw bytes and conveys through gRPC unchanged);
// [ParseCallToolResult] gives callers structured access to that raw payload.
//
// Source: codex-rs/protocol/src/mcp.rs (CallToolResult),
// codex-rs/core/src/tools/handlers/mcp.rs
type CallToolResult struct {
	// Content holds the MCP content blocks, each preserved as raw JSON.
	Content []json.RawMessage `json:"-"`
	// StructuredContent is the optional structured result, nil when absent.
	StructuredContent json.RawMessage `json:"-"`
	// IsError reports whether the tool call failed, nil when absent.
	IsError *bool `json:"-"`
	// Meta is the optional "_meta" payload, nil when absent.
	Meta json.RawMessage `json:"-"`

	// raw preserves the original tool_response bytes for a byte-exact re-marshal.
	raw json.RawMessage
}

// ParseCallToolResult decodes an MCP tool_response object (for example the raw
// bytes of an SDK ToolOutputUnknown variant) into a CallToolResult.
func ParseCallToolResult(data []byte) (CallToolResult, error) {
	var r CallToolResult
	if err := r.UnmarshalJSON(data); err != nil {
		return CallToolResult{}, err
	}
	return r, nil
}

// RawJSON returns a copy of the original tool_response bytes, or nil for a
// value that wasn't decoded from JSON.
func (r CallToolResult) RawJSON() json.RawMessage { return cloneRaw(r.raw) }

// MarshalJSON re-emits the original tool_response bytes when present, otherwise
// builds the CallToolResult object from the decoded fields.
func (r CallToolResult) MarshalJSON() ([]byte, error) {
	if len(r.raw) > 0 {
		return cloneRaw(r.raw), nil
	}
	return json.Marshal(struct {
		Content           []json.RawMessage `json:"content"`
		StructuredContent json.RawMessage   `json:"structuredContent,omitzero"`
		IsError           *bool             `json:"isError,omitzero"`
		Meta              json.RawMessage   `json:"_meta,omitzero"`
	}{Content: r.Content, StructuredContent: r.StructuredContent, IsError: r.IsError, Meta: r.Meta})
}

// UnmarshalJSON decodes the CallToolResult object and preserves the raw bytes.
func (r *CallToolResult) UnmarshalJSON(data []byte) error {
	var v struct {
		Content           []json.RawMessage `json:"content"`
		StructuredContent json.RawMessage   `json:"structuredContent"`
		IsError           *bool             `json:"isError"`
		Meta              json.RawMessage   `json:"_meta"`
	}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	r.raw = cloneRaw(data)
	r.Content = v.Content
	r.StructuredContent = v.StructuredContent
	r.IsError = v.IsError
	r.Meta = v.Meta
	return nil
}
