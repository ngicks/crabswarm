package model

// ApiKeySource represents the source of an API key.
type ApiKeySource string

const (
	ApiKeySourceUser      ApiKeySource = "user"
	ApiKeySourceProject   ApiKeySource = "project"
	ApiKeySourceOrg       ApiKeySource = "org"
	ApiKeySourceTemporary ApiKeySource = "temporary"
)

// SdkBeta represents an SDK beta feature flag.
type SdkBeta string

const (
	SdkBetaContext1m20250807 SdkBeta = "context-1m-2025-08-07"
)

// ConfigScope represents the scope of a configuration setting.
type ConfigScope string

const (
	ConfigScopeLocal   ConfigScope = "local"
	ConfigScopeUser    ConfigScope = "user"
	ConfigScopeProject ConfigScope = "project"
)

// McpServerConnectionStatus represents the connection status of an MCP server.
type McpServerConnectionStatus string

const (
	McpServerConnectionStatusConnected McpServerConnectionStatus = "connected"
	McpServerConnectionStatusFailed    McpServerConnectionStatus = "failed"
	McpServerConnectionStatusNeedsAuth McpServerConnectionStatus = "needs-auth"
	McpServerConnectionStatusPending   McpServerConnectionStatus = "pending"
)

// CallToolResultContentType represents the type of content in a call tool result.
type CallToolResultContentType string

const (
	CallToolResultContentTypeText     CallToolResultContentType = "text"
	CallToolResultContentTypeImage    CallToolResultContentType = "image"
	CallToolResultContentTypeResource CallToolResultContentType = "resource"
)
