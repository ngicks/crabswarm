package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/ngicks/crabswarm/pkg/cmdman"
	"github.com/spf13/cobra"
	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/ngicks/crabswarm/pkg/api/gen/proto/go/cmdman/v1"
)

func init() {
	rootCmd.AddCommand(attachCmd)
	attachCmd.Flags().Bool("no-stdin", false, "Output-only mode")
	attachCmd.Flags().Bool("sig-proxy", true, "Forward signals to command")
	attachCmd.Flags().String("detach-keys", "ctrl-p,ctrl-q", "Key sequence to detach")
}

var attachCmd = &cobra.Command{
	Use:   "attach [flags] ID|NAME",
	Short: "Attach to a running command's PTY",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAttach(cmd, args[0])
	},
}

// signals forwarded to the remote command during attach.
var forwardedSignals = []os.Signal{
	syscall.SIGINT,
	syscall.SIGTERM,
	syscall.SIGHUP,
	syscall.SIGQUIT,
	syscall.SIGUSR1,
	syscall.SIGUSR2,
	syscall.SIGTSTP,
	syscall.SIGCONT,
	syscall.SIGWINCH,
}

func runAttach(cmd *cobra.Command, idOrName string) error {
	noStdin, _ := cmd.Flags().GetBool("no-stdin")
	sigProxy, _ := cmd.Flags().GetBool("sig-proxy")

	store, err := cmdman.OpenStore(cmdman.DBPath())
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer store.Close()

	id, err := store.ResolveID(idOrName)
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

	client := pb.NewCommandMonitorClient(conn)
	ctx := cmd.Context()

	stream, err := client.Attach(ctx)
	if err != nil {
		return fmt.Errorf("attach: %w", err)
	}

	// Put terminal into raw mode so keystrokes pass through to the remote PTY.
	var restoreTerminal func()
	if !noStdin {
		fd := int(os.Stdin.Fd())
		if term.IsTerminal(fd) {
			oldState, err := term.MakeRaw(fd)
			if err == nil {
				restoreTerminal = func() { term.Restore(fd, oldState) }
				defer restoreTerminal()
			}
		}
	}

	// Take over all signal handling. signal.Reset() undoes any prior
	// signal.Notify registrations (including main.go's NotifyContext),
	// giving us full control.
	signal.Reset()

	// Send initial terminal size.
	sendResize(stream)

	// HandleAllSignals: forward signals to remote command, handle SIGWINCH
	// locally as resize, and force-exit after 3 consecutive SIGINT/SIGTERM.
	if sigProxy {
		sigCh := make(chan os.Signal, 4)
		signal.Notify(sigCh, forwardedSignals...)
		defer signal.Stop(sigCh)

		go handleAllSignals(ctx, sigCh, client, stream, restoreTerminal)
	}

	// Read from stream -> stdout.
	errCh := make(chan error, 2)
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			os.Stdout.Write(msg.Stdout)
		}
	}()

	// Read stdin -> stream.
	if !noStdin {
		go func() {
			buf := make([]byte, 32*1024)
			for {
				n, err := os.Stdin.Read(buf)
				if n > 0 {
					data := make([]byte, n)
					copy(data, buf[:n])
					if sendErr := stream.Send(&pb.AttachInput{
						Input: &pb.AttachInput_Stdin{Stdin: data},
					}); sendErr != nil {
						errCh <- sendErr
						return
					}
				}
				if err != nil {
					if err != io.EOF {
						errCh <- err
					}
					return
				}
			}
		}()
	}

	// Wait for either direction to finish.
	select {
	case err := <-errCh:
		if err == io.EOF {
			return nil
		}
		return err
	case <-ctx.Done():
		return nil
	}
}

// handleAllSignals processes signals during attach:
//   - SIGWINCH → send resize event
//   - SIGINT/SIGTERM → forward to remote; after 3 consecutive, force exit with terminal restore
//   - All others → forward to remote
func handleAllSignals(
	ctx context.Context,
	sigCh <-chan os.Signal,
	client pb.CommandMonitorClient,
	stream pb.CommandMonitor_AttachClient,
	restoreTerminal func(),
) {
	forceCount := 0
	for {
		select {
		case sig, ok := <-sigCh:
			if !ok {
				return
			}
			sigNum, ok := sig.(syscall.Signal)
			if !ok {
				continue
			}

			if sigNum == syscall.SIGWINCH {
				sendResize(stream)
				forceCount = 0
				continue
			}

			// Forward to remote command.
			client.Signal(ctx, &pb.SignalRequest{Signal: int32(sigNum)})

			// Count consecutive SIGINT/SIGTERM for forced exit.
			if sigNum == syscall.SIGINT || sigNum == syscall.SIGTERM {
				forceCount++
				if forceCount >= 3 {
					// Force exit: restore terminal before exiting.
					if restoreTerminal != nil {
						restoreTerminal()
					}
					os.Exit(1)
				}
			} else {
				forceCount = 0
			}

		case <-ctx.Done():
			return
		}
	}
}

func sendResize(stream pb.CommandMonitor_AttachClient) {
	rows, cols := getTerminalSize()
	if rows > 0 && cols > 0 {
		stream.Send(&pb.AttachInput{
			Input: &pb.AttachInput_Resize{
				Resize: &pb.ResizeEvent{
					Rows: uint32(rows),
					Cols: uint32(cols),
				},
			},
		})
	}
}

func getTerminalSize() (rows, cols int) {
	return getTerminalSizeImpl()
}
