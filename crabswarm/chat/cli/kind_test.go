package cli

import (
	"strings"
	"testing"

	"gotest.tools/v3/assert"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

func TestParseMemberKind(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want chatv1.MemberKind
	}{
		{"agent", chatv1.MemberKind_MEMBER_KIND_AGENT},
		{"human", chatv1.MemberKind_MEMBER_KIND_HUMAN},
	} {
		got, err := ParseMemberKind(tc.in)
		assert.NilError(t, err)
		assert.Equal(t, got, tc.want)
	}

	// A join that declares nothing is refused, and the refusal names the flag
	// and both words, so a caller still wired to a flagless join is told what to
	// add rather than admitted as the wrong sort of member.
	_, err := ParseMemberKind("")
	assert.ErrorContains(t, err, "--kind")
	assert.ErrorContains(t, err, "agent or human")

	for _, bad := range []string{
		"AGENT", "Human", "harness", "person", "bot", "unspecified", "unknown",
	} {
		_, err := ParseMemberKind(bad)
		assert.Assert(t, err != nil, "kind %q should not parse", bad)
		assert.Assert(t, strings.Contains(err.Error(), "--kind"),
			"refusal of %q should name the flag, got %q", bad, err)
	}
}

func TestMemberKindName(t *testing.T) {
	assert.Equal(t, MemberKindName(chatv1.MemberKind_MEMBER_KIND_AGENT), "agent")
	assert.Equal(t, MemberKindName(chatv1.MemberKind_MEMBER_KIND_HUMAN), "human")
	assert.Equal(t, MemberKindName(chatv1.MemberKind_MEMBER_KIND_UNSPECIFIED), "unknown")
}

// The names offered for completion must be exactly the ones the parser accepts,
// or a shell-completed value would be rejected.
func TestMemberKindNamesMatchParser(t *testing.T) {
	names := MemberKindNames()
	assert.DeepEqual(t, names, []string{"agent", "human"})
	assert.Equal(t, len(names), len(memberKinds))
	for _, name := range names {
		kind, err := ParseMemberKind(name)
		assert.NilError(t, err)
		assert.Equal(t, MemberKindName(kind), name)
	}
}
