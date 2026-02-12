package tool_iov1

import "fmt"

func (x ApiKeySource) MarshalText() ([]byte, error) {
	switch x {
	case ApiKeySource_API_KEY_SOURCE_USER:
		return []byte("user"), nil
	case ApiKeySource_API_KEY_SOURCE_PROJECT:
		return []byte("project"), nil
	case ApiKeySource_API_KEY_SOURCE_ORG:
		return []byte("org"), nil
	case ApiKeySource_API_KEY_SOURCE_TEMPORARY:
		return []byte("temporary"), nil
	default:
		return nil, nil
	}
}

func (x *ApiKeySource) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = ApiKeySource_API_KEY_SOURCE_UNSPECIFIED
	case "user":
		*x = ApiKeySource_API_KEY_SOURCE_USER
	case "project":
		*x = ApiKeySource_API_KEY_SOURCE_PROJECT
	case "org":
		*x = ApiKeySource_API_KEY_SOURCE_ORG
	case "temporary":
		*x = ApiKeySource_API_KEY_SOURCE_TEMPORARY
	default:
		return fmt.Errorf("unknown ApiKeySource text: %q", string(b))
	}
	return nil
}

func (x SdkBeta) MarshalText() ([]byte, error) {
	switch x {
	case SdkBeta_SDK_BETA_CONTEXT_1M_2025_08_07:
		return []byte("context-1m-2025-08-07"), nil
	default:
		return nil, nil
	}
}

func (x *SdkBeta) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = SdkBeta_SDK_BETA_UNSPECIFIED
	case "context-1m-2025-08-07":
		*x = SdkBeta_SDK_BETA_CONTEXT_1M_2025_08_07
	default:
		return fmt.Errorf("unknown SdkBeta text: %q", string(b))
	}
	return nil
}

func (x ConfigScope) MarshalText() ([]byte, error) {
	switch x {
	case ConfigScope_CONFIG_SCOPE_LOCAL:
		return []byte("local"), nil
	case ConfigScope_CONFIG_SCOPE_USER:
		return []byte("user"), nil
	case ConfigScope_CONFIG_SCOPE_PROJECT:
		return []byte("project"), nil
	default:
		return nil, nil
	}
}

func (x *ConfigScope) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = ConfigScope_CONFIG_SCOPE_UNSPECIFIED
	case "local":
		*x = ConfigScope_CONFIG_SCOPE_LOCAL
	case "user":
		*x = ConfigScope_CONFIG_SCOPE_USER
	case "project":
		*x = ConfigScope_CONFIG_SCOPE_PROJECT
	default:
		return fmt.Errorf("unknown ConfigScope text: %q", string(b))
	}
	return nil
}

func (x McpServerConnectionStatus) MarshalText() ([]byte, error) {
	switch x {
	case McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_CONNECTED:
		return []byte("connected"), nil
	case McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_FAILED:
		return []byte("failed"), nil
	case McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_NEEDS_AUTH:
		return []byte("needs-auth"), nil
	case McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_PENDING:
		return []byte("pending"), nil
	default:
		return nil, nil
	}
}

func (x *McpServerConnectionStatus) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_UNSPECIFIED
	case "connected":
		*x = McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_CONNECTED
	case "failed":
		*x = McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_FAILED
	case "needs-auth":
		*x = McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_NEEDS_AUTH
	case "pending":
		*x = McpServerConnectionStatus_MCP_SERVER_CONNECTION_STATUS_PENDING
	default:
		return fmt.Errorf("unknown McpServerConnectionStatus text: %q", string(b))
	}
	return nil
}

func (x CallToolResultContentType) MarshalText() ([]byte, error) {
	switch x {
	case CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_TEXT:
		return []byte("text"), nil
	case CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_IMAGE:
		return []byte("image"), nil
	case CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_RESOURCE:
		return []byte("resource"), nil
	default:
		return nil, nil
	}
}

func (x *CallToolResultContentType) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_UNSPECIFIED
	case "text":
		*x = CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_TEXT
	case "image":
		*x = CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_IMAGE
	case "resource":
		*x = CallToolResultContentType_CALL_TOOL_RESULT_CONTENT_TYPE_RESOURCE
	default:
		return fmt.Errorf("unknown CallToolResultContentType text: %q", string(b))
	}
	return nil
}
