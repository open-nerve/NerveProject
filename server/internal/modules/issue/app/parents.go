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

// inReach reports whether the caller actor may put the undeleted work
// item i under another of the project projectID, or another under it,
// under h, his write's lock on the project (M4 design 3.4, 3.8, 3.9): it
// is the project's, not archived, and he sees it (the guest rule with h's
// grant, Issue.HiddenFrom). Whether that makes a loop is the write's to
// check (domain.Loops).
func inReach(h held, actor, projectID uuid.UUID, i domain.Issue) bool {
	return i.ProjectID == projectID && i.ArchivedAt == nil && !i.HiddenFrom(actor, h.grant.OwnOnly)
}

// parentOf reads the work item id that the caller actor asks for as the
// parent of a work item of the project projectID, under h, his write's
// lock on it, and answers it when it may be one: an undeleted work item in
// his reach (inReach); ok is false for any other.
func parentOf(ctx context.Context, issues IssueReader, h held, actor, projectID, id uuid.UUID) (p domain.Issue, ok bool, err error) {
	p, found, err := issues.GetIssue(ctx, id)
	if err != nil || !found || !inReach(h, actor, projectID, p) {
		return domain.Issue{}, false, err
	}
	return p, true, nil
}

// parentChange checks the parent to, nil for none, that the caller actor's
// update sets on the work item h holds, under h, and names the change as
// the work item's activity writes it (row 5 of M4 design 3.13). A new
// parent may be one (parentOf) that is neither the work item nor a work
// item under it, the work item not in its ancestry (domain.Loops); ok is
// false otherwise. The old parent is the work item's parent before
// (parentsBefore).
func parentChange(ctx context.Context, issues IssueUpdater, h held, actor uuid.UUID, to *uuid.UUID) (c domain.Changed[*domain.Ref], ok bool,
	err error) {
	i := h.issue
	if to != nil {
		parent, ok, err := parentOf(ctx, issues, h, actor, i.ProjectID, *to)
		if err != nil || !ok {
			return domain.Changed[*domain.Ref]{}, false, err
		}
		ancestry, err := issues.Ancestry(ctx, parent.ID)
		if err != nil || domain.Loops(i.ID, ancestry) {
			return domain.Changed[*domain.Ref]{}, false, err
		}
		c.New = refOf(h.project, parent)
	}
	before, err := parentsBefore(ctx, issues, h, []domain.Issue{i})
	if err != nil {
		return domain.Changed[*domain.Ref]{}, false, err
	}
	c.Old = before[0]
	return c, true, nil
}

// parentsBefore reads the parents that the work items items, of the
// project h holds locked, are under before a write takes them from there
// (M4 design 3.9, 3.13): under the lock, each parent once, in id order,
// and none when no work item is under one. It answers each work item's as
// its activity names it, by its identifier (refOf), nil for none. A parent
// before that is not there is an error: deleting a work item deletes those
// under it (3.6).
func parentsBefore(ctx context.Context, issues IssueReader, h held, items []domain.Issue) ([]*domain.Ref, error) {
	var ids []uuid.UUID
	for _, i := range items {
		if i.ParentID != nil && !slices.Contains(ids, *i.ParentID) {
			ids = append(ids, *i.ParentID)
		}
	}
	before := make([]*domain.Ref, len(items))
	if len(ids) == 0 {
		return before, nil
	}
	parents, err := issues.GetIssues(ctx, slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare))
	if err != nil {
		return nil, err
	}
	for n, i := range items {
		if i.ParentID == nil {
			continue
		}
		at := slices.IndexFunc(parents, func(p domain.Issue) bool { return p.ID == *i.ParentID })
		if at < 0 {
			return nil, fmt.Errorf("work item %s is under %s, which is not there", i.ID, *i.ParentID)
		}
		before[n] = refOf(h.project, parents[at])
	}
	return before, nil
}

// refOf is the work item i, of the project p, locked, as an activity names
// it: by its identifier under the lock (M4 design 3.13).
func refOf(p LockedProject, i domain.Issue) *domain.Ref {
	return &domain.Ref{ID: i.ID, Identifier: domain.Identifier(p.Identifier, i.SequenceID)}
}
