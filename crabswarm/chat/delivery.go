package chat

import (
	"context"
	"log/slog"
	"time"
)

// deliverer puts a message in the inbox of everyone it is addressed to and
// reports each delivery to the notifier.
//
// It is a value the services hold rather than a method on one of them: the
// member-facing and the host-facing half put the same message in the same
// inbox, and only who is allowed to ask differs. Keeping the step here lets the
// second caller deliver without reaching into the first.
type deliverer struct {
	store    *Store
	notifier Notifier
	logger   *slog.Logger
}

// newDeliverer returns the delivery step over store, reporting deliveries to
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

// send delivers text to the one member addr resolves to within from's room and
// returns that member. The error is the store's own; mapping it onto a status
// belongs to the RPC that asked.
func (d deliverer) send(
	ctx context.Context,
	from Member,
	addr, text string,
	sentAt time.Time,
) (Member, error) {
	recipient, err := d.store.Send(ctx, from.Token, addr, text, sentAt)
	if err != nil {
		return Member{}, err
	}
	d.notify(ctx, recipient, senderOf(from), text)
	return recipient, nil
}

// broadcast delivers text to every member of from's room and returns the
// recipients. excludeSender leaves from out, as [Store.Broadcast] describes.
func (d deliverer) broadcast(
	ctx context.Context,
	from Member,
	text string,
	sentAt time.Time,
	excludeSender bool,
) ([]Member, error) {
	recipients, err := d.store.Broadcast(ctx, from.Token, text, sentAt, excludeSender)
	if err != nil {
		return nil, err
	}
	sender := senderOf(from)
	for _, r := range recipients {
		d.notify(ctx, r, sender, text)
	}
	return recipients, nil
}

// sendAs is [deliverer.send] for a sender that holds no member row: from is
// both the perspective addr resolves from and the attribution the message
// keeps.
func (d deliverer) sendAs(
	ctx context.Context,
	from Sender,
	addr, text string,
	sentAt time.Time,
) (Member, error) {
	recipient, err := d.store.sendAs(ctx, from, addr, text, sentAt)
	if err != nil {
		return Member{}, err
	}
	d.notify(ctx, recipient, from, text)
	return recipient, nil
}

// broadcastAs is [deliverer.broadcast] for a sender that holds no member row.
// Nobody is left out: a sender outside the room is not in it to be excluded.
func (d deliverer) broadcastAs(
	ctx context.Context,
	from Sender,
	text string,
	sentAt time.Time,
) ([]Member, error) {
	recipients, err := d.store.broadcastAs(ctx, from, text, sentAt)
	if err != nil {
		return nil, err
	}
	for _, r := range recipients {
		d.notify(ctx, r, from, text)
	}
	return recipients, nil
}

// notify reports one delivery, logging what the notifier could not do. The
// message is already stored, so a failed nudge costs the recipient a late read,
// not the message.
func (d deliverer) notify(ctx context.Context, recipient Member, from Sender, text string) {
	if err := d.notifier.Notify(ctx, recipient, from, text); err != nil {
		d.logger.Warn("chat: notifying recipient failed",
			"recipient", recipient.Team+"/"+recipient.Name, "err", err)
	}
}
