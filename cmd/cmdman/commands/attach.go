package commands

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
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

// errDetached is returned by escapeReader when the detach key sequence is detected.
var errDetached = errors.New("detached")

func runAttach(cmd *cobra.Command, idOrName string) error {
	noStdin, _ := cmd.Flags().GetBool("no-stdin")
	sigProxy, _ := cmd.Flags().GetBool("sig-proxy")
	detachKeysStr, _ := cmd.Flags().GetString("detach-keys")

	detachKeys, err := parseDetachKeys(detachKeysStr)
	if err != nil {
		return fmt.Errorf("invalid detach-keys: %w", err)
	}

	store, err := cmdman.OpenStore(cmdman.DBPath(), true)
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

	// Read stdin -> stream, with detach key detection.
	if !noStdin {
		go func() {
			var r io.Reader = os.Stdin
			if len(detachKeys) > 0 {
				r = &escapeReader{r: os.Stdin, keys: detachKeys}
			}
			buf := make([]byte, 32*1024)
			for {
				n, err := r.Read(buf)
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
		if err == io.EOF || errors.Is(err, errDetached) {
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

// escapeReader wraps a reader and detects a detach key sequence.
// When the full sequence is matched, Read returns errDetached.
// Partial matches are buffered and flushed if the sequence breaks.
type escapeReader struct {
	r    io.Reader
	keys []byte
	pos  int    // position in the escape sequence
	buf  []byte // buffered partial match bytes
}

func (e *escapeReader) Read(p []byte) (int, error) {
	// Flush any buffered partial-match bytes first.
	if len(e.buf) > 0 {
		n := copy(p, e.buf)
		e.buf = e.buf[n:]
		return n, nil
	}

	nr, err := e.r.Read(p)
	if nr == 0 {
		return 0, err
	}

	// Scan read bytes for the escape sequence.
	out := 0
	for i := 0; i < nr; i++ {
		b := p[i]
		if b == e.keys[e.pos] {
			e.pos++
			if e.pos == len(e.keys) {
				// Full escape sequence detected.
				// Return any output accumulated before the sequence,
				// then errDetached on the next call.
				if out > 0 {
					e.pos = 0
					// Store errDetached for the next Read call
					// by keeping pos at len(keys)... actually,
					// simpler: return what we have and signal detach.
					// We need to return the detach on *this* call if no output.
				}
				return out, errDetached
			}
		} else if e.pos > 0 {
			// Partial match broken. Flush the buffered escape prefix.
			// The bytes that matched so far need to be output.
			prefix := e.keys[:e.pos]
			e.pos = 0

			// Check if current byte starts a new match.
			if b == e.keys[0] {
				e.pos = 1
			}

			// Output the prefix + (if not starting new match) current byte.
			remaining := p[out:]
			n := copy(remaining, prefix)
			out += n
			if n < len(prefix) {
				// Not enough space in p. Buffer the rest.
				e.buf = append(e.buf, prefix[n:]...)
				if e.pos == 0 {
					e.buf = append(e.buf, b)
				}
				// Also buffer unprocessed input.
				e.buf = append(e.buf, p[i+1:nr]...)
				return out, err
			}
			if e.pos == 0 {
				if out < len(p) {
					p[out] = b
					out++
				} else {
					e.buf = append(e.buf, b)
					e.buf = append(e.buf, p[i+1:nr]...)
					return out, err
				}
			}
		} else {
			p[out] = b
			out++
		}
	}

	return out, err
}

// parseDetachKeys parses a detach key sequence string like "ctrl-p,ctrl-q"
// into a byte slice.
func parseDetachKeys(s string) ([]byte, error) {
	if s == "" {
		return nil, nil
	}
	parts := strings.Split(s, ",")
	keys := make([]byte, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "ctrl-") {
			ch := strings.TrimPrefix(part, "ctrl-")
			if len(ch) != 1 {
				return nil, fmt.Errorf("invalid ctrl sequence: %s", part)
			}
			// ctrl-a = 0x01, ctrl-z = 0x1a
			b := ch[0]
			if b >= 'a' && b <= 'z' {
				keys = append(keys, b-'a'+1)
			} else if b >= 'A' && b <= 'Z' {
				keys = append(keys, b-'A'+1)
			} else {
				return nil, fmt.Errorf("invalid ctrl sequence: %s", part)
			}
		} else if len(part) == 1 {
			keys = append(keys, part[0])
		} else {
			return nil, fmt.Errorf("invalid key: %s", part)
		}
	}
	return keys, nil
}
