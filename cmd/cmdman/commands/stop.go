package commands

import (
	"fmt"
	"strings"
	"syscall"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/cmdman/v1"
)

func init() {
	rootCmd.AddCommand(stopCmd)
	stopCmd.Flags().StringArrayP("label", "l", nil, "Target commands matching labels")
	stopCmd.Flags().StringP("signal", "s", "SIGTERM", "Signal to send")
}

var stopCmd = &cobra.Command{
	Use:   "stop [flags] [ID|NAME]",
	Short: "Send signal to a running command",
	RunE:  runStop,
}

func runStop(cmd *cobra.Command, args []string) error {
	sigName, _ := cmd.Flags().GetString("signal")
	labelSlice, _ := cmd.Flags().GetStringArray("label")

	sig := parseSignal(sigName)

	store, err := cmdman.OpenStore(cmdman.DBPath(), true)
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	ids, err := resolveTargets(store, args, labelSlice)
	if err != nil {
		return err
	}

	for _, id := range ids {
		if err := stopOne(cmd, store, id, sig); err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "stop %s: %v\n", id, err)
		}
	}
	return nil
}

func stopOne(cmd *cobra.Command, store *cmdman.Store, id string, sig int32) error {
	_, _, stateJSON, err := store.GetCommandState(id)
	if err != nil {
		return err
	}
	if stateJSON.SocketPath == "" {
		return fmt.Errorf("no socket path")
	}

	conn, err := grpc.NewClient(
		"unix://"+stateJSON.SocketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return err
	}
	defer conn.Close()

	client := pb.NewCommandMonitorClient(conn)
	_, err = client.Signal(cmd.Context(), &pb.SignalRequest{Signal: sig})
	return err
}

func parseSignal(s string) int32 {
	s = strings.ToUpper(s)
	s = strings.TrimPrefix(s, "SIG")
	switch s {
	case "HUP":
		return int32(syscall.SIGHUP)
	case "INT":
		return int32(syscall.SIGINT)
	case "QUIT":
		return int32(syscall.SIGQUIT)
	case "KILL":
		return int32(syscall.SIGKILL)
	case "TERM":
		return int32(syscall.SIGTERM)
	case "USR1":
		return int32(syscall.SIGUSR1)
	case "USR2":
		return int32(syscall.SIGUSR2)
	default:
		return int32(syscall.SIGTERM)
	}
}

// resolveTargets resolves command IDs from args and/or labels.
func resolveTargets(store *cmdman.Store, args []string, labelSlice []string) ([]string, error) {
	var ids []string

	for _, a := range args {
		id, err := store.ResolveID(a)
		if err != nil {
			return nil, fmt.Errorf("resolve %q: %w", a, err)
		}
		ids = append(ids, id)
	}

	labels, err := parseLabels(labelSlice)
	if err != nil {
		return nil, err
	}
	if len(labels) > 0 {
		labelIDs, err := store.FindByLabels(labels)
		if err != nil {
			return nil, fmt.Errorf("find by labels: %w", err)
		}
		ids = append(ids, labelIDs...)
	}

	if len(ids) == 0 {
		return nil, fmt.Errorf("no commands specified")
	}
	return ids, nil
}
