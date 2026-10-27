package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateIssueDeps supplies lock orchestration and the shared creation core.
type CreateIssueDeps struct {
	Locks    Locks
	Creation *IssueCreation
}

// CreateIssue creates a work item in a project.
type CreateIssue struct{ d CreateIssueDeps }

// NewCreateIssue returns the use case.
func NewCreateIssue(d CreateIssueDeps) *CreateIssue { return &CreateIssue{d: d} }

// Execute checks in (domain.CheckNewIssue), before any read; then Locks
// runs the write (M4 design 3.5, 11.11): its precheck, before any lock;
// then the project's write gate, which refuses a write that may wait no
// longer (503 server_busy); then in one transaction, the project
// FOR NO KEY UPDATE and the decision on issue.create, and under them, in
// this order: an archived project refuses it (409 project.archived); the
// dependent rows (check), every problem in one 422; the project's next
// number; the clock; the sort order after the greatest of the state's work
// items; the work item, under its parent when it has one; its assignees,
// the default one too; its labels; its subscribers, the caller and the
// assignees, in id order; its activity: the creation, then each assignee
// by his display name (domain.CreationActivity), and no row of its parent
// (3.13). All of it is the caller's at that moment. The answer is the work
// item as stored, read back in the transaction, which has no sub-items to
// count yet (Issue.SubIssuesCount).
func (u *CreateIssue) Execute(ctx context.Context, projectID uuid.UUID, in domain.NewIssue) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	if in, err = domain.CheckNewIssue(in); err != nil {
		return domain.Issue{}, err
	}
	w := write{project: projectID, action: domain.ActionCreate, notFound: domain.ErrProjectNotFound}
	var created domain.Issue
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		var err error
		created, _, err = u.d.Creation.createLocked(ctx, actor.UserID, projectID, h, in)
		return err
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return created, nil
}
