package model

// SlashCommand represents a slash command.
type SlashCommand struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	ArgumentHint string `json:"argument_hint"`
}

// ModelInfo represents information about a model.
type ModelInfo struct {
	Value       string `json:"value"`
	DisplayName string `json:"display_name"`
	Description string `json:"description"`
}

// McpServerInfo represents information about an MCP server.
type McpServerInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// McpServerStatus represents the status of an MCP server.
type McpServerStatus struct {
	Name       string         `json:"name"`
	Status     string         `json:"status"`
	ServerInfo *McpServerInfo `json:"server_info,omitempty"`
}

// AccountInfo represents account information.
type AccountInfo struct {
	Email            *string `json:"email,omitempty"`
	Organization     *string `json:"organization,omitempty"`
	SubscriptionType *string `json:"subscription_type,omitempty"`
	TokenSource      *string `json:"token_source,omitempty"`
	ApiKeySource     *string `json:"api_key_source,omitempty"`
}

// ModelUsage represents usage statistics for a model.
type ModelUsage struct {
	InputTokens               int64   `json:"input_tokens"`
	OutputTokens              int64   `json:"output_tokens"`
	CacheReadInputTokens      int64   `json:"cache_read_input_tokens"`
	CacheCreationInputTokens  int64   `json:"cache_creation_input_tokens"`
	WebSearchRequests         int64   `json:"web_search_requests"`
	CostUsd                   float64 `json:"cost_usd"`
	ContextWindow             int64   `json:"context_window"`
}

// NonNullableUsage represents non-nullable usage statistics.
type NonNullableUsage struct {
	InputTokens               int64  `json:"input_tokens"`
	OutputTokens              int64  `json:"output_tokens"`
	CacheCreationInputTokens  *int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens      *int64 `json:"cache_read_input_tokens,omitempty"`
}

// Usage represents nullable usage statistics.
type Usage struct {
	InputTokens               *int64 `json:"input_tokens,omitempty"`
	OutputTokens              *int64 `json:"output_tokens,omitempty"`
	CacheCreationInputTokens  *int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens      *int64 `json:"cache_read_input_tokens,omitempty"`
}

// CallToolResultContent represents content in a call tool result.
type CallToolResultContent struct {
	Type string         `json:"type"`
	Data map[string]any `json:"data,omitempty"`
}

// CallToolResult represents the result of a tool call.
type CallToolResult struct {
	Content []*CallToolResultContent `json:"content,omitempty"`
	IsError *bool                    `json:"is_error,omitempty"`
}
