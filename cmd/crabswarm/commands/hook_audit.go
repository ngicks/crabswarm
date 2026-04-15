package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/crabhook/v1"
	crabswarmhook "github.com/ngicks/crabswarm/pkg/crabswarm/hook"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
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

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	sockPath := resolveSocketPath(cmd)
	_ = os.MkdirAll(filepath.Dir(sockPath), fs.ModePerm)

	conn, err := grpc.NewClient(
		"unix://"+sockPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connecting to server: %w", err)
	}
	defer conn.Close()

	client := pb.NewAuditServiceClient(conn)

	return crabswarmhook.Audit(ctx, reader, client)
}
