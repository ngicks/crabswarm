package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func PreToolUseHookInputFromProto(pb *sdktypes.PreToolUseHookInput) *PreToolUseHookInput {
	if pb == nil {
		return nil
	}
	return &PreToolUseHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		ToolName:       pb.GetToolName(),
		ToolInput:      ToolInputFromProto(pb.GetToolInput()),
	}
}

func (m *PreToolUseHookInput) ToProto() *sdktypes.PreToolUseHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.PreToolUseHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		ToolName:       m.ToolName,
		ToolInput:      m.ToolInput.ToProto(),
	}
}

func PostToolUseHookInputFromProto(pb *sdktypes.PostToolUseHookInput) *PostToolUseHookInput {
	if pb == nil {
		return nil
	}
	return &PostToolUseHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		ToolName:       pb.GetToolName(),
		ToolInput:      ToolInputFromProto(pb.GetToolInput()),
		ToolResponse:   ToolOutputFromProto(pb.GetToolResponse()),
	}
}

func (m *PostToolUseHookInput) ToProto() *sdktypes.PostToolUseHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.PostToolUseHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		ToolName:       m.ToolName,
		ToolInput:      m.ToolInput.ToProto(),
		ToolResponse:   m.ToolResponse.ToProto(),
	}
}

func PostToolUseFailureHookInputFromProto(pb *sdktypes.PostToolUseFailureHookInput) *PostToolUseFailureHookInput {
	if pb == nil {
		return nil
	}
	return &PostToolUseFailureHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		ToolName:       pb.GetToolName(),
		ToolInput:      ToolInputFromProto(pb.GetToolInput()),
		Error:          pb.GetError(),
		IsInterrupt:    optionalBoolFromProto(pb.IsInterrupt),
	}
}

func (m *PostToolUseFailureHookInput) ToProto() *sdktypes.PostToolUseFailureHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.PostToolUseFailureHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		ToolName:       m.ToolName,
		ToolInput:      m.ToolInput.ToProto(),
		Error:          m.Error,
		IsInterrupt:    optionalBoolToProto(m.IsInterrupt),
	}
}

func NotificationHookInputFromProto(pb *sdktypes.NotificationHookInput) *NotificationHookInput {
	if pb == nil {
		return nil
	}
	return &NotificationHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		Message:        pb.GetMessage(),
		Title:          optionalStringFromProto(pb.Title),
	}
}

func (m *NotificationHookInput) ToProto() *sdktypes.NotificationHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.NotificationHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		Message:        m.Message,
		Title:          optionalStringToProto(m.Title),
	}
}

func UserPromptSubmitHookInputFromProto(pb *sdktypes.UserPromptSubmitHookInput) *UserPromptSubmitHookInput {
	if pb == nil {
		return nil
	}
	return &UserPromptSubmitHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		Prompt:         pb.GetPrompt(),
	}
}

func (m *UserPromptSubmitHookInput) ToProto() *sdktypes.UserPromptSubmitHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.UserPromptSubmitHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		Prompt:         m.Prompt,
	}
}

func SessionStartHookInputFromProto(pb *sdktypes.SessionStartHookInput) *SessionStartHookInput {
	if pb == nil {
		return nil
	}
	return &SessionStartHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		Source:         SessionStartSourceFromProto(pb.GetSource()),
	}
}

func (m *SessionStartHookInput) ToProto() *sdktypes.SessionStartHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.SessionStartHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		Source:         m.Source.ToProto(),
	}
}

func SessionEndHookInputFromProto(pb *sdktypes.SessionEndHookInput) *SessionEndHookInput {
	if pb == nil {
		return nil
	}
	return &SessionEndHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		Reason:         pb.GetReason(),
	}
}

func (m *SessionEndHookInput) ToProto() *sdktypes.SessionEndHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.SessionEndHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		Reason:         m.Reason,
	}
}

func StopHookInputFromProto(pb *sdktypes.StopHookInput) *StopHookInput {
	if pb == nil {
		return nil
	}
	return &StopHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		StopHookActive: pb.GetStopHookActive(),
	}
}

func (m *StopHookInput) ToProto() *sdktypes.StopHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.StopHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		StopHookActive: m.StopHookActive,
	}
}

func SubagentStartHookInputFromProto(pb *sdktypes.SubagentStartHookInput) *SubagentStartHookInput {
	if pb == nil {
		return nil
	}
	return &SubagentStartHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		AgentID:        pb.GetAgentId(),
		AgentType:      pb.GetAgentType(),
	}
}

func (m *SubagentStartHookInput) ToProto() *sdktypes.SubagentStartHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.SubagentStartHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		AgentId:        m.AgentID,
		AgentType:      m.AgentType,
	}
}

func SubagentStopHookInputFromProto(pb *sdktypes.SubagentStopHookInput) *SubagentStopHookInput {
	if pb == nil {
		return nil
	}
	return &SubagentStopHookInput{
		SessionID:      pb.GetSessionId(),
		TranscriptPath: pb.GetTranscriptPath(),
		Cwd:            pb.GetCwd(),
		PermissionMode: optionalStringFromProto(pb.PermissionMode),
		StopHookActive: pb.GetStopHookActive(),
	}
}

func (m *SubagentStopHookInput) ToProto() *sdktypes.SubagentStopHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.SubagentStopHookInput{
		SessionId:      m.SessionID,
		TranscriptPath: m.TranscriptPath,
		Cwd:            m.Cwd,
		PermissionMode: optionalStringToProto(m.PermissionMode),
		StopHookActive: m.StopHookActive,
	}
}

func PreCompactHookInputFromProto(pb *sdktypes.PreCompactHookInput) *PreCompactHookInput {
	if pb == nil {
		return nil
	}
	return &PreCompactHookInput{
		SessionID:          pb.GetSessionId(),
		TranscriptPath:     pb.GetTranscriptPath(),
		Cwd:                pb.GetCwd(),
		PermissionMode:     optionalStringFromProto(pb.PermissionMode),
		Trigger:            PreCompactTriggerFromProto(pb.GetTrigger()),
		CustomInstructions: optionalStringFromProto(pb.CustomInstructions),
	}
}

func (m *PreCompactHookInput) ToProto() *sdktypes.PreCompactHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.PreCompactHookInput{
		SessionId:          m.SessionID,
		TranscriptPath:     m.TranscriptPath,
		Cwd:                m.Cwd,
		PermissionMode:     optionalStringToProto(m.PermissionMode),
		Trigger:            m.Trigger.ToProto(),
		CustomInstructions: optionalStringToProto(m.CustomInstructions),
	}
}

func PermissionRequestHookInputFromProto(pb *sdktypes.PermissionRequestHookInput) *PermissionRequestHookInput {
	if pb == nil {
		return nil
	}
	return &PermissionRequestHookInput{
		SessionID:             pb.GetSessionId(),
		TranscriptPath:        pb.GetTranscriptPath(),
		Cwd:                   pb.GetCwd(),
		PermissionMode:        optionalStringFromProto(pb.PermissionMode),
		ToolName:              pb.GetToolName(),
		ToolInput:             ToolInputFromProto(pb.GetToolInput()),
		PermissionSuggestions: permissionUpdatesFromProto(pb.GetPermissionSuggestions()),
	}
}

func (m *PermissionRequestHookInput) ToProto() *sdktypes.PermissionRequestHookInput {
	if m == nil {
		return nil
	}
	return &sdktypes.PermissionRequestHookInput{
		SessionId:             m.SessionID,
		TranscriptPath:        m.TranscriptPath,
		Cwd:                   m.Cwd,
		PermissionMode:        optionalStringToProto(m.PermissionMode),
		ToolName:              m.ToolName,
		ToolInput:             m.ToolInput.ToProto(),
		PermissionSuggestions: permissionUpdatesToProto(m.PermissionSuggestions),
	}
}

func HookInputFromProto(pb *sdktypes.HookInput) *HookInput {
	if pb == nil {
		return nil
	}
	m := &HookInput{}
	switch v := pb.GetInput().(type) {
	case *sdktypes.HookInput_PreToolUse:
		m.PreToolUse = PreToolUseHookInputFromProto(v.PreToolUse)
	case *sdktypes.HookInput_PostToolUse:
		m.PostToolUse = PostToolUseHookInputFromProto(v.PostToolUse)
	case *sdktypes.HookInput_PostToolUseFailure:
		m.PostToolUseFailure = PostToolUseFailureHookInputFromProto(v.PostToolUseFailure)
	case *sdktypes.HookInput_Notification:
		m.Notification = NotificationHookInputFromProto(v.Notification)
	case *sdktypes.HookInput_UserPromptSubmit:
		m.UserPromptSubmit = UserPromptSubmitHookInputFromProto(v.UserPromptSubmit)
	case *sdktypes.HookInput_SessionStart:
		m.SessionStart = SessionStartHookInputFromProto(v.SessionStart)
	case *sdktypes.HookInput_SessionEnd:
		m.SessionEnd = SessionEndHookInputFromProto(v.SessionEnd)
	case *sdktypes.HookInput_Stop:
		m.Stop = StopHookInputFromProto(v.Stop)
	case *sdktypes.HookInput_SubagentStart:
		m.SubagentStart = SubagentStartHookInputFromProto(v.SubagentStart)
	case *sdktypes.HookInput_SubagentStop:
		m.SubagentStop = SubagentStopHookInputFromProto(v.SubagentStop)
	case *sdktypes.HookInput_PreCompact:
		m.PreCompact = PreCompactHookInputFromProto(v.PreCompact)
	case *sdktypes.HookInput_PermissionRequest:
		m.PermissionRequest = PermissionRequestHookInputFromProto(v.PermissionRequest)
	}
	return m
}

func (m *HookInput) ToProto() *sdktypes.HookInput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.HookInput{}
	switch {
	case m.PreToolUse != nil:
		pb.Input = &sdktypes.HookInput_PreToolUse{PreToolUse: m.PreToolUse.ToProto()}
	case m.PostToolUse != nil:
		pb.Input = &sdktypes.HookInput_PostToolUse{PostToolUse: m.PostToolUse.ToProto()}
	case m.PostToolUseFailure != nil:
		pb.Input = &sdktypes.HookInput_PostToolUseFailure{PostToolUseFailure: m.PostToolUseFailure.ToProto()}
	case m.Notification != nil:
		pb.Input = &sdktypes.HookInput_Notification{Notification: m.Notification.ToProto()}
	case m.UserPromptSubmit != nil:
		pb.Input = &sdktypes.HookInput_UserPromptSubmit{UserPromptSubmit: m.UserPromptSubmit.ToProto()}
	case m.SessionStart != nil:
		pb.Input = &sdktypes.HookInput_SessionStart{SessionStart: m.SessionStart.ToProto()}
	case m.SessionEnd != nil:
		pb.Input = &sdktypes.HookInput_SessionEnd{SessionEnd: m.SessionEnd.ToProto()}
	case m.Stop != nil:
		pb.Input = &sdktypes.HookInput_Stop{Stop: m.Stop.ToProto()}
	case m.SubagentStart != nil:
		pb.Input = &sdktypes.HookInput_SubagentStart{SubagentStart: m.SubagentStart.ToProto()}
	case m.SubagentStop != nil:
		pb.Input = &sdktypes.HookInput_SubagentStop{SubagentStop: m.SubagentStop.ToProto()}
	case m.PreCompact != nil:
		pb.Input = &sdktypes.HookInput_PreCompact{PreCompact: m.PreCompact.ToProto()}
	case m.PermissionRequest != nil:
		pb.Input = &sdktypes.HookInput_PermissionRequest{PermissionRequest: m.PermissionRequest.ToProto()}
	}
	return pb
}

func AsyncHookJSONOutputFromProto(pb *sdktypes.AsyncHookJSONOutput) *AsyncHookJSONOutput {
	if pb == nil {
		return nil
	}
	return &AsyncHookJSONOutput{
		AsyncTimeout: optionalInt32FromProto(pb.AsyncTimeout),
	}
}

func (m *AsyncHookJSONOutput) ToProto() *sdktypes.AsyncHookJSONOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.AsyncHookJSONOutput{
		AsyncTimeout: optionalInt32ToProto(m.AsyncTimeout),
	}
}

func PreToolUseHookSpecificOutputFromProto(pb *sdktypes.PreToolUseHookSpecificOutput) *PreToolUseHookSpecificOutput {
	if pb == nil {
		return nil
	}
	return &PreToolUseHookSpecificOutput{
		PermissionDecision:       optionalStringFromProto(pb.PermissionDecision),
		PermissionDecisionReason: optionalStringFromProto(pb.PermissionDecisionReason),
		UpdatedInput:             ToolInputFromProto(pb.GetUpdatedInput()),
	}
}

func (m *PreToolUseHookSpecificOutput) ToProto() *sdktypes.PreToolUseHookSpecificOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.PreToolUseHookSpecificOutput{
		PermissionDecision:       optionalStringToProto(m.PermissionDecision),
		PermissionDecisionReason: optionalStringToProto(m.PermissionDecisionReason),
		UpdatedInput:             m.UpdatedInput.ToProto(),
	}
}

func UserPromptSubmitHookSpecificOutputFromProto(pb *sdktypes.UserPromptSubmitHookSpecificOutput) *UserPromptSubmitHookSpecificOutput {
	if pb == nil {
		return nil
	}
	return &UserPromptSubmitHookSpecificOutput{
		AdditionalContext: optionalStringFromProto(pb.AdditionalContext),
	}
}

func (m *UserPromptSubmitHookSpecificOutput) ToProto() *sdktypes.UserPromptSubmitHookSpecificOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.UserPromptSubmitHookSpecificOutput{
		AdditionalContext: optionalStringToProto(m.AdditionalContext),
	}
}

func SessionStartHookSpecificOutputFromProto(pb *sdktypes.SessionStartHookSpecificOutput) *SessionStartHookSpecificOutput {
	if pb == nil {
		return nil
	}
	return &SessionStartHookSpecificOutput{
		AdditionalContext: optionalStringFromProto(pb.AdditionalContext),
	}
}

func (m *SessionStartHookSpecificOutput) ToProto() *sdktypes.SessionStartHookSpecificOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.SessionStartHookSpecificOutput{
		AdditionalContext: optionalStringToProto(m.AdditionalContext),
	}
}

func PostToolUseHookSpecificOutputFromProto(pb *sdktypes.PostToolUseHookSpecificOutput) *PostToolUseHookSpecificOutput {
	if pb == nil {
		return nil
	}
	return &PostToolUseHookSpecificOutput{
		AdditionalContext: optionalStringFromProto(pb.AdditionalContext),
	}
}

func (m *PostToolUseHookSpecificOutput) ToProto() *sdktypes.PostToolUseHookSpecificOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.PostToolUseHookSpecificOutput{
		AdditionalContext: optionalStringToProto(m.AdditionalContext),
	}
}

func HookSpecificOutputFromProto(pb *sdktypes.HookSpecificOutput) *HookSpecificOutput {
	if pb == nil {
		return nil
	}
	m := &HookSpecificOutput{}
	switch v := pb.GetOutput().(type) {
	case *sdktypes.HookSpecificOutput_PreToolUse:
		m.PreToolUse = PreToolUseHookSpecificOutputFromProto(v.PreToolUse)
	case *sdktypes.HookSpecificOutput_UserPromptSubmit:
		m.UserPromptSubmit = UserPromptSubmitHookSpecificOutputFromProto(v.UserPromptSubmit)
	case *sdktypes.HookSpecificOutput_SessionStart:
		m.SessionStart = SessionStartHookSpecificOutputFromProto(v.SessionStart)
	case *sdktypes.HookSpecificOutput_PostToolUse:
		m.PostToolUse = PostToolUseHookSpecificOutputFromProto(v.PostToolUse)
	}
	return m
}

func (m *HookSpecificOutput) ToProto() *sdktypes.HookSpecificOutput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.HookSpecificOutput{}
	switch {
	case m.PreToolUse != nil:
		pb.Output = &sdktypes.HookSpecificOutput_PreToolUse{PreToolUse: m.PreToolUse.ToProto()}
	case m.UserPromptSubmit != nil:
		pb.Output = &sdktypes.HookSpecificOutput_UserPromptSubmit{UserPromptSubmit: m.UserPromptSubmit.ToProto()}
	case m.SessionStart != nil:
		pb.Output = &sdktypes.HookSpecificOutput_SessionStart{SessionStart: m.SessionStart.ToProto()}
	case m.PostToolUse != nil:
		pb.Output = &sdktypes.HookSpecificOutput_PostToolUse{PostToolUse: m.PostToolUse.ToProto()}
	}
	return pb
}

func SyncHookJSONOutputFromProto(pb *sdktypes.SyncHookJSONOutput) *SyncHookJSONOutput {
	if pb == nil {
		return nil
	}
	return &SyncHookJSONOutput{
		Continue:           optionalBoolFromProto(pb.Continue),
		SuppressOutput:     optionalBoolFromProto(pb.SuppressOutput),
		StopReason:         optionalStringFromProto(pb.StopReason),
		Decision:           optionalStringFromProto(pb.Decision),
		SystemMessage:      optionalStringFromProto(pb.SystemMessage),
		Reason:             optionalStringFromProto(pb.Reason),
		HookSpecificOutput: HookSpecificOutputFromProto(pb.GetHookSpecificOutput()),
	}
}

func (m *SyncHookJSONOutput) ToProto() *sdktypes.SyncHookJSONOutput {
	if m == nil {
		return nil
	}
	return &sdktypes.SyncHookJSONOutput{
		Continue:           optionalBoolToProto(m.Continue),
		SuppressOutput:     optionalBoolToProto(m.SuppressOutput),
		StopReason:         optionalStringToProto(m.StopReason),
		Decision:           optionalStringToProto(m.Decision),
		SystemMessage:      optionalStringToProto(m.SystemMessage),
		Reason:             optionalStringToProto(m.Reason),
		HookSpecificOutput: m.HookSpecificOutput.ToProto(),
	}
}

func HookJSONOutputFromProto(pb *sdktypes.HookJSONOutput) *HookJSONOutput {
	if pb == nil {
		return nil
	}
	m := &HookJSONOutput{}
	switch v := pb.GetOutput().(type) {
	case *sdktypes.HookJSONOutput_AsyncOutput:
		m.AsyncOutput = AsyncHookJSONOutputFromProto(v.AsyncOutput)
	case *sdktypes.HookJSONOutput_SyncOutput:
		m.SyncOutput = SyncHookJSONOutputFromProto(v.SyncOutput)
	}
	return m
}

func (m *HookJSONOutput) ToProto() *sdktypes.HookJSONOutput {
	if m == nil {
		return nil
	}
	pb := &sdktypes.HookJSONOutput{}
	switch {
	case m.AsyncOutput != nil:
		pb.Output = &sdktypes.HookJSONOutput_AsyncOutput{AsyncOutput: m.AsyncOutput.ToProto()}
	case m.SyncOutput != nil:
		pb.Output = &sdktypes.HookJSONOutput_SyncOutput{SyncOutput: m.SyncOutput.ToProto()}
	}
	return pb
}
