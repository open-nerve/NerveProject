package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// NewNotifyOnIssueEvents enqueues, within the write's transaction, the
// fan-out of the notifications of each event of the module's types that
// notifies anyone but its actor of anything (M7 P1b spec 2.9; Notifiable.
// AnyBut, from the event's changes as published): the event, every change at
// its position, the HTML of a description and a comment blanked, so the
// positions the job numbers the notifications with are the event's and
// river_job keeps no HTML (R4). It takes a publication's events at once and
// enqueues theirs at once, in their order (R12). An event to enqueue that the
// fan-out would refuse fails the write, before any is enqueued.
func NewNotifyOnIssueEvents(fanouts Fanouts) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "notify_on_issue_events", Types: domain.EventTypes(), HandleAll: func(ctx context.Context, events []shared.Event) error {
		var sent []shared.Event
		for _, e := range events {
			if !domain.NotifiableOf(e).AnyBut(e.ActorID) {
				continue
			}
			if _, err := workItemOf(e); err != nil {
				return err
			}
			e.Changes = domain.WithoutHTML(e.Changes)
			sent = append(sent, e)
		}
		if len(sent) == 0 {
			return nil
		}
		return fanouts.Enqueue(ctx, sent)
	}}
}
