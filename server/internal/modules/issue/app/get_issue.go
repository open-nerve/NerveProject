package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetIssue reads a work item: GET /api/v0/issues/{issue_id} (M4 design
// 5.1).
type GetIssue struct {
	locks Locks
}

// NewGetIssue returns the use case.
func NewGetIssue(locks Locks) *GetIssue {
	return &GetIssue{locks: locks}
}

// Execute reads the undeleted work item id, archived or not, with its
// assignees and labels, and decides issue.read on it for the caller,
// without a lock or a transaction (Locks.see): a work item that is not
// there, whose project the caller does not see, or that the guest rule
// hides from him, is issue.not_found; a role that may not read it,
// forbidden.
func (u *GetIssue) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	return u.locks.see(ctx, actor, id, domain.ActionRead)
}
