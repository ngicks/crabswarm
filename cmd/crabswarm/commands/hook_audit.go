package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/ngicks/crabswarm/cmd/internal/stdiopipe"
	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/crabhook/v1"
	"github.com/ngicks/crabswarm/pkg/crabswarm"
	crabswarmhook "github.com/ngicks/crabswarm/pkg/crabswarm/hook"
)

func hookAuditCmd(parent *cobra.Command, flagSock *string) {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Forward Claude Code hook events to the crabswarm server",
		Long: "A hook for claude code's PreToolUse event. " +
			"Sends all hook inputs to backend crabswarm server so we can audit them.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHookAudit(cmd, args, *flagSock)
		},
	}

	parent.AddCommand(cmd)
}

func runHookAudit(cmd *cobra.Command, _ []string, flagSock string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	cfg, err := crabswarm.Config{Sock: flagSock}.Load()
	if err != nil {
		return err
	}
	sockPath := cfg.Sock
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
