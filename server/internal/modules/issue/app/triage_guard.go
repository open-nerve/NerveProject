package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// refuseTriage is the check (write.check) of a write that changes its own
// work item i, as Locks reads it before the lock and again under it (M7
// P2a spec 2.6, R4, DP-6): a work item in its project's intake, its intake
// row not accepted (domain.Issue.InTriage), is 409 issue.in_triage. Locks
// runs it after the decision, so after the 404 and the 403, and before
// every refusal of the write's own under the lock: project.archived,
// issue.archived, and an archiving's or unarchiving's answer as read of a
// work item already as asked. A write takes it by choice, not by its
// action: updateIssue, archiveIssue and unarchiveIssue, deleteIssue,
// addSubIssues of the parent, createIssueRelations of the source. The
// writes a work item in the intake takes as any other, its links' though
// issue.update decides them, its comments', its reactions', its
// subscription, do not; nor does deleteIssueRelation, whose write on
// either work item runs no check, a work item in the intake having no
// relation (pre-flight re-85 I3).
func refuseTriage(_ context.Context, i domain.Issue, _ shared.Grant) error {
	if i.InTriage() {
		return domain.ErrInTriage
	}
	return nil
}
