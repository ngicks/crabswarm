package chat

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"

	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// statusTimeout bounds one status write. [Service] mirrors inside the RPC that
// changed the state, so a cmdman that hangs would otherwise hang the caller's
// Join or ReportState.
const statusTimeout = 3 * time.Second

// statusDetail rides along with every published state to say who published it,
// so an operator reading a command's status can tell a chat-reported state from
// one the harness or they themselves set.
const statusDetail = "crabswarm chat"

// CmdmanStatusMirror publishes member state onto the cmdman command a member's
// token names: `cmdman status set <state> <token> --detail ...`, and
// `cmdman status delete <token>` once the member is gone.
//
// The vocabulary needs no translation. [MemberState] is spelled with the same
// three words cmdman's status takes — working, waiting, done — which is why the
// state reaches the CLI as itself rather than through a mapping table.
//
// Only agents are published. A human's token is minted by the daemon and names
// no command, so cmdman could only reject it; the guard is what keeps a
// daemon-issued secret off a cmdman command line.
type CmdmanStatusMirror struct {
	bin    string
	logger *slog.Logger
}

var _ StatusMirror = (*CmdmanStatusMirror)(nil)

// NewCmdmanStatusMirror returns a mirror that shells out to the cmdman binary
// named by bin. An empty bin means "cmdman", resolved on PATH, the same default
// the token resolver uses; a nil logger discards logs.
func NewCmdmanStatusMirror(bin string, logger *slog.Logger) *CmdmanStatusMirror {
	if bin == "" {
		bin = resolver.DefaultCmdmanBin
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &CmdmanStatusMirror{bin: bin, logger: logger}
}

// Set publishes state as the status of m's command. A member cmdman cannot be
// told about is skipped rather than reported as an error: there is nothing
// wrong, the member simply has no harness whose state a display would mean
// anything about.
func (m *CmdmanStatusMirror) Set(ctx context.Context, member Member, state MemberState) error {
	if !m.publishable(member, "publish state for") {
		return nil
	}
	ctx, cancel := m.detach(ctx)
	defer cancel()

	return m.run(ctx, "set", string(state), member.Token, "--detail", statusDetail)
}

// Clear withdraws the status of m's command, which is what keeps a departed
// member from sitting in an operator's display forever.
func (m *CmdmanStatusMirror) Clear(ctx context.Context, member Member) error {
	if !m.publishable(member, "withdraw state for") {
		return nil
	}
	ctx, cancel := m.detach(ctx)
	defer cancel()

	return m.run(ctx, "delete", member.Token)
}

// publishable reports whether cmdman can be told about member at all, logging
// why not. what names the attempted action, so the log line reads as a
// sentence.
func (m *CmdmanStatusMirror) publishable(member Member, what string) bool {
	who := member.Team + "/" + member.Name

	// Not "== KindHuman": a member kind this mirror has never heard of has no
	// harness state to label a command with either. A member that declared no
	// harness reports none, so its command would only ever show the state it
	// was admitted in.
	if member.Kind != KindAgent {
		m.logger.Debug("chat: not asked to "+what+" a member that runs no command",
			"member", who, "kind", member.Kind)
		return false
	}
	if err := resolver.ValidateToken(member.Token); err != nil {
		m.logger.Warn("chat: not asked to "+what+" a member whose token cmdman cannot take",
			"member", who, "err", err)
		return false
	}
	return true
}

// detach separates the status write from the RPC that triggered it. A client
// that gives up mid-call still changed the store, so the display has to catch
// up regardless; the timeout replaces the cancellation the RPC context would
// have provided.
func (m *CmdmanStatusMirror) detach(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), statusTimeout)
}

// run invokes `cmdman status <args...>`.
func (m *CmdmanStatusMirror) run(ctx context.Context, args ...string) error {
	argv := append([]string{"status"}, args...)
	out, err := exec.CommandContext(ctx, m.bin, argv...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cmdman %s: %w: %s",
			strings.Join(argv, " "), err, strings.TrimSpace(string(out)))
	}
	return nil
}
