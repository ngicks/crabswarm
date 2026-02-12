// Package internal contains the cobra commands for crabhook.
package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	pb "github.com/ngicks/crabswarm/hook/api/gen/go/permission/v1"
	"github.com/ngicks/crabswarm/hook/model"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/timestamppb"
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
		HookEventName:  string(input.HookEventName),
		ToolName:       string(input.ToolName),
		ToolInputJson:  toolInputJSON,
		SessionId:      input.SessionID,
		MessageId:      input.MessageID,
		Cwd:            input.Cwd,
		TranscriptPath: input.TranscriptPath,
	}

	// Send the request
	resp, err := client.RequestPermission(ctx, req)
	if err != nil {
		return fmt.Errorf("permission request failed: %w", err)
	}

	// Convert the gRPC response to hook output
	output := pbResponseToHookOutput(resp, input.HookEventName)

	// Write the output to stdout
	encoder := json.NewEncoder(os.Stdout)
	if err := encoder.Encode(output); err != nil {
		return fmt.Errorf("failed to encode hook output: %w", err)
	}

	// Send audit event (best-effort, failures only logged to stderr)
	sendAuditEvent(client, req)

	return nil
}

// pbResponseToHookOutput converts a protobuf PermissionResponse to a model.HookOutput.
func pbResponseToHookOutput(resp *pb.PermissionResponse, eventName model.HookEventName) model.HookOutput {
	output := model.HookOutput{
		SuppressOutput: resp.SuppressOutput,
		SystemMessage:  resp.SystemMessage,
	}

	if !resp.ShouldContinue {
		f := false
		output.Continue = &f
		output.StopReason = resp.StopReason
	}

	if hso := resp.HookSpecificOutput; hso != nil {
		hookEventName := model.HookEventName(hso.HookEventName)
		if hookEventName == "" {
			hookEventName = eventName
		}
		specific := &model.HookSpecificOutput{
			HookEventName:           hookEventName,
			PermissionDecision:      pbDecisionToModel(hso.PermissionDecision),
			PermissionDecisionReason: hso.PermissionDecisionReason,
			AdditionalContext:       hso.AdditionalContext,
		}
		if hso.UpdatedInputJson != "" {
			specific.UpdatedInput = json.RawMessage(hso.UpdatedInputJson)
		}
		output.HookSpecificOutput = specific
	}

	return output
}

// pbDecisionToModel converts a protobuf PermissionDecision to model.PermissionDecision.
func pbDecisionToModel(d pb.PermissionDecision) model.PermissionDecision {
	switch d {
	case pb.PermissionDecision_PERMISSION_DECISION_ALLOW:
		return model.PermissionAllow
	case pb.PermissionDecision_PERMISSION_DECISION_DENY:
		return model.PermissionDeny
	case pb.PermissionDecision_PERMISSION_DECISION_ASK:
		return model.PermissionAsk
	default:
		return model.PermissionDeny
	}
}

// sendAuditEvent sends a single audit event to the server. This is best-effort;
// failures are logged to stderr but do not affect the hook outcome.
func sendAuditEvent(client pb.PermissionServiceClient, req *pb.PermissionRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	stream, err := client.Audit(ctx)
	if err != nil {
		slog.Warn("failed to open audit stream", "error", err)
		return
	}

	event := &pb.AuditEvent{
		Request:   req,
		Timestamp: timestamppb.Now(),
	}

	if err := stream.Send(event); err != nil {
		slog.Warn("failed to send audit event", "error", err)
		return
	}

	if _, err := stream.CloseAndRecv(); err != nil {
		slog.Warn("audit stream close error", "error", err)
	}
}
