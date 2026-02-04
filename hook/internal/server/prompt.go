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

// Prompter handles interactive prompts for permission decisions.
type Prompter struct {
	reader io.Reader
	writer io.Writer
}

// NewPrompter creates a new Prompter with the given reader and writer.
func NewPrompter(reader io.Reader, writer io.Writer) *Prompter {
	return &Prompter{
		reader: reader,
		writer: writer,
	}
}

// Prompt displays the permission request and prompts the user for a decision.
func (p *Prompter) Prompt(ctx context.Context, req *pb.PermissionRequest) (*pb.PermissionResponse, error) {
	// Display the request information
	fmt.Fprintf(p.writer, "\n%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(p.writer, "Permission Request: %s\n", req.HookName)
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Tool:       %s\n", req.ToolName)
	fmt.Fprintf(p.writer, "Session:    %s\n", req.SessionId)
	fmt.Fprintf(p.writer, "Message:    %s\n", req.MessageId)
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

	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("-", 60))
	fmt.Fprintf(p.writer, "Options:\n")
	fmt.Fprintf(p.writer, "  [a] Allow         - Allow this tool execution\n")
	fmt.Fprintf(p.writer, "  [b] Block         - Block this tool execution\n")
	fmt.Fprintf(p.writer, "  [A] Allow Always  - Allow all similar requests\n")
	fmt.Fprintf(p.writer, "%s\n", strings.Repeat("=", 60))
	fmt.Fprintf(p.writer, "Your choice [a/b/A]: ")

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
	response := &pb.PermissionResponse{}
	switch choice {
	case "a", "y", "yes", "allow":
		response.Decision = pb.Decision_DECISION_ALLOW
		fmt.Fprintf(p.writer, "-> Allowed\n")
	case "b", "n", "no", "block":
		response.Decision = pb.Decision_DECISION_BLOCK
		fmt.Fprintf(p.writer, "Reason (optional, press Enter to skip): ")
		if scanner.Scan() {
			response.Reason = strings.TrimSpace(scanner.Text())
		}
		fmt.Fprintf(p.writer, "-> Blocked\n")
	case "A", "always", "allow_always":
		response.Decision = pb.Decision_DECISION_ALLOW_ALWAYS
		fmt.Fprintf(p.writer, "-> Allowed (always)\n")
	default:
		// Default to block for safety
		response.Decision = pb.Decision_DECISION_BLOCK
		response.Reason = "Invalid choice - defaulting to block"
		fmt.Fprintf(p.writer, "-> Invalid choice, defaulting to block\n")
	}

	return response, nil
}
