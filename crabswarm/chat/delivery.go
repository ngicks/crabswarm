package chat

import (
	"context"
	"log/slog"
	"time"
)

// deliverer appends a message to a room, announces it there and nudges the
// agents it woke.
//
// It is a value the services hold rather than a method on one of them: the
// member-facing and the host-facing half write into the same rooms, and only
// who is allowed to ask differs. Keeping the step here lets the second caller
// send without reaching into the first.
type deliverer struct {
	store    *Store
	notifier Notifier
	logger   *slog.Logger
}

// newDeliverer returns the send step over store, reporting mentions to
// notifier. A nil notifier means [NopNotifier], a nil logger discards logs.
func newDeliverer(store *Store, notifier Notifier, logger *slog.Logger) deliverer {
	if notifier == nil {
		notifier = NopNotifier{}
	}
	if logger == nil {
		logger = slog.New(slog.DiscardHandler)
	}
	return deliverer{store: store, notifier: notifier, logger: logger}
}

// send appends text to from's room and nudges whoever it woke. The error is the
// store's own; mapping it onto a status belongs to the RPC that asked.
//
// The room hears about every message, nudged or not: the feed is what an open
// stream reads the conversation from, and a board post is still something that
// happened in the room.
func (d deliverer) send(
	ctx context.Context,
	from Sender,
	target Target,
	text string,
	sentAt time.Time,
) (Sent, error) {
	sent, err := d.store.Send(ctx, from, target, text, sentAt)
	if err != nil {
		return Sent{}, err
	}
	d.store.events.publish(from.Room, messageAppendedEvent(sent.Message))
	for _, m := range d.nudged(ctx, from, sent) {
		d.notify(ctx, m, from, text)
	}
	return sent, nil
}

// nudged is who the message is worth interrupting for.
//
// Only a mention nudges, and only an agent is nudgeable — a nudge is keystrokes
// typed into a terminal, which anyone else neither has nor wants. Whether the
// terminal may be typed into right now is the notifier's call, made from the
// state the member last reported.
func (d deliverer) nudged(ctx context.Context, from Sender, sent Sent) []Member {
	switch sent.Message.Target.Kind {
	case TargetRoles:
		var nudged []Member
		for _, m := range sent.Mentioned {
			// The absent need no filtering of their own: a role nobody attends
			// under comes back carrying no kind at all, so it is not an agent and
			// falls out here. Their mention waits at their read position.
			if m.Kind == KindAgent {
				nudged = append(nudged, m)
			}
		}
		return nudged
	case TargetEveryone:
		return d.attendingAgents(ctx, from)
	default:
		// A board post is addressed to nobody, so it interrupts nobody: it sits
		// in the room for whoever comes looking.
		return nil
	}
}

// attendingAgents is every agent attending from's room but from itself: an
// agent announcing something already knows it, and the echo would cost it a
// nudge and a read.
//
// The sender is matched by role rather than by token, since the host operator
// sends into a room without holding one. One session per role makes that exact.
func (d deliverer) attendingAgents(ctx context.Context, from Sender) []Member {
	members, err := d.store.ListMembers(ctx, from.Room)
	if err != nil {
		// The message is already in the room, so this costs its readers a late
		// read rather than the message.
		d.logger.Warn("chat: listing a room to nudge it failed",
			"room", from.Room, "err", err)
		return nil
	}
	var nudged []Member
	for _, m := range members {
		if m.Kind != KindAgent || (m.Team == from.Team && m.Name == from.Name) {
			continue
		}
		nudged = append(nudged, m)
	}
	return nudged
}

// notify reports one mention, logging what the notifier could not do. The
// message is already recorded, so a failed nudge costs the recipient a late
// read, not the message.
func (d deliverer) notify(ctx context.Context, recipient Member, from Sender, text string) {
	if err := d.notifier.Notify(ctx, recipient, from, text); err != nil {
		d.logger.Warn("chat: notifying recipient failed",
			"recipient", recipient.Team+"/"+recipient.Name, "err", err)
	}
}
