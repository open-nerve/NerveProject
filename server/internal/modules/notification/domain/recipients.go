package domain

import (
	"slices"
	"uuid"
)

// Sender is why a receiver is notified (spec 2.4, M7 design 3.3), settled
// when the notification is sent.
type Sender string

// The senders of notifications, values of notifications.sender.
const (
	SenderCreated    Sender = "created"
	SenderAssigned   Sender = "assigned"
	SenderSubscribed Sender = "subscribed"
	SenderMentioned  Sender = "mentioned"
)

// Issue is the work item as the rules read it under its row's lock: its
// creator, nil when not known, and its current assignees.
type Issue struct {
	CreatedByID *uuid.UUID
	AssigneeIDs []uuid.UUID
}

// Row is a notification to send: its receiver, its sender and the change it
// is of.
type Row struct {
	Receiver uuid.UUID
	Sender   Sender
	Item     Item
}

// Recipients are the notifications n is sent as, before the recheck (spec
// 2.4), of the work item i, whose subscribers are subscribers, written by
// actor, nil for nerve's own write: each mention its account's, as
// mentioned, but the actor's; then each subscriber, but the actor and those
// n mentions, one for each change n notifies of, as created when he created
// i, else as assigned when he is one of its assignees, else as subscribed.
func Recipients(n Notifiable, actor *uuid.UUID, subscribers []uuid.UUID, i Issue) []Row {
	var rows []Row
	mentioned := map[uuid.UUID]bool{}
	for _, m := range n.Mentions {
		if actor != nil && m.Account == *actor {
			continue
		}
		mentioned[m.Account] = true
		rows = append(rows, Row{Receiver: m.Account, Sender: SenderMentioned, Item: m.Item})
	}
	for _, s := range subscribers {
		if (actor != nil && s == *actor) || mentioned[s] {
			continue
		}
		sender := senderOf(s, i)
		for _, item := range n.Changes {
			rows = append(rows, Row{Receiver: s, Sender: sender, Item: item})
		}
	}
	return rows
}

// senderOf is the sender of a subscriber's notification of i: created for
// its creator, else assigned for one of its assignees, else subscribed.
func senderOf(account uuid.UUID, i Issue) Sender {
	switch {
	case i.CreatedByID != nil && *i.CreatedByID == account:
		return SenderCreated
	case slices.Contains(i.AssigneeIDs, account):
		return SenderAssigned
	}
	return SenderSubscribed
}

// Receivers are the accounts of rows, each once, in id order: those the
// recheck decides on.
func Receivers(rows []Row) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.Receiver)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	return slices.Compact(ids)
}

// Decision is whether an account may read the work item's project's work
// items at the moment of the recheck, and whether only those he created:
// the access module's decision of issue.read (spec 2.7).
type Decision struct {
	Allowed bool
	OwnOnly bool
}

// HiddenFrom is the guest rule (M4 design 3.4): whether a work item whose
// creator is creator, nil when not known, is hidden from the account, whose
// grant on its project is ownOnly. The issue module's Issue.HiddenFrom is
// the one predicate; the fan-out hands it to Visible.
type HiddenFrom func(creator *uuid.UUID, account uuid.UUID, ownOnly bool) bool

// Visible are the rows whose receivers decisions allow to read the work
// item i (spec 2.4) and from whom hiddenFrom does not hide it, by the
// creator i names; one decisions do not name is not allowed.
func Visible(rows []Row, decisions map[uuid.UUID]Decision, i Issue, hiddenFrom HiddenFrom) []Row {
	var kept []Row
	for _, r := range rows {
		if d := decisions[r.Receiver]; d.Allowed && !hiddenFrom(i.CreatedByID, r.Receiver, d.OwnOnly) {
			kept = append(kept, r)
		}
	}
	return kept
}
