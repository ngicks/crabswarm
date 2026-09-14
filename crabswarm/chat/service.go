package chat

import (
	"context"
	"log/slog"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// Notifier is told that a member was mentioned, so a harness that has finished
// its turn can be woken instead of waiting for its agent to poll. It is the
// seam the keystroke injector plugs into; the service only reports mentions.
//
// Notify is called once per nudged member, inside the RPC that recorded the
// message, with that RPC's context. An implementation that needs to outlive the
// call — queueing, retrying, shelling out to a terminal — detaches on its own
// (context.WithoutCancel); putting a queue in the seam would force one on
// implementations that push synchronously.
//
// A returned error never fails the send: the message is already in the room by
// then, waiting at the recipient's read position, and the service only logs it.
type Notifier interface {
	Notify(ctx context.Context, recipient Member, from Sender, text string) error
}

// NopNotifier drops every notification. It is what a [Service] runs with until
// a real notifier is configured: messages then wait in the room until the
// mentioned role reads them.
type NopNotifier struct{}

var _ Notifier = NopNotifier{}

// Notify does nothing and never fails.
func (NopNotifier) Notify(context.Context, Member, Sender, string) error { return nil }

// StatusMirror publishes a member's harness state somewhere the host can see
// it, so an operator watching their commands reads the same words the chat
// broker holds. It is the seam the cmdman status display plugs into.
//
// Set is called after the store has recorded state, Clear after a member is
// gone. Both run inside the RPC that caused them, with that RPC's context; an
// implementation that must outlive the call detaches on its own.
//
// A returned error never fails the RPC: the store is already authoritative by
// then, and a display that lags behind it costs nothing but the display.
type StatusMirror interface {
	Set(ctx context.Context, m Member, state MemberState) error
	Clear(ctx context.Context, m Member) error
}

// NopStatusMirror publishes nothing. It is what a [Service] runs with until a
// real mirror is configured, which is every deployment that is not driving
// cmdman.
type NopStatusMirror struct{}

var _ StatusMirror = NopStatusMirror{}

// Set does nothing and never fails.
func (NopStatusMirror) Set(context.Context, Member, MemberState) error { return nil }

// Clear does nothing and never fails.
func (NopStatusMirror) Clear(context.Context, Member) error { return nil }

// tokenNamePrefixLen is how much of a token a generated member name carries.
// Long enough to stay unique among the handful of agents in a room, short
// enough for another agent to type as an address.
const tokenNamePrefixLen = 8

// TeamInfoProvider resolves an identity token to the placement of its holder,
// which is what decides whether the token may attend and where.
//
// Resolve returns an error wrapping [resolver.ErrUnknownToken] when the token
// cannot be placed at all; any other error means the lookup itself failed and
// carries no verdict about the token. [resolver.CmdmanCompose] is the
// implementation the daemon runs.
type TeamInfoProvider interface {
	Resolve(ctx context.Context, token string) (resolver.TeamInfo, error)
}

var _ TeamInfoProvider = (*resolver.CmdmanCompose)(nil)

// Service is the member-facing half of the chat broker: the ChatService gRPC
// implementation over the [Store], gated by the [TeamInfoProvider] that decides
// which tokens may attend.
//
// It reads the caller's token from the request context, where
// [UnaryTokenInterceptor] puts it, so a server registering a Service must
// install that interceptor.
type Service struct {
	chatv1.UnimplementedChatServiceServer

	store    *Store
	provider TeamInfoProvider
	deliver  deliverer
	mirror   StatusMirror
	logger   *slog.Logger
}

var _ chatv1.ChatServiceServer = (*Service)(nil)

// NewService returns the ChatService implementation over store, admitting the
// members provider knows, reporting mentions to notifier and member state to
// mirror. A nil notifier means [NopNotifier], a nil mirror [NopStatusMirror];
// a nil logger discards logs.
func NewService(
	store *Store,
	provider TeamInfoProvider,
	notifier Notifier,
	mirror StatusMirror,
	logger *slog.Logger,
) *Service {
	if mirror == nil {
		mirror = NopStatusMirror{}
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		store:    store,
		provider: provider,
		deliver:  newDeliverer(store, notifier, logger),
		mirror:   mirror,
		logger:   logger,
	}
}

// caller resolves the member behind the request's token for every RPC but
// [Service.Attend], which has no attendance to require yet because it is what
// establishes one.
//
// Attendance is the whole of the check: it lasts exactly as long as the stream
// that declared it, so a token the store still holds is a token whose session
// is still running, and nothing has to be asked of the team-info provider on
// the way through.
//
// A token nobody attends under is Unauthenticated, not NotFound: NotFound on
// these RPCs means the role the caller addressed does not exist, and the two
// must not read alike.
func (s *Service) caller(ctx context.Context) (Member, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return Member{}, err
	}
	m, err := s.store.Member(ctx, token)
	if err != nil {
		return Member{}, storeStatus(err)
	}
	return m, nil
}

// mirrorState publishes m's state, logging what the mirror could not do. The
// store already holds the state, so a failed mirror costs an operator a stale
// display and the member nothing.
func (s *Service) mirrorState(ctx context.Context, m Member, state MemberState) {
	if err := s.mirror.Set(ctx, m, state); err != nil {
		s.logger.Warn("chat: mirroring member state failed",
			"member", m.Team+"/"+m.Name, "state", state, "err", err)
	}
}

// mirrorGone withdraws m's published state. Failing is ordinary rather than
// notable: a member stops attending because its session is ending, so whatever
// held the display is often gone before the withdrawal reaches it.
func (s *Service) mirrorGone(ctx context.Context, m Member) {
	if err := s.mirror.Clear(ctx, m); err != nil {
		s.logger.Debug("chat: withdrawing member state failed",
			"member", m.Team+"/"+m.Name, "err", err)
	}
}

// defaultName names an attendee after its kind and its token, the last things
// left to name it by once neither the request nor the team-info provider
// supplied a name.
//
// The kind leads because the name is what everyone else in the room reads: a
// member that declared itself a human and answers at its own pace must not be
// addressed as an agent whose terminal is typed into. The stored kind is the
// word itself, so the prefix is spelled from it rather than mapped again.
func defaultName(token string, kind MemberKind) string {
	if len(token) > tokenNamePrefixLen {
		token = token[:tokenNamePrefixLen]
	}
	return string(kind) + "-" + token
}
