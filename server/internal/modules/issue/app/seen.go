package app

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// grants are the caller's grants on the projects of the work items an
// answer shows, by project, as their decisions answered them.
type grants map[uuid.UUID]shared.Grant

// seen is items as the caller actor sees them (M4 design 3.4, 3.9), by
// his grants on their projects: each with the number of its undeleted,
// unarchived direct sub-items he sees, every one, or those he created
// when the guest rule hides others' from him in its project
// (Grant.OwnOnly), read in one count; and with its parent only when he
// sees it (hideParents). A work item of a project without a grant in
// grants is an error: every answer is decided first.
func (l Locks) seen(ctx context.Context, actor uuid.UUID, g grants, items ...domain.Issue) ([]domain.Issue, error) {
	if len(items) == 0 {
		return nil, nil
	}
	ids := make([]uuid.UUID, len(items))
	for n, i := range items {
		ids[n] = i.ID
	}
	counts, err := l.issues.SubIssueCounts(ctx, actor, ids)
	if err != nil {
		return nil, err
	}
	out := slices.Clone(items)
	var guarded []int
	for n, i := range out {
		grant, ok := g[i.ProjectID]
		if !ok {
			return nil, fmt.Errorf("no grant on project %s, of work item %s", i.ProjectID, i.ID)
		}
		out[n].SubIssuesCount = counts[i.ID].All
		if grant.OwnOnly {
			out[n].SubIssuesCount = counts[i.ID].Own
		}
		if grant.OwnOnly && i.ParentID != nil {
			guarded = append(guarded, n)
		}
	}
	if err := l.hideParents(ctx, actor, out, guarded); err != nil {
		return nil, err
	}
	return out, nil
}

// hideParents answers as none (nil) the parent of each work item at the
// places guarded in items that the guest rule hides from the caller actor
// (M4 design 3.4): their grants hide others' work items from him
// (Grant.OwnOnly), so their parents are read, in one read, where the count
// is read, and one he did not create (domain.Issue.HiddenFrom), or one the
// read does not find, is hidden. Nothing is read when no place is guarded,
// as under a grant that shows him every work item. A failing read is its
// error.
func (l Locks) hideParents(ctx context.Context, actor uuid.UUID, items []domain.Issue, guarded []int) error {
	if len(guarded) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(guarded))
	for k, n := range guarded {
		ids[k] = *items[n].ParentID
	}
	parents, err := l.issues.GetIssues(ctx, slices.Compact(slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare)))
	if err != nil {
		return err
	}
	shown := make(map[uuid.UUID]bool, len(parents))
	for _, p := range parents {
		shown[p.ID] = !p.HiddenFrom(actor, true)
	}
	for _, n := range guarded {
		if !shown[*items[n].ParentID] {
			items[n].ParentID = nil
		}
	}
	return nil
}

// seenOne is the work item i as the caller actor sees it (seen), by his
// grant on its project.
func (l Locks) seenOne(ctx context.Context, actor uuid.UUID, grant shared.Grant, i domain.Issue) (domain.Issue, error) {
	out, err := l.seen(ctx, actor, grants{i.ProjectID: grant}, i)
	if err != nil {
		return domain.Issue{}, err
	}
	return out[0], nil
}
