package commands

import (
	"fmt"
	"io"
	"os"
	"time"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/claude_hook/v1"
	sdktypes "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/sdk_types/v1"
	"github.com/ngicks/crabswarm/pkg/claudehook/handler"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func init() {
	hookCmd.AddCommand(hookAuditCmd)
}

// hookAuditCmd is the audit subcommand under hook.
var hookAuditCmd = &cobra.Command{
	Use:   "audit",
	Short: "A hook for claude code's PreToolUse event. Sends all hook inputs to backend crabswarm server so we can audit them",
	RunE:  runHookAuditCmd,
}

func runHookAuditCmd(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	raw, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("reading stdin: %w", err)
	}

	var input sdktypes.PreToolUseHookInput
	if err := protojson.Unmarshal(raw, &input); err != nil {
		return fmt.Errorf("parsing PreToolUseHookInput: %w", err)
	}

	sockPath := resolveSocketPath(cmd)

	conn, err := grpc.NewClient(
		"unix://"+sockPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connecting to server: %w", err)
	}
	defer conn.Close()

	client := pb.NewAuditServiceClient(conn)

	_, err = client.SendAuditEvent(ctx, &pb.AuditRequest{
		Input:     &input,
		Timestamp: timestamppb.New(time.Now()),
	})
	if err != nil {
		return fmt.Errorf("sending audit event: %w", err)
	}

	return &handler.HandlerError{}
}
