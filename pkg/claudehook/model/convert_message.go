package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func SDKAssistantMessageFromProto(pb *sdktypes.SDKAssistantMessage) *SDKAssistantMessage {
	if pb == nil {
		return nil
	}
	return &SDKAssistantMessage{
		UUID:            pb.GetUuid(),
		SessionID:       pb.GetSessionId(),
		Message:         structFromProto(pb.GetMessage()),
		ParentToolUseID: optionalStringFromProto(pb.ParentToolUseId),
	}
}

func (m *SDKAssistantMessage) ToProto() *sdktypes.SDKAssistantMessage {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKAssistantMessage{
		Uuid:            m.UUID,
		SessionId:       m.SessionID,
		Message:         structToProto(m.Message),
		ParentToolUseId: optionalStringToProto(m.ParentToolUseID),
	}
}

func SDKUserMessageFromProto(pb *sdktypes.SDKUserMessage) *SDKUserMessage {
	if pb == nil {
		return nil
	}
	return &SDKUserMessage{
		UUID:            optionalStringFromProto(pb.Uuid),
		SessionID:       pb.GetSessionId(),
		Message:         structFromProto(pb.GetMessage()),
		ParentToolUseID: optionalStringFromProto(pb.ParentToolUseId),
	}
}

func (m *SDKUserMessage) ToProto() *sdktypes.SDKUserMessage {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKUserMessage{
		Uuid:            optionalStringToProto(m.UUID),
		SessionId:       m.SessionID,
		Message:         structToProto(m.Message),
		ParentToolUseId: optionalStringToProto(m.ParentToolUseID),
	}
}

func SDKUserMessageReplayFromProto(pb *sdktypes.SDKUserMessageReplay) *SDKUserMessageReplay {
	if pb == nil {
		return nil
	}
	return &SDKUserMessageReplay{
		UUID:            pb.GetUuid(),
		SessionID:       pb.GetSessionId(),
		Message:         structFromProto(pb.GetMessage()),
		ParentToolUseID: optionalStringFromProto(pb.ParentToolUseId),
	}
}

func (m *SDKUserMessageReplay) ToProto() *sdktypes.SDKUserMessageReplay {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKUserMessageReplay{
		Uuid:            m.UUID,
		SessionId:       m.SessionID,
		Message:         structToProto(m.Message),
		ParentToolUseId: optionalStringToProto(m.ParentToolUseID),
	}
}

func SDKPermissionDenialFromProto(pb *sdktypes.SDKPermissionDenial) *SDKPermissionDenial {
	if pb == nil {
		return nil
	}
	return &SDKPermissionDenial{
		ToolName:  pb.GetToolName(),
		ToolUseID: pb.GetToolUseId(),
		ToolInput: ToolInputFromProto(pb.GetToolInput()),
	}
}

func (m *SDKPermissionDenial) ToProto() *sdktypes.SDKPermissionDenial {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKPermissionDenial{
		ToolName:  m.ToolName,
		ToolUseId: m.ToolUseID,
		ToolInput: m.ToolInput.ToProto(),
	}
}

func sdkPermissionDenialsFromProto(pbs []*sdktypes.SDKPermissionDenial) []*SDKPermissionDenial {
	if pbs == nil {
		return nil
	}
	result := make([]*SDKPermissionDenial, len(pbs))
	for i, pb := range pbs {
		result[i] = SDKPermissionDenialFromProto(pb)
	}
	return result
}

func sdkPermissionDenialsToProto(ms []*SDKPermissionDenial) []*sdktypes.SDKPermissionDenial {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.SDKPermissionDenial, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func modelUsageMapFromProto(m map[string]*sdktypes.ModelUsage) map[string]*ModelUsage {
	if m == nil {
		return nil
	}
	result := make(map[string]*ModelUsage, len(m))
	for k, v := range m {
		result[k] = ModelUsageFromProto(v)
	}
	return result
}

func modelUsageMapToProto(m map[string]*ModelUsage) map[string]*sdktypes.ModelUsage {
	if m == nil {
		return nil
	}
	result := make(map[string]*sdktypes.ModelUsage, len(m))
	for k, v := range m {
		result[k] = v.ToProto()
	}
	return result
}

func SDKResultMessageFromProto(pb *sdktypes.SDKResultMessage) *SDKResultMessage {
	if pb == nil {
		return nil
	}
	return &SDKResultMessage{
		Subtype:           SDKResultSubtypeFromProto(pb.GetSubtype()),
		UUID:              pb.GetUuid(),
		SessionID:         pb.GetSessionId(),
		DurationMs:        pb.GetDurationMs(),
		DurationApiMs:     pb.GetDurationApiMs(),
		IsError:           pb.GetIsError(),
		NumTurns:          pb.GetNumTurns(),
		TotalCostUsd:      pb.GetTotalCostUsd(),
		Usage:             NonNullableUsageFromProto(pb.GetUsage()),
		ModelUsage:        modelUsageMapFromProto(pb.GetModelUsage()),
		PermissionDenials: sdkPermissionDenialsFromProto(pb.GetPermissionDenials()),
		Result:            optionalStringFromProto(pb.Result),
		StructuredOutput:  valueFromProto(pb.GetStructuredOutput()),
		Errors:            pb.GetErrors(),
	}
}

func (m *SDKResultMessage) ToProto() *sdktypes.SDKResultMessage {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKResultMessage{
		Subtype:           m.Subtype.ToProto(),
		Uuid:              m.UUID,
		SessionId:         m.SessionID,
		DurationMs:        m.DurationMs,
		DurationApiMs:     m.DurationApiMs,
		IsError:           m.IsError,
		NumTurns:          m.NumTurns,
		TotalCostUsd:      m.TotalCostUsd,
		Usage:             m.Usage.ToProto(),
		ModelUsage:        modelUsageMapToProto(m.ModelUsage),
		PermissionDenials: sdkPermissionDenialsToProto(m.PermissionDenials),
		Result:            optionalStringToProto(m.Result),
		StructuredOutput:  valueToProto(m.StructuredOutput),
		Errors:            m.Errors,
	}
}

func SDKSystemMcpServerFromProto(pb *sdktypes.SDKSystemMcpServer) *SDKSystemMcpServer {
	if pb == nil {
		return nil
	}
	return &SDKSystemMcpServer{
		Name:   pb.GetName(),
		Status: pb.GetStatus(),
	}
}

func (m *SDKSystemMcpServer) ToProto() *sdktypes.SDKSystemMcpServer {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKSystemMcpServer{
		Name:   m.Name,
		Status: m.Status,
	}
}

func sdkSystemMcpServersFromProto(pbs []*sdktypes.SDKSystemMcpServer) []*SDKSystemMcpServer {
	if pbs == nil {
		return nil
	}
	result := make([]*SDKSystemMcpServer, len(pbs))
	for i, pb := range pbs {
		result[i] = SDKSystemMcpServerFromProto(pb)
	}
	return result
}

func sdkSystemMcpServersToProto(ms []*SDKSystemMcpServer) []*sdktypes.SDKSystemMcpServer {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.SDKSystemMcpServer, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func SDKSystemMessageFromProto(pb *sdktypes.SDKSystemMessage) *SDKSystemMessage {
	if pb == nil {
		return nil
	}
	return &SDKSystemMessage{
		UUID:           pb.GetUuid(),
		SessionID:      pb.GetSessionId(),
		ApiKeySource:   ApiKeySourceFromProto(pb.GetApiKeySource()),
		Cwd:            pb.GetCwd(),
		Tools:          pb.GetTools(),
		McpServers:     sdkSystemMcpServersFromProto(pb.GetMcpServers()),
		Model:          pb.GetModel(),
		PermissionMode: PermissionModeFromProto(pb.GetPermissionMode()),
		SlashCommands:  pb.GetSlashCommands(),
		OutputStyle:    pb.GetOutputStyle(),
	}
}

func (m *SDKSystemMessage) ToProto() *sdktypes.SDKSystemMessage {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKSystemMessage{
		Uuid:           m.UUID,
		SessionId:      m.SessionID,
		ApiKeySource:   m.ApiKeySource.ToProto(),
		Cwd:            m.Cwd,
		Tools:          m.Tools,
		McpServers:     sdkSystemMcpServersToProto(m.McpServers),
		Model:          m.Model,
		PermissionMode: m.PermissionMode.ToProto(),
		SlashCommands:  m.SlashCommands,
		OutputStyle:    m.OutputStyle,
	}
}

func SDKPartialAssistantMessageFromProto(pb *sdktypes.SDKPartialAssistantMessage) *SDKPartialAssistantMessage {
	if pb == nil {
		return nil
	}
	return &SDKPartialAssistantMessage{
		UUID:            pb.GetUuid(),
		SessionID:       pb.GetSessionId(),
		Event:           structFromProto(pb.GetEvent()),
		ParentToolUseID: optionalStringFromProto(pb.ParentToolUseId),
	}
}

func (m *SDKPartialAssistantMessage) ToProto() *sdktypes.SDKPartialAssistantMessage {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKPartialAssistantMessage{
		Uuid:            m.UUID,
		SessionId:       m.SessionID,
		Event:           structToProto(m.Event),
		ParentToolUseId: optionalStringToProto(m.ParentToolUseID),
	}
}

func CompactMetadataFromProto(pb *sdktypes.CompactMetadata) *CompactMetadata {
	if pb == nil {
		return nil
	}
	return &CompactMetadata{
		Trigger:   pb.GetTrigger(),
		PreTokens: pb.GetPreTokens(),
	}
}

func (m *CompactMetadata) ToProto() *sdktypes.CompactMetadata {
	if m == nil {
		return nil
	}
	return &sdktypes.CompactMetadata{
		Trigger:   m.Trigger,
		PreTokens: m.PreTokens,
	}
}

func SDKCompactBoundaryMessageFromProto(pb *sdktypes.SDKCompactBoundaryMessage) *SDKCompactBoundaryMessage {
	if pb == nil {
		return nil
	}
	return &SDKCompactBoundaryMessage{
		UUID:            pb.GetUuid(),
		SessionID:       pb.GetSessionId(),
		CompactMetadata: CompactMetadataFromProto(pb.GetCompactMetadata()),
	}
}

func (m *SDKCompactBoundaryMessage) ToProto() *sdktypes.SDKCompactBoundaryMessage {
	if m == nil {
		return nil
	}
	return &sdktypes.SDKCompactBoundaryMessage{
		Uuid:            m.UUID,
		SessionId:       m.SessionID,
		CompactMetadata: m.CompactMetadata.ToProto(),
	}
}

func SDKMessageFromProto(pb *sdktypes.SDKMessage) *SDKMessage {
	if pb == nil {
		return nil
	}
	m := &SDKMessage{}
	switch v := pb.GetMessage().(type) {
	case *sdktypes.SDKMessage_Assistant:
		m.Assistant = SDKAssistantMessageFromProto(v.Assistant)
	case *sdktypes.SDKMessage_User:
		m.User = SDKUserMessageFromProto(v.User)
	case *sdktypes.SDKMessage_UserReplay:
		m.UserReplay = SDKUserMessageReplayFromProto(v.UserReplay)
	case *sdktypes.SDKMessage_Result:
		m.Result = SDKResultMessageFromProto(v.Result)
	case *sdktypes.SDKMessage_System:
		m.System = SDKSystemMessageFromProto(v.System)
	case *sdktypes.SDKMessage_Partial:
		m.Partial = SDKPartialAssistantMessageFromProto(v.Partial)
	case *sdktypes.SDKMessage_CompactBoundary:
		m.CompactBoundary = SDKCompactBoundaryMessageFromProto(v.CompactBoundary)
	}
	return m
}

func (m *SDKMessage) ToProto() *sdktypes.SDKMessage {
	if m == nil {
		return nil
	}
	pb := &sdktypes.SDKMessage{}
	switch {
	case m.Assistant != nil:
		pb.Message = &sdktypes.SDKMessage_Assistant{Assistant: m.Assistant.ToProto()}
	case m.User != nil:
		pb.Message = &sdktypes.SDKMessage_User{User: m.User.ToProto()}
	case m.UserReplay != nil:
		pb.Message = &sdktypes.SDKMessage_UserReplay{UserReplay: m.UserReplay.ToProto()}
	case m.Result != nil:
		pb.Message = &sdktypes.SDKMessage_Result{Result: m.Result.ToProto()}
	case m.System != nil:
		pb.Message = &sdktypes.SDKMessage_System{System: m.System.ToProto()}
	case m.Partial != nil:
		pb.Message = &sdktypes.SDKMessage_Partial{Partial: m.Partial.ToProto()}
	case m.CompactBoundary != nil:
		pb.Message = &sdktypes.SDKMessage_CompactBoundary{CompactBoundary: m.CompactBoundary.ToProto()}
	}
	return pb
}
