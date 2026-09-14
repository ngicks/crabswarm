package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/cli"
)

// The room's attendance is a resource rather than a further tool because it is
// what a harness keeps in view: a tool result is read once, in the turn that
// asked for it, while a resource is there to be re-read whenever the room has
// moved on underneath the agent. The chat_members tool stays for the turn that
// just wants the list.

// membersURI names the caller's room roster. The scheme is the daemon's rather
// than a file or http one: nothing about the roster is fetchable by an address,
// and naming a resource after the thing it holds is what lets the daemon's own
// documents sit under one scheme.
const membersURI = "crabswarm://chat/members"

// The roster is answered as structured data because its reader is the harness
// rather than the model: it gets an address, a team, a name, a kind and a state
// as fields it can act on, where the CLI's listing would have to be split back
// into columns first.
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
// assembling one and getting the collision rule wrong. Kind says whether a
// message reaches the member on its own: an agent is typed into when one
// arrives, a human reads its room when it asks.
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
		Description: "Everyone attending your room: the role chat_send " +
			"addresses, the team and name it is made of, the kind — agent for " +
			"a harness a message is typed into, human for someone who reads " +
			"when it asks — and the state each member's harness last " +
			"reported: working, waiting or done, and unknown where the daemon " +
			"reported none. Subscribe to be told when somebody starts or " +
			"stops attending, or changes state.",
	}, s.readMembers)
}

// readMembers answers with the room as it stands. Every read lists the room
// again rather than replaying what the event feed said: the feed is a nudge to
// look, not a record to accumulate, so a reader that missed an event still gets
// the truth here.
func (s *Server) readMembers(
	ctx context.Context, _ *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	token, err := cli.ResolveToken(s.token)
	if err != nil {
		return nil, err
	}
	if err := s.awaitAttendance(ctx); err != nil {
		return nil, err
	}
	members, err := s.client.Members(ctx, token)
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
			Kind:    cli.MemberKindName(m.GetKind()),
			State:   cli.HarnessStateName(m.GetState()),
		})
	}
	return out
}

// subscribed accepts a request to be told about the roster. The room's feed is
// already up — it is the attendance the bridge holds for the whole session — so
// there is nothing to start here; what is left is deciding whether the bridge
// can keep the promise a subscription asks for.
func (s *Server) subscribed(_ context.Context, req *mcp.SubscribeRequest) error {
	return announceable(req.Params.URI)
}

// announceable returns nil when the bridge can tell a harness that uri changed,
// and the refusal otherwise.
//
// A URI this bridge does not serve is refused rather than accepted quietly. The
// SDK records a subscription for whatever URI it is handed, so accepting a typo
// would leave the harness waiting on news that could never come.
func announceable(uri string) error {
	if uri == membersURI {
		return nil
	}
	return mcp.ResourceNotFoundError(uri)
}

// unsubscribed acknowledges the withdrawal and leaves the feed running.
//
// The feed is not the subscription's to end: it is the attendance itself, which
// runs for as long as the session does whether or not anything is listening.
// Nothing is announced to a harness that withdrew — the SDK sends a resource
// update to the sessions that subscribed and to no others. The SDK also
// requires this handler as soon as [Server.subscribed] exists.
func (s *Server) unsubscribed(_ context.Context, req *mcp.UnsubscribeRequest) error {
	return announceable(req.Params.URI)
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
// the same states. It changes the room's conversation, but nothing subscribes
// to that — chat_read is what a member reads its messages with, and a
// subscription the bridge accepted would be one it could not serve.
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
