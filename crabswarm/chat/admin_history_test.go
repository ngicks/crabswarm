package chat

import (
	"fmt"
	"testing"

	"filippo.io/age"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// seedConversation puts n numbered messages in the room, addressed to everyone,
// and returns the member that said them.
func seedConversation(t *testing.T, svc *AdminService, n int) Member {
	t.Helper()
	ana := attend(t, svc.store, "tok-a", testRoom, "alpha", "ana")
	for i := range n {
		send(t, svc.store, senderOf(ana), Target{Kind: TargetEveryone},
			fmt.Sprintf("note-%d", i))
	}
	return ana
}

// adminHistory is [AdminService.History] with a fresh credential, failing the
// test on error.
func adminHistory(
	t *testing.T,
	svc *AdminService,
	id age.Identity,
	filter *chatv1.ReadFilter,
) []string {
	t.Helper()
	res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{Room: testRoom, Filter: filter})
	assert.NilError(t, err)
	return messageTexts(res.GetMessages())
}

// A reader opening a room wants its end, so a request that names no cursor gets
// the last ten rather than the whole retained conversation.
func TestAdminService_HistoryDefaultsToTheLastTen(t *testing.T) {
	svc, id := newTestAdminService(t)
	seedConversation(t, svc, 12)

	got := adminHistory(t, svc, id, nil)
	assert.Equal(t, len(got), 10)
	assert.Equal(t, got[0], "note-2")
	assert.Equal(t, got[9], "note-11")
}

func TestAdminService_HistoryHonoursTheFilter(t *testing.T) {
	svc, id := newTestAdminService(t)
	seedConversation(t, svc, 5)

	assert.DeepEqual(t,
		adminHistory(t, svc, id, &chatv1.ReadFilter{
			Cursor: chatv1.ReadCursor_READ_CURSOR_HEAD, Range: 2,
		}),
		[]string{"note-0", "note-1"})

	assert.DeepEqual(t,
		adminHistory(t, svc, id, &chatv1.ReadFilter{
			Cursor: chatv1.ReadCursor_READ_CURSOR_TAIL, Range: -2,
		}),
		[]string{"note-3", "note-4"})

	assert.DeepEqual(t,
		adminHistory(t, svc, id, &chatv1.ReadFilter{
			Cursor: chatv1.ReadCursor_READ_CURSOR_HEAD, Since: 1, Until: 4,
		}),
		[]string{"note-1", "note-2"})
}

// The read moves nothing, so the same stretch reads the same way twice, and it
// marks nothing as mentioning the reader: an operator is not in the room.
func TestAdminService_HistoryConsumesNothing(t *testing.T) {
	svc, id := newTestAdminService(t)
	ana := seedConversation(t, svc, 3)

	first := adminHistory(t, svc, id, nil)
	assert.DeepEqual(t, adminHistory(t, svc, id, nil), first)

	res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{Room: testRoom})
	assert.NilError(t, err)
	for _, m := range res.GetMessages() {
		assert.Assert(t, !m.GetMentionedYou())
	}

	// Ana's own position never moved, so what she was away for is still hers
	// to read.
	msgs, _, err := svc.store.Read(t.Context(), senderOf(ana), ReadFilter{})
	assert.NilError(t, err)
	assert.Equal(t, len(msgs), 0, "the sender's own messages are not her unread")
}

func TestAdminService_HistoryRefuses(t *testing.T) {
	svc, id := newTestAdminService(t)
	seedConversation(t, svc, 2)

	_, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	// Unread is measured from a role's read position, and this read has none.
	_, err = svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{
			Room:   testRoom,
			Filter: &chatv1.ReadFilter{Cursor: chatv1.ReadCursor_READ_CURSOR_UNREAD},
		})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)

	// A range against its cursor asks for what cannot exist.
	_, err = svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{
			Room: testRoom,
			Filter: &chatv1.ReadFilter{
				Cursor: chatv1.ReadCursor_READ_CURSOR_TAIL, Range: 1,
			},
		})
	assert.Equal(t, status.Code(err), codes.InvalidArgument)
}

// A room is what was said in it, so one nobody has spoken in answers with
// nothing rather than with NotFound.
func TestAdminService_HistoryOfARoomNobodyUsed(t *testing.T) {
	svc, id := newTestAdminService(t)

	res, err := svc.History(adminCtx(t, adminNonce(t, svc, id)),
		&chatv1.AdminHistoryRequest{Room: "/work/nowhere"})
	assert.NilError(t, err)
	assert.Equal(t, len(res.GetMessages()), 0)
}
