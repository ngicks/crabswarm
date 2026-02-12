package tool_iov1

import "fmt"

func (x SettingSource) MarshalText() ([]byte, error) {
	switch x {
	case SettingSource_SETTING_SOURCE_USER:
		return []byte("user"), nil
	case SettingSource_SETTING_SOURCE_PROJECT:
		return []byte("project"), nil
	case SettingSource_SETTING_SOURCE_LOCAL:
		return []byte("local"), nil
	default:
		return nil, nil
	}
}

func (x *SettingSource) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = SettingSource_SETTING_SOURCE_UNSPECIFIED
	case "user":
		*x = SettingSource_SETTING_SOURCE_USER
	case "project":
		*x = SettingSource_SETTING_SOURCE_PROJECT
	case "local":
		*x = SettingSource_SETTING_SOURCE_LOCAL
	default:
		return fmt.Errorf("unknown SettingSource text: %q", string(b))
	}
	return nil
}

func (x AgentModel) MarshalText() ([]byte, error) {
	switch x {
	case AgentModel_AGENT_MODEL_SONNET:
		return []byte("sonnet"), nil
	case AgentModel_AGENT_MODEL_OPUS:
		return []byte("opus"), nil
	case AgentModel_AGENT_MODEL_HAIKU:
		return []byte("haiku"), nil
	case AgentModel_AGENT_MODEL_INHERIT:
		return []byte("inherit"), nil
	default:
		return nil, nil
	}
}

func (x *AgentModel) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = AgentModel_AGENT_MODEL_UNSPECIFIED
	case "sonnet":
		*x = AgentModel_AGENT_MODEL_SONNET
	case "opus":
		*x = AgentModel_AGENT_MODEL_OPUS
	case "haiku":
		*x = AgentModel_AGENT_MODEL_HAIKU
	case "inherit":
		*x = AgentModel_AGENT_MODEL_INHERIT
	default:
		return fmt.Errorf("unknown AgentModel text: %q", string(b))
	}
	return nil
}
