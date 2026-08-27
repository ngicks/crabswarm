package types

// codex_extension.go adds Codex (OpenAI codex-rs) tool input/output variants to
// the Claude SDK's ToolInputSchemas / ToolOutputSchemas unions.
//
// Codex emits hook payloads in the Claude hook envelope but a few tools carry
// shapes the Claude SDK never produces (apply_patch, the bare-string "Bash"
// tool_response, update_plan). The structural types and parsing live in the
// ext/codex package; the union variant wrappers must live here because the
// union marker interfaces (toolInputSchemas / toolOutputSchemas) are sealed to
// this package. Tools whose Codex shape already matches a Claude type — Bash
// tool_input (BashInput) and MCP tool_input (McpInput) — reuse the Claude
// dispatch and have no wrapper here; MCP tool_response stays ToolOutputUnknown
// (its raw bytes round-trip unchanged and codex.ParseCallToolResult decodes it).
//
// Source: pkg/claudehook/types/ext/codex, codex-rs/core/src/tools.

import (
	"encoding/json"

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/sdktypes/v1"
	codexpb "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/sdktypes/v1/ext/codex"
	"github.com/ngicks/crabswarm/pkg/claudehook/types/ext/codex"
)

// CodexApplyPatchInput is the Codex apply_patch tool_input variant of
// [ToolInputSchemas] (tool_name "apply_patch").
//
// Source: pkg/claudehook/types/ext/codex (ApplyPatchInput)
type CodexApplyPatchInput struct{ codex.ApplyPatchInput }

func (CodexApplyPatchInput) toolInputSchemas() {}

// CodexUpdatePlanInput is the Codex update_plan tool_input variant of
// [ToolInputSchemas] (tool_name "update_plan").
//
// Source: pkg/claudehook/types/ext/codex (UpdatePlanInput)
type CodexUpdatePlanInput struct{ codex.UpdatePlanInput }

func (CodexUpdatePlanInput) toolInputSchemas() {}

// CodexApplyPatchOutput is the Codex apply_patch tool_response variant of
// [ToolOutputSchemas] (tool_name "apply_patch"), a JSON string of exec-formatted
// output.
//
// Source: pkg/claudehook/types/ext/codex (ApplyPatchOutput)
type CodexApplyPatchOutput struct{ codex.ApplyPatchOutput }

func (CodexApplyPatchOutput) toolOutputSchemas() {}

// CodexShellOutput is the Codex shell tool_response variant of
// [ToolOutputSchemas] (tool_name "Bash"), a bare JSON string of raw output that
// Claude's object-shaped BashOutput cannot decode.
//
// Source: pkg/claudehook/types/ext/codex (ShellOutput)
type CodexShellOutput struct{ codex.ShellOutput }

func (CodexShellOutput) toolOutputSchemas() {}

var (
	_ ToolInputSchemas_Value  = (*CodexApplyPatchInput)(nil)
	_ ToolInputSchemas_Value  = (*CodexUpdatePlanInput)(nil)
	_ ToolOutputSchemas_Value = (*CodexApplyPatchOutput)(nil)
	_ ToolOutputSchemas_Value = (*CodexShellOutput)(nil)
)

// GetCodexApplyPatchInput returns the active variant as a [CodexApplyPatchInput].
func (o ToolInputSchemas) GetCodexApplyPatchInput() (*CodexApplyPatchInput, bool) {
	v, ok := o.value.(*CodexApplyPatchInput)
	return v, ok
}

// GetCodexUpdatePlanInput returns the active variant as a [CodexUpdatePlanInput].
func (o ToolInputSchemas) GetCodexUpdatePlanInput() (*CodexUpdatePlanInput, bool) {
	v, ok := o.value.(*CodexUpdatePlanInput)
	return v, ok
}

// GetCodexApplyPatchOutput returns the active variant as a [CodexApplyPatchOutput].
func (o ToolOutputSchemas) GetCodexApplyPatchOutput() (*CodexApplyPatchOutput, bool) {
	v, ok := o.value.(*CodexApplyPatchOutput)
	return v, ok
}

// GetCodexShellOutput returns the active variant as a [CodexShellOutput].
func (o ToolOutputSchemas) GetCodexShellOutput() (*CodexShellOutput, bool) {
	v, ok := o.value.(*CodexShellOutput)
	return v, ok
}

// unmarshalCodexInputForTool decodes Codex-specific tool inputs the Claude SDK
// doesn't define, returning done=true when toolName is handled here. Codex tools
// whose input shape matches a Claude type (Bash, mcp__*) are left to the Claude
// dispatch.
func (o *ToolInputSchemas) unmarshalCodexInputForTool(toolName string, data []byte) (bool, error) {
	switch toolName {
	case codex.ToolNameApplyPatch:
		var v CodexApplyPatchInput
		o.value = &v
		return true, v.UnmarshalJSON(data)
	case codex.ToolNameUpdatePlan:
		var v CodexUpdatePlanInput
		o.value = &v
		return true, v.UnmarshalJSON(data)
	default:
		return false, nil
	}
}

// unmarshalCodexOutputForTool decodes Codex-specific tool responses, returning
// done=true when handled here. apply_patch is always Codex-shaped; "Bash" is
// shared with Claude, so only its bare-string form is taken here and the object
// form is left to Claude's BashOutput.
func (o *ToolOutputSchemas) unmarshalCodexOutputForTool(
	toolName string,
	data []byte,
) (bool, error) {
	switch toolName {
	case codex.ToolNameApplyPatch:
		var v CodexApplyPatchOutput
		o.value = &v
		return true, v.UnmarshalJSON(data)
	case codex.ToolNameBash:
		if firstJSONToken(data) == '"' {
			var v CodexShellOutput
			o.value = &v
			return true, v.UnmarshalJSON(data)
		}
		return false, nil
	default:
		return false, nil
	}
}

// firstJSONToken returns the first non-whitespace byte of data, or 0 when data
// holds only whitespace. It distinguishes a JSON string ('"') from an object
// ('{') without a full decode.
func firstJSONToken(data []byte) byte {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		default:
			return b
		}
	}
	return 0
}

// codexToolInputToProto builds the proto ToolInput for a Codex input variant, or
// nil when v is not a Codex variant (the caller falls back to the raw carrier).
func codexToolInputToProto(v ToolInputSchemas_Value) *pb.ToolInput {
	switch x := v.(type) {
	case *CodexApplyPatchInput:
		return &pb.ToolInput{
			Input: &pb.ToolInput_CodexApplyPatch{CodexApplyPatch: codexApplyPatchInputToProto(x)},
		}
	case *CodexUpdatePlanInput:
		return &pb.ToolInput{
			Input: &pb.ToolInput_CodexUpdatePlan{CodexUpdatePlan: codexUpdatePlanInputToProto(x)},
		}
	default:
		return nil
	}
}

// codexToolInputFromProto reconstructs a Codex input variant from the proto
// ToolInput, returning ok=false when no Codex oneof variant is set.
func codexToolInputFromProto(v *pb.ToolInput) (ToolInputSchemas, bool, error) {
	switch {
	case v.GetCodexApplyPatch() != nil:
		c, err := codexApplyPatchInputFromProto(v.GetCodexApplyPatch())
		if err != nil {
			return ToolInputSchemas{}, true, err
		}
		return NewToolInputSchemas(c), true, nil
	case v.GetCodexUpdatePlan() != nil:
		c, err := codexUpdatePlanInputFromProto(v.GetCodexUpdatePlan())
		if err != nil {
			return ToolInputSchemas{}, true, err
		}
		return NewToolInputSchemas(c), true, nil
	default:
		return ToolInputSchemas{}, false, nil
	}
}

// codexToolOutputToProto builds the proto ToolOutput for a Codex output variant,
// or nil when v is not a Codex variant.
func codexToolOutputToProto(v ToolOutputSchemas_Value) *pb.ToolOutput {
	switch x := v.(type) {
	case *CodexApplyPatchOutput:
		return &pb.ToolOutput{
			Output: &pb.ToolOutput_CodexApplyPatch{
				CodexApplyPatch: codexApplyPatchOutputToProto(x),
			},
		}
	case *CodexShellOutput:
		return &pb.ToolOutput{
			Output: &pb.ToolOutput_CodexShell{CodexShell: codexShellOutputToProto(x)},
		}
	default:
		return nil
	}
}

// codexToolOutputFromProto reconstructs a Codex output variant from the proto
// ToolOutput, returning ok=false when no Codex oneof variant is set.
func codexToolOutputFromProto(v *pb.ToolOutput) (ToolOutputSchemas, bool, error) {
	switch {
	case v.GetCodexApplyPatch() != nil:
		c, err := codexApplyPatchOutputFromProto(v.GetCodexApplyPatch())
		if err != nil {
			return ToolOutputSchemas{}, true, err
		}
		return NewToolOutputSchemas(c), true, nil
	case v.GetCodexShell() != nil:
		c, err := codexShellOutputFromProto(v.GetCodexShell())
		if err != nil {
			return ToolOutputSchemas{}, true, err
		}
		return NewToolOutputSchemas(c), true, nil
	default:
		return ToolOutputSchemas{}, false, nil
	}
}

func codexApplyPatchInputToProto(x *CodexApplyPatchInput) *codexpb.CodexApplyPatchInput {
	return &codexpb.CodexApplyPatchInput{
		RawJson:    x.RawJSON(),
		Command:    x.Command,
		Operations: codexPatchOpsToProto(x.Patch.Operations),
	}
}

func codexApplyPatchInputFromProto(p *codexpb.CodexApplyPatchInput) (*CodexApplyPatchInput, error) {
	var v CodexApplyPatchInput
	if raw := p.GetRawJson(); len(raw) > 0 {
		return &v, v.UnmarshalJSON(raw)
	}
	v.Command = p.GetCommand()
	v.Patch = codex.Patch{Operations: codexPatchOpsFromProto(p.GetOperations())}
	return &v, nil
}

func codexApplyPatchOutputToProto(x *CodexApplyPatchOutput) *codexpb.CodexApplyPatchOutput {
	return &codexpb.CodexApplyPatchOutput{
		RawJson:          x.RawJSON(),
		Text:             x.Text,
		ExitCode:         x.ExitCode,
		WallTimeSeconds:  x.WallTimeSeconds,
		TotalOutputLines: x.TotalOutputLines,
		Output:           x.Output,
		Summary:          codexSummaryToProto(x.Summary),
	}
}

func codexApplyPatchOutputFromProto(
	p *codexpb.CodexApplyPatchOutput,
) (*CodexApplyPatchOutput, error) {
	var v CodexApplyPatchOutput
	if raw := p.GetRawJson(); len(raw) > 0 {
		return &v, v.UnmarshalJSON(raw)
	}
	// Rebuild from the decoded text so every field stays consistent.
	b, err := json.Marshal(p.GetText())
	if err != nil {
		return nil, err
	}
	return &v, v.UnmarshalJSON(b)
}

func codexShellOutputToProto(x *CodexShellOutput) *codexpb.CodexShellOutput {
	return &codexpb.CodexShellOutput{RawJson: x.RawJSON(), Output: x.Output}
}

func codexShellOutputFromProto(p *codexpb.CodexShellOutput) (*CodexShellOutput, error) {
	var v CodexShellOutput
	if raw := p.GetRawJson(); len(raw) > 0 {
		return &v, v.UnmarshalJSON(raw)
	}
	b, err := json.Marshal(p.GetOutput())
	if err != nil {
		return nil, err
	}
	return &v, v.UnmarshalJSON(b)
}

func codexUpdatePlanInputToProto(x *CodexUpdatePlanInput) *codexpb.CodexUpdatePlanInput {
	return &codexpb.CodexUpdatePlanInput{
		RawJson:     x.RawJSON(),
		Explanation: x.Explanation,
		Plan:        codexPlanItemsToProto(x.Plan),
	}
}

func codexUpdatePlanInputFromProto(p *codexpb.CodexUpdatePlanInput) (*CodexUpdatePlanInput, error) {
	var v CodexUpdatePlanInput
	if raw := p.GetRawJson(); len(raw) > 0 {
		return &v, v.UnmarshalJSON(raw)
	}
	v.Explanation = protoOpt(p, "explanation", p.GetExplanation())
	v.Plan = codexPlanItemsFromProto(p.GetPlan())
	return &v, nil
}

func codexPatchOpsToProto(ops []codex.PatchOperation) []*codexpb.CodexPatchOperation {
	if ops == nil {
		return nil
	}
	out := make([]*codexpb.CodexPatchOperation, len(ops))
	for i, op := range ops {
		out[i] = &codexpb.CodexPatchOperation{
			Kind:   codexPatchKindToProto(op.Kind),
			Path:   op.Path,
			MoveTo: op.MoveTo,
		}
	}
	return out
}

func codexPatchOpsFromProto(in []*codexpb.CodexPatchOperation) []codex.PatchOperation {
	if in == nil {
		return nil
	}
	out := make([]codex.PatchOperation, len(in))
	for i, op := range in {
		out[i] = codex.PatchOperation{
			Kind:   codexPatchKindFromProto(op.GetKind()),
			Path:   op.GetPath(),
			MoveTo: op.GetMoveTo(),
		}
	}
	return out
}

func codexSummaryToProto(summary []codex.SummaryEntry) []*codexpb.CodexApplyPatchSummaryEntry {
	if summary == nil {
		return nil
	}
	out := make([]*codexpb.CodexApplyPatchSummaryEntry, len(summary))
	for i, e := range summary {
		out[i] = &codexpb.CodexApplyPatchSummaryEntry{
			Kind: codexPatchKindToProto(e.Kind),
			Path: e.Path,
		}
	}
	return out
}

func codexPlanItemsToProto(plan []codex.PlanItem) []*codexpb.CodexPlanItem {
	if plan == nil {
		return nil
	}
	out := make([]*codexpb.CodexPlanItem, len(plan))
	for i, item := range plan {
		out[i] = &codexpb.CodexPlanItem{
			Step:   item.Step,
			Status: codexPlanStatusToProto(item.Status),
		}
	}
	return out
}

func codexPlanItemsFromProto(in []*codexpb.CodexPlanItem) []codex.PlanItem {
	if in == nil {
		return nil
	}
	out := make([]codex.PlanItem, len(in))
	for i, item := range in {
		out[i] = codex.PlanItem{
			Step:   item.GetStep(),
			Status: codexPlanStatusFromProto(item.GetStatus()),
		}
	}
	return out
}

func codexPatchKindToProto(k codex.PatchKind) codexpb.CodexPatchKind {
	switch k {
	case codex.PatchKindAdd:
		return codexpb.CodexPatchKind_CODEX_PATCH_KIND_ADD
	case codex.PatchKindUpdate:
		return codexpb.CodexPatchKind_CODEX_PATCH_KIND_UPDATE
	case codex.PatchKindDelete:
		return codexpb.CodexPatchKind_CODEX_PATCH_KIND_DELETE
	default:
		return codexpb.CodexPatchKind_CODEX_PATCH_KIND_UNSPECIFIED
	}
}

func codexPatchKindFromProto(k codexpb.CodexPatchKind) codex.PatchKind {
	switch k {
	case codexpb.CodexPatchKind_CODEX_PATCH_KIND_ADD:
		return codex.PatchKindAdd
	case codexpb.CodexPatchKind_CODEX_PATCH_KIND_UPDATE:
		return codex.PatchKindUpdate
	case codexpb.CodexPatchKind_CODEX_PATCH_KIND_DELETE:
		return codex.PatchKindDelete
	default:
		return ""
	}
}

func codexPlanStatusToProto(s codex.PlanStatus) codexpb.CodexPlanStatus {
	switch s {
	case codex.PlanStatusPending:
		return codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_PENDING
	case codex.PlanStatusInProgress:
		return codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_IN_PROGRESS
	case codex.PlanStatusCompleted:
		return codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_COMPLETED
	default:
		return codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_UNSPECIFIED
	}
}

func codexPlanStatusFromProto(s codexpb.CodexPlanStatus) codex.PlanStatus {
	switch s {
	case codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_PENDING:
		return codex.PlanStatusPending
	case codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_IN_PROGRESS:
		return codex.PlanStatusInProgress
	case codexpb.CodexPlanStatus_CODEX_PLAN_STATUS_COMPLETED:
		return codex.PlanStatusCompleted
	default:
		return ""
	}
}
