package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SubscriptionDeps are the ports of subscribeIssue and unsubscribeIssue:
// the locks, the subscribers' store and the clock. Neither marks the work
// item changed, writes its activity or publishes an event (M7 P1a spec
// 2.4, R6), so neither holds a port to.
type SubscriptionDeps struct {
	Locks         Locks
	Subscriptions SubscriptionWriter
	Clock         Clock
}

// SubscribeIssue subscribes the caller to a work item he may read (M7 P1a
// spec 2.3, 2.4): under its row FOR SHARE (Locks.share), his row of its
// subscribers, at the moment and by him; one who subscribes already keeps
// his row, and the answer is the same. An archived work item, or one of an
// archived project, is subscribed to as any other (R3).
type SubscribeIssue struct{ d SubscriptionDeps }

// NewSubscribeIssue returns subscribeIssue over d's ports.
func NewSubscribeIssue(d SubscriptionDeps) *SubscribeIssue { return &SubscribeIssue{d} }

// Execute subscribes the caller to the work item id.
func (u *SubscribeIssue) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.share(ctx, actor, id, domain.ActionRead, func(ctx context.Context, i SharedIssue) error {
		s := stamp{workspace: i.WorkspaceID, project: i.ProjectID, issue: i.ID, by: actor.UserID, at: u.d.Clock.Now()}
		return u.d.Subscriptions.InsertSubscribers(ctx, s.links([]uuid.UUID{actor.UserID}))
	})
}

// UnsubscribeIssue unsubscribes the caller from a work item he may read
// (M7 P1a spec 2.3, 2.4): under its row FOR SHARE (Locks.share), his one
// undeleted row of its subscribers soft-deleted, at the moment and by him;
// one who does not subscribe has none, and the answer is the same. An
// archived work item, or one of an archived project, as any other (R3).
type UnsubscribeIssue struct{ d SubscriptionDeps }

// NewUnsubscribeIssue returns unsubscribeIssue over d's ports.
func NewUnsubscribeIssue(d SubscriptionDeps) *UnsubscribeIssue { return &UnsubscribeIssue{d} }

// Execute unsubscribes the caller from the work item id.
func (u *UnsubscribeIssue) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	return u.d.Locks.share(ctx, actor, id, domain.ActionRead, func(ctx context.Context, i SharedIssue) error {
		return u.d.Subscriptions.RemoveSubscriber(ctx, SubscriberRemoval{IssueID: i.ID, SubscriberID: actor.UserID, By: actor.UserID, At: u.d.Clock.Now()})
	})
}
