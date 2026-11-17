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
// river_job keeps no HTML (R4). An event to enqueue that the fan-out would
// refuse fails the write.
func NewNotifyOnIssueEvents(fanouts Fanouts) shared.EventSubscriber {
	return shared.EventSubscriber{Name: "notify_on_issue_events", Types: domain.EventTypes(), Handle: func(ctx context.Context, e shared.Event) error {
		if !domain.NotifiableOf(e).AnyBut(e.ActorID) {
			return nil
		}
		if _, err := workItemOf(e); err != nil {
			return err
		}
		args := e
		args.Changes = domain.WithoutHTML(e.Changes)
		return fanouts.Enqueue(ctx, args)
	}}
}
