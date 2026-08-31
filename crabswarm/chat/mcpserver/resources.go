package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The room's attendance is a resource rather than a fifth tool because it is
// something the harness can keep in view: a tool result is read once, in the
// turn that asked for it, while a subscribed resource is re-read whenever the
// room changes underneath the agent. The chat_members tool stays for the turn
// that just wants the list, and answers in the CLI's own words; this answers in
// a structured form, since the reader here is the harness rather than the model.

// membersURI names the caller's room roster. The scheme is the daemon's rather
// than a file or http one: nothing about this is fetchable by an address, and
// naming it after the thing it holds keeps the door open for the room's history
// to arrive beside it under the same scheme.
const membersURI = "crabswarm://chat/members"

const membersMIMEType = "application/json"

// roster is the members resource. It carries the room once rather than on every
// member: a caller only ever sees its own room, so repeating it per line would
// say the same thing as many times as there are members.
type roster struct {
	Room    string         `json:"room"`
	Members []rosterMember `json:"members"`
}

// rosterMember is one attendee. Address is spelled out beside the team and name
// it is made of, so the reader can hand it straight to chat_send instead of
// assembling one and getting the collision rule wrong.
type rosterMember struct {
	Address string `json:"address"`
	Team    string `json:"team"`
	Name    string `json:"name"`
	State   string `json:"state"`
}

// addResources registers the members resource before the first session, so the
// resources capability is advertised during the handshake rather than announced
// as a change the harness has to notice.
//
// The room's history is deliberately absent: the daemon has no RPC to read it
// with, and a resource that could only answer "not yet" would be worse than no
// resource at all.
func (s *Server) addResources() {
	s.mcp.AddResource(&mcp.Resource{
		Name:     "members",
		Title:    "Room members",
		URI:      membersURI,
		MIMEType: membersMIMEType,
		Description: "Everyone attending your room: the address chat_send " +
			"takes, the team and name it is made of, and the state each " +
			"member's harness last reported — working, waiting or done, and " +
			"unknown where the daemon reported none. Subscribe to be told " +
			"when someone joins, leaves, or changes state.",
	}, s.readMembers)
}

// readMembers answers with the room as it stands. Every read lists the room
// again rather than replaying what the event feed said: the feed is a nudge to
// look, not a record to accumulate, so a reader that missed an event still gets
// the truth here.
func (s *Server) readMembers(
	ctx context.Context, _ *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	if err := s.ensureJoined(ctx); err != nil {
		return nil, err
	}
	members, err := s.client.Members(ctx, s.token)
	if err != nil {
		return nil, err
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
			State:   cli.HarnessStateName(m.GetState()),
		})
	}
	return out
}

// subscribed starts watching the room the first time anything asks to be told
// about it. Nothing is watched before that: a harness that lists the tools and
// never subscribes should cost the daemon no stream at all.
//
// A URI this bridge does not serve is refused rather than accepted quietly. The
// SDK records a subscription for whatever URI it is handed, so accepting a
// typo would leave the harness waiting on news that could never come.
func (s *Server) subscribed(_ context.Context, req *mcp.SubscribeRequest) error {
	if req.Params.URI != membersURI {
		return mcp.ResourceNotFoundError(req.Params.URI)
	}
	s.watchOnce.Do(func() { close(s.watchWanted) })
	return nil
}

// unsubscribed acknowledges the withdrawal and leaves the feed running.
//
// The stream is not torn down with the last subscription because this bridge
// serves exactly one stdio session, which ends with the process: a harness that
// stops and starts watching mid-session is cheaper to serve from a feed that is
// already up than from one that has to be dialled again, and a feed nobody
// subscribes to announces nothing. The SDK also requires this handler as soon
// as [Server.subscribed] exists.
func (s *Server) unsubscribed(_ context.Context, req *mcp.UnsubscribeRequest) error {
	if req.Params.URI != membersURI {
		return mcp.ResourceNotFoundError(req.Params.URI)
	}
	return nil
}

// How long the bridge waits before watching the room again after its feed
// ended. The daemon drops a watcher that falls behind, so ending is ordinary
// enough that the first retry is quick; the ceiling is what keeps a daemon that
// is down from being asked in a loop.
const (
	watchBackoffBase = 200 * time.Millisecond
	watchBackoffMax  = 5 * time.Second
)

// watchMembers keeps the room's event feed up for as long as the session lasts,
// once something has subscribed to the roster.
//
// It never gives up the way the startup join does. A feed that ended is the one
// thing a subscriber cannot notice for itself: the harness is holding a view of
// the room that would quietly stop being true, with no call of its own to fail
// and tell it so.
func (s *Server) watchMembers(ctx context.Context) {
	select {
	case <-ctx.Done():
		return
	case <-s.watchWanted:
	}

	backoff := watchBackoffBase
	for attempt := 0; ; attempt++ {
		started := time.Now()
		err := s.streamRoom(ctx, attempt > 0)
		if ctx.Err() != nil {
			return
		}
		s.logger.Warn("the room event feed ended; watching again",
			"backoff", backoff, "error", err)
		// A feed that stayed up for a while and then broke is not the trouble a
		// feed that never got going is, so it starts its retries over rather
		// than inheriting the wait the previous failure had climbed to.
		if time.Since(started) >= watchBackoffMax {
			backoff = watchBackoffBase
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
		}
		backoff = min(2*backoff, watchBackoffMax)
	}
}

// streamRoom watches the room until the feed ends, announcing the roster as
// changed for every event that changes it. It returns why the feed ended.
//
// resumed says an earlier feed had already ended, which makes the roster stale
// by definition: whatever happened while nothing was watching went unannounced.
// Saying so as soon as the new feed is up is what closes that gap — the
// subscriber reads the resource, and the read lists the room afresh.
//
// Attendance is declared first, as it is for a tool call. A subscription can
// arrive before the startup join has landed, and watching a room on behalf of a
// member the daemon does not acknowledge only spends the backoff on a refusal
// the join would have cleared.
func (s *Server) streamRoom(ctx context.Context, resumed bool) error {
	if err := s.ensureJoined(ctx); err != nil {
		return err
	}
	stream, err := s.client.WatchRoom(ctx, s.token)
	if err != nil {
		return err
	}
	if resumed {
		s.membersChanged(ctx)
	}
	for {
		ev, err := stream.Recv()
		if err != nil {
			return err
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
// the same states, and the message itself is read out of an inbox rather than
// off a resource.
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
