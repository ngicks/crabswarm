package mcp

import (
	"context"
	"log/slog"
	"os"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
	"github.com/ngicks/crabswarm/internal/libver"
	"github.com/ngicks/crabswarm/pkg/harnessctl"
)

// host is what a server offers whichever transport it serves: the SDK server
// the families register onto, the connection to the daemon every tool acts
// through, and the resources a harness may subscribe to. [Server] and
// [HTTPServer] each hold one and differ only in how a session finds the member
// it acts as.
type host struct {
	logger *slog.Logger
	client *cli.Client
	mcp    *mcpsdk.Server

	// getenv reads the environment the harness started this server in, which is
	// where a deliverer finds what its channel needs. It is a field rather than
	// a call to [os.Getenv] so a test can play a harness without setting a
	// variable on the process running it. [New] takes the real one, and so does
	// [NewHTTP] when it is handed none.
	getenv func(string) string

	// detect names the harness behind the client that handshook. It is a field
	// for the reason getenv is one: a harness that carries a channel or a state
	// feed of its own speaks to a CLI this process cannot start, so a test hands
	// the server one that plays the part. Both constructors take
	// [harnessctl.Detect].
	detect func(clientName string, getenv func(string) string) harnessctl.Harness

	// subscribable are the resource URIs a family registered, which are the ones
	// a harness may ask to be told about. Written while the families register
	// and only read once the session is up.
	subscribable map[string]struct{}
	// rosterResources are the resources a change in who attends the room makes
	// stale. Written and read like subscribable.
	rosterResources []string

	// pace is the schedule every member this server makes runs its loops at.
	pace pace
}

// newHost dials sockPath and prepares the SDK server. getenv nil reads the
// process environment. A keepAlive above zero pings every session that often
// and closes one that stopped answering; zero pings nothing.
func newHost(
	logger *slog.Logger, sockPath string, getenv func(string) string, keepAlive time.Duration,
) (*host, error) {
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	if getenv == nil {
		getenv = os.Getenv
	}
	client, err := cli.Dial(sockPath)
	if err != nil {
		return nil, err
	}
	h := &host{
		logger:       logger,
		client:       client,
		getenv:       getenv,
		detect:       harnessctl.Detect,
		subscribable: map[string]struct{}{},
		pace:         defaultPace,
	}
	opts := &mcpsdk.ServerOptions{
		Logger:                    logger,
		SubscribeHandler:          h.subscribed,
		UnsubscribeHandler:        h.unsubscribed,
		KeepAlive:                 keepAlive,
		KeepAliveFailureThreshold: keepAliveFailures,
	}
	// What the server declares about itself is settled here, because the
	// handshake carries it and the handshake is what tells the server which
	// harness it is serving — the answer arrives after the question.
	if harnessctl.ClaudeChannelEnabled(h.getenv) {
		opts.Instructions = claudeChannelInstructions
	}
	h.mcp = mcpsdk.NewServer(
		&mcpsdk.Implementation{Name: serverName, Version: libver.Version},
		opts,
	)
	return h, nil
}

// keepAliveFailures is how many pings in a row a session may leave unanswered
// before it is closed. One is not enough: a client whose event stream just
// dropped answers nothing until it has opened the stream again, and that is a
// live session.
const keepAliveFailures = 3

// newMember makes a member attending as token on this server's connection.
func (h *host) newMember(token string) *Member {
	return newMember(h.logger, h.client, token, h.announceRoster, h.pace)
}

// harnessOf reads the harness off the handshake a session made. A session
// names one client for its whole life, so its caller asks once per session.
func (h *host) harnessOf(params *mcpsdk.InitializeParams) harnessctl.Harness {
	var client string
	if params.ClientInfo != nil {
		client = params.ClientInfo.Name
	}
	harness := h.detect(client, h.getenv)
	h.logger.Info("the harness named itself",
		"client", client,
		"harness", cli.HarnessName(harness.Kind()),
		"nudge", cli.NudgeDeliveryName(harness.Nudge()))
	return harness
}

// addResource registers a resource and lets a harness subscribe to it.
//
// Subscribing goes through the server because the SDK takes one handler for the
// whole session: a URI nothing registered is refused rather than accepted
// quietly, which is what keeps a harness from waiting on news that could never
// come.
func (h *host) addResource(res *mcpsdk.Resource, handler mcpsdk.ResourceHandler) {
	h.subscribable[res.URI] = struct{}{}
	h.mcp.AddResource(res, handler)
}

// announceOnRosterChange has the subscribed sessions told to read uri again
// whenever the room's attendance or anyone's state changes.
func (h *host) announceOnRosterChange(uri string) {
	h.rosterResources = append(h.rosterResources, uri)
}

// announceRoster tells the subscribed sessions to read the resources a roster
// change made stale.
//
// Every subscribed session of the SDK server is told, whichever member it acts
// as: the SDK sends a resource update to all of them and to no chosen one. A
// session told about a room it is not in reads its own room again, which costs
// it a read and tells it nothing wrong.
func (h *host) announceRoster(ctx context.Context) {
	for _, uri := range h.rosterResources {
		err := h.mcp.ResourceUpdated(ctx,
			&mcpsdk.ResourceUpdatedNotificationParams{URI: uri})
		if err != nil {
			h.logger.Warn("announcing the changed room roster failed",
				"uri", uri, "error", err)
		}
	}
}

// subscribed accepts a request to be told about a resource. The room's feed is
// already up — it is the attendance the member holds for as long as it runs — so
// there is nothing to start here; what is left is deciding whether the server
// can keep the promise a subscription asks for.
func (h *host) subscribed(_ context.Context, req *mcpsdk.SubscribeRequest) error {
	return h.announceable(req.Params.URI)
}

// unsubscribed acknowledges the withdrawal and leaves the feed running.
//
// The feed is not the subscription's to end: it is the attendance itself, which
// runs for as long as the member does whether or not anything is listening.
// Nothing is announced to a harness that withdrew — the SDK sends a resource
// update to the sessions that subscribed and to no others. The SDK also
// requires this handler as soon as [host.subscribed] exists.
func (h *host) unsubscribed(_ context.Context, req *mcpsdk.UnsubscribeRequest) error {
	return h.announceable(req.Params.URI)
}

// announceable returns nil when the server can tell a harness that uri changed,
// and the refusal otherwise.
//
// A URI no family registered is refused rather than accepted quietly. The SDK
// records a subscription for whatever URI it is handed, so accepting a typo
// would leave the harness waiting on news that could never come.
func (h *host) announceable(uri string) error {
	if _, ok := h.subscribable[uri]; ok {
		return nil
	}
	return mcpsdk.ResourceNotFoundError(uri)
}
