package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func AllowResultFromProto(pb *sdktypes.AllowResult) *AllowResult {
	if pb == nil {
		return nil
	}
	return &AllowResult{
		UpdatedInput:       ToolInputFromProto(pb.GetUpdatedInput()),
		UpdatedPermissions: permissionUpdatesFromProto(pb.GetUpdatedPermissions()),
	}
}

func (m *AllowResult) ToProto() *sdktypes.AllowResult {
	if m == nil {
		return nil
	}
	return &sdktypes.AllowResult{
		UpdatedInput:       m.UpdatedInput.ToProto(),
		UpdatedPermissions: permissionUpdatesToProto(m.UpdatedPermissions),
	}
}

func DenyResultFromProto(pb *sdktypes.DenyResult) *DenyResult {
	if pb == nil {
		return nil
	}
	return &DenyResult{
		Message:   pb.GetMessage(),
		Interrupt: optionalBoolFromProto(pb.Interrupt),
	}
}

func (m *DenyResult) ToProto() *sdktypes.DenyResult {
	if m == nil {
		return nil
	}
	return &sdktypes.DenyResult{
		Message:   m.Message,
		Interrupt: optionalBoolToProto(m.Interrupt),
	}
}

func PermissionResultFromProto(pb *sdktypes.PermissionResult) *PermissionResult {
	if pb == nil {
		return nil
	}
	m := &PermissionResult{}
	switch v := pb.GetResult().(type) {
	case *sdktypes.PermissionResult_Allow:
		m.Allow = AllowResultFromProto(v.Allow)
	case *sdktypes.PermissionResult_Deny:
		m.Deny = DenyResultFromProto(v.Deny)
	}
	return m
}

func (m *PermissionResult) ToProto() *sdktypes.PermissionResult {
	if m == nil {
		return nil
	}
	pb := &sdktypes.PermissionResult{}
	switch {
	case m.Allow != nil:
		pb.Result = &sdktypes.PermissionResult_Allow{Allow: m.Allow.ToProto()}
	case m.Deny != nil:
		pb.Result = &sdktypes.PermissionResult_Deny{Deny: m.Deny.ToProto()}
	}
	return pb
}

func McpStdioServerConfigFromProto(pb *sdktypes.McpStdioServerConfig) *McpStdioServerConfig {
	if pb == nil {
		return nil
	}
	return &McpStdioServerConfig{
		Command: pb.GetCommand(),
		Args:    pb.GetArgs(),
		Env:     pb.GetEnv(),
	}
}

func (m *McpStdioServerConfig) ToProto() *sdktypes.McpStdioServerConfig {
	if m == nil {
		return nil
	}
	return &sdktypes.McpStdioServerConfig{
		Command: m.Command,
		Args:    m.Args,
		Env:     m.Env,
	}
}

func McpSSEServerConfigFromProto(pb *sdktypes.McpSSEServerConfig) *McpSSEServerConfig {
	if pb == nil {
		return nil
	}
	return &McpSSEServerConfig{
		URL:     pb.GetUrl(),
		Headers: pb.GetHeaders(),
	}
}

func (m *McpSSEServerConfig) ToProto() *sdktypes.McpSSEServerConfig {
	if m == nil {
		return nil
	}
	return &sdktypes.McpSSEServerConfig{
		Url:     m.URL,
		Headers: m.Headers,
	}
}

func McpHttpServerConfigFromProto(pb *sdktypes.McpHttpServerConfig) *McpHttpServerConfig {
	if pb == nil {
		return nil
	}
	return &McpHttpServerConfig{
		URL:     pb.GetUrl(),
		Headers: pb.GetHeaders(),
	}
}

func (m *McpHttpServerConfig) ToProto() *sdktypes.McpHttpServerConfig {
	if m == nil {
		return nil
	}
	return &sdktypes.McpHttpServerConfig{
		Url:     m.URL,
		Headers: m.Headers,
	}
}

func McpServerConfigFromProto(pb *sdktypes.McpServerConfig) *McpServerConfig {
	if pb == nil {
		return nil
	}
	m := &McpServerConfig{}
	switch v := pb.GetConfig().(type) {
	case *sdktypes.McpServerConfig_Stdio:
		m.Stdio = McpStdioServerConfigFromProto(v.Stdio)
	case *sdktypes.McpServerConfig_Sse:
		m.Sse = McpSSEServerConfigFromProto(v.Sse)
	case *sdktypes.McpServerConfig_Http:
		m.Http = McpHttpServerConfigFromProto(v.Http)
	}
	return m
}

func (m *McpServerConfig) ToProto() *sdktypes.McpServerConfig {
	if m == nil {
		return nil
	}
	pb := &sdktypes.McpServerConfig{}
	switch {
	case m.Stdio != nil:
		pb.Config = &sdktypes.McpServerConfig_Stdio{Stdio: m.Stdio.ToProto()}
	case m.Sse != nil:
		pb.Config = &sdktypes.McpServerConfig_Sse{Sse: m.Sse.ToProto()}
	case m.Http != nil:
		pb.Config = &sdktypes.McpServerConfig_Http{Http: m.Http.ToProto()}
	}
	return pb
}

func SdkPluginConfigFromProto(pb *sdktypes.SdkPluginConfig) *SdkPluginConfig {
	if pb == nil {
		return nil
	}
	return &SdkPluginConfig{
		Path: pb.GetPath(),
	}
}

func (m *SdkPluginConfig) ToProto() *sdktypes.SdkPluginConfig {
	if m == nil {
		return nil
	}
	return &sdktypes.SdkPluginConfig{
		Path: m.Path,
	}
}

func sdkPluginConfigsFromProto(pbs []*sdktypes.SdkPluginConfig) []*SdkPluginConfig {
	if pbs == nil {
		return nil
	}
	result := make([]*SdkPluginConfig, len(pbs))
	for i, pb := range pbs {
		result[i] = SdkPluginConfigFromProto(pb)
	}
	return result
}

func sdkPluginConfigsToProto(ms []*SdkPluginConfig) []*sdktypes.SdkPluginConfig {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.SdkPluginConfig, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func AgentDefinitionFromProto(pb *sdktypes.AgentDefinition) *AgentDefinition {
	if pb == nil {
		return nil
	}
	return &AgentDefinition{
		Description: pb.GetDescription(),
		Tools:       pb.GetTools(),
		Prompt:      pb.GetPrompt(),
		Model:       AgentModelFromProto(pb.GetModel()),
	}
}

func (m *AgentDefinition) ToProto() *sdktypes.AgentDefinition {
	if m == nil {
		return nil
	}
	return &sdktypes.AgentDefinition{
		Description: m.Description,
		Tools:       m.Tools,
		Prompt:      m.Prompt,
		Model:       m.Model.ToProto(),
	}
}

func agentDefinitionsMapFromProto(m map[string]*sdktypes.AgentDefinition) map[string]*AgentDefinition {
	if m == nil {
		return nil
	}
	result := make(map[string]*AgentDefinition, len(m))
	for k, v := range m {
		result[k] = AgentDefinitionFromProto(v)
	}
	return result
}

func agentDefinitionsMapToProto(m map[string]*AgentDefinition) map[string]*sdktypes.AgentDefinition {
	if m == nil {
		return nil
	}
	result := make(map[string]*sdktypes.AgentDefinition, len(m))
	for k, v := range m {
		result[k] = v.ToProto()
	}
	return result
}

func SystemPromptPresetFromProto(pb *sdktypes.SystemPromptPreset) *SystemPromptPreset {
	if pb == nil {
		return nil
	}
	return &SystemPromptPreset{
		Append: pb.GetAppend(),
	}
}

func (m *SystemPromptPreset) ToProto() *sdktypes.SystemPromptPreset {
	if m == nil {
		return nil
	}
	return &sdktypes.SystemPromptPreset{
		Append: m.Append,
	}
}

func SystemPromptFromProto(pb *sdktypes.SystemPrompt) *SystemPrompt {
	if pb == nil {
		return nil
	}
	m := &SystemPrompt{}
	switch v := pb.GetPrompt().(type) {
	case *sdktypes.SystemPrompt_Custom:
		s := v.Custom
		m.Custom = &s
	case *sdktypes.SystemPrompt_Preset:
		m.Preset = SystemPromptPresetFromProto(v.Preset)
	}
	return m
}

func (m *SystemPrompt) ToProto() *sdktypes.SystemPrompt {
	if m == nil {
		return nil
	}
	pb := &sdktypes.SystemPrompt{}
	switch {
	case m.Custom != nil:
		pb.Prompt = &sdktypes.SystemPrompt_Custom{Custom: *m.Custom}
	case m.Preset != nil:
		pb.Prompt = &sdktypes.SystemPrompt_Preset{Preset: m.Preset.ToProto()}
	}
	return pb
}

func OutputFormatFromProto(pb *sdktypes.OutputFormat) *OutputFormat {
	if pb == nil {
		return nil
	}
	return &OutputFormat{
		Schema: structFromProto(pb.GetSchema()),
	}
}

func (m *OutputFormat) ToProto() *sdktypes.OutputFormat {
	if m == nil {
		return nil
	}
	return &sdktypes.OutputFormat{
		Schema: structToProto(m.Schema),
	}
}

func sdkBetasFromProto(pbs []sdktypes.SdkBeta) []SdkBeta {
	if pbs == nil {
		return nil
	}
	result := make([]SdkBeta, len(pbs))
	for i, pb := range pbs {
		result[i] = SdkBetaFromProto(pb)
	}
	return result
}

func sdkBetasToProto(ms []SdkBeta) []sdktypes.SdkBeta {
	if ms == nil {
		return nil
	}
	result := make([]sdktypes.SdkBeta, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func settingSourcesFromProto(pbs []sdktypes.SettingSource) []SettingSource {
	if pbs == nil {
		return nil
	}
	result := make([]SettingSource, len(pbs))
	for i, pb := range pbs {
		result[i] = SettingSourceFromProto(pb)
	}
	return result
}

func settingSourcesToProto(ms []SettingSource) []sdktypes.SettingSource {
	if ms == nil {
		return nil
	}
	result := make([]sdktypes.SettingSource, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func mcpServerConfigsMapFromProto(m map[string]*sdktypes.McpServerConfig) map[string]*McpServerConfig {
	if m == nil {
		return nil
	}
	result := make(map[string]*McpServerConfig, len(m))
	for k, v := range m {
		result[k] = McpServerConfigFromProto(v)
	}
	return result
}

func mcpServerConfigsMapToProto(m map[string]*McpServerConfig) map[string]*sdktypes.McpServerConfig {
	if m == nil {
		return nil
	}
	result := make(map[string]*sdktypes.McpServerConfig, len(m))
	for k, v := range m {
		result[k] = v.ToProto()
	}
	return result
}

func SandboxIgnoreViolationsFromProto(pb *sdktypes.SandboxIgnoreViolations) *SandboxIgnoreViolations {
	if pb == nil {
		return nil
	}
	return &SandboxIgnoreViolations{
		File:    pb.GetFile(),
		Network: pb.GetNetwork(),
	}
}

func (m *SandboxIgnoreViolations) ToProto() *sdktypes.SandboxIgnoreViolations {
	if m == nil {
		return nil
	}
	return &sdktypes.SandboxIgnoreViolations{
		File:    m.File,
		Network: m.Network,
	}
}

func NetworkSandboxSettingsFromProto(pb *sdktypes.NetworkSandboxSettings) *NetworkSandboxSettings {
	if pb == nil {
		return nil
	}
	return &NetworkSandboxSettings{
		AllowLocalBinding:   optionalBoolFromProto(pb.AllowLocalBinding),
		AllowUnixSockets:    pb.GetAllowUnixSockets(),
		AllowAllUnixSockets: optionalBoolFromProto(pb.AllowAllUnixSockets),
		HttpProxyPort:       optionalInt32FromProto(pb.HttpProxyPort),
		SocksProxyPort:      optionalInt32FromProto(pb.SocksProxyPort),
	}
}

func (m *NetworkSandboxSettings) ToProto() *sdktypes.NetworkSandboxSettings {
	if m == nil {
		return nil
	}
	return &sdktypes.NetworkSandboxSettings{
		AllowLocalBinding:   optionalBoolToProto(m.AllowLocalBinding),
		AllowUnixSockets:    m.AllowUnixSockets,
		AllowAllUnixSockets: optionalBoolToProto(m.AllowAllUnixSockets),
		HttpProxyPort:       optionalInt32ToProto(m.HttpProxyPort),
		SocksProxyPort:      optionalInt32ToProto(m.SocksProxyPort),
	}
}

func SandboxSettingsFromProto(pb *sdktypes.SandboxSettings) *SandboxSettings {
	if pb == nil {
		return nil
	}
	return &SandboxSettings{
		Enabled:                   optionalBoolFromProto(pb.Enabled),
		AutoAllowBashIfSandboxed:  optionalBoolFromProto(pb.AutoAllowBashIfSandboxed),
		ExcludedCommands:          pb.GetExcludedCommands(),
		AllowUnsandboxedCommands:  optionalBoolFromProto(pb.AllowUnsandboxedCommands),
		Network:                   NetworkSandboxSettingsFromProto(pb.GetNetwork()),
		IgnoreViolations:          SandboxIgnoreViolationsFromProto(pb.GetIgnoreViolations()),
		EnableWeakerNestedSandbox: optionalBoolFromProto(pb.EnableWeakerNestedSandbox),
	}
}

func (m *SandboxSettings) ToProto() *sdktypes.SandboxSettings {
	if m == nil {
		return nil
	}
	return &sdktypes.SandboxSettings{
		Enabled:                   optionalBoolToProto(m.Enabled),
		AutoAllowBashIfSandboxed:  optionalBoolToProto(m.AutoAllowBashIfSandboxed),
		ExcludedCommands:          m.ExcludedCommands,
		AllowUnsandboxedCommands:  optionalBoolToProto(m.AllowUnsandboxedCommands),
		Network:                   m.Network.ToProto(),
		IgnoreViolations:          m.IgnoreViolations.ToProto(),
		EnableWeakerNestedSandbox: optionalBoolToProto(m.EnableWeakerNestedSandbox),
	}
}

func OptionsFromProto(pb *sdktypes.Options) *Options {
	if pb == nil {
		return nil
	}
	return &Options{
		AdditionalDirectories:           pb.GetAdditionalDirectories(),
		Agents:                          agentDefinitionsMapFromProto(pb.GetAgents()),
		AllowDangerouslySkipPermissions: pb.GetAllowDangerouslySkipPermissions(),
		AllowedTools:                    pb.GetAllowedTools(),
		Betas:                           sdkBetasFromProto(pb.GetBetas()),
		Continue:                        pb.GetContinue(),
		Cwd:                             pb.GetCwd(),
		DisallowedTools:                 pb.GetDisallowedTools(),
		EnableFileCheckpointing:         pb.GetEnableFileCheckpointing(),
		Env:                             pb.GetEnv(),
		Executable:                      pb.GetExecutable(),
		ExecutableArgs:                  pb.GetExecutableArgs(),
		ExtraArgs:                       pb.GetExtraArgs(),
		FallbackModel:                   pb.GetFallbackModel(),
		ForkSession:                     pb.GetForkSession(),
		IncludePartialMessages:          pb.GetIncludePartialMessages(),
		MaxBudgetUsd:                    optionalFloat64FromProto(pb.MaxBudgetUsd),
		MaxThinkingTokens:               optionalInt32FromProto(pb.MaxThinkingTokens),
		MaxTurns:                        optionalInt32FromProto(pb.MaxTurns),
		McpServers:                      mcpServerConfigsMapFromProto(pb.GetMcpServers()),
		Model:                           pb.GetModel(),
		OutputFormat:                    OutputFormatFromProto(pb.GetOutputFormat()),
		PathToClaudeCodeExecutable:      pb.GetPathToClaudeCodeExecutable(),
		PermissionMode:                  PermissionModeFromProto(pb.GetPermissionMode()),
		PermissionPromptToolName:        pb.GetPermissionPromptToolName(),
		Plugins:                         sdkPluginConfigsFromProto(pb.GetPlugins()),
		Resume:                          pb.GetResume(),
		ResumeSessionAt:                 pb.GetResumeSessionAt(),
		Sandbox:                         SandboxSettingsFromProto(pb.GetSandbox()),
		SettingSources:                  settingSourcesFromProto(pb.GetSettingSources()),
		StrictMcpConfig:                 pb.GetStrictMcpConfig(),
		SystemPrompt:                    SystemPromptFromProto(pb.GetSystemPrompt()),
		ToolsList:                       pb.GetToolsList(),
		ToolsPreset:                     pb.GetToolsPreset(),
	}
}

func (m *Options) ToProto() *sdktypes.Options {
	if m == nil {
		return nil
	}
	return &sdktypes.Options{
		AdditionalDirectories:           m.AdditionalDirectories,
		Agents:                          agentDefinitionsMapToProto(m.Agents),
		AllowDangerouslySkipPermissions: m.AllowDangerouslySkipPermissions,
		AllowedTools:                    m.AllowedTools,
		Betas:                           sdkBetasToProto(m.Betas),
		Continue:                        m.Continue,
		Cwd:                             m.Cwd,
		DisallowedTools:                 m.DisallowedTools,
		EnableFileCheckpointing:         m.EnableFileCheckpointing,
		Env:                             m.Env,
		Executable:                      m.Executable,
		ExecutableArgs:                  m.ExecutableArgs,
		ExtraArgs:                       m.ExtraArgs,
		FallbackModel:                   m.FallbackModel,
		ForkSession:                     m.ForkSession,
		IncludePartialMessages:          m.IncludePartialMessages,
		MaxBudgetUsd:                    optionalFloat64ToProto(m.MaxBudgetUsd),
		MaxThinkingTokens:               optionalInt32ToProto(m.MaxThinkingTokens),
		MaxTurns:                        optionalInt32ToProto(m.MaxTurns),
		McpServers:                      mcpServerConfigsMapToProto(m.McpServers),
		Model:                           m.Model,
		OutputFormat:                    m.OutputFormat.ToProto(),
		PathToClaudeCodeExecutable:      m.PathToClaudeCodeExecutable,
		PermissionMode:                  m.PermissionMode.ToProto(),
		PermissionPromptToolName:        m.PermissionPromptToolName,
		Plugins:                         sdkPluginConfigsToProto(m.Plugins),
		Resume:                          m.Resume,
		ResumeSessionAt:                 m.ResumeSessionAt,
		Sandbox:                         m.Sandbox.ToProto(),
		SettingSources:                  settingSourcesToProto(m.SettingSources),
		StrictMcpConfig:                 m.StrictMcpConfig,
		SystemPrompt:                    m.SystemPrompt.ToProto(),
		ToolsList:                       m.ToolsList,
		ToolsPreset:                     m.ToolsPreset,
	}
}
