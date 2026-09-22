// Package nudge holds the words a member is woken with and the address those
// words name the sender by.
//
// Two things deliver a nudge: the daemon, typing a line into an agent's
// terminal, and the agent's own MCP server, pushing one through the harness's
// notification channel. The words live here so that both say the same thing —
// a member that changed how it is reached would otherwise find the room
// addressing it in a different voice.
//
// What a notice says is who wrote, or how much is waiting, how to read it, and
// where the answer goes. Never the message itself: the text is sender-controlled
// content, and a line typed into a terminal or pushed into a context is the last
// place to repeat it.
package nudge

import (
	"strconv"
	"strings"
	"unicode"
)

// MaxAddrLen caps the sender address a notice carries. A member name comes from
// the agent that attends and nothing upstream bounds it, so the cap is what
// keeps one line one line; a longer address is cut short.
const MaxAddrLen = 64

// prefix opens every notice, so the agent can tell a line the room put in front
// of it from one it wrote itself.
const prefix = "[crabswarm chat] "

// The notices name the chat_read tool rather than the `crabswarm chat read`
// command. Every harness the room reaches is served by the crabswarm MCP server
// and so has the tool, while some of them decline to run a command nobody asked
// them to run.
//
// They also name chat_send and the server both tools come from. A member
// reached through a harness channel sees that channel's own instructions beside
// the notice, and a channel plugin that tells the model to answer with its own
// reply tool would carry the answer away from the room.

// NewMessage is the notice for one mention that just arrived: who wrote, what
// hands it over, and what answers it.
func NewMessage(from string) string {
	return prefix + "new message from " + from +
		" — read it with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"
}

// Waiting is the notice for mentions that piled up while the member could not
// be interrupted. It counts them rather than naming their senders: by the time
// it is delivered, which of them wrote is what the read itself says.
func Waiting(n int) string {
	if n == 1 {
		return prefix + "1 unread message mentions you" +
			" — read it with the chat_read tool and respond with chat_send," +
			" both from crabswarm-mcp"
	}
	return prefix + strconv.Itoa(n) + " unread messages mention you" +
		" — read them with the chat_read tool and respond with chat_send," +
		" both from crabswarm-mcp"
}

// Address spells a sender the way every chat verb addresses one. A sender with
// no team is the host operator, whose messages read as coming from "admin"
// rather than from "/admin": nobody can attend a room without a team, so there
// is no other unteamed sender to confuse it with.
func Address(team, name string) string {
	if team == "" {
		return name
	}
	return team + "/" + name
}

// Sanitize makes s safe to carry as part of one line. It drops control
// characters — a carriage return typed into a terminal would submit the line
// early and leave the rest of it running as a command of its own — and
// truncates at [MaxAddrLen]. Neither bound is guaranteed upstream: a name is
// whatever the attending agent asked to be called.
func Sanitize(s string) string {
	var cleaned strings.Builder
	cleaned.Grow(len(s))
	runes := 0
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		cleaned.WriteRune(r)
		runes++
		if runes == MaxAddrLen {
			break
		}
	}
	return cleaned.String()
}
