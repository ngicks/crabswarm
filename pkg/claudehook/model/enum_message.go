package model

// SDKResultSubtype represents the subtype of an SDK result message.
type SDKResultSubtype string

const (
	SDKResultSubtypeSuccess                       SDKResultSubtype = "success"
	SDKResultSubtypeErrorMaxTurns                 SDKResultSubtype = "error_max_turns"
	SDKResultSubtypeErrorDuringExecution          SDKResultSubtype = "error_during_execution"
	SDKResultSubtypeErrorMaxBudgetUsd             SDKResultSubtype = "error_max_budget_usd"
	SDKResultSubtypeErrorMaxStructuredOutputRetries SDKResultSubtype = "error_max_structured_output_retries"
)
