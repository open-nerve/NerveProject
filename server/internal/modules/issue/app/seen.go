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
	if err := hideParents(ctx, l.issues, actor, out, guarded); err != nil {
		return nil, err
	}
	if err := projectAttachments(ctx, l.attachments, out); err != nil {
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
func hideParents(ctx context.Context, issues IssueReader, actor uuid.UUID, items []domain.Issue, guarded []int) error {
	if len(guarded) == 0 {
		return nil
	}
	ids := make([]uuid.UUID, len(guarded))
	for k, n := range guarded {
		ids[k] = *items[n].ParentID
	}
	parents, err := issues.GetIssues(ctx, ascending(ids))
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

// seenOne is the work item i as the caller actor sees it, by his grant on
// its project, in full, at the moment Locks' clock answers (seenAt, M7 P2a
// spec 2.3): a read's answer, and that of a write that wrote nothing and
// so has no moment of its own.
func (l Locks) seenOne(ctx context.Context, actor uuid.UUID, grant shared.Grant, i domain.Issue) (domain.Issue, error) {
	return l.seenAt(ctx, actor, grant, i, l.clock)
}

// seenAt is the work item i as the caller actor sees it (seen), by his
// grant on its project, in full, at the moment clock answers: with its
// intake row as he is answered it then (intakeSeen); and with whether he
// subscribes to it, read last, in the same transaction or read snapshot
// (M7 P1a spec 2.3, R5). A write that wrote at a moment answers at it
// (moment), so that its answer shows what it wrote. Every answer of one
// work item in full but a creation's, which has no intake row, is this
// one.
func (l Locks) seenAt(ctx context.Context, actor uuid.UUID, grant shared.Grant, i domain.Issue, clock Clock) (domain.Issue, error) {
	out, err := l.seen(ctx, actor, grants{i.ProjectID: grant}, i)
	if err != nil {
		return domain.Issue{}, err
	}
	if out[0].Intake, err = l.intakeSeen(ctx, actor, grant, i, clock); err != nil {
		return domain.Issue{}, err
	}
	if out[0].Subscribed, err = l.issues.IsSubscribed(ctx, i.ID, actor); err != nil {
		return domain.Issue{}, err
	}
	return out[0], nil
}

// intakeSeen is the intake row of the work item i as the caller actor, by
// his grant on its project, is answered it at the moment clock answers (M7
// P2a spec 2.8): none when i has none; else as it is then
// (domain.IntakeFacts.AsOf), its effective status, the clock read once and
// for a snooze alone, so that an answer that does not need the moment
// reads none; and the work item it duplicates only when that is an
// undeleted work item of i's project he sees (Issue.HiddenFrom), read by
// its id, for a duplicate alone: one not there, deleted since (design 15
// D2), of another project or hidden from him is none, so that the answer
// tells nothing of it.
func (l Locks) intakeSeen(ctx context.Context, actor uuid.UUID, grant shared.Grant, i domain.Issue, clock Clock) (*domain.IntakeFacts, error) {
	if i.Intake == nil {
		return nil, nil
	}
	f := i.Intake.AsOf(clock.Now)
	if f.DuplicateToID == nil {
		return &f, nil
	}
	target, found, err := l.issues.GetIssue(ctx, *f.DuplicateToID)
	if err != nil {
		return nil, err
	}
	if !found || target.ProjectID != i.ProjectID || target.HiddenFrom(actor, grant.OwnOnly) {
		f.DuplicateToID = nil
	}
	return &f, nil
}
