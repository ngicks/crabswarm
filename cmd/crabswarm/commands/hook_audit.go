package commands

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ngicks/crabswarm/api/gen/proto/go/crabhook/v1"
	"github.com/ngicks/crabswarm/crabswarm"
	crabswarmhook "github.com/ngicks/crabswarm/crabswarm/hook"
	"github.com/ngicks/crabswarm/internal/stdiopipe"
)

func hookAuditCmd(parent *cobra.Command, flagSock, flagConfig *string) {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Forward Claude Code hook events to the crabswarm server",
		Long: "A hook for claude code's PreToolUse event. " +
			"Sends all hook inputs to backend crabswarm server so we can audit them.",
		Args:              cobra.NoArgs,
		ValidArgsFunction: cobra.NoFileCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runHookAudit(cmd, args, *flagSock, *flagConfig)
		},
	}

	parent.AddCommand(cmd)
}

func runHookAudit(cmd *cobra.Command, _ []string, flagSock, flagConfig string) error {
	ctx := cmd.Context()

	reader := stdiopipe.Stdin(ctx)
	defer reader.Close()

	cfg, err := crabswarm.LoadConfig(flagConfig)
	if err != nil {
		return err
	}
	if cmd.Flags().Changed("sock") {
		cfg.Sock = flagSock
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
