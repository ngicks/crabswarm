package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The room's attendance and its transcript are resources rather than further
// tools because they are what a harness keeps in view: a tool result is read
// once, in the turn that asked for it, while a resource is there to be re-read
// whenever the room has moved on underneath the agent. The chat_members tool
// stays for the turn that just wants the list.

// membersURI names the caller's room roster and historyURI its conversation.
// The scheme is the daemon's rather than a file or http one: nothing about
// either is fetchable by an address, and naming a resource after the thing it
// holds is what lets the two sit beside each other under one scheme.
const (
	membersURI = "crabswarm://chat/members"
	historyURI = "crabswarm://chat/history"
)

// The roster is answered as structured data because its reader is the harness
// rather than the model, and because a member's state is not in the CLI's
// listing at all. The transcript is answered in the CLI's own words: every line
// of it already carries everything an entry holds, so a second spelling of the
// same conversation would be one more thing to keep in step.
const (
	membersMIMEType = "application/json"
	historyMIMEType = "text/plain"
)

// roster is the members resource. It carries the room once rather than on every
// member: a caller only ever sees its own room, so repeating it per line would
// say the same thing as many times as there are members.
type roster struct {
	Room    string         `json:"room"`
	Members []rosterMember `json:"members"`
}

// rosterMember is one attendee. Address is spelled out beside the team and name
// it is made of, so the reader can hand it straight to chat_send instead of
// assembling one and getting the collision rule wrong. Kind says whether a
// message reaches the member on its own: an agent is typed into when one
// arrives, a human is only ever handed its inbox when it asks.
type rosterMember struct {
	Address string `json:"address"`
	Team    string `json:"team"`
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	State   string `json:"state"`
}

// addResources registers the room's resources before the first session, so the
// resources capability is advertised during the handshake rather than announced
// as a change the harness has to notice.
func (s *Server) addResources() {
	s.mcp.AddResource(&mcp.Resource{
		Name:     "members",
		Title:    "Room members",
		URI:      membersURI,
		MIMEType: membersMIMEType,
		Description: "Everyone attending your room: the address chat_send " +
			"takes, the team and name it is made of, the kind — agent for a " +
			"harness a message is typed into, human for someone who reads an " +
			"inbox — and the state each member's harness last reported: " +
			"working, waiting or done, and unknown where the daemon reported " +
			"none. Subscribe to be told when someone joins, leaves, or " +
			"changes state.",
	}, s.readMembers)
	s.mcp.AddResource(&mcp.Resource{
		Name:     "history",
		Title:    "Room transcript",
		URI:      historyURI,
		MIMEType: historyMIMEType,
		Description: "The recent conversation of your room, oldest first: " +
			"when each message was sent, who sent it, and who it went to — " +
			"\"*\" for one addressed to the whole room. It is what the room " +
			"said rather than what was addressed to you, so reading it " +
			"consumes nothing and shows messages chat_read has already " +
			"handed over. Read it again to catch up; nothing announces it " +
			"as changed.",
	}, s.readHistory)
}

// readMembers answers with the room as it stands. Every read lists the room
// again rather than replaying what the event feed said: the feed is a nudge to
// look, not a record to accumulate, so a reader that missed an event still gets
// the truth here.
func (s *Server) readMembers(
	ctx context.Context, _ *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	token, err := s.ensureJoined(ctx)
	if err != nil {
		return nil, err
	}
	members, err := s.client.Members(ctx, token)
	if err != nil {
		return nil, s.forgetJoined(err)
	}
	body, err := json.MarshalIndent(rosterOf(members), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encoding the room roster: %w", err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      membersURI,
			MIMEType: membersMIMEType,
			Text:     string(body),
		}},
	}, nil
}

// readHistory answers with the tail of the room's conversation as the CLI
// prints it, rendered through [cli.Client] for the same reason a tool result is:
// the transcript reads the same whether a member is wired to its room through
// MCP or through a shell.
//
// The window is whichever one the daemon defaults to. A resource read carries
// no arguments to ask for another with, and the default is already what a
// member catching up is after: the recent conversation rather than everything
// the room kept.
func (s *Server) readHistory(
	ctx context.Context, _ *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	token, err := s.ensureJoined(ctx)
	if err != nil {
		return nil, err
	}
	var rendered bytes.Buffer
	if err := s.client.History(ctx, &rendered, token, 0); err != nil {
		return nil, s.forgetJoined(err)
	}
	return &mcp.ReadResourceResult{
		Contents: []*mcp.ResourceContents{{
			URI:      historyURI,
			MIMEType: historyMIMEType,
			Text:     rendered.String(),
		}},
	}, nil
}

// rosterOf shapes what the daemon reported into what the resource says.
func rosterOf(members []*chatv1.Member) roster {
	out := roster{Members: make([]rosterMember, 0, len(members))}
	for _, m := range members {
		if out.Room == "" {
			out.Room = m.GetRoom()
		}
		out.Members = append(out.Members, rosterMember{
			Address: cli.Address(m),
			Team:    m.GetTeam(),
			Name:    m.GetName(),
			Kind:    cli.MemberKindName(m.GetKind()),
			State:   cli.HarnessStateName(m.GetState()),
		})
	}
	return out
}

// subscribed accepts a request to be told about the roster. The room's feed is
// already up — the bridge watches it for the whole session — so there is
// nothing to start here; what is left is deciding whether the bridge can keep
// the promise a subscription asks for.
func (s *Server) subscribed(_ context.Context, req *mcp.SubscribeRequest) error {
	return announceable(req.Params.URI)
}

// announceable returns nil when the bridge can tell a harness that uri changed,
// and the refusal otherwise.
//
// A URI this bridge does not serve is refused rather than accepted quietly. The
// SDK records a subscription for whatever URI it is handed, so accepting a typo
// would leave the harness waiting on news that could never come.
//
// The transcript is refused for the same reason even though it is served: the
// room's feed carries members joining, leaving and changing state, and nothing
// at all when a message is appended, so a subscription to it would be a promise
// no event could keep. The refusal says so, since a harness that asked has to
// decide what to do instead — and re-reading is the answer. It says it without
// naming an operation, because [Server.unsubscribed] refuses on these same
// grounds and a withdrawal answered in the words of a subscription reads as an
// answer to something else.
func announceable(uri string) error {
	switch uri {
	case membersURI:
		return nil
	case historyURI:
		return fmt.Errorf(
			"%s cannot be watched: nothing announces it as changed; read it again to catch up",
			historyURI)
	default:
		return mcp.ResourceNotFoundError(uri)
	}
}

// unsubscribed acknowledges the withdrawal and leaves the feed running.
//
// The feed is not the subscription's to end: it is what tells the bridge its
// own membership has lapsed, so it runs for as long as the session does
// whether or not anything is listening. Nothing is announced to a harness that
// withdrew — the SDK sends a resource update to the sessions that subscribed
// and to no others. The SDK also requires this handler as soon as
// [Server.subscribed] exists.
func (s *Server) unsubscribed(_ context.Context, req *mcp.UnsubscribeRequest) error {
	return announceable(req.Params.URI)
}

// How long the bridge waits before watching the room again after its feed
// ended. The daemon drops a watcher that falls behind, so ending is ordinary
// enough that the first retry is quick; the ceiling is what keeps a daemon that
// is down from being asked in a loop.
const (
	watchBackoffBase = 200 * time.Millisecond
	watchBackoffMax  = 5 * time.Second
)

// watchMembers keeps the room's event feed up for as long as the session lasts.
//
// It watches whether or not anything has subscribed, and never gives up. A feed
// that ended is the one thing its readers cannot notice for themselves: a
// subscribed harness is holding a view of the room that would quietly stop
// being true, with no call of its own to fail and tell it so. The bridge is in
// the same position about its own membership — the daemon can forget it between
// one turn and the next — and a feed the daemon refuses is what tells it, since
// there may be no tool call for hours. That is why the feed runs from the start
// rather than from the first subscription: an unsubscribed session is told
// nothing either way, so nothing is spent on it but the stream.
func (s *Server) watchMembers(ctx context.Context) {
	backoff := watchBackoffBase
	failures := 0
	for resumed := false; ; resumed = true {
		started := time.Now()
		err := s.streamRoom(ctx, resumed)
		if ctx.Err() != nil {
			return
		}
		failures++
		// A feed that stayed up for a while and then broke is not the trouble a
		// feed that never got going is, so it starts its retries over rather
		// than inheriting the wait the previous failure had climbed to. Settled
		// before the log, so the wait it reports is the one it takes.
		if time.Since(started) >= watchBackoffMax {
			backoff = watchBackoffBase
			failures = 1
		}
		// A feed that could not be opened because the bridge is not attending
		// yet is the attend loop's news to report, and it reports it: saying it
		// again here would double every line a bridge waiting for its daemon
		// writes.
		if !errors.Is(err, errNotAttending) {
			s.warnRetry("the room event feed ended; watching again", failures, backoff, err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, watchBackoffMax)
	}
}

// errNotAttending marks a feed attempt that never reached the daemon because
// the bridge has no attendance to watch on behalf of.
var errNotAttending = errors.New("not attending the chat room")

// streamRoom watches the room until the feed ends, announcing the roster as
// changed for every event that changes it. It returns why the feed ended.
//
// resumed says an earlier feed had already ended, which makes the roster stale
// by definition: whatever happened while nothing was watching went unannounced.
// Saying so as soon as the new feed is up is what closes that gap — the
// subscriber reads the resource, and the read lists the room afresh.
//
// Attendance is declared first, as it is for a tool call. The feed starts with
// the process, so it regularly gets there before the attend loop has landed a
// join, and watching a room on behalf of a member the daemon does not
// acknowledge only spends the backoff on a refusal the join would have cleared.
// A feed refused for that reason gives the declared attendance back up, so the
// next attempt declares it again instead of looping on the same no.
func (s *Server) streamRoom(ctx context.Context, resumed bool) error {
	token, err := s.ensureJoined(ctx)
	if err != nil {
		return fmt.Errorf("%w: %w", errNotAttending, err)
	}
	stream, err := s.client.WatchRoom(ctx, token)
	if err != nil {
		return s.forgetJoined(err)
	}
	if resumed {
		s.membersChanged(ctx)
	}
	for {
		ev, err := stream.Recv()
		if err != nil {
			return s.forgetJoined(err)
		}
		if rosterChanged(ev) {
			s.membersChanged(ctx)
		}
	}
}

// membersChanged tells the subscribed sessions to read the roster again.
func (s *Server) membersChanged(ctx context.Context) {
	err := s.mcp.ResourceUpdated(ctx, &mcp.ResourceUpdatedNotificationParams{URI: membersURI})
	if err != nil {
		s.logger.Warn("announcing the changed room roster failed", "error", err)
	}
}

// rosterChanged reports whether ev changes who attends the room or what state
// they are in. A message being appended does not: the same members are there in
// the same states. It changes the transcript, but nothing subscribes to that —
// the daemon never publishes such an event today, so a subscription the bridge
// accepted would be one it could not serve.
func rosterChanged(ev *chatv1.RoomEvent) bool {
	switch ev.GetEvent().(type) {
	case *chatv1.RoomEvent_MemberStateChanged,
		*chatv1.RoomEvent_MemberJoined,
		*chatv1.RoomEvent_MemberLeft:
		return true
	default:
		return false
	}
}
