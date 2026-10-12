package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// A channel is the transport the server runs its session on, kept hold of so
// the server can also push a notification of its own down it.
//
// It exists because the SDK serves the methods it knows and nothing else: a
// harness channel is a notification the SDK has no call for, and the only
// honest place to write one is the connection the SDK is already reading and
// writing. Wrapping the transport is what gets hold of that connection —
// [mcpsdk.Server.Connect] asks the transport for it and keeps it to itself
// otherwise.
//
// The connection is written to directly rather than through a second stream of
// its own. Its Write serialises against the traffic the SDK is writing, so a
// notification pushed mid-call lands between frames instead of inside one.
type channel struct {
	inner mcpsdk.Transport

	// mu guards conn, which the SDK's connect writes and a delivery reads. The
	// two happen on different goroutines: the delivery runs off the attendance
	// feed, which is up before the harness has finished connecting.
	mu   sync.Mutex
	conn mcpsdk.Connection
}

// newChannel wraps inner so the server can push its own notifications through
// whatever the SDK connects.
func newChannel(inner mcpsdk.Transport) *channel {
	return &channel{inner: inner}
}

// Connect implements [mcpsdk.Transport]. The connection is handed to the SDK
// unwrapped: everything this type adds is written alongside the SDK's own
// traffic, not layered over it.
func (c *channel) Connect(ctx context.Context) (mcpsdk.Connection, error) {
	conn, err := c.inner.Connect(ctx)
	if err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.conn = conn
	c.mu.Unlock()
	return conn, nil
}

// maxProtocolVersion is the newest MCP revision this server offers.
//
// The cap is there for Claude Code: a session that negotiated the revision
// after this one drops the channel events its server pushes, and the SDK offers
// that newer revision by default. Every harness crabswarm has been recorded
// speaking to asks for this revision anyway, so capping costs nothing and is
// the difference between a channel that delivers and one that is silently
// ignored.
const maxProtocolVersion = "2025-11-25"

// SupportsProtocolVersion implements [mcpsdk.ProtocolVersionSupporter], which
// is what the SDK filters its offered revisions through.
//
// The revisions are dates, so they sort as strings — the comparison the SDK
// makes of them itself.
func (c *channel) SupportsProtocolVersion(version string) bool {
	return version <= maxProtocolVersion
}

// discoverMethod is the handshake the revision above the cap replaces
// [mcpsdk.Server]'s initialize with.
const discoverMethod = "server/discover"

// refusesTheNewerHandshake answers [discoverMethod] as a method this server does
// not have, which is what sends a client back to the handshake the cap leaves it.
//
// The cap a transport declares reaches the revisions that handshake answers
// with and no further: the SDK serves the call either way, and serving it marks
// the session initialized. A client that read the capped list, found nothing it
// could use and fell back to initialize is then refused for initializing twice,
// and the session ends before it began. Refusing the call outright is what a
// client reads as a server older than itself, which is the fallback the spec
// describes.
func refusesTheNewerHandshake(next mcpsdk.MethodHandler) mcpsdk.MethodHandler {
	return func(
		ctx context.Context, method string, req mcpsdk.Request,
	) (mcpsdk.Result, error) {
		if method == discoverMethod {
			return nil, &jsonrpc.Error{
				Code: jsonrpc.CodeMethodNotFound,
				Message: discoverMethod + " is not served above protocol revision " +
					maxProtocolVersion,
			}
		}
		return next(ctx, method, req)
	}
}

// errNoChannelYet refuses a notification sent before the harness has connected.
// Nothing has been written to yet, so there is nowhere for it to go; the caller
// logs it and comes back to the notice on the next chance to deliver it.
var errNoChannelYet = errors.New(
	"the harness has not connected yet, so there is no channel to notify it through")

// Notify writes one JSON-RPC notification to the harness. It implements
// [harnessctl.Session], which is how a channel reaches its own harness.
func (c *channel) Notify(ctx context.Context, method string, params any) error {
	c.mu.Lock()
	conn := c.conn
	c.mu.Unlock()
	if conn == nil {
		return errNoChannelYet
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("encoding the params of %s: %w", method, err)
	}
	// A request carrying no id is a notification, which is what a channel event
	// is: the harness acts on it and answers nothing.
	if err := conn.Write(ctx, &jsonrpc.Request{Method: method, Params: raw}); err != nil {
		return fmt.Errorf("writing %s to the harness: %w", method, err)
	}
	return nil
}
