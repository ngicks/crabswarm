// Package api anchors code generation for the crabswarm proto schema; it holds
// no Go source of its own. The go:generate directive below runs buf from this
// directory, where buf.gen.yaml lives, so `go generate ./...` reaches it: buf
// reads the schema under pkg/api/schema and emits the connect-go/gRPC Go
// bindings into pkg/api/gen and the TypeScript client types into web/src/gen
// (see buf.gen.yaml).
package api

//go:generate buf generate
