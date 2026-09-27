package codexproxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/ngicks/crabswarm/crabswarm/mcp"
)

// The thread requests that start or reload a thread, and with it the thread's
// MCP servers. Their params.config is where the token goes.
const (
	methodThreadStart  = "thread/start"
	methodThreadResume = "thread/resume"
	methodThreadFork   = "thread/fork"
)

var threadMethods = []string{methodThreadStart, methodThreadResume, methodThreadFork}

// focusIdPrefix begins the id of every request the proxy makes itself. The
// TUI numbers its own requests or names them after what they do, so the
// prefix is how an answer is told apart as the proxy's.
const focusIdPrefix = "crabswarm-proxy-"

// focusTool is the crabswarm MCP tool the proxy calls on a resumed thread.
const focusTool = "crabswarm_focus"

// tokenKey is the per-thread config key that sets the token header on every
// request the thread's MCP server named server sends. The dotted spelling
// layers over the app server's own overrides of that server, where a nested
// object would replace them.
func tokenKey(server string) string {
	return "mcp_servers." + server + ".http_headers." + mcp.TokenHeader
}

// threadRequest is the method of msg when it is one of the thread requests,
// and "" for anything else, JSON or not.
func threadRequest(msg []byte) string {
	var req struct {
		Method string `json:"method"`
	}
	if json.Unmarshal(msg, &req) != nil || !slices.Contains(threadMethods, req.Method) {
		return ""
	}
	return req.Method
}

// stamp returns the request msg with params.config[key] set to token. An
// absent or null params or config is created; every other member keeps the
// value the TUI wrote.
func stamp(msg []byte, key, token string) ([]byte, error) {
	var req map[string]json.RawMessage
	if err := json.Unmarshal(msg, &req); err != nil {
		return nil, err
	}
	params, err := object(req["params"])
	if err != nil {
		return nil, fmt.Errorf("reading params: %w", err)
	}
	config, err := object(params["config"])
	if err != nil {
		return nil, fmt.Errorf("reading params.config: %w", err)
	}
	if config[key], err = marshal(token); err != nil {
		return nil, err
	}
	if params["config"], err = marshal(config); err != nil {
		return nil, err
	}
	if req["params"], err = marshal(params); err != nil {
		return nil, err
	}
	return marshal(req)
}

// resumedThread is the params.threadId of a thread/resume request, or "" when
// it names none.
func resumedThread(msg []byte) string {
	var req struct {
		Params struct {
			ThreadId string `json:"threadId"`
		} `json:"params"`
	}
	if json.Unmarshal(msg, &req) != nil {
		return ""
	}
	return req.Params.ThreadId
}

// focusCall is the request calling the focus tool of the MCP server named
// server on threadId, numbered n among the calls one connection made.
func focusCall(n int, server, threadId string) ([]byte, error) {
	type params struct {
		Server    string   `json:"server"`
		ThreadId  string   `json:"threadId"`
		Tool      string   `json:"tool"`
		Arguments struct{} `json:"arguments"`
	}
	return marshal(struct {
		Jsonrpc string `json:"jsonrpc"`
		Id      string `json:"id"`
		Method  string `json:"method"`
		Params  params `json:"params"`
	}{
		Jsonrpc: "2.0",
		Id:      focusIdPrefix + strconv.Itoa(n),
		Method:  "mcpServer/tool/call",
		Params:  params{Server: server, ThreadId: threadId, Tool: focusTool},
	})
}

// ownAnswer reports whether msg answers a request the proxy made itself.
//
// A server message the size of a thread's whole history is common, so the
// prefix is looked for in the raw bytes before anything is decoded.
func ownAnswer(msg []byte) bool {
	if !bytes.Contains(msg, []byte(focusIdPrefix)) {
		return false
	}
	var frame struct {
		Id     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	if json.Unmarshal(msg, &frame) != nil || frame.Method != "" {
		return false
	}
	var id string
	if json.Unmarshal(frame.Id, &id) != nil {
		return false
	}
	return strings.HasPrefix(id, focusIdPrefix)
}

// object decodes a JSON object member, reading an absent or null one as an
// empty object.
func object(raw json.RawMessage) (map[string]json.RawMessage, error) {
	var m map[string]json.RawMessage
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
	}
	if m == nil {
		m = map[string]json.RawMessage{}
	}
	return m, nil
}

// marshal is json.Marshal without HTML escaping, so the members the proxy
// passes through keep the spelling the TUI gave them: json.Marshal would
// rewrite every <, > and & inside them.
func marshal(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimSuffix(buf.Bytes(), []byte("\n")), nil
}
