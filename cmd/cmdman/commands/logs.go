package commands

import (
	"fmt"
	"io"
	"os"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/cmdman/v1"
)

func init() {
	rootCmd.AddCommand(logsCmd)
	logsCmd.Flags().BoolP("follow", "f", false, "Follow output")
}

var logsCmd = &cobra.Command{
	Use:   "logs [flags] ID|NAME",
	Short: "Show command output from scrollback buffer",
	Args:  cobra.ExactArgs(1),
	RunE:  runLogs,
}

func runLogs(cmd *cobra.Command, args []string) error {
	follow, _ := cmd.Flags().GetBool("follow")

	store, err := cmdman.OpenStore(cmdman.DBPath(), true)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	id, err := store.ResolveID(args[0])
	if err != nil {
		return fmt.Errorf("resolve command: %w", err)
	}

	_, _, stateJSON, err := store.GetCommandState(id)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	if stateJSON.SocketPath == "" {
		return fmt.Errorf("no socket path for command %s", id)
	}

	conn, err := grpc.NewClient(
		"unix://"+stateJSON.SocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return fmt.Errorf("connect to monitor: %w", err)
	}
	defer conn.Close()

	client := pb.NewCommandMonitorServiceClient(conn)
	stream, err := client.Logs(cmd.Context(), &pb.LogsRequest{Follow: follow})
	if err != nil {
		return fmt.Errorf("logs: %w", err)
	}

	for {
		msg, err := stream.Recv()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		os.Stdout.Write(msg.Data)
	}
}
