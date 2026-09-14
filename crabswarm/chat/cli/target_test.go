package cli

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// Every written target parses into the case it means and spells itself back as
// what was written, which is what lets a rendered line be typed back in.
func TestParseTarget(t *testing.T) {
	for _, tc := range []struct {
		name    string
		written string
		// want is how the parsed target reads back; a post reads as "-" rather
		// than as the empty string it was written with.
		want  string
		roles []string
	}{
		{"the whole room", "everyone", "everyone", nil},
		{"the whole room with spaces around it", " everyone ", "everyone", nil},
		{"one qualified role", "devenv/claude-1", "devenv/claude-1", []string{"devenv/claude-1"}},
		{"one bare role", "claude-1", "claude-1", []string{"claude-1"}},
		{
			"several roles",
			"claude-1,devenv/claude-2",
			"claude-1,devenv/claude-2",
			[]string{"claude-1", "devenv/claude-2"},
		},
		{
			// A list written for a human to read has spaces in it, and a target
			// is not worth refusing over one.
			"spaces around the roles",
			" claude-1 , devenv/claude-2 ",
			"claude-1,devenv/claude-2",
			[]string{"claude-1", "devenv/claude-2"},
		},
		{"a board post", "", "-", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseTarget(tc.written)
			assert.NilError(t, err)
			assert.Equal(t, TargetString(got), tc.want)

			var roles []string
			for _, r := range got.GetRoles().GetRoles() {
				roles = append(roles, roleAddress(r.GetTeam(), r.GetName()))
			}
			assert.DeepEqual(t, roles, tc.roles)
		})
	}
}

// A post is the absence of a target, which is how the wire says "for nobody".
func TestParseTarget_PostCarriesNothing(t *testing.T) {
	got, err := ParseTarget("")
	assert.NilError(t, err)
	assert.Assert(t, got == nil)
}

// A bare name keeps its team empty rather than being guessed at here: resolving
// it — the sender's team first, then across the room — is the daemon's job, and
// only the daemon knows who is in the room.
func TestParseTarget_BareNameLeavesTheTeamToTheDaemon(t *testing.T) {
	got, err := ParseTarget("alice")
	assert.NilError(t, err)
	roles := got.GetRoles().GetRoles()
	assert.Equal(t, len(roles), 1)
	assert.Equal(t, roles[0].GetTeam(), "")
	assert.Equal(t, roles[0].GetName(), "alice")
}

// A half-written or repeated role is refused here rather than sent: the daemon
// answers an address nothing holds with NotFound, which reads as "no such
// member" and sends the writer looking for the wrong mistake.
func TestParseTarget_Rejects(t *testing.T) {
	for _, tc := range []struct {
		written string
		// says is what the message has to name, so the writer knows what to fix.
		says string
	}{
		{" ", "board post"},
		{"a,,b", "board post"},
		{"alice,", "board post"},
		{"/alice", "team/name"},
		{"backend/", "team/name"},
		{"a/b/c", "team/name"},
		{"alice,alice", "twice"},
		{"backend/alice,backend/alice", "twice"},
		// The star grammar the admin verbs used to take is gone, and the error
		// names the word that replaced it.
		{"*", "everyone"},
		{"backend/*", "everyone"},
		{"everyone,alice", "everyone"},
	} {
		t.Run(tc.written, func(t *testing.T) {
			_, err := ParseTarget(tc.written)
			assert.Assert(t, err != nil)
			assert.Assert(t, strings.Contains(err.Error(), tc.says),
				"error %q does not name %q", err, tc.says)
		})
	}
}

// A sender with no team is the host operator, whose lines read as "admin"
// rather than as "/admin": no attendee is in no team, so there is nothing else
// it could be.
func TestAddress(t *testing.T) {
	assert.Equal(t, Address(member("backend", "alice", "/work")), "backend/alice")
	assert.Equal(t, Address(member("", "admin", "/work")), "admin")
}

// A roles case that names nobody addresses nobody, which is what a post is; the
// daemon never writes one, and a target column that went empty would leave the
// line a field short.
func TestTargetString_EmptyRolesReadAsAPost(t *testing.T) {
	empty := &chatv1.Target{Target: &chatv1.Target_Roles{Roles: &chatv1.Roles{}}}
	assert.Equal(t, TargetString(empty), PostTarget)
	assert.Equal(t, TargetString(nil), PostTarget)
}
