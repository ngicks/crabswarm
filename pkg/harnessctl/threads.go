package harnessctl

// A Codex app server hosts every thread its TUIs open, and one app server may
// be shared by several agents, each with TUIs of its own. The server a harness
// is detected by serves one MCP session per thread, so it is the one that can
// say which of those threads an agent is: a session learns its thread from the
// tool calls it receives. What follows is how the two ends meet. The harness
// asks [AgentThreads] which thread an agent is in front of, and the MCP server
// answers from the sessions it holds.

// AgentThreads is one agent's side of finding the thread a harness shared by
// several agents delivers to and reports on.
//
// The agent is served by one or more MCP sessions, one per thread its TUIs
// opened. A session names its thread once a tool call reaches it, and until
// then it is unbound. The agent is in front of the bound thread it was last
// seen in front of.
//
// Every method answers from memory and returns at once. A harness calls them
// with none of its own locks held, and the answer may change the moment after.
type AgentThreads interface {
	// Target is the bound thread the agent was last seen in front of, and
	// whether any of the agent's sessions is bound at all.
	Target() (thread string, ok bool)
	// Unbound reports whether a session of the agent has not named its thread
	// yet. A harness that needs the target pings the threads it does not know
	// first, since the unbound session may be the one the agent is in front of.
	Unbound() bool
	// Bound reports whether a session of any agent already named thread, so a
	// ping to it would learn nothing.
	Bound(thread string) bool
	// Changed is signalled whenever Target or Unbound may answer differently. It
	// is the same channel on every call, and one signal may stand for several
	// changes.
	Changed() <-chan struct{}
	// Ping is the tool call that asks the MCP session of thread which agent it
	// serves. The session it reaches is bound to thread. done withdraws the
	// call once its answer is in, whatever the answer was.
	Ping(thread string) (call ToolCall, done func())
}

// ToolCall is one tool call a harness has its server make on the MCP server of
// a thread: the server as the harness's own configuration names it, the tool,
// and its arguments.
type ToolCall struct {
	Server    string
	Tool      string
	Arguments map[string]any
}

// ThreadScoper is a harness whose server hosts several agents at once, as a
// shared Codex app server does. Unscoped, it delivers to and reports on the
// thread a person used last, which is right only while one agent uses the
// server.
type ThreadScoper interface {
	Harness
	// ScopeTo returns the harness that delivers to and reports on the thread
	// threads names, and nothing else. Every harness it returns shares the
	// connection of the one it was called on.
	ScopeTo(threads AgentThreads) Harness
}
