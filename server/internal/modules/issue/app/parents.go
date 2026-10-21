package app

import (
	"context"
	"fmt"
	"slices"
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
// Issue.HiddenFrom); ok is false for any other. Whether the work item
// would be under itself is its update's to check (Ancestry).
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

// parentChange checks the parent to, nil for none, that the caller actor's
// update sets on the work item h holds, under h, and names the change as
// the work item's activity writes it (row 5 of M4 design 3.13). A new
// parent may be one (parentOf) that is neither the work item nor a work
// item under it, the work item not in its ancestry; ok is false otherwise.
// The old parent, when there is one, is read under the lock: an undeleted
// work item of the project, since deleting a work item deletes those under
// it (3.6), so one not there is an error.
func parentChange(ctx context.Context, issues IssueUpdater, h held, actor uuid.UUID, to *uuid.UUID) (c domain.Changed[*domain.Ref], ok bool,
	err error) {
	i := h.issue
	if to != nil {
		parent, ok, err := parentOf(ctx, issues, h, actor, i.ProjectID, *to)
		if err != nil || !ok {
			return domain.Changed[*domain.Ref]{}, false, err
		}
		ancestry, err := issues.Ancestry(ctx, parent.ID)
		if err != nil || slices.Contains(ancestry, i.ID) {
			return domain.Changed[*domain.Ref]{}, false, err
		}
		c.New = refOf(h, parent)
	}
	if i.ParentID != nil {
		old, found, err := issues.GetIssue(ctx, *i.ParentID)
		switch {
		case err != nil:
			return domain.Changed[*domain.Ref]{}, false, err
		case !found:
			return domain.Changed[*domain.Ref]{}, false, fmt.Errorf("work item %s is under %s, which is not there", i.ID, *i.ParentID)
		}
		c.Old = refOf(h, old)
	}
	return c, true, nil
}

// refOf is the work item i, of the project h holds locked, as an activity
// names it: by its identifier under the lock (M4 design 3.13).
func refOf(h held, i domain.Issue) *domain.Ref {
	return &domain.Ref{ID: i.ID, Identifier: domain.Identifier(h.project.Identifier, i.SequenceID)}
}
