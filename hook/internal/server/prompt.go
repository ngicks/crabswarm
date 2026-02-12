package server

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
)

// PlainPrompter handles interactive prompts for permission decisions via plain text stdin/stdout.
type PlainPrompter struct {
	reader io.Reader
	writer io.Writer
}

// NewPlainPrompter creates a new PlainPrompter with the given reader and writer.
func NewPlainPrompter(reader io.Reader, writer io.Writer) *PlainPrompter {
	return &PlainPrompter{
		reader: reader,
		writer: writer,
	}
}

// Prompt displays the permission request and prompts the user for a decision.
func (p *PlainPrompter) Prompt(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	// Display the request information
	fmt.Fprintf(p.writer, "\n%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(p.writer, "Permission Request: %s\n", req.HookEventName)
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Tool:       %s\n", req.ToolName)
	fmt.Fprintf(p.writer, "Session:    %s\n", req.SessionId)
	fmt.Fprintf(p.writer, "Message:    %s\n", req.MessageId)
	if req.Cwd != "" {
		fmt.Fprintf(p.writer, "Cwd:        %s\n", req.Cwd)
	}
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))

	// Pretty print the tool input
	if req.ToolInputJson != "" {
		var prettyJSON map[string]any
		if err := json.Unmarshal([]byte(req.ToolInputJson), &prettyJSON); err == nil {
			formatted, _ := json.MarshalIndent(prettyJSON, "", "  ")
			fmt.Fprintf(p.writer, "Input:\n%s\n", string(formatted))
		} else {
			fmt.Fprintf(p.writer, "Input: %s\n", req.ToolInputJson)
		}
	}

	// Check if this is an AskUserQuestion tool
	if req.ToolName == "AskUserQuestion" && req.ToolInputJson != "" {
		return p.promptAskUserQuestion(ctx, req)
	}

	// Check if this is an ExitPlanMode tool
	if req.ToolName == "ExitPlanMode" && req.ToolInputJson != "" {
		return p.promptExitPlanMode(ctx, req)
	}

	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Options:\n")
	fmt.Fprintf(p.writer, "  [a] Allow  - Allow this tool execution\n")
	fmt.Fprintf(p.writer, "  [d] Deny   - Deny this tool execution\n")
	fmt.Fprintf(p.writer, "  [k] Ask    - Prompt user for confirmation\n")
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(p.writer, "Your choice [a/d/k]: ")

	// Read user input
	scanner := bufio.NewScanner(p.reader)
	responseCh := make(chan string, 1)

	go func() {
		if scanner.Scan() {
			responseCh <- strings.TrimSpace(scanner.Text())
		} else {
			responseCh <- ""
		}
	}()

	var choice string
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case choice = <-responseCh:
	}

	// Parse the choice
	var decision pb.PermissionDecision
	var reason string

	switch choice {
	case "a", "y", "yes", "allow":
		decision = pb.PermissionDecision_PERMISSION_DECISION_ALLOW
		fmt.Fprintf(p.writer, "-> Allowed\n")
	case "d", "n", "no", "deny", "b", "block":
		decision = pb.PermissionDecision_PERMISSION_DECISION_DENY
		fmt.Fprintf(p.writer, "Reason (optional, press Enter to skip): ")
		if scanner.Scan() {
			reason = strings.TrimSpace(scanner.Text())
		}
		fmt.Fprintf(p.writer, "-> Denied\n")
	case "k", "ask":
		decision = pb.PermissionDecision_PERMISSION_DECISION_ASK
		fmt.Fprintf(p.writer, "-> Ask (prompt for confirmation)\n")
	default:
		decision = pb.PermissionDecision_PERMISSION_DECISION_DENY
		reason = "Invalid choice - defaulting to deny"
		fmt.Fprintf(p.writer, "-> Invalid choice, defaulting to deny\n")
	}

	return BuildPermissionResponse(req, decision, reason), nil
}

// promptAskUserQuestion handles AskUserQuestion tool inputs.
func (p *PlainPrompter) promptAskUserQuestion(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	input, err := ParseAskUserInput(req.ToolInputJson)
	if err != nil {
		fmt.Fprintf(p.writer, "  (Failed to parse AskUserQuestion input, falling back to standard prompt)\n")
		return p.promptStandard(ctx, req)
	}

	scanner := bufio.NewScanner(p.reader)
	answers := make(map[string]string)

	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "AskUserQuestion: Answer the questions below\n")
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))

	for _, q := range input.Questions {
		fmt.Fprintf(p.writer, "\n[%s] %s\n", q.Header, q.Question)
		for j, opt := range q.Options {
			if opt.Description != "" {
				fmt.Fprintf(p.writer, "  %d) %s - %s\n", j+1, opt.Label, opt.Description)
			} else {
				fmt.Fprintf(p.writer, "  %d) %s\n", j+1, opt.Label)
			}
		}
		fmt.Fprintf(p.writer, "  0) Other (type custom answer)\n")

		if q.MultiSelect {
			fmt.Fprintf(p.writer, "Enter choices (comma-separated numbers, 0 for custom, or type text): ")
		} else {
			fmt.Fprintf(p.writer, "Enter choice (number, 0 for custom, or type text): ")
		}

		responseCh := make(chan string, 1)
		go func() {
			if scanner.Scan() {
				responseCh <- strings.TrimSpace(scanner.Text())
			} else {
				responseCh <- ""
			}
		}()

		var choice string
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case choice = <-responseCh:
		}

		key := q.Question
		if choice == "0" || choice == "" {
			fmt.Fprintf(p.writer, "Enter your answer: ")
			customCh := make(chan string, 1)
			go func() {
				if scanner.Scan() {
					customCh <- strings.TrimSpace(scanner.Text())
				} else {
					customCh <- ""
				}
			}()
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case answers[key] = <-customCh:
			}
		} else {
			answers[key] = ResolveAnswers(choice, q.Options)
		}
	}

	fmt.Fprintf(p.writer, "\n-> Allowed with answers\n")
	return BuildAskUserResponse(req, input, answers)
}

// promptExitPlanMode handles ExitPlanMode tool inputs with a dedicated plan approval prompt.
func (p *PlainPrompter) promptExitPlanMode(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	input, err := ParseExitPlanModeInput(req.ToolInputJson)
	if err != nil {
		fmt.Fprintf(p.writer, "  (Failed to parse ExitPlanMode input, falling back to standard prompt)\n")
		return p.promptStandard(ctx, req)
	}

	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Plan Approval\n")
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))

	if len(input.AllowedPrompts) > 0 {
		fmt.Fprintf(p.writer, "The plan requests the following permissions:\n")
		for _, ap := range input.AllowedPrompts {
			fmt.Fprintf(p.writer, "  [%s] %s\n", ap.Tool, ap.Prompt)
		}
	} else {
		fmt.Fprintf(p.writer, "The plan requests no additional permissions.\n")
	}

	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Options:\n")
	fmt.Fprintf(p.writer, "  [a] Allow  - Approve the plan\n")
	fmt.Fprintf(p.writer, "  [d] Deny   - Reject the plan\n")
	fmt.Fprintf(p.writer, "  [k] Ask    - Prompt user for confirmation\n")
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(p.writer, "Your choice [a/d/k]: ")

	scanner := bufio.NewScanner(p.reader)
	responseCh := make(chan string, 1)
	go func() {
		if scanner.Scan() {
			responseCh <- strings.TrimSpace(scanner.Text())
		} else {
			responseCh <- ""
		}
	}()

	var choice string
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case choice = <-responseCh:
	}

	var decision pb.PermissionDecision
	var reason string

	switch choice {
	case "a", "y", "yes", "allow":
		decision = pb.PermissionDecision_PERMISSION_DECISION_ALLOW
		fmt.Fprintf(p.writer, "-> Plan approved\n")
	case "d", "n", "no", "deny", "b", "block":
		decision = pb.PermissionDecision_PERMISSION_DECISION_DENY
		fmt.Fprintf(p.writer, "Reason (optional, press Enter to skip): ")
		if scanner.Scan() {
			reason = strings.TrimSpace(scanner.Text())
		}
		fmt.Fprintf(p.writer, "-> Plan rejected\n")
	case "k", "ask":
		decision = pb.PermissionDecision_PERMISSION_DECISION_ASK
		fmt.Fprintf(p.writer, "-> Ask (prompt for confirmation)\n")
	default:
		decision = pb.PermissionDecision_PERMISSION_DECISION_DENY
		reason = "Invalid choice - defaulting to deny"
		fmt.Fprintf(p.writer, "-> Invalid choice, defaulting to deny\n")
	}

	return BuildPermissionResponse(req, decision, reason), nil
}

// promptStandard is the standard permission prompt (fallback).
func (p *PlainPrompter) promptStandard(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Options:\n")
	fmt.Fprintf(p.writer, "  [a] Allow  - Allow this tool execution\n")
	fmt.Fprintf(p.writer, "  [d] Deny   - Deny this tool execution\n")
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(p.writer, "Your choice [a/d]: ")

	scanner := bufio.NewScanner(p.reader)
	responseCh := make(chan string, 1)
	go func() {
		if scanner.Scan() {
			responseCh <- strings.TrimSpace(scanner.Text())
		} else {
			responseCh <- ""
		}
	}()

	var choice string
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case choice = <-responseCh:
	}

	var decision pb.PermissionDecision
	switch choice {
	case "a", "y", "yes", "allow":
		decision = pb.PermissionDecision_PERMISSION_DECISION_ALLOW
		fmt.Fprintf(p.writer, "-> Allowed\n")
	default:
		decision = pb.PermissionDecision_PERMISSION_DECISION_DENY
		fmt.Fprintf(p.writer, "-> Denied\n")
	}

	return BuildPermissionResponse(req, decision, ""), nil
}
