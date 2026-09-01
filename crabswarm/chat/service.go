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
	"github.com/ngicks/crabswarm/crabswarm/chat/resolver"
)

// Notifier is told that a member has mail, so a harness that has finished its
// turn can be woken instead of waiting for its agent to poll. It is the seam
// the keystroke injector plugs into; the service only reports deliveries.
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

	mu sync.Mutex
	// verified holds, per token, when the provider last vouched for it. See
	// [providerCheckTTL].
	verified map[string]time.Time
}

var _ chatv1.ChatServiceServer = (*Service)(nil)

// NewService returns the ChatService implementation over store, admitting the
// members provider knows, reporting deliveries to notifier and member state to
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
// Only an agent is checked, and only past [providerCheckTTL]: a member that
// declared no harness stays until it leaves, whatever the provider makes of
// its token. A lookup that carried no verdict is not remembered either, so the
// next RPC asks again instead of riding an answer nobody gave.
func (s *Service) stillKnown(ctx context.Context, m Member) bool {
	if m.Kind != KindAgent || s.recentlyVerified(m.Token) {
		return true
	}
	switch checkLiveness(ctx, s.store, s.provider, s.logger, s.forgetVerified, m) {
	case memberVouchedFor:
		s.recordVerified(m.Token)
	case memberReaped:
		return false
	}
	return true
}

// livenessVerdict is what the team-info provider had to say about a member,
// as [checkLiveness] reports it.
type livenessVerdict int

const (
	// memberVouchedFor: the provider placed the token, so whoever holds it is
	// still running.
	memberVouchedFor livenessVerdict = iota
	// memberUnjudged: the lookup itself failed, so nothing was learned about
	// the token and its holder stays.
	memberUnjudged
	// memberReaped: the provider no longer knows the token, and its holder is
	// gone from the store.
	memberReaped
)

// checkLiveness asks the provider about m and drops m from the store when the
// provider no longer knows its token. It is the one definition of a member
// being gone, shared by the lazy reap the member half runs before every RPC and
// by the name-collision paths of both halves: a flaky cmdman must not free
// names any more than it may empty rooms.
//
// Only an agent is asked about: an agent is gone when the session that carried
// it is, while anyone else stays until they say otherwise, and a name they hold
// is theirs until an operator moves them. A lookup that fails without a verdict
// keeps the member — a missing cmdman binary or a locked cmdman store would
// otherwise empty every room at once, and a stale member costs far less than
// that.
//
// forget is handed the token of a member that is reaped, so no cached verdict
// outlives the member it vouched for. It is nil for a caller holding no such
// cache, which the admin half is.
func checkLiveness(
	ctx context.Context,
	store *Store,
	provider TeamInfoProvider,
	logger *slog.Logger,
	forget func(token string),
	m Member,
) livenessVerdict {
	if m.Kind != KindAgent {
		return memberVouchedFor
	}
	_, err := provider.Resolve(ctx, m.Token)
	switch {
	case err == nil:
		return memberVouchedFor
	case !errors.Is(err, resolver.ErrUnknownToken):
		logger.Warn("chat: team-info lookup failed, keeping member",
			"member", m.Team+"/"+m.Name, "err", err)
		return memberUnjudged
	}
	// No status is withdrawn here: the member is reaped because the provider
	// no longer knows its token, which means the command that carried the
	// display is already gone.
	logger.Info("chat: reaping member the provider no longer knows",
		"member", m.Team+"/"+m.Name, "room", m.Room, "err", err)
	if _, err := store.RemoveMember(ctx, m.Token); err != nil {
		logger.Warn("chat: removing reaped member failed",
			"member", m.Team+"/"+m.Name, "err", err)
	} else {
		// Unlike the status display, the room hears about a reap: the watchers
		// are the other members' sessions, which are still running and would
		// otherwise keep a vanished member on their list forever.
		store.events.publish(m.Room, memberLeftEvent(m))
	}
	// The verdict is what the cache holds, so it goes even when the removal
	// did not: a token this call has judged gone must not be vouched for by
	// what an earlier call cached about it.
	if forget != nil {
		forget(m.Token)
	}
	return memberReaped
}

// reclaimName frees name within team of room when the member holding it has
// vanished, and reports whether the name is there for the taking. It is what
// tells the ghost of a command that no longer exists from a member that is
// still in the room: a recreated compose replica derives the exact name its
// predecessor left behind, and nothing else would ever free it.
//
// The provider is asked afresh rather than through the member half's cached
// verdicts: a collision is rare enough to be worth a lookup, the answer decides
// whether the caller gets in at all, and a verdict cached moments ago would
// vouch for precisely the holder a recreated replica has just replaced.
func reclaimName(
	ctx context.Context,
	store *Store,
	provider TeamInfoProvider,
	logger *slog.Logger,
	forget func(token string),
	room, team, name string,
) bool {
	holder, err := store.memberNamed(ctx, room, team, name)
	if err != nil {
		// Nobody holds the name any more: whoever did left between the refusal
		// and this lookup. Any other failure leaves the refusal standing.
		return errors.Is(err, ErrNotFound)
	}
	return checkLiveness(ctx, store, provider, logger, forget, holder) == memberReaped
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
// notable: a member leaves because its session is ending, so whatever held the
// display is often gone before the withdrawal reaches it.
func (s *Service) mirrorGone(ctx context.Context, m Member) {
	if err := s.mirror.Clear(ctx, m); err != nil {
		s.logger.Debug("chat: withdrawing member state failed",
			"member", m.Team+"/"+m.Name, "err", err)
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

// defaultName names a joiner after its token, the last thing left to name it by
// once neither the join request nor the team-info provider supplied a name.
func defaultName(token string) string {
	if len(token) > tokenNamePrefixLen {
		token = token[:tokenNamePrefixLen]
	}
	return "agent-" + token
}

// memberState maps the reported harness state onto the stored one. The
// unspecified state is rejected rather than defaulted: a hook that failed to
// fill it in must not silently mark its agent done, which is the one state a
// keystroke nudge is sent to on sight rather than only once the report has
// gone stale.
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

// harnessStateProto maps the stored state back onto the wire enum. Unlike
// [memberState] it refuses nothing: a [Member] carrying no state is one nobody
// recorded a state for, which is a member to describe rather than a request to
// turn down.
func harnessStateProto(state MemberState) chatv1.HarnessState {
	switch state {
	case StateWorking:
		return chatv1.HarnessState_HARNESS_STATE_WORKING
	case StateWaiting:
		return chatv1.HarnessState_HARNESS_STATE_WAITING
	case StateDone:
		return chatv1.HarnessState_HARNESS_STATE_DONE
	default:
		return chatv1.HarnessState_HARNESS_STATE_UNSPECIFIED
	}
}

func memberProto(m Member) *chatv1.Member {
	return &chatv1.Member{
		Name:  m.Name,
		Team:  m.Team,
		Room:  m.Room,
		State: harnessStateProto(m.State),
	}
}

func senderProto(s Sender) *chatv1.Member {
	return &chatv1.Member{Name: s.Name, Team: s.Team, Room: s.Room}
}
