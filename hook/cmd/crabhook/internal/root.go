// Package internal contains the cobra commands for crabhook.
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/model"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	serverAddr string
	timeout    time.Duration
)

// rootCmd is the root command (hook client mode).
var rootCmd = &cobra.Command{
	Use:   "crabhook",
	Short: "Claude Code permission hook handler",
	Long: `crabhook is a Claude Code hook handler that forwards permission requests
to an interactive server for user approval.

When run without a subcommand, it acts as a hook client:
- Reads hook input from stdin (JSON)
- Sends a gRPC request to the permission server
- Writes the decision to stdout (JSON)

Use 'crabhook serve' to start the interactive permission server.`,
	RunE: runHookClient,
}

func init() {
	rootCmd.Flags().StringVarP(&serverAddr, "server", "s", "localhost:50051", "Permission server address")
	rootCmd.Flags().DurationVarP(&timeout, "timeout", "t", 5*time.Minute, "Request timeout")
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

// runHookClient is the main logic for the hook client mode.
func runHookClient(cmd *cobra.Command, args []string) error {
	// Read hook input from stdin
	var input model.HookInput
	decoder := json.NewDecoder(os.Stdin)
	if err := decoder.Decode(&input); err != nil {
		return fmt.Errorf("failed to decode hook input: %w", err)
	}

	// Connect to the permission server
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	conn, err := grpc.NewClient(serverAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("failed to connect to server: %w", err)
	}
	defer conn.Close()

	client := pb.NewPermissionServiceClient(conn)

	// Build the permission request
	toolInputJSON := ""
	if input.ToolInput != nil {
		toolInputJSON = string(input.ToolInput)
	}

	req := &pb.PermissionRequest{
		HookName:      input.HookName,
		ToolName:      input.ToolName,
		ToolInputJson: toolInputJSON,
		SessionId:     input.SessionID,
		MessageId:     input.MessageID,
	}

	// Send the request
	resp, err := client.RequestPermission(ctx, req)
	if err != nil {
		return fmt.Errorf("permission request failed: %w", err)
	}

	// Convert the response to hook output
	output := model.HookOutput{
		Decision: decisionToString(resp.Decision),
		Reason:   resp.Reason,
	}

	// Write the output to stdout
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode hook output: %w", err)
	}

	return nil
}

// decisionToString converts a protobuf Decision to a string for the hook output.
func decisionToString(d pb.Decision) string {
	switch d {
	case pb.Decision_DECISION_ALLOW:
		return model.DecisionAllow
	case pb.Decision_DECISION_BLOCK:
		return model.DecisionBlock
	case pb.Decision_DECISION_ALLOW_ALWAYS:
		return model.DecisionAllowAlways
	default:
		return model.DecisionBlock
	}
}
