package commands

import (
	"encoding/json"
	"fmt"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/cmdman/v1"
)

func init() {
	rootCmd.AddCommand(inspectCmd)
}

var inspectCmd = &cobra.Command{
	Use:   "inspect ID|NAME",
	Short: "Show merged command definition, runtime state, and exit history",
	Args:  cobra.ExactArgs(1),
	RunE:  runInspect,
}

// InspectOutput is the JSON output of inspect.
type InspectOutput struct {
	ID          string                    `json:"id"`
	Name        string                    `json:"name,omitempty"`
	Config      *cmdman.CommandConfigJSON `json:"config"`
	State       string                    `json:"state"`
	ExitCode    *int                      `json:"exit_code,omitempty"`
	StateJSON   *cmdman.CommandStateJSON  `json:"state_detail"`
	ExitHistory []cmdman.ExitRecord       `json:"exit_history,omitempty"`
	ConfigPath  string                    `json:"config_path,omitempty"`
	LiveStatus  *LiveStatusInfo           `json:"live_status,omitempty"`
}

// LiveStatusInfo is the live status from the monitor gRPC Status RPC.
type LiveStatusInfo struct {
	State    string `json:"state"`
	ExitCode int32  `json:"exit_code"`
	PID      int32  `json:"pid"`
}

func runInspect(cmd *cobra.Command, args []string) error {
	store, err := cmdman.OpenStore(cmdman.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	id, name, cfg, err := store.GetCommandConfig(args[0])
	if err != nil {
		return fmt.Errorf("resolve command: %w", err)
	}

	state, exitCode, stateJSON, err := store.GetCommandState(id)
	if err != nil {
		return fmt.Errorf("get state: %w", err)
	}

	exitHistory, _ := store.GetExitHistory(id)

	out := InspectOutput{
		ID:          id,
		Name:        name,
		Config:      cfg,
		State:       state,
		ExitCode:    exitCode,
		StateJSON:   stateJSON,
		ExitHistory: exitHistory,
		ConfigPath:  cfg.ConfigPath(),
	}

	// Try to get live status from monitor.
	if stateJSON.SocketPath != "" {
		if live := getLiveStatus(cmd, stateJSON.SocketPath); live != nil {
			out.LiveStatus = live
		}
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func getLiveStatus(cmd *cobra.Command, sockPath string) *LiveStatusInfo {
	conn, err := grpc.NewClient(
		"unix://"+sockPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil
	}
	defer conn.Close()

	client := pb.NewCommandMonitorClient(conn)
	resp, err := client.Status(cmd.Context(), &pb.StatusRequest{})
	if err != nil {
		return nil
	}
	return &LiveStatusInfo{
		State:    resp.State,
		ExitCode: resp.ExitCode,
		PID:      resp.Pid,
	}
}
