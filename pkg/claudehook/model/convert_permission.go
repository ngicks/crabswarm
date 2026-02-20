package model

import (
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
)

func PermissionRuleValueFromProto(pb *sdktypes.PermissionRuleValue) *PermissionRuleValue {
	if pb == nil {
		return nil
	}
	return &PermissionRuleValue{
		ToolName:    pb.GetToolName(),
		RuleContent: optionalStringFromProto(pb.RuleContent),
	}
}

func (m *PermissionRuleValue) ToProto() *sdktypes.PermissionRuleValue {
	if m == nil {
		return nil
	}
	return &sdktypes.PermissionRuleValue{
		ToolName:    m.ToolName,
		RuleContent: optionalStringToProto(m.RuleContent),
	}
}

func permissionRuleValuesFromProto(pbs []*sdktypes.PermissionRuleValue) []*PermissionRuleValue {
	if pbs == nil {
		return nil
	}
	result := make([]*PermissionRuleValue, len(pbs))
	for i, pb := range pbs {
		result[i] = PermissionRuleValueFromProto(pb)
	}
	return result
}

func permissionRuleValuesToProto(ms []*PermissionRuleValue) []*sdktypes.PermissionRuleValue {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.PermissionRuleValue, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}

func AddRulesUpdateFromProto(pb *sdktypes.AddRulesUpdate) *AddRulesUpdate {
	if pb == nil {
		return nil
	}
	return &AddRulesUpdate{
		Rules:       permissionRuleValuesFromProto(pb.GetRules()),
		Behavior:    PermissionBehaviorFromProto(pb.GetBehavior()),
		Destination: PermissionUpdateDestinationFromProto(pb.GetDestination()),
	}
}

func (m *AddRulesUpdate) ToProto() *sdktypes.AddRulesUpdate {
	if m == nil {
		return nil
	}
	return &sdktypes.AddRulesUpdate{
		Rules:       permissionRuleValuesToProto(m.Rules),
		Behavior:    m.Behavior.ToProto(),
		Destination: m.Destination.ToProto(),
	}
}

func ReplaceRulesUpdateFromProto(pb *sdktypes.ReplaceRulesUpdate) *ReplaceRulesUpdate {
	if pb == nil {
		return nil
	}
	return &ReplaceRulesUpdate{
		Rules:       permissionRuleValuesFromProto(pb.GetRules()),
		Behavior:    PermissionBehaviorFromProto(pb.GetBehavior()),
		Destination: PermissionUpdateDestinationFromProto(pb.GetDestination()),
	}
}

func (m *ReplaceRulesUpdate) ToProto() *sdktypes.ReplaceRulesUpdate {
	if m == nil {
		return nil
	}
	return &sdktypes.ReplaceRulesUpdate{
		Rules:       permissionRuleValuesToProto(m.Rules),
		Behavior:    m.Behavior.ToProto(),
		Destination: m.Destination.ToProto(),
	}
}

func RemoveRulesUpdateFromProto(pb *sdktypes.RemoveRulesUpdate) *RemoveRulesUpdate {
	if pb == nil {
		return nil
	}
	return &RemoveRulesUpdate{
		Rules:       permissionRuleValuesFromProto(pb.GetRules()),
		Behavior:    PermissionBehaviorFromProto(pb.GetBehavior()),
		Destination: PermissionUpdateDestinationFromProto(pb.GetDestination()),
	}
}

func (m *RemoveRulesUpdate) ToProto() *sdktypes.RemoveRulesUpdate {
	if m == nil {
		return nil
	}
	return &sdktypes.RemoveRulesUpdate{
		Rules:       permissionRuleValuesToProto(m.Rules),
		Behavior:    m.Behavior.ToProto(),
		Destination: m.Destination.ToProto(),
	}
}

func SetModeUpdateFromProto(pb *sdktypes.SetModeUpdate) *SetModeUpdate {
	if pb == nil {
		return nil
	}
	return &SetModeUpdate{
		Mode:        PermissionModeFromProto(pb.GetMode()),
		Destination: PermissionUpdateDestinationFromProto(pb.GetDestination()),
	}
}

func (m *SetModeUpdate) ToProto() *sdktypes.SetModeUpdate {
	if m == nil {
		return nil
	}
	return &sdktypes.SetModeUpdate{
		Mode:        m.Mode.ToProto(),
		Destination: m.Destination.ToProto(),
	}
}

func AddDirectoriesUpdateFromProto(pb *sdktypes.AddDirectoriesUpdate) *AddDirectoriesUpdate {
	if pb == nil {
		return nil
	}
	return &AddDirectoriesUpdate{
		Directories: pb.GetDirectories(),
		Destination: PermissionUpdateDestinationFromProto(pb.GetDestination()),
	}
}

func (m *AddDirectoriesUpdate) ToProto() *sdktypes.AddDirectoriesUpdate {
	if m == nil {
		return nil
	}
	return &sdktypes.AddDirectoriesUpdate{
		Directories: m.Directories,
		Destination: m.Destination.ToProto(),
	}
}

func RemoveDirectoriesUpdateFromProto(pb *sdktypes.RemoveDirectoriesUpdate) *RemoveDirectoriesUpdate {
	if pb == nil {
		return nil
	}
	return &RemoveDirectoriesUpdate{
		Directories: pb.GetDirectories(),
		Destination: PermissionUpdateDestinationFromProto(pb.GetDestination()),
	}
}

func (m *RemoveDirectoriesUpdate) ToProto() *sdktypes.RemoveDirectoriesUpdate {
	if m == nil {
		return nil
	}
	return &sdktypes.RemoveDirectoriesUpdate{
		Directories: m.Directories,
		Destination: m.Destination.ToProto(),
	}
}

func PermissionUpdateFromProto(pb *sdktypes.PermissionUpdate) *PermissionUpdate {
	if pb == nil {
		return nil
	}
	m := &PermissionUpdate{}
	switch v := pb.GetUpdate().(type) {
	case *sdktypes.PermissionUpdate_AddRules:
		m.AddRules = AddRulesUpdateFromProto(v.AddRules)
	case *sdktypes.PermissionUpdate_ReplaceRules:
		m.ReplaceRules = ReplaceRulesUpdateFromProto(v.ReplaceRules)
	case *sdktypes.PermissionUpdate_RemoveRules:
		m.RemoveRules = RemoveRulesUpdateFromProto(v.RemoveRules)
	case *sdktypes.PermissionUpdate_SetMode:
		m.SetMode = SetModeUpdateFromProto(v.SetMode)
	case *sdktypes.PermissionUpdate_AddDirectories:
		m.AddDirectories = AddDirectoriesUpdateFromProto(v.AddDirectories)
	case *sdktypes.PermissionUpdate_RemoveDirectories:
		m.RemoveDirectories = RemoveDirectoriesUpdateFromProto(v.RemoveDirectories)
	}
	return m
}

func (m *PermissionUpdate) ToProto() *sdktypes.PermissionUpdate {
	if m == nil {
		return nil
	}
	pb := &sdktypes.PermissionUpdate{}
	switch {
	case m.AddRules != nil:
		pb.Update = &sdktypes.PermissionUpdate_AddRules{AddRules: m.AddRules.ToProto()}
	case m.ReplaceRules != nil:
		pb.Update = &sdktypes.PermissionUpdate_ReplaceRules{ReplaceRules: m.ReplaceRules.ToProto()}
	case m.RemoveRules != nil:
		pb.Update = &sdktypes.PermissionUpdate_RemoveRules{RemoveRules: m.RemoveRules.ToProto()}
	case m.SetMode != nil:
		pb.Update = &sdktypes.PermissionUpdate_SetMode{SetMode: m.SetMode.ToProto()}
	case m.AddDirectories != nil:
		pb.Update = &sdktypes.PermissionUpdate_AddDirectories{AddDirectories: m.AddDirectories.ToProto()}
	case m.RemoveDirectories != nil:
		pb.Update = &sdktypes.PermissionUpdate_RemoveDirectories{RemoveDirectories: m.RemoveDirectories.ToProto()}
	}
	return pb
}

func permissionUpdatesFromProto(pbs []*sdktypes.PermissionUpdate) []*PermissionUpdate {
	if pbs == nil {
		return nil
	}
	result := make([]*PermissionUpdate, len(pbs))
	for i, pb := range pbs {
		result[i] = PermissionUpdateFromProto(pb)
	}
	return result
}

func permissionUpdatesToProto(ms []*PermissionUpdate) []*sdktypes.PermissionUpdate {
	if ms == nil {
		return nil
	}
	result := make([]*sdktypes.PermissionUpdate, len(ms))
	for i, m := range ms {
		result[i] = m.ToProto()
	}
	return result
}
