package codex

import "encoding/json"

// ShellOutput is a decoded Codex shell tool_response. Codex's shell_command and
// exec_command tools both report the hook tool_name "Bash" and return their
// output as a bare JSON string of the raw aggregated stdout+stderr — with no
// "Exit code:" / "Wall time:" headers (unlike apply_patch). Claude's BashOutput
// is a JSON object, so the string form needs this Codex-specific type.
//
// The decoded Output exposes the text; the original string round-trips
// byte-for-byte via the preserved raw bytes.
//
// Source: codex-rs/core/src/tools/handlers/shell.rs (format_exec_output_str)
type ShellOutput struct {
	// Output is the decoded raw aggregated stdout+stderr.
	Output string `json:"-"`

	// raw preserves the original tool_response bytes for a byte-exact re-marshal.
	raw json.RawMessage
}

// RawJSON returns a copy of the original tool_response bytes, or nil for a
// value that wasn't decoded from JSON.
func (o ShellOutput) RawJSON() json.RawMessage { return cloneRaw(o.raw) }

// MarshalJSON re-emits the original tool_response bytes when present, otherwise
// encodes Output as a JSON string.
func (o ShellOutput) MarshalJSON() ([]byte, error) {
	if len(o.raw) > 0 {
		return cloneRaw(o.raw), nil
	}
	return json.Marshal(o.Output)
}

// UnmarshalJSON preserves the raw bytes and, when the tool_response is a JSON
// string, records the decoded text. A non-string value is tolerated (raw is
// still preserved) so a malformed payload never aborts the surrounding hook
// parse.
func (o *ShellOutput) UnmarshalJSON(data []byte) error {
	o.raw = cloneRaw(data)
	var s string
	if json.Unmarshal(data, &s) != nil {
		return nil
	}
	o.Output = s
	return nil
}
