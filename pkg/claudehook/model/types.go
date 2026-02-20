package model

// PermissionResult is a union type for permission results.
type PermissionResult struct {
	Allow *AllowResult `json:"allow,omitempty"`
	Deny  *DenyResult  `json:"deny,omitempty"`
}

// AllowResult represents an allowed permission result.
type AllowResult struct {
	UpdatedInput       *ToolInput          `json:"updated_input,omitempty"`
	UpdatedPermissions []*PermissionUpdate `json:"updated_permissions,omitempty"`
}

// DenyResult represents a denied permission result.
type DenyResult struct {
	Message   string `json:"message"`
	Interrupt *bool  `json:"interrupt,omitempty"`
}

// McpServerConfig is a union type for MCP server configurations.
type McpServerConfig struct {
	Stdio *McpStdioServerConfig `json:"stdio,omitempty"`
	Sse   *McpSSEServerConfig   `json:"sse,omitempty"`
	Http  *McpHttpServerConfig  `json:"http,omitempty"`
}

// McpStdioServerConfig represents a stdio MCP server configuration.
type McpStdioServerConfig struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// McpSSEServerConfig represents an SSE MCP server configuration.
type McpSSEServerConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// McpHttpServerConfig represents an HTTP MCP server configuration.
type McpHttpServerConfig struct {
	URL     string            `json:"url"`
	Headers map[string]string `json:"headers,omitempty"`
}

// SdkPluginConfig represents an SDK plugin configuration.
type SdkPluginConfig struct {
	Path string `json:"path"`
}

// AgentDefinition represents the definition of an agent.
type AgentDefinition struct {
	Description string   `json:"description"`
	Tools       []string `json:"tools,omitempty"`
	Prompt      string   `json:"prompt"`
	Model       string   `json:"model"`
}

// SystemPromptPreset represents a system prompt preset.
type SystemPromptPreset struct {
	Append string `json:"append"`
}

// SystemPrompt is a union type for system prompts.
type SystemPrompt struct {
	Custom *string             `json:"custom,omitempty"`
	Preset *SystemPromptPreset `json:"preset,omitempty"`
}

// OutputFormat represents output format options.
type OutputFormat struct {
	Schema map[string]any `json:"schema,omitempty"`
}

// SandboxIgnoreViolations represents sandbox violations to ignore.
type SandboxIgnoreViolations struct {
	File    []string `json:"file,omitempty"`
	Network []string `json:"network,omitempty"`
}

// NetworkSandboxSettings represents network sandbox settings.
type NetworkSandboxSettings struct {
	AllowLocalBinding   *bool    `json:"allow_local_binding,omitempty"`
	AllowUnixSockets    []string `json:"allow_unix_sockets,omitempty"`
	AllowAllUnixSockets *bool    `json:"allow_all_unix_sockets,omitempty"`
	HttpProxyPort       *int32   `json:"http_proxy_port,omitempty"`
	SocksProxyPort      *int32   `json:"socks_proxy_port,omitempty"`
}

// SandboxSettings represents sandbox settings.
type SandboxSettings struct {
	Enabled                   *bool                    `json:"enabled,omitempty"`
	AutoAllowBashIfSandboxed  *bool                    `json:"auto_allow_bash_if_sandboxed,omitempty"`
	ExcludedCommands          []string                 `json:"excluded_commands,omitempty"`
	AllowUnsandboxedCommands  *bool                    `json:"allow_unsandboxed_commands,omitempty"`
	Network                   *NetworkSandboxSettings  `json:"network,omitempty"`
	IgnoreViolations          *SandboxIgnoreViolations `json:"ignore_violations,omitempty"`
	EnableWeakerNestedSandbox *bool                    `json:"enable_weaker_nested_sandbox,omitempty"`
}

// Options represents SDK options.
type Options struct {
	AdditionalDirectories          []string                    `json:"additional_directories,omitempty"`
	Agents                         map[string]*AgentDefinition `json:"agents,omitempty"`
	AllowDangerouslySkipPermissions bool                       `json:"allow_dangerously_skip_permissions"`
	AllowedTools                   []string                    `json:"allowed_tools,omitempty"`
	Betas                          []string                    `json:"betas,omitempty"`
	Continue                       bool                        `json:"continue"`
	Cwd                            string                      `json:"cwd"`
	DisallowedTools                []string                    `json:"disallowed_tools,omitempty"`
	EnableFileCheckpointing        bool                        `json:"enable_file_checkpointing"`
	Env                            map[string]string           `json:"env,omitempty"`
	Executable                     string                      `json:"executable"`
	ExecutableArgs                 []string                    `json:"executable_args,omitempty"`
	ExtraArgs                      map[string]string           `json:"extra_args,omitempty"`
	FallbackModel                  string                      `json:"fallback_model"`
	ForkSession                    bool                        `json:"fork_session"`
	IncludePartialMessages         bool                        `json:"include_partial_messages"`
	MaxBudgetUsd                   *float64                    `json:"max_budget_usd,omitempty"`
	MaxThinkingTokens              *int32                      `json:"max_thinking_tokens,omitempty"`
	MaxTurns                       *int32                      `json:"max_turns,omitempty"`
	McpServers                     map[string]*McpServerConfig `json:"mcp_servers,omitempty"`
	Model                          string                      `json:"model"`
	OutputFormat                   *OutputFormat               `json:"output_format,omitempty"`
	PathToClaudeCodeExecutable     string                      `json:"path_to_claude_code_executable"`
	PermissionMode                 string                      `json:"permission_mode"`
	PermissionPromptToolName       string                      `json:"permission_prompt_tool_name"`
	Plugins                        []*SdkPluginConfig          `json:"plugins,omitempty"`
	Resume                         string                      `json:"resume"`
	ResumeSessionAt                string                      `json:"resume_session_at"`
	Sandbox                        *SandboxSettings            `json:"sandbox,omitempty"`
	SettingSources                 []string                    `json:"setting_sources,omitempty"`
	StrictMcpConfig                bool                        `json:"strict_mcp_config"`
	SystemPrompt                   *SystemPrompt               `json:"system_prompt,omitempty"`
	ToolsList                      []string                    `json:"tools_list,omitempty"`
	ToolsPreset                    bool                        `json:"tools_preset"`
}
