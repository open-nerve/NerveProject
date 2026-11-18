package app

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateNotificationDeps supply a change of a notification: the decision
// on its workspace, the projects the caller reads there and the guest rule,
// all before the lock; the business clock; the store's single read, lock
// and write; the work items' names; and its transaction, read-write, READ
// COMMITTED, each lock wait at most issue.write_lock_timeout, a wait ended
// so answered as 503 server_busy (bootstrap's busyTx, spec 2.7 step 4).
type UpdateNotificationDeps struct {
	Authorizer shared.Authorizer
	Readable   ReadableProjects
	GuestRule  GuestRule
	Clock      Clock
	Rows       NotificationRows
	Refs       IssueRefs
	Tx         shared.TxManager
}

// UpdateNotification changes the caller's notification's three states
// (spec 2.2, 2.7): PATCH /notifications/{notification_id}.
type UpdateNotification struct{ d UpdateNotificationDeps }

// NewUpdateNotification constructs the change.
func NewUpdateNotification(d UpdateNotificationDeps) *UpdateNotification {
	return &UpdateNotification{d: d}
}

// Execute is spec 2.7's seven steps, in their order, so that its answer
// never tells whether a notification the caller cannot reach exists (R5):
// every such one is ErrNotificationNotFound, the same value, before the
// clock is read and before p is checked, so even p's 422 comes only after.
// Steps 1 to 3 run before any lock and outside any transaction (reach). In
// one transaction then: the row locked, again the caller's and undeleted
// (4); the clock read, after the lock (5); p applied at that moment (6),
// the only place a 422 comes from; and, when that changes a state, the
// write at that moment by the caller, else the row as the lock read it
// (7). The caller is answered the row with its view, the work items it
// names asked for in the transaction with the projects of step 3.
func (u *UpdateNotification) Execute(ctx context.Context, id uuid.UUID, p domain.Patch) (NotificationView, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return NotificationView{}, err
	}
	grants, err := u.reach(ctx, actor, id)
	if err != nil {
		return NotificationView{}, err
	}
	var view NotificationView
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		locked, found, err := u.d.Rows.LockNotification(ctx, id, actor.UserID)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrNotificationNotFound
		}
		now := u.d.Clock.Now()
		next, changed, err := p.Apply(locked.State, now)
		if err != nil {
			return err
		}
		answered := locked.Notification
		if changed {
			if answered, err = u.d.Rows.UpdateNotification(ctx, StateWrite{ID: id, ReceiverID: actor.UserID, State: next, At: now}); err != nil {
				return err
			}
		}
		views, err := viewsOf(ctx, u.d.Refs, actor.UserID, grants, []Notification{answered})
		if err != nil {
			return err
		}
		view = views[0]
		return nil
	})
	if err != nil {
		return NotificationView{}, err
	}
	return view, nil
}

// reach is steps 1 to 3 (spec 2.7), before any lock and outside any
// transaction: the caller's undeleted notification id read without a
// lock; the decision of notification.write on its workspace; and the
// projects he reads there, which its project must be among, its work item
// not hidden from him by the guest rule, the issue module's one predicate,
// with his grant there. It answers those projects. A notification that is
// not his or not there, a workspace Authorize does not let him see, a
// project he does not read and a work item the rule hides are each
// ErrNotificationNotFound, the same value; another refusal is the
// Authorizer's.
func (u *UpdateNotification) reach(ctx context.Context, actor shared.Actor, id uuid.UUID) ([]Grant, error) {
	n, found, err := u.d.Rows.GetNotification(ctx, id, actor.UserID)
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, domain.ErrNotificationNotFound
	}
	_, err = u.d.Authorizer.Authorize(ctx, actor, domain.ActionNotificationWrite, shared.Target{WorkspaceID: n.WorkspaceID})
	if errors.Is(err, shared.ErrNotVisible) {
		return nil, domain.ErrNotificationNotFound
	}
	if err != nil {
		return nil, err
	}
	grants, err := u.d.Readable.Readable(ctx, actor, n.WorkspaceID)
	if err != nil {
		return nil, err
	}
	at := slices.IndexFunc(grants, func(g Grant) bool { return g.ProjectID == n.ProjectID })
	if at < 0 || u.d.GuestRule.HiddenFrom(n.IssueCreatedByID, actor.UserID, grants[at].OwnOnly) {
		return nil, domain.ErrNotificationNotFound
	}
	return grants, nil
}
