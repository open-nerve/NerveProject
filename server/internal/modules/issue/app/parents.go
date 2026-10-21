package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The parent a write puts a work item under (M4 design 3.4, 3.8, 3.9),
// read under the project's lock, which every write of a parent holds (3.5),
// so that the work items' lines stay as read until the write commits.

// parentOf reads the work item id that the caller actor asks for as the
// parent of a work item of the project projectID, under h, his write's
// lock on it, and answers it when it may be one: an undeleted work item of
// the project, not archived, that he sees (the guest rule with h's grant,
// Issue.HiddenFrom); ok is false for any other.
func parentOf(ctx context.Context, issues IssueReader, h held, actor, projectID, id uuid.UUID) (p domain.Issue, ok bool, err error) {
	p, found, err := issues.GetIssue(ctx, id)
	if err != nil || !found {
		return domain.Issue{}, false, err
	}
	if p.ProjectID != projectID || p.ArchivedAt != nil || p.HiddenFrom(actor, h.grant.OwnOnly) {
		return domain.Issue{}, false, nil
	}
	return p, true, nil
}
