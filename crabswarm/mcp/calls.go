package mcp

import (
	"context"
	"encoding/json"
	"strings"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// A Codex app server can call any tool on the MCP server of a thread it hosts,
// listed or not, and the call reaches the session serving that thread with the
// thread's id in its metadata. That is how the side of a harness server that
// sits outside the MCP session tells the session something about itself: which
// thread it serves, and that the person is in front of it now. Such a call is
// the server's own business, so it is answered here and never reaches a tool
// or the model.

// reservedToolPrefix begins the name of every tool call the server answers
// itself. No family registers a tool under it.
const reservedToolPrefix = "crabswarm_"

const (
	// focusTool says the person is in front of the thread the call reached. A
	// proxy in front of a TUI calls it after the TUI resumes a thread, which
	// the app server says nothing about on its feed.
	focusTool = "crabswarm_focus"
	// pingTool asks the session it reaches to take the thread it was sent to
	// as its own. Its one argument, pingNonceArg, names the ping, so the thread
	// is known even from a call that carried no metadata.
	pingTool     = "crabswarm_ping"
	pingNonceArg = "nonce"
)

// methodCallTool is the MCP request a tool call arrives as.
const methodCallTool = "tools/call"

// callWatcher is what a server learns about its sessions from the tool calls
// they receive. Each method returns at once: it runs inside the request that
// made the call.
type callWatcher interface {
	// toolCalled is any tool call that named thread in its metadata.
	toolCalled(session *mcpsdk.ServerSession, thread string)
	// focused is a focus call, which named thread in its metadata or nothing.
	focused(session *mcpsdk.ServerSession, thread string)
	// pinged is a ping call, which named thread in its metadata or nothing, and
	// carried nonce.
	pinged(session *mcpsdk.ServerSession, thread, nonce string)
}

// answersReservedCalls answers every reserved tool call itself, with a success
// that carries nothing, and tells w what each tool call said about the session
// it arrived on. A nil w learns nothing.
//
// The result is written out here because the call never reaches the SDK's own
// handler, which is what fills in empty content: a result with none would go
// out as null, which a strict client refuses.
func answersReservedCalls(w callWatcher) mcpsdk.Middleware {
	return func(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
		return func(
			ctx context.Context, method string, req mcpsdk.Request,
		) (mcpsdk.Result, error) {
			if method != methodCallTool {
				return next(ctx, method, req)
			}
			session, _ := req.GetSession().(*mcpsdk.ServerSession)
			params, _ := req.GetParams().(*mcpsdk.CallToolParamsRaw)
			if session == nil || params == nil {
				return next(ctx, method, req)
			}
			thread := metaThread(params.Meta)
			if !strings.HasPrefix(params.Name, reservedToolPrefix) {
				if w != nil && thread != "" {
					w.toolCalled(session, thread)
				}
				return next(ctx, method, req)
			}
			if w != nil {
				switch params.Name {
				case focusTool:
					w.focused(session, thread)
				case pingTool:
					w.pinged(session, thread, pingNonce(params.Arguments))
				}
			}
			return &mcpsdk.CallToolResult{Content: []mcpsdk.Content{}}, nil
		}
	}
}

// metaThread is the Codex thread a tool call's metadata names, or "".
func metaThread(meta mcpsdk.Meta) string {
	thread, _ := meta["threadId"].(string)
	return thread
}

// pingNonce is the nonce a ping carried, or "" when it carried none.
func pingNonce(args json.RawMessage) string {
	var ping map[string]any
	if json.Unmarshal(args, &ping) != nil {
		return ""
	}
	nonce, _ := ping[pingNonceArg].(string)
	return nonce
}
