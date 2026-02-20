// Package hookhandler defines handler for claude hook
package hookhandler

type HookDecision string

type HandlerError struct {
	Decision HookDecision
}

func (e *HandlerError) Error() string {
	return ""
}

func (e *HandlerError) Handle() {
}
