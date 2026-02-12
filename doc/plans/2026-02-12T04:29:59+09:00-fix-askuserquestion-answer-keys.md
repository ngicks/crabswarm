# Fix: AskUserQuestion answer key format does not match official protocol

**Sources**:
- https://platform.claude.com/docs/en/agent-sdk/user-input
- https://platform.claude.com/docs/en/agent-sdk/hooks

Plan also written to `doc/plans/2026-02-12T03:53:04+09:00-fix-askuserquestion-answer-keys.md`.

## Context

The previous commit fixed hook I/O field names and output structure to match the official protocol. However, the `AskUserQuestion` handling in `prompt.go` still doesn't conform to the official user-input protocol described at the user-input docs page. The answer map keys and response format are wrong.

## Problem

In `hook/internal/server/prompt.go:187`, the answer map key is:

```go
key := fmt.Sprintf("question_%d", i)  // WRONG: "question_0", "question_1", ...
```

The official protocol (from the user-input docs) requires the **question text** as the key:

```json
{
  "questions": [...],
  "answers": {
    "How should I format the output?": "Summary",
    "Which sections should I include?": "Introduction, Conclusion"
  }
}
```

Additionally, non-numeric input (free text typed directly) should be used as-is as the answer value, rather than requiring the user to explicitly choose "0) Other" first.

## Fix

### File: `hook/internal/server/prompt.go`

**Change 1**: Use `q.Question` as the answer map key instead of `question_0`, `question_1`.

Line 187: `key := fmt.Sprintf("question_%d", i)` → `key := q.Question`

**Change 2**: Support direct free-text input. Currently, non-numeric input falls through to `resolveAnswers` which tries to parse as numbers and falls back to raw text. This works but the UX prompt says "0 for custom" implying that's the only way. Change the prompt text and the `choice == "0"` condition to also handle direct free-text:

- If the input is numeric and matches an option → use the option label
- If the input is "0" → prompt for custom text
- If the input is non-numeric text → use it directly as the answer

The `resolveAnswers` function already handles the non-numeric fallback correctly (returns `choice` if no valid indices), so we just need to update the prompt text.

### File: `hook/internal/server/prompt_test.go` (new)

Add a test for `promptAskUserQuestion` and `resolveAnswers` that verifies:
- Answer keys are question text, not `question_0`
- Numeric selection maps to the option label
- Free-text input is passed through directly
- Multi-select comma-separated numbers produce joined labels

## Verification

1. `go build ./...` — compiles
2. `go test ./...` — all tests pass
3. Verify JSON output matches: `{"questions": [...], "answers": {"<question text>": "<label or free text>"}}`
