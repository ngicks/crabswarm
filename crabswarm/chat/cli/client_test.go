package cli

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
)

// The token has to travel as request metadata: the daemon reads it there and
// nowhere else, so a context value would arrive as no token at all.
func TestClient_CarriesTokenAsMetadata(t *testing.T) {
	fake := &fakeChatService{members: []*chatv1.Member{member("backend", "alice", "/work")}}
	d := serveTestDaemon(t, fake, nil)

	_, err := d.client.Members(t.Context(), "tok-a")
	assert.NilError(t, err)
	assert.DeepEqual(t, d.seenTokens(), []string{"tok-a"})
}

// A caller with no token never gets past the interceptor, and the refusal is
// reported in the daemon's own words. The streaming half is checked too: it has
// an interceptor of its own, and attendance is the one call that would be
// served by a server that forgot to install it.
func TestClient_EmptyTokenIsRejected(t *testing.T) {
	for _, tc := range []struct {
		name string
		call func(c *Client) error
	}{
		{"unary", func(c *Client) error {
			_, err := c.Members(t.Context(), "")
			return err
		}},
		{"stream", func(c *Client) error {
			_, err := c.Attend(t.Context(), "", "alice",
				chatv1.MemberKind_MEMBER_KIND_HUMAN)
			return err
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			d := serveTestDaemon(t, &fakeChatService{}, nil)

			err := tc.call(d.client)
			assert.Assert(t, err != nil)
			assert.Assert(t, strings.Contains(err.Error(), chat.TokenMetadataKey))
			assert.Equal(t, status.Code(errors.Unwrap(err)), codes.Unauthenticated)
		})
	}
}

// An error the daemon returns is surfaced as the message it wrote, without the
// "rpc error: code = ..." envelope: an ambiguous address names the qualified
// form to retry with, and that sentence is the whole point of the error.
func TestClient_SurfacesServerMessageVerbatim(t *testing.T) {
	const msg = `"alice" is attended by teams "backend" and "frontend": address one as "team/name"`
	fake := &fakeChatService{err: status.Error(codes.InvalidArgument, msg)}
	d := serveTestDaemon(t, fake, nil)

	target, err := ParseTarget("alice")
	assert.NilError(t, err)
	_, err = d.client.Send(t.Context(), "tok-a", target, "hi")
	assert.Assert(t, err != nil)
	assert.Equal(t, err.Error(), msg)
	assert.Equal(t, status.Code(errors.Unwrap(err)), codes.InvalidArgument)
}

// A running daemon answers Unavailable when its team-info provider could not be
// asked, and that is not the daemon being absent: telling the operator to start
// one would send them to restart what is already running. The answer reaches
// them as the daemon wrote it.
func TestClient_ProviderUnavailableIsNotTheDaemonBeingDown(t *testing.T) {
	const msg = chat.ProviderUnavailableMessage + ": cmdman: connection refused"
	fake := &fakeChatService{err: status.Error(codes.Unavailable, msg)}
	d := serveTestDaemon(t, fake, nil)

	_, err := d.client.Read(t.Context(), "tok-a", nil)
	assert.Assert(t, err != nil)
	assert.Equal(t, err.Error(), msg)
	assert.Assert(t, !errors.Is(err, ErrDaemonUnreachable))
	assert.Equal(t, status.Code(errors.Unwrap(err)), codes.Unavailable)
}

// Nothing listening on the socket is a different failure from a refused
// request, and the CLI says how to fix it.
func TestClient_UnreachableDaemonHint(t *testing.T) {
	client, err := Dial(filepath.Join(t.TempDir(), "absent.sock"))
	assert.NilError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	err = client.ReadInto(t.Context(), &strings.Builder{}, "tok-a", ReadOptions{})
	assert.ErrorIs(t, err, ErrDaemonUnreachable)
	assert.Assert(t, strings.Contains(err.Error(), "crabswarm serve"))
}

func TestDial_RejectsEmptySocketPath(t *testing.T) {
	_, err := Dial("")
	assert.Assert(t, err != nil)
}

// daemonDelay is how long the daemon in the case below takes to arrive: long
// enough that the client has failed to reach it at least once and is waiting
// between attempts of its own when it does, short enough not to slow the
// package's tests.
const daemonDelay = 500 * time.Millisecond

// A client dialled at a socket nothing is listening on reaches the daemon that
// turns up later, over the connection it already has. That is the shape every
// long-lived caller here is in: a bridge is started by its harness before the
// daemon is up and holds one client for the whole session, so the connection
// behind it has to become usable on its own rather than by being dialled again.
//
// The call is retried rather than left to wait. A call made while nothing is
// listening is refused at once — that immediacy is what lets the CLI answer a
// missing daemon with the line that says to start one — so what is watched here
// is the connection underneath recovering, which is the half a caller cannot do
// anything about.
func TestClient_ReachesADaemonThatArrivesLate(t *testing.T) {
	sock := filepath.Join(t.TempDir(), "chat.sock")
	client, err := Dial(sock)
	assert.NilError(t, err)
	t.Cleanup(func() { _ = client.Close() })

	srv := grpc.NewServer(grpc.ChainUnaryInterceptor(chat.UnaryTokenInterceptor()))
	chatv1.RegisterChatServiceServer(srv, &fakeChatService{})

	// Generous against the wait the client may be sitting in when the socket
	// appears; a client that gave up entirely fails here rather than hanging.
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()

	var serving errgroup.Group
	serving.Go(func() error {
		select {
		case <-time.After(daemonDelay):
		case <-ctx.Done():
			return nil
		}
		lis, err := net.Listen("unix", sock)
		if err != nil {
			return fmt.Errorf("listening on %s: %w", sock, err)
		}
		// Serve ends when the cleanup below stops the server, which is the end
		// of the case rather than a failure of it.
		_ = srv.Serve(lis)
		return nil
	})
	t.Cleanup(func() {
		cancel()
		srv.Stop()
		assert.NilError(t, serving.Wait())
	})

	var last error
	for ctx.Err() == nil {
		if _, last = client.Read(ctx, "tok-a", nil); last == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("the client never reached the daemon; the last attempt said: %v", last)
}
