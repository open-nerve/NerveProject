package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateIssueDeps supplies lock orchestration, the shared update core and
// the clock an update's moment is read from.
type UpdateIssueDeps struct {
	Locks  Locks
	Update *IssueUpdate
	Clock  Clock
}

// UpdateIssue changes a work item: PATCH /api/v0/issues/{issue_id} (M4
// design 3.5, 3.7, 5.1, 6.7).
type UpdateIssue struct {
	d UpdateIssueDeps
}

// NewUpdateIssue returns the use case.
func NewUpdateIssue(d UpdateIssueDeps) *UpdateIssue {
	return &UpdateIssue{d: d}
}

// Execute checks p (domain.CheckIssuePatch), before any read; then Locks
// runs the write on the work item id (M4 design 3.5, 6.7, 11.11): before
// any lock, the work item read, the decision on issue.update, the guest
// rule first, then refuseTriage (409 issue.in_triage, M7 P2a spec 2.6);
// the project's write gate (503 server_busy); then in one transaction, the
// project FOR NO KEY UPDATE, the work item read again, the decision and
// refuseTriage again, and under them the update itself (updateLocked), its
// moment read from the use case's clock after its last lock.
func (u *UpdateIssue) Execute(ctx context.Context, id uuid.UUID, p domain.IssuePatch) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if p, err = domain.CheckIssuePatch(p); err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionUpdate, notFound: domain.ErrIssueNotFound, check: refuseTriage}
	var updated domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		updated, err = u.d.Update.updateLocked(ctx, actor.UserID, h, p, u.d.Clock)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return updated, nil
}
