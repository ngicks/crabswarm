package model

// SDKAssistantMessage represents an assistant message.
type SDKAssistantMessage struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Message         map[string]any `json:"message,omitempty"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

// SDKUserMessage represents a user message.
type SDKUserMessage struct {
	UUID            *string        `json:"uuid,omitempty"`
	SessionID       string         `json:"session_id"`
	Message         map[string]any `json:"message,omitempty"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

// SDKUserMessageReplay represents a replayed user message.
type SDKUserMessageReplay struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Message         map[string]any `json:"message,omitempty"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

// SDKPermissionDenial represents a permission denial.
type SDKPermissionDenial struct {
	ToolName  string     `json:"tool_name"`
	ToolUseID string     `json:"tool_use_id"`
	ToolInput *ToolInput `json:"tool_input,omitempty"`
}

// SDKResultMessage represents a result message.
type SDKResultMessage struct {
	Subtype            SDKResultSubtype       `json:"subtype"`
	UUID               string                 `json:"uuid"`
	SessionID          string                 `json:"session_id"`
	DurationMs         int64                  `json:"duration_ms"`
	DurationApiMs      int64                  `json:"duration_api_ms"`
	IsError            bool                   `json:"is_error"`
	NumTurns           int32                  `json:"num_turns"`
	TotalCostUsd       float64                `json:"total_cost_usd"`
	Usage              *NonNullableUsage      `json:"usage,omitempty"`
	ModelUsage         map[string]*ModelUsage `json:"model_usage,omitempty"`
	PermissionDenials  []*SDKPermissionDenial `json:"permission_denials,omitempty"`
	Result             *string                `json:"result,omitempty"`
	StructuredOutput   any                    `json:"structured_output,omitempty"`
	Errors             []string               `json:"errors,omitempty"`
}

// SDKSystemMcpServer represents an MCP server in a system message.
type SDKSystemMcpServer struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// SDKSystemMessage represents a system message.
type SDKSystemMessage struct {
	UUID           string                `json:"uuid"`
	SessionID      string                `json:"session_id"`
	ApiKeySource   ApiKeySource          `json:"api_key_source"`
	Cwd            string                `json:"cwd"`
	Tools          []string              `json:"tools,omitempty"`
	McpServers     []*SDKSystemMcpServer `json:"mcp_servers,omitempty"`
	Model          string                `json:"model"`
	PermissionMode PermissionMode        `json:"permission_mode"`
	SlashCommands  []string              `json:"slash_commands,omitempty"`
	OutputStyle    string                `json:"output_style"`
}

// SDKPartialAssistantMessage represents a partial assistant message.
type SDKPartialAssistantMessage struct {
	UUID            string         `json:"uuid"`
	SessionID       string         `json:"session_id"`
	Event           map[string]any `json:"event,omitempty"`
	ParentToolUseID *string        `json:"parent_tool_use_id,omitempty"`
}

// CompactMetadata represents metadata about a compaction event.
type CompactMetadata struct {
	Trigger   string `json:"trigger"`
	PreTokens int32  `json:"pre_tokens"`
}

// SDKCompactBoundaryMessage represents a compact boundary message.
type SDKCompactBoundaryMessage struct {
	UUID            string           `json:"uuid"`
	SessionID       string           `json:"session_id"`
	CompactMetadata *CompactMetadata `json:"compact_metadata,omitempty"`
}

// SDKMessage is a union type for all SDK messages.
type SDKMessage struct {
	Assistant       *SDKAssistantMessage       `json:"assistant,omitempty"`
	User            *SDKUserMessage            `json:"user,omitempty"`
	UserReplay      *SDKUserMessageReplay      `json:"user_replay,omitempty"`
	Result          *SDKResultMessage          `json:"result,omitempty"`
	System          *SDKSystemMessage          `json:"system,omitempty"`
	Partial         *SDKPartialAssistantMessage `json:"partial,omitempty"`
	CompactBoundary *SDKCompactBoundaryMessage `json:"compact_boundary,omitempty"`
}
