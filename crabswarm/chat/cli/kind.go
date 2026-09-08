package cli

import (
	"fmt"
	"strings"

	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// memberKinds maps the words the CLI accepts for a member kind onto the enum.
// The unspecified kind has no word: the daemon refuses a join that declares
// none, and a word for it would be a way to ask for that refusal.
var memberKinds = map[string]chatv1.MemberKind{
	"agent": chatv1.MemberKind_MEMBER_KIND_AGENT,
	"human": chatv1.MemberKind_MEMBER_KIND_HUMAN,
}

// MemberKindNames returns the accepted member-kind words, in the order they are
// documented. The command wiring uses them for shell completion, so the offer
// cannot drift from what [ParseMemberKind] accepts.
func MemberKindNames() []string {
	return []string{"agent", "human"}
}

// ParseMemberKind maps a kind word onto the declared enum.
//
// Both refusals name the flag rather than only the value. Being typed into is
// not a thing to guess at: whoever is repairing a join that declared no kind —
// a harness still wired to an older spelling of this command, say — has to be
// told which flag to add, and a default would have quietly attended as the
// wrong sort of member instead.
func ParseMemberKind(s string) (chatv1.MemberKind, error) {
	if s == "" {
		return chatv1.MemberKind_MEMBER_KIND_UNSPECIFIED,
			fmt.Errorf("--kind is required: %s", strings.Join(MemberKindNames(), " or "))
	}
	kind, ok := memberKinds[s]
	if !ok {
		return chatv1.MemberKind_MEMBER_KIND_UNSPECIFIED,
			fmt.Errorf("unknown member kind %q: --kind takes %s",
				s, strings.Join(MemberKindNames(), " or "))
	}
	return kind, nil
}

// MemberKindName spells a member's kind as the word [ParseMemberKind] takes, so
// a caller presenting the roster and a caller joining use the same two words. A
// kind outside them — the unspecified one, which is what a member built without
// a kind carries — reads as "unknown", since that is what it tells whoever is
// reading.
func MemberKindName(k chatv1.MemberKind) string {
	for _, name := range MemberKindNames() {
		if memberKinds[name] == k {
			return name
		}
	}
	return "unknown"
}
