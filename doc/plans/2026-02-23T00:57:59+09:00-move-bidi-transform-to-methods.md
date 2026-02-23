# Move bi-directional transformation functions to methods

## Context

The `pkg/claudesdk/models/convert.go` file currently defines two public free functions for converting between Go model types and protobuf types:
- `PreToolUseHookInputToProto(m *PreToolUseHookInput) (*pb.PreToolUseHookInput, error)`
- `PreToolUseHookInputFromProto(p *pb.PreToolUseHookInput) (*PreToolUseHookInput, error)`

These should be methods on `*PreToolUseHookInput` instead, which is more idiomatic Go.

## Changes

### 1. Convert `PreToolUseHookInputToProto` to method `ToProto`

**File:** `pkg/claudesdk/models/convert.go:16`

```go
// Before
func PreToolUseHookInputToProto(m *PreToolUseHookInput) (*pb.PreToolUseHookInput, error) {

// After
func (m *PreToolUseHookInput) ToProto() (*pb.PreToolUseHookInput, error) {
```

### 2. Convert `PreToolUseHookInputFromProto` to method `FromProto`

**File:** `pkg/claudesdk/models/convert.go:36`

```go
// Before
func PreToolUseHookInputFromProto(p *pb.PreToolUseHookInput) (*PreToolUseHookInput, error) {
    // ...builds and returns new *PreToolUseHookInput
}

// After
func (m *PreToolUseHookInput) FromProto(p *pb.PreToolUseHookInput) error {
    // ...populates m's fields from p, returns only error
}
```

`FromProto` becomes a mutating-receiver method. The caller creates the struct and calls `FromProto` on it.

### 3. Update callers

**File:** `pkg/crabswarm/hook.go:29`
```go
// Before
protoInput, err := models.PreToolUseHookInputToProto(&input)
// After
protoInput, err := input.ToProto()
```

**File:** `pkg/claudesdk/models/models_test.go:171`
```go
// Before
p, err := models.PreToolUseHookInputToProto(&m)
// After
p, err := m.ToProto()
```

**File:** `pkg/claudesdk/models/models_test.go:180`
```go
// Before
m2, err := models.PreToolUseHookInputFromProto(p)
// After
var m2 models.PreToolUseHookInput
err = m2.FromProto(p)
```

### 4. Update comments

Update doc comments on the methods to reflect the new signatures.

### Files to modify
- `pkg/claudesdk/models/convert.go` — refactor 2 functions to methods
- `pkg/crabswarm/hook.go` — update 1 call site
- `pkg/claudesdk/models/models_test.go` — update 2 call sites

## Verification

```sh
cd /home/watage/gitrepo/github.com/ngicks/crabswarm
go build ./...
go test ./pkg/claudesdk/models/... ./pkg/crabswarm/...
```
