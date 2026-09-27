package codexproxy

import (
	"encoding/json"
	"log/slog"
	"testing"

	"github.com/coder/websocket"
	"gotest.tools/v3/assert"
)

const headerKey = "mcp_servers.crabswarm-mcp.http_headers.X-Crabswarm-Token"

// The recorded proxy stamped each thread request with an env key. Taking that
// key out gives the request as the TUI sent it; stamping that has to give the
// recorded request back, with the header key where the env key was and every
// other member as recorded, the helper thread's nested mcp_servers included.
func TestStamp_ReproducesTheRecordedRewrites(t *testing.T) {
	const envKey = "mcp_servers.crabswarm-mcp.env.CRABSWARM_CHAT_TOKEN"

	var lines int
	for _, name := range []string{
		"codex-proxy-injected-tui.ndjson",
		"codex-proxy-injected-resume.ndjson",
	} {
		for _, recorded := range fixtureLines(t, name) {
			lines++
			want := decoded(t, recorded)
			config := want["params"].(map[string]any)["config"].(map[string]any)
			token, ok := config[envKey].(string)
			assert.Assert(t, ok, "%s holds a line without %s", name, envKey)

			delete(config, envKey)
			sent, err := json.Marshal(want)
			assert.NilError(t, err)
			config[headerKey] = token

			stamped, err := stamp(sent, headerKey, token)
			assert.NilError(t, err)
			assert.DeepEqual(t, decoded(t, stamped), want)
		}
	}
	assert.Equal(t, lines, 4)
}

// The helper thread Codex starts beside a TUI's own disables the crabswarm
// MCP server with a nested object. The dotted key lands beside it and leaves
// it as it was.
func TestStamp_LeavesTheHelperThreadsNestedServerConfig(t *testing.T) {
	var helper []byte
	for _, line := range fixtureLines(t, "codex-tui-thread-lifecycle.client.ndjson") {
		var req struct {
			Params struct {
				ThreadSource string `json:"threadSource"`
			} `json:"params"`
		}
		assert.NilError(t, json.Unmarshal(line, &req))
		if req.Params.ThreadSource == "thread_title" {
			helper = line
		}
	}
	assert.Assert(t, helper != nil, "the recording holds no helper thread")

	stamped, err := stamp(helper, headerKey, "tokA")
	assert.NilError(t, err)
	config := decoded(t, stamped)["params"].(map[string]any)["config"].(map[string]any)
	assert.DeepEqual(t, config["mcp_servers"],
		map[string]any{"crabswarm-mcp": map[string]any{"enabled": false}})
	assert.Equal(t, config[headerKey], "tokA")
	assert.DeepEqual(t, decoded(t, stamped), withToken(t, helper, "tokA"))
}

// A request without params or config gets both, and every member the TUI
// wrote keeps its spelling: no HTML escaping, no reformatted numbers.
func TestStamp_CreatesWhatIsMissingAndKeepsTheRest(t *testing.T) {
	for _, tc := range []struct {
		name string
		msg  string
		want string
	}{
		{
			name: "no params",
			msg:  `{"id":1,"method":"thread/start"}`,
			want: `{"id":1,"method":"thread/start","params":{"config":{"` + headerKey + `":"tok"}}}`,
		},
		{
			name: "null params",
			msg:  `{"id":1,"method":"thread/fork","params":null}`,
			want: `{"id":1,"method":"thread/fork","params":{"config":{"` + headerKey + `":"tok"}}}`,
		},
		{
			name: "null config",
			msg:  `{"id":2,"method":"thread/resume","params":{"threadId":"t","config":null}}`,
			want: `{"id":2,"method":"thread/resume",` +
				`"params":{"config":{"` + headerKey + `":"tok"},"threadId":"t"}}`,
		},
		{
			name: "spelling kept",
			msg: `{"id":3,"method":"thread/start","params":{"config":{"x":1.50},` +
				`"developerInstructions":"a<b>&c é"}}`,
			want: `{"id":3,"method":"thread/start","params":{"config":{"` + headerKey +
				`":"tok","x":1.50},"developerInstructions":"a<b>&c é"}}`,
		},
		{
			name: "a token already there is replaced",
			msg:  `{"id":4,"method":"thread/start","params":{"config":{"` + headerKey + `":"old"}}}`,
			want: `{"id":4,"method":"thread/start","params":{"config":{"` + headerKey + `":"tok"}}}`,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			stamped, err := stamp([]byte(tc.msg), headerKey, "tok")
			assert.NilError(t, err)
			assert.Equal(t, string(stamped), tc.want)
		})
	}
}

// params or config that is no object cannot take the key, and is reported
// rather than overwritten.
func TestStamp_RefusesANonObject(t *testing.T) {
	for _, msg := range []string{
		`{"id":1,"method":"thread/start","params":[1]}`,
		`{"id":1,"method":"thread/start","params":{"config":"x"}}`,
	} {
		_, err := stamp([]byte(msg), headerKey, "tok")
		assert.Assert(t, err != nil, "%s was stamped", msg)
	}
}

func TestThreadRequest_NamesTheThreeThreadMethodsOnly(t *testing.T) {
	for msg, want := range map[string]string{
		`{"id":1,"method":"thread/start","params":{}}`:           "thread/start",
		`{"id":1,"method":"thread/resume","params":{}}`:          "thread/resume",
		`{"id":1,"method":"thread/fork","params":{}}`:            "thread/fork",
		`{"id":1,"method":"thread/unsubscribe","params":{}}`:     "",
		`{"id":1,"method":"turn/start","params":{}}`:             "",
		`{"id":1,"result":{"method":"thread/start"}}`:            "",
		`[{"id":1,"method":"thread/start"}]`:                     "",
		`not json "method":"thread/start"`:                       "",
		`{"id":1,"method":"thread/start/extra","params":{}}`:     "",
		`{"id":"x","method":"thread/resume","params":{"a":1}}`:   "thread/resume",
		`{"id":1,"jsonrpc":"2.0","method":"thread/fork"}`:        "thread/fork",
		`{"id":1,"method":"THREAD/START","params":{"config":1}}`: "",
	} {
		assert.Equal(t, threadRequest([]byte(msg)), want, msg)
	}
}

// Only an answer carrying one of the proxy's own ids is the proxy's. A server
// request with such an id, or any message merely mentioning the prefix, is
// not.
func TestOwnAnswer_RecognisesTheAnswersToTheProxysOwnRequests(t *testing.T) {
	for msg, want := range map[string]bool{
		`{"id":"crabswarm-proxy-1","result":{}}`:                             true,
		`{"id":"crabswarm-proxy-12","error":{"code":-32602,"message":"no"}}`: true,
		`{"id":"crabswarm-proxy-1","method":"item/tool/call","params":{}}`:   false,
		`{"id":3,"result":{"text":"crabswarm-proxy-3"}}`:                     false,
		`{"id":"startup-thread-start-1","result":{}}`:                        false,
		`{"method":"thread/started","params":{"id":"crabswarm-proxy-1"}}`:    false,
		`crabswarm-proxy-1`: false,
	} {
		assert.Equal(t, ownAnswer([]byte(msg)), want, msg)
	}
}

// One connection's rewrites: nothing without a token or for a binary frame, a
// thread request that cannot take the key goes on as the TUI sent it, and a
// resume naming no thread is stamped without a focus call.
func TestStamper_Rewrite(t *testing.T) {
	logger := slog.New(slog.DiscardHandler)
	start := []byte(`{"id":1,"method":"thread/start","params":{}}`)
	resume := []byte(`{"id":2,"method":"thread/resume","params":{"threadId":"t1"}}`)

	s := &stamper{logger: logger, server: DefaultServer}
	assert.DeepEqual(t, s.rewrite(websocket.MessageText, resume), [][]byte{resume})

	s = &stamper{logger: logger, server: DefaultServer, token: "tok"}
	assert.DeepEqual(t, s.rewrite(websocket.MessageBinary, start), [][]byte{start})

	badConfig := []byte(`{"id":3,"method":"thread/resume","params":{"threadId":"t1","config":7}}`)
	assert.DeepEqual(t, s.rewrite(websocket.MessageText, badConfig), [][]byte{badConfig})

	byPath := []byte(`{"id":4,"method":"thread/resume","params":{"path":"/r.jsonl"}}`)
	out := s.rewrite(websocket.MessageText, byPath)
	assert.Equal(t, len(out), 1)
	assert.DeepEqual(t, decoded(t, out[0]), withToken(t, byPath, "tok"))

	out = s.rewrite(websocket.MessageText, resume)
	assert.Equal(t, len(out), 2)
	assert.Equal(t, string(out[1]),
		`{"jsonrpc":"2.0","id":"crabswarm-proxy-1","method":"mcpServer/tool/call",`+
			`"params":{"server":"crabswarm-mcp","threadId":"t1",`+
			`"tool":"crabswarm_focus","arguments":{}}}`)

	s = &stamper{logger: logger, server: "other", token: "tok"}
	out = s.rewrite(websocket.MessageText, resume)
	assert.Equal(t, len(out), 2)
	config := decoded(t, out[0])["params"].(map[string]any)["config"].(map[string]any)
	assert.Equal(t, config["mcp_servers.other.http_headers.X-Crabswarm-Token"], "tok")
	assert.Equal(t, decoded(t, out[1])["params"].(map[string]any)["server"], "other")
}
