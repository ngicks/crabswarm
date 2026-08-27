package chat

import (
	"context"
	"errors"
	"log/slog"
	"sync"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Notifier is told that a member has mail, so a harness that has finished its
// turn can be woken instead of waiting for its agent to poll. It is the seam the keystroke
// injector plugs into; the service only reports deliveries.
//
// Notify is called once per recipient, inside the RPC that delivered the
// message, with that RPC's context. An implementation that needs to outlive
// the call — queueing, retrying, shelling out to a terminal — detaches on its
// own (context.WithoutCancel); putting a queue in the seam would force one on
// implementations that push synchronously.
//
// A returned error never fails the delivery: the message is already in the
// recipient's inbox by then, and the service only logs it.
type Notifier interface {
	Notify(ctx context.Context, recipient Member, from Sender, text string) error
}

// NopNotifier drops every notification. It is what a [Service] runs with until
// a real notifier is configured: messages then wait in the inbox until the
// recipient reads them.
type NopNotifier struct{}

var _ Notifier = NopNotifier{}

// Notify does nothing and never fails.
func (NopNotifier) Notify(context.Context, Member, Sender, string) error { return nil }

// providerCheckTTL is how long a successful team-info lookup vouches for an
// agent's token. Re-checking on every RPC would put a cmdman process launch in
// front of every message; re-checking never would keep vanished agents in the
// room forever. Not configurable: it trades reap latency against RPC latency,
// and neither side of that trade is worth a config key yet.
const providerCheckTTL = 30 * time.Second

// tokenNamePrefixLen is how much of a token a generated member name carries.
// Long enough to stay unique among the handful of agents in a room, short
// enough for another agent to type as an address.
const tokenNamePrefixLen = 8

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
	notifier Notifier
	logger   *slog.Logger

	mu sync.Mutex
	// verified holds, per token, when the provider last vouched for it. See
	// [providerCheckTTL].
	verified map[string]time.Time
}

var _ chatv1.ChatServiceServer = (*Service)(nil)

// NewService returns the ChatService implementation over store, admitting the
// members provider knows and reporting deliveries to notifier. A nil notifier
// means [NopNotifier]; a nil logger discards logs.
func NewService(
	store *Store,
	provider TeamInfoProvider,
	notifier Notifier,
	logger *slog.Logger,
) *Service {
	if notifier == nil {
		notifier = NopNotifier{}
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return &Service{
		store:    store,
		provider: provider,
		notifier: notifier,
		logger:   logger,
		verified: make(map[string]time.Time),
	}
}

// caller resolves the member behind the request's token for every RPC but
// Join, which has no membership to require yet.
//
// A token the store does not hold is Unauthenticated, not NotFound: NotFound on
// these RPCs means the member the caller addressed does not exist, and the two
// must not read alike.
func (s *Service) caller(ctx context.Context) (Member, error) {
	token, err := tokenFromContext(ctx)
	if err != nil {
		return Member{}, err
	}
	m, err := s.store.Member(ctx, token)
	if errors.Is(err, ErrNotFound) {
		return Member{}, status.Error(codes.Unauthenticated,
			"token is not attending any room; join first")
	}
	if err != nil {
		return Member{}, storeStatus(err)
	}
	if !s.stillKnown(ctx, m) {
		return Member{}, status.Error(codes.Unauthenticated,
			"token is no longer known to the team-info provider")
	}
	return m, nil
}

// stillKnown reports whether m may keep attending, dropping it from the store
// when it may not.
//
// Only an agent is checked, and only past [providerCheckTTL]: a human's token
// was minted by the daemon, so no provider has ever heard of it. A lookup that
// fails without a verdict keeps the member — a missing cmdman binary or a
// locked cmdman store would otherwise empty every room at once, and a stale
// member costs far less than that.
func (s *Service) stillKnown(ctx context.Context, m Member) bool {
	if m.Kind != KindAgent || s.recentlyVerified(m.Token) {
		return true
	}
	_, err := s.provider.Resolve(ctx, m.Token)
	switch {
	case err == nil:
		s.recordVerified(m.Token)
		return true
	case !errors.Is(err, ErrUnknownToken):
		s.logger.Warn("chat: team-info lookup failed, keeping member",
			"member", m.Team+"/"+m.Name, "err", err)
		return true
	}
	s.logger.Info("chat: reaping member the provider no longer knows",
		"member", m.Team+"/"+m.Name, "room", m.Room, "err", err)
	if _, err := s.store.RemoveMember(ctx, m.Token); err != nil {
		s.logger.Warn("chat: removing reaped member failed",
			"member", m.Team+"/"+m.Name, "err", err)
	}
	s.forgetVerified(m.Token)
	return false
}

// notify reports one delivery, logging what the notifier could not do. The
// message is already stored, so a failed nudge costs the recipient a late read,
// not the message.
func (s *Service) notify(ctx context.Context, recipient Member, from Sender, text string) {
	if err := s.notifier.Notify(ctx, recipient, from, text); err != nil {
		s.logger.Warn("chat: notifying recipient failed",
			"recipient", recipient.Team+"/"+recipient.Name, "err", err)
	}
}

func (s *Service) recentlyVerified(token string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	at, ok := s.verified[token]
	return ok && time.Since(at) < providerCheckTTL
}

// recordVerified stamps token as vouched for, dropping the entries that have
// expired. Without that sweep the map would keep every token the daemon ever
// admitted: a member removed by an admin RPC passes through neither Leave nor
// the reap path.
func (s *Service) recordVerified(token string) {
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	for t, at := range s.verified {
		if now.Sub(at) >= providerCheckTTL {
			delete(s.verified, t)
		}
	}
	s.verified[token] = now
}

func (s *Service) forgetVerified(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.verified, token)
}

// defaultName names a joiner that reported none after its token, which is the
// only thing about it the daemon knows.
func defaultName(token string) string {
	if len(token) > tokenNamePrefixLen {
		token = token[:tokenNamePrefixLen]
	}
	return "agent-" + token
}

// memberState maps the reported harness state onto the stored one. The
// unspecified state is rejected rather than defaulted: a hook that failed to
// fill it in must not silently mark its agent done, which is the one state that
// invites a keystroke nudge.
func memberState(state chatv1.HarnessState) (MemberState, error) {
	switch state {
	case chatv1.HarnessState_HARNESS_STATE_DONE:
		return StateDone, nil
	case chatv1.HarnessState_HARNESS_STATE_WORKING:
		return StateWorking, nil
	case chatv1.HarnessState_HARNESS_STATE_WAITING:
		return StateWaiting, nil
	default:
		return "", status.Errorf(codes.InvalidArgument,
			"unknown harness state %q", state)
	}
}

// storeStatus maps a store error onto the status code its sentinel means. The
// store's message is kept: it names the address, the room and the colliding
// teams, which is exactly what the caller has to act on.
func storeStatus(err error) error {
	switch {
	case errors.Is(err, ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, ErrAmbiguousName), errors.Is(err, ErrInvalidName):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, ErrNameTaken):
		return status.Error(codes.AlreadyExists, err.Error())
	default:
		return status.Error(codes.Internal, err.Error())
	}
}

func memberProto(m Member) *chatv1.Member {
	return &chatv1.Member{Name: m.Name, Team: m.Team, Room: m.Room}
}

func senderProto(s Sender) *chatv1.Member {
	return &chatv1.Member{Name: s.Name, Team: s.Team, Room: s.Room}
}
