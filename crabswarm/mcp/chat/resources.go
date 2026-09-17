package chat

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

// addResources registers the room's resources and asks the server to announce
// the roster whenever the attendance it holds says the room changed.
func (f *family) addResources() {
	f.server.AddResource(&mcp.Resource{
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
	}, f.readMembers)
	f.server.AnnounceOnRosterChange(membersURI)
}

// readMembers answers with the room as it stands. Every read lists the room
// again rather than replaying what the event feed said: the feed is a nudge to
// look, not a record to accumulate, so a reader that missed an event still gets
// the truth here.
func (f *family) readMembers(
	ctx context.Context, _ *mcp.ReadResourceRequest,
) (*mcp.ReadResourceResult, error) {
	token, err := f.server.ResolveToken()
	if err != nil {
		return nil, err
	}
	if err := f.server.AwaitAttendance(ctx); err != nil {
		return nil, err
	}
	members, err := f.server.Client().Members(ctx, token)
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
