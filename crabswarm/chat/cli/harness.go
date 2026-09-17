package cli

import (
	chatv1 "github.com/ngicks/crabswarm/api/gen/proto/go/ngicks/crabswarm/chat/v1"
)

// unnamedValue stands in for a value nobody declared, wherever a roster column
// has nothing to put. A dash rather than a word: the columns are read by
// cutting a line on whitespace, so a blank would shift every column after it,
// and a word like "unknown" would read as something the member said about
// itself. Nothing said it.
const unnamedValue = "-"

// HarnessName spells the CLI a member runs, for the rosters that show it. The
// words are the ones the harnesses are known by rather than the enum's
// spelling, since a reader comparing a roster with the processes on their host
// is looking for those.
func HarnessName(h chatv1.Harness) string {
	switch h {
	case chatv1.Harness_HARNESS_CLAUDE_CODE:
		return "claude-code"
	case chatv1.Harness_HARNESS_CODEX:
		return "codex"
	case chatv1.Harness_HARNESS_OPENCODE:
		return "opencode"
	case chatv1.Harness_HARNESS_OTHER:
		return "other"
	default:
		return unnamedValue
	}
}

// NudgeDeliveryName spells how a mention reaches a member: typed into its
// terminal by the daemon, or pushed to it by its own server through the
// harness's notification channel.
//
// A member with no delivery at all is one nothing is ever pushed to, which is
// what a human is — they read their room when they choose to.
func NudgeDeliveryName(n chatv1.NudgeDelivery) string {
	switch n {
	case chatv1.NudgeDelivery_NUDGE_DELIVERY_TERMINAL:
		return "terminal"
	case chatv1.NudgeDelivery_NUDGE_DELIVERY_NATIVE:
		return "native"
	default:
		return unnamedValue
	}
}
