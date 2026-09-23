package chat

// What a member says about the harness it runs, and about how it wants to be
// reached. The two are declared separately on purpose: a harness this list does
// not yet name can still say by what route a mention gets to it, and a harness
// it does name may be reached either way depending on what its server can do.

// Harness is the CLI a member runs, as the member's own MCP server read it off
// the MCP handshake. The empty value is a member nothing said this about: a
// human, or an agent whose server declared no harness at all. A server that
// handshaked under a name nothing recognises declares [HarnessOther] instead.
//
// The daemon records it and shows it. One thing turns on it: the screen poller
// reads the terminal of an agent attending as [HarnessOther] and of no other
// member, since every harness this list names reports on a feed of its own.
type Harness string

const (
	// HarnessClaudeCode is Anthropic's Claude Code.
	HarnessClaudeCode Harness = "claude-code"
	// HarnessCodex is OpenAI's Codex CLI.
	HarnessCodex Harness = "codex"
	// HarnessOpenCode is the OpenCode CLI.
	HarnessOpenCode Harness = "opencode"
	// HarnessOther is a harness the daemon has no name for.
	HarnessOther Harness = "other"
)

// NudgeDelivery is how a mention reaches a member: by the daemon typing into
// its terminal, or by the member's own MCP server pushing it through the
// harness's notification channel.
//
// The empty value is a member that is never nudged at all, which is what a
// human is: they read their room when they choose to.
type NudgeDelivery string

const (
	// NudgeTerminal has the daemon type the mention into the member's terminal.
	// It is what every agent was before a harness could deliver its own, and so
	// what an agent declaring nothing attends as.
	NudgeTerminal NudgeDelivery = "terminal"
	// NudgeNative has the member's MCP server deliver the mention through the
	// harness's own channel. The daemon records the message, announces it to the
	// room and types nothing: the server is watching the same feed.
	NudgeNative NudgeDelivery = "native"
)
