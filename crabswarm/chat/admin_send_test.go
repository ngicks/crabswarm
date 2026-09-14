package chat

import (
	"testing"

	"filippo.io/age"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// attendHuman is [attend] for someone whose terminal is never typed into.
func attendHuman(t *testing.T, s *Store, token, room, team, name string) Member {
	t.Helper()
	m, err := s.Attend(t.Context(), Member{
		Token: token, Name: name, Team: team, Room: room, Kind: KindHuman,
	})
	assert.NilError(t, err)
	return m
}

// adminSend is [AdminService.Send] with a fresh credential, failing the test on
// error.
func adminSend(
	t *testing.T,
	svc *AdminService,
	id age.Identity,
	room string,
	target *chatv1.Target,
	text string,
) *chatv1.AdminSendResponse {
	t.Helper()
	res, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), &chatv1.AdminSendRequest{
		Room: room, Target: target, Text: text,
	})
	assert.NilError(t, err)
	return res
}

func TestAdminService_SendToRoles(t *testing.T) {
	svc, id, notifier := newTestAdminServiceWithNotifier(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")
	attendHuman(t, svc.store, "tok-h", testRoom, "hosts", "hana")
	carl := attend(t, svc.store, "tok-c", testRoom, "alpha", "carl")
	_, err := svc.store.Detach(t.Context(), carl.Token)
	assert.NilError(t, err)

	// Bare names, resolved across the whole room: the operator sits in no team
	// whose colleague could win the tie.
	res := adminSend(t, svc, id, testRoom, to("ana", "hana", "carl"), "ship it")
	assert.DeepEqual(t, addresses(res.GetMentioned()),
		[]string{"alpha/ana", "hosts/hana", "alpha/carl"})
	assert.DeepEqual(t, addresses(res.GetAbsent()), []string{"alpha/carl"})
	assert.DeepEqual(t, notifier.nudged(), []string{"alpha/ana"})

	// Attributed to the host: the name is reserved by the empty team, which no
	// member can have, so nobody in the room can write under it.
	said, err := svc.store.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, said[0].From.Name, adminName)
	assert.Equal(t, said[0].From.Team, "")
}

func TestAdminService_SendToEveryone(t *testing.T) {
	svc, id, notifier := newTestAdminServiceWithNotifier(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")
	attend(t, svc.store, "tok-b", testRoom, "beta", "bob")
	attendHuman(t, svc.store, "tok-h", testRoom, "hosts", "hana")

	// Nobody in particular is named, and the operator is nobody's colleague, so
	// every attending agent hears it.
	res := adminSend(t, svc, id, testRoom, everyone(), "the host is restarting")
	assert.Equal(t, len(res.GetMentioned()), 0)
	assert.DeepEqual(t, notifier.nudged(), []string{"alpha/ana", "beta/bob"})
}

func TestAdminService_SendPostNudgesNobody(t *testing.T) {
	svc, id, notifier := newTestAdminServiceWithNotifier(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")

	res := adminSend(t, svc, id, testRoom, nil, "maintenance window at 18:00")
	assert.Equal(t, len(res.GetMentioned()), 0)
	assert.Equal(t, len(notifier.nudged()), 0)

	said, err := svc.store.ReadRoom(t.Context(), testRoom, ReadFilter{})
	assert.NilError(t, err)
	assert.DeepEqual(t, texts(said), []string{"maintenance window at 18:00"})
}

func TestAdminService_SendRefuses(t *testing.T) {
	svc, id, _ := newTestAdminServiceWithNotifier(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "bob")
	attend(t, svc.store, "tok-b", testRoom, "beta", "bob")

	for _, tc := range []struct {
		name string
		req  *chatv1.AdminSendRequest
		want codes.Code
	}{
		{
			"no room",
			&chatv1.AdminSendRequest{Target: everyone(), Text: "hi"},
			codes.InvalidArgument,
		},
		{
			"no text",
			&chatv1.AdminSendRequest{Room: testRoom, Target: everyone()},
			codes.InvalidArgument,
		},
		{
			"a role the room has never had",
			&chatv1.AdminSendRequest{
				Room: testRoom, Target: to("nobody"), Text: "hi",
			},
			codes.NotFound,
		},
		{
			// With no team of its own the sender has no colleague to prefer, so
			// the operator is told to write it as "team/name".
			"a bare name two teams carry",
			&chatv1.AdminSendRequest{
				Room: testRoom, Target: to("bob"), Text: "hi",
			},
			codes.InvalidArgument,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := svc.Send(adminCtx(t, adminNonce(t, svc, id)), tc.req)
			assert.Equal(t, status.Code(err), tc.want)
		})
	}

	// A qualified name gets through where the bare one could not.
	res := adminSend(t, svc, id, testRoom, to("beta/bob"), "hi")
	assert.DeepEqual(t, addresses(res.GetMentioned()), []string{"beta/bob"})
}

// The room hears about the message whoever sent it: an open stream reads the
// conversation off the feed, and an operator's message is part of it.
func TestAdminService_SendIsAnnouncedToTheRoom(t *testing.T) {
	svc, id, _ := newTestAdminServiceWithNotifier(t)
	attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")
	watcher := svc.store.events.subscribe(testRoom)
	defer svc.store.events.unsubscribe(watcher)

	adminSend(t, svc, id, testRoom, to("ana"), "ping")
	assert.Equal(t, describeEvent(nextEvent(t, watcher.events)), "message:/admin:ping")
}
