package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/jsonrpc"
	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/sync/errgroup"
	"gotest.tools/v3/assert"
)

// connectedChannel wraps one end of an in-memory pair and connects both, so a
// case can read what the channel writes straight off the wire.
//
// Both ends are connected here because the pair is a synchronous pipe: a write
// with nobody reading blocks until somebody does.
func connectedChannel(t *testing.T) (*channel, mcpsdk.Connection) {
	t.Helper()

	serverSide, clientSide := mcpsdk.NewInMemoryTransports()
	ch := newChannel(serverSide)
	served, err := ch.Connect(t.Context())
	assert.NilError(t, err)
	t.Cleanup(func() { _ = served.Close() })

	peer, err := clientSide.Connect(t.Context())
	assert.NilError(t, err)
	t.Cleanup(func() { _ = peer.Close() })
	return ch, peer
}

// A notification the SDK has no call for still goes out, as a JSON-RPC request
// with no id — which is what the harness on the other end reads as an event it
// answers nothing to.
func TestChannel_NotifyWritesANotification(t *testing.T) {
	ch, peer := connectedChannel(t)

	const method = "notifications/claude/channel"
	params := map[string]any{
		"content": "[crabswarm chat] new message from alpha/bob",
		"meta":    map[string]any{"from": "alpha/bob", "room": testRoom},
	}

	// The pipe hands the write over to the read, so the two have to be running
	// at once.
	var notified errgroup.Group
	notified.Go(func() error { return ch.Notify(t.Context(), method, params) })

	msg, err := peer.Read(t.Context())
	assert.NilError(t, err)
	assert.NilError(t, notified.Wait())

	req, ok := msg.(*jsonrpc.Request)
	assert.Assert(t, ok, "the harness was sent %T, want a request", msg)
	assert.Assert(t, !req.IsCall(), "the notification carries an id, so it asks for an answer")
	assert.Equal(t, req.Method, method)

	var got map[string]any
	assert.NilError(t, json.Unmarshal(req.Params, &got))
	assert.DeepEqual(t, got, params)
}

// A notification written before the harness connected is refused rather than
// dropped: there is no connection to write to yet, and the caller comes back to
// the notice on the next chance to deliver it.
func TestChannel_RefusesToNotifyBeforeItIsConnected(t *testing.T) {
	serverSide, _ := mcpsdk.NewInMemoryTransports()
	err := newChannel(serverSide).Notify(t.Context(), "notifications/claude/channel", nil)
	assert.ErrorIs(t, err, errNoChannelYet)
}

// negotiated runs a plain SDK server and answers with the protocol revision the
// SDK's own client settled on with it. capped is whether the server is given
// what [New] gives its own: the transport that declares the cap, and the
// middleware that turns the newer handshake away.
func negotiated(t *testing.T, capped bool) string {
	t.Helper()

	serverSide, clientSide := mcpsdk.NewInMemoryTransports()
	var transport mcpsdk.Transport = serverSide
	server := mcpsdk.NewServer(&mcpsdk.Implementation{Name: "probe", Version: "v0"}, nil)
	if capped {
		transport = newChannel(serverSide)
		server.AddReceivingMiddleware(refusesTheNewerHandshake)
	}

	ctx, cancel := context.WithCancel(t.Context())
	var served errgroup.Group
	served.Go(func() error { return server.Run(ctx, transport) })

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "probe-client", Version: "v0"}, nil)
	session, err := client.Connect(t.Context(), clientSide, nil)
	assert.NilError(t, err)
	t.Cleanup(func() {
		_ = session.Close()
		cancel()
		_ = served.Wait()
	})
	return session.InitializeResult().ProtocolVersion
}

// The server offers nothing above the cap, so no client can talk it into the
// revision that would drop the channel events it pushes — and a client that
// asks for the newer one still gets a session rather than a refusal.
//
// The same exchange uncapped is the control: it settles on something above the
// cap, which is what says the cap is doing the capping rather than the SDK
// happening to agree.
func TestChannel_CapsTheProtocolRevisionItOffers(t *testing.T) {
	assert.Equal(t, negotiated(t, true), maxProtocolVersion)

	uncapped := negotiated(t, false)
	assert.Assert(t, uncapped > maxProtocolVersion,
		"an uncapped server settled on %s, which is not above the cap of %s",
		uncapped, maxProtocolVersion)
}

// The cap is a date comparison, so it holds for every revision either side of
// it rather than for the one that prompted it.
func TestChannel_SupportsEveryRevisionUpToTheCap(t *testing.T) {
	ch := newChannel(nil)
	for _, version := range []string{"2024-11-05", "2025-03-26", "2025-06-18", maxProtocolVersion} {
		assert.Assert(t, ch.SupportsProtocolVersion(version), "%s is not offered", version)
	}
	for _, version := range []string{"2026-07-28", "2027-01-01"} {
		assert.Assert(t, !ch.SupportsProtocolVersion(version), "%s is offered", version)
	}
}
