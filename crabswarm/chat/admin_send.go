package chat

import (
	"context"
	"strings"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Send delivers a message into a room the caller does not attend, addressed to
// the roles it names, to everyone there, or to nobody.
//
// The message is attributed to the reserved [adminName] identity, and nothing
// is recorded as attending for it: the operator speaks into the room without
// entering it, so there is nothing there to address back and nothing to nudge
// on the operator's own behalf.
//
// That identity carries no team, which is what decides how a bare name in the
// target resolves: with no team of its own the sender has no colleague to
// prefer, so the name is looked for across the whole room and a name two teams
// carry is refused as ambiguous rather than guessed at.
func (a *AdminService) Send(
	ctx context.Context,
	req *chatv1.AdminSendRequest,
) (*chatv1.AdminSendResponse, error) {
	if err := a.authenticate(ctx); err != nil {
		return nil, err
	}
	switch {
	case req.GetRoom() == "":
		return nil, status.Error(codes.InvalidArgument, "empty room")
	case req.GetText() == "":
		return nil, status.Error(codes.InvalidArgument, "empty message text")
	}
	from := adminSender(req.GetRoom())
	sent, err := a.deliver.send(
		ctx, from, targetOf(req.GetTarget()), req.GetText(), time.Now())
	if err != nil {
		return nil, storeStatus(err)
	}
	a.logger.InfoContext(ctx, "chat: admin sent message",
		"room", req.GetRoom(), "target", targetAddr(sent.Message.Target),
		"mentioned", len(sent.Mentioned), "absent", len(sent.Absent))
	return &chatv1.AdminSendResponse{
		Mentioned: membersProto(sent.Mentioned),
		Absent:    membersProto(sent.Absent),
	}, nil
}

// targetAddr renders a written target for the log line, in the grammar an
// operator writes one in: "everyone" for the whole room, the roles by address,
// and "-" for a board post, which names nobody to render.
func targetAddr(t Target) string {
	switch t.Kind {
	case TargetEveryone:
		return "everyone"
	case TargetRoles:
		addrs := make([]string, len(t.Roles))
		for i, r := range t.Roles {
			addrs[i] = r.Team + "/" + r.Name
		}
		return strings.Join(addrs, ",")
	default:
		return "-"
	}
}
