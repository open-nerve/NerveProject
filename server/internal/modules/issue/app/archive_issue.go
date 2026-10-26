package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ArchiveIssueDeps are what archiveIssue and unarchiveIssue read and write
// through.
type ArchiveIssueDeps struct {
	Events shared.EventPublisher
	Locks  Locks
	Issues IssueArchiver
	States StateCatalog
	Clock  Clock
}

// ArchiveIssue archives a work item: POST
// /api/v0/issues/{issue_id}/archive (M4 design 3.8, 5.1).
type ArchiveIssue struct {
	d ArchiveIssueDeps
}

// NewArchiveIssue returns the use case.
func NewArchiveIssue(d ArchiveIssueDeps) *ArchiveIssue {
	return &ArchiveIssue{d: d}
}

// Execute archives the work item id (archive), or answers it as it is when
// it is archived already; a work item in a state of neither the completed
// nor the cancelled group is refused (409 issue.not_archivable), its state
// read under the project's lock.
func (u *ArchiveIssue) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	return u.d.archive(ctx, id, true)
}

// UnarchiveIssue unarchives a work item: POST
// /api/v0/issues/{issue_id}/unarchive (M4 design 3.8, 5.1).
type UnarchiveIssue struct {
	d ArchiveIssueDeps
}

// NewUnarchiveIssue returns the use case.
func NewUnarchiveIssue(d ArchiveIssueDeps) *UnarchiveIssue {
	return &UnarchiveIssue{d: d}
}

// Execute unarchives the work item id (archive), whatever its state, or
// answers it as it is when it is not archived.
func (u *UnarchiveIssue) Execute(ctx context.Context, id uuid.UUID) (domain.Issue, error) {
	return u.d.archive(ctx, id, false)
}

// archive archives the work item id, or unarchives it when on is false (M4
// design 3.5, 3.8, 3.13): Locks runs the write, issue.archive decided on
// it, the guest rule first, before any lock; then the project's write
// gate, which refuses a write that may wait no longer (503 server_busy);
// then in one transaction, the project FOR NO KEY UPDATE, the work item
// read again and the decision again, and under them, in this order: an
// archived project refuses it (409 project.archived); a work item already
// as on would make it is answered as read, and nothing is written; an
// archiving checks the work item's state (archivable); then the clock; the
// work item's archived_at, the moment or none, and its last writer and
// moment, the caller's; its activity's row, 14 or 16. The answer is the
// work item as stored, read back in the transaction, as the caller sees
// it (Locks.seen), the work item as read when nothing is written.
func (d ArchiveIssueDeps) archive(ctx context.Context, id uuid.UUID, on bool) (domain.Issue, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Issue{}, err
	}
	w := write{issue: &id, action: domain.ActionArchive, notFound: domain.ErrIssueNotFound}
	var answer domain.Issue
	err = d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		switch {
		case h.project.Archived:
			return domain.ErrProjectArchived
		case (h.issue.ArchivedAt != nil) == on:
			answer, err = d.Locks.seenOne(ctx, actor.UserID, h.grant, h.issue)
			return err
		}
		if on {
			if err := d.archivable(ctx, h.issue); err != nil {
				return err
			}
		}
		s := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: d.Clock.Now()}
		a, activity := IssueArchival{ID: id, By: s.by, At: s.at}, domain.UnarchiveActivity()
		if on {
			a.ArchivedAt, activity = &s.at, domain.ArchiveActivity()
		}
		if err := d.Issues.SetArchivedAt(ctx, a); err != nil {
			return err
		}
		if err := d.Issues.InsertActivities(ctx, s.activity([]domain.Activity{activity})); err != nil {
			return err
		}
		if answer, err = readBack(ctx, d.Issues, id); err != nil {
			return err
		}
		if answer, err = d.Locks.seenOne(ctx, actor.UserID, h.grant, answer); err != nil {
			return err
		}
		kind := domain.EventIssueUnarchived
		if on {
			kind = domain.EventIssueArchived
		}
		return d.Events.Publish(ctx, s.event(kind, []shared.Change{{Field: *activity.Field, Old: activity.OldValue, New: activity.NewValue}}))
	})
	if err != nil {
		return domain.Issue{}, err
	}
	return answer, nil
}

// archivable refuses to archive the work item i unless its state, one of
// its project's read under the project's lock (issueState), is of a group a
// work item is archived in (domain.Archivable): 409 issue.not_archivable.
func (d ArchiveIssueDeps) archivable(ctx context.Context, i domain.Issue) error {
	states, err := d.States.States(ctx, i.ProjectID)
	if err != nil {
		return err
	}
	state, err := issueState(states, i)
	switch {
	case err != nil:
		return err
	case !domain.Archivable(state.Group):
		return domain.ErrNotArchivable
	}
	return nil
}
