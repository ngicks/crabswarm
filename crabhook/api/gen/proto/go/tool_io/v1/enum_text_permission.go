package tool_iov1

import "fmt"

func (x PermissionMode) MarshalText() ([]byte, error) {
	switch x {
	case PermissionMode_PERMISSION_MODE_DEFAULT:
		return []byte("default"), nil
	case PermissionMode_PERMISSION_MODE_ACCEPT_EDITS:
		return []byte("acceptEdits"), nil
	case PermissionMode_PERMISSION_MODE_BYPASS_PERMISSIONS:
		return []byte("bypassPermissions"), nil
	case PermissionMode_PERMISSION_MODE_PLAN:
		return []byte("plan"), nil
	default:
		return nil, nil
	}
}

func (x *PermissionMode) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = PermissionMode_PERMISSION_MODE_UNSPECIFIED
	case "default":
		*x = PermissionMode_PERMISSION_MODE_DEFAULT
	case "acceptEdits":
		*x = PermissionMode_PERMISSION_MODE_ACCEPT_EDITS
	case "bypassPermissions":
		*x = PermissionMode_PERMISSION_MODE_BYPASS_PERMISSIONS
	case "plan":
		*x = PermissionMode_PERMISSION_MODE_PLAN
	default:
		return fmt.Errorf("unknown PermissionMode text: %q", string(b))
	}
	return nil
}

func (x PermissionBehavior) MarshalText() ([]byte, error) {
	switch x {
	case PermissionBehavior_PERMISSION_BEHAVIOR_ALLOW:
		return []byte("allow"), nil
	case PermissionBehavior_PERMISSION_BEHAVIOR_DENY:
		return []byte("deny"), nil
	case PermissionBehavior_PERMISSION_BEHAVIOR_ASK:
		return []byte("ask"), nil
	default:
		return nil, nil
	}
}

func (x *PermissionBehavior) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = PermissionBehavior_PERMISSION_BEHAVIOR_UNSPECIFIED
	case "allow":
		*x = PermissionBehavior_PERMISSION_BEHAVIOR_ALLOW
	case "deny":
		*x = PermissionBehavior_PERMISSION_BEHAVIOR_DENY
	case "ask":
		*x = PermissionBehavior_PERMISSION_BEHAVIOR_ASK
	default:
		return fmt.Errorf("unknown PermissionBehavior text: %q", string(b))
	}
	return nil
}

func (x PermissionUpdateDestination) MarshalText() ([]byte, error) {
	switch x {
	case PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_USER_SETTINGS:
		return []byte("userSettings"), nil
	case PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_PROJECT_SETTINGS:
		return []byte("projectSettings"), nil
	case PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_LOCAL_SETTINGS:
		return []byte("localSettings"), nil
	case PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_SESSION:
		return []byte("session"), nil
	default:
		return nil, nil
	}
}

func (x *PermissionUpdateDestination) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_UNSPECIFIED
	case "userSettings":
		*x = PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_USER_SETTINGS
	case "projectSettings":
		*x = PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_PROJECT_SETTINGS
	case "localSettings":
		*x = PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_LOCAL_SETTINGS
	case "session":
		*x = PermissionUpdateDestination_PERMISSION_UPDATE_DESTINATION_SESSION
	default:
		return fmt.Errorf("unknown PermissionUpdateDestination text: %q", string(b))
	}
	return nil
}
