package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func SlashCommandFromProto(pb *sdktypes.SlashCommand) *SlashCommand {
	if pb == nil {
		return nil
	}
	return &SlashCommand{
		Name:         pb.GetName(),
		Description:  pb.GetDescription(),
		ArgumentHint: pb.GetArgumentHint(),
	}
}

func (m *SlashCommand) ToProto() *sdktypes.SlashCommand {
	if m == nil {
		return nil
	}
	return &sdktypes.SlashCommand{
		Name:         m.Name,
		Description:  m.Description,
		ArgumentHint: m.ArgumentHint,
	}
}

func ModelInfoFromProto(pb *sdktypes.ModelInfo) *ModelInfo {
	if pb == nil {
		return nil
	}
	return &ModelInfo{
		Value:       pb.GetValue(),
		DisplayName: pb.GetDisplayName(),
		Description: pb.GetDescription(),
	}
}

func (m *ModelInfo) ToProto() *sdktypes.ModelInfo {
	if m == nil {
		return nil
	}
	return &sdktypes.ModelInfo{
		Value:       m.Value,
		DisplayName: m.DisplayName,
		Description: m.Description,
	}
}

func McpServerInfoFromProto(pb *sdktypes.McpServerInfo) *McpServerInfo {
	if pb == nil {
		return nil
	}
	return &McpServerInfo{
		Name:    pb.GetName(),
		Version: pb.GetVersion(),
	}
}

func (m *McpServerInfo) ToProto() *sdktypes.McpServerInfo {
	if m == nil {
		return nil
	}
	return &sdktypes.McpServerInfo{
		Name:    m.Name,
		Version: m.Version,
	}
}

func McpServerStatusFromProto(pb *sdktypes.McpServerStatus) *McpServerStatus {
	if pb == nil {
		return nil
	}
	return &McpServerStatus{
		Name:       pb.GetName(),
		Status:     McpServerConnectionStatusFromProto(pb.GetStatus()),
		ServerInfo: McpServerInfoFromProto(pb.GetServerInfo()),
	}
}

func (m *McpServerStatus) ToProto() *sdktypes.McpServerStatus {
	if m == nil {
		return nil
	}
	return &sdktypes.McpServerStatus{
		Name:       m.Name,
		Status:     m.Status.ToProto(),
		ServerInfo: m.ServerInfo.ToProto(),
	}
}

func AccountInfoFromProto(pb *sdktypes.AccountInfo) *AccountInfo {
	if pb == nil {
		return nil
	}
	return &AccountInfo{
		Email:            optionalStringFromProto(pb.Email),
		Organization:     optionalStringFromProto(pb.Organization),
		SubscriptionType: optionalStringFromProto(pb.SubscriptionType),
		TokenSource:      optionalStringFromProto(pb.TokenSource),
		ApiKeySource:     optionalStringFromProto(pb.ApiKeySource),
	}
}

func (m *AccountInfo) ToProto() *sdktypes.AccountInfo {
	if m == nil {
		return nil
	}
	return &sdktypes.AccountInfo{
		Email:            optionalStringToProto(m.Email),
		Organization:     optionalStringToProto(m.Organization),
		SubscriptionType: optionalStringToProto(m.SubscriptionType),
		TokenSource:      optionalStringToProto(m.TokenSource),
		ApiKeySource:     optionalStringToProto(m.ApiKeySource),
	}
}

func ModelUsageFromProto(pb *sdktypes.ModelUsage) *ModelUsage {
	if pb == nil {
		return nil
	}
	return &ModelUsage{
		InputTokens:              pb.GetInputTokens(),
		OutputTokens:             pb.GetOutputTokens(),
		CacheReadInputTokens:     pb.GetCacheReadInputTokens(),
		CacheCreationInputTokens: pb.GetCacheCreationInputTokens(),
		WebSearchRequests:        pb.GetWebSearchRequests(),
		CostUsd:                  pb.GetCostUsd(),
		ContextWindow:            pb.GetContextWindow(),
	}
}

func (m *ModelUsage) ToProto() *sdktypes.ModelUsage {
	if m == nil {
		return nil
	}
	return &sdktypes.ModelUsage{
		InputTokens:              m.InputTokens,
		OutputTokens:             m.OutputTokens,
		CacheReadInputTokens:     m.CacheReadInputTokens,
		CacheCreationInputTokens: m.CacheCreationInputTokens,
		WebSearchRequests:        m.WebSearchRequests,
		CostUsd:                  m.CostUsd,
		ContextWindow:            m.ContextWindow,
	}
}

func NonNullableUsageFromProto(pb *sdktypes.NonNullableUsage) *NonNullableUsage {
	if pb == nil {
		return nil
	}
	return &NonNullableUsage{
		InputTokens:              pb.GetInputTokens(),
		OutputTokens:             pb.GetOutputTokens(),
		CacheCreationInputTokens: optionalInt64FromProto(pb.CacheCreationInputTokens),
		CacheReadInputTokens:     optionalInt64FromProto(pb.CacheReadInputTokens),
	}
}

func (m *NonNullableUsage) ToProto() *sdktypes.NonNullableUsage {
	if m == nil {
		return nil
	}
	return &sdktypes.NonNullableUsage{
		InputTokens:              m.InputTokens,
		OutputTokens:             m.OutputTokens,
		CacheCreationInputTokens: optionalInt64ToProto(m.CacheCreationInputTokens),
		CacheReadInputTokens:     optionalInt64ToProto(m.CacheReadInputTokens),
	}
}

func UsageFromProto(pb *sdktypes.Usage) *Usage {
	if pb == nil {
		return nil
	}
	return &Usage{
		InputTokens:              optionalInt64FromProto(pb.InputTokens),
		OutputTokens:             optionalInt64FromProto(pb.OutputTokens),
		CacheCreationInputTokens: optionalInt64FromProto(pb.CacheCreationInputTokens),
		CacheReadInputTokens:     optionalInt64FromProto(pb.CacheReadInputTokens),
	}
}

func (m *Usage) ToProto() *sdktypes.Usage {
	if m == nil {
		return nil
	}
	return &sdktypes.Usage{
		InputTokens:              optionalInt64ToProto(m.InputTokens),
		OutputTokens:             optionalInt64ToProto(m.OutputTokens),
		CacheCreationInputTokens: optionalInt64ToProto(m.CacheCreationInputTokens),
		CacheReadInputTokens:     optionalInt64ToProto(m.CacheReadInputTokens),
	}
}

func CallToolResultContentFromProto(pb *sdktypes.CallToolResultContent) *CallToolResultContent {
	if pb == nil {
		return nil
	}
	return &CallToolResultContent{
		Type: CallToolResultContentTypeFromProto(pb.GetType()),
		Data: structFromProto(pb.GetData()),
	}
}

func (m *CallToolResultContent) ToProto() *sdktypes.CallToolResultContent {
	if m == nil {
		return nil
	}
	return &sdktypes.CallToolResultContent{
		Type: m.Type.ToProto(),
		Data: structToProto(m.Data),
	}
}

func callToolResultContentsFromProto(pbs []*sdktypes.CallToolResultContent) []*CallToolResultContent {
	if pbs == nil {
		return nil
	}
	result := make([]*CallToolResultContent, len(pbs))
	for i, pb := range pbs {
		result[i] = CallToolResultContentFromProto(pb)
	}
	return result
}

func callToolResultContentsToProto(ms []*CallToolResultContent) []*sdktypes.CallToolResultContent {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.CallToolResultContent, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func CallToolResultFromProto(pb *sdktypes.CallToolResult) *CallToolResult {
	if pb == nil {
		return nil
	}
	return &CallToolResult{
		Content: callToolResultContentsFromProto(pb.GetContent()),
		IsError: optionalBoolFromProto(pb.IsError),
	}
}

func (m *CallToolResult) ToProto() *sdktypes.CallToolResult {
	if m == nil {
		return nil
	}
	return &sdktypes.CallToolResult{
		Content: callToolResultContentsToProto(m.Content),
		IsError: optionalBoolToProto(m.IsError),
	}
}
