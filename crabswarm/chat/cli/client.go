// Package cli is the caller's half of the chat broker: it dials the daemon
// socket, carries the caller's credential on every RPC, and renders what comes
// back as plain text that an agent can parse as easily as a human can read it.
//
// Everything the `crabswarm chat` subcommands do lives here; the ./cmd layer
// only parses flags and hands off. The rendering functions take the generated
// chat types directly rather than a view model of their own: this package
// already speaks the wire schema to make the call in the first place, so a
// second copy of every message shape would buy nothing.
package cli

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat"
)

// Client talks to the chat broker the crabswarm daemon hosts on its Unix
// socket. It holds both halves of the schema: the member-facing ChatService,
// authenticated per call by an identity token, and the host-only
// ChatAdminService, authenticated per call by a decrypted challenge nonce.
type Client struct {
	conn  *grpc.ClientConn
	chat  chatv1.ChatServiceClient
	admin chatv1.ChatAdminServiceClient
}

// Dial returns a client for the daemon listening on the Unix socket sockPath.
// It does not connect: grpc.NewClient is lazy, so an unreachable daemon
// surfaces on the first RPC as [ErrDaemonUnreachable] rather than here.
func Dial(sockPath string) (*Client, error) {
	if sockPath == "" {
		return nil, errors.New("no crabswarm socket path configured")
	}
	conn, err := grpc.NewClient(
		"unix://"+sockPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, fmt.Errorf("connecting to the crabswarm daemon at %q: %w", sockPath, err)
	}
	return newClient(conn), nil
}

// newClient wraps an established connection. Dial builds the socket one; tests
// pass a connection to an in-process server.
func newClient(conn *grpc.ClientConn) *Client {
	return &Client{
		conn:  conn,
		chat:  chatv1.NewChatServiceClient(conn),
		admin: chatv1.NewChatAdminServiceClient(conn),
	}
}

// Close releases the connection.
func (c *Client) Close() error {
	return c.conn.Close()
}

// withToken attaches the caller's identity to an outgoing call.
//
// This is the client-side counterpart of the daemon's token interceptor, and
// deliberately not [chat.ContextWithToken]: that one stores a context value for
// code calling the service in-process, while a real RPC has to put the token in
// the request metadata for the interceptor to find it.
func withToken(ctx context.Context, token string) context.Context {
	return metadata.AppendToOutgoingContext(ctx, chat.TokenMetadataKey, token)
}

// ErrDaemonUnreachable reports that no daemon answered on the socket. It is
// what every RPC in this package returns when the transport fails, so a caller
// can tell "nobody is listening" from an answer it did not like.
var ErrDaemonUnreachable = errors.New("chat daemon unreachable")

// rpcError presents a gRPC failure the way a CLI user wants to read it — the
// server's message alone, without the "rpc error: code = ..." envelope — while
// keeping the original error reachable through errors.Is/As, so a caller can
// still inspect the status code.
//
// The daemon's messages are written for this audience: an ambiguous address
// names the qualified form to retry with, an unknown member names the room. The
// envelope would only bury them.
type rpcError struct {
	msg string
	err error
}

func (e *rpcError) Error() string { return e.msg }

func (e *rpcError) Unwrap() error { return e.err }

// callError maps an RPC failure onto the error the CLI reports. An unavailable
// daemon gets the hint that starts one; anything else keeps the server's own
// wording.
func callError(err error) error {
	if err == nil {
		return nil
	}
	st, ok := status.FromError(err)
	if !ok {
		return err
	}
	if st.Code() == codes.Unavailable {
		return &rpcError{
			msg: fmt.Sprintf("%s: %s\nhint: start it by running `crabswarm serve`",
				ErrDaemonUnreachable, st.Message()),
			err: fmt.Errorf("%w: %w", ErrDaemonUnreachable, err),
		}
	}
	return &rpcError{msg: st.Message(), err: err}
}
