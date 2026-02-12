package tool_iov1

import "fmt"

func (x SDKResultSubtype) MarshalText() ([]byte, error) {
	switch x {
	case SDKResultSubtype_SDK_RESULT_SUBTYPE_SUCCESS:
		return []byte("success"), nil
	case SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_TURNS:
		return []byte("error_max_turns"), nil
	case SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_DURING_EXECUTION:
		return []byte("error_during_execution"), nil
	case SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_BUDGET_USD:
		return []byte("error_max_budget_usd"), nil
	case SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_STRUCTURED_OUTPUT_RETRIES:
		return []byte("error_max_structured_output_retries"), nil
	default:
		return nil, nil
	}
}

func (x *SDKResultSubtype) UnmarshalText(b []byte) error {
	switch string(b) {
	case "":
		*x = SDKResultSubtype_SDK_RESULT_SUBTYPE_UNSPECIFIED
	case "success":
		*x = SDKResultSubtype_SDK_RESULT_SUBTYPE_SUCCESS
	case "error_max_turns":
		*x = SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_TURNS
	case "error_during_execution":
		*x = SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_DURING_EXECUTION
	case "error_max_budget_usd":
		*x = SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_BUDGET_USD
	case "error_max_structured_output_retries":
		*x = SDKResultSubtype_SDK_RESULT_SUBTYPE_ERROR_MAX_STRUCTURED_OUTPUT_RETRIES
	default:
		return fmt.Errorf("unknown SDKResultSubtype text: %q", string(b))
	}
	return nil
}
