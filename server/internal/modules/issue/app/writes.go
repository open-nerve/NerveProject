package app

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The pieces every write on a work item's rows shares (M4 design 3.5):
// where its rows go, the accounts its activity names, and its answer.

// stamp is where a write's rows under a work item go, and whose and when
// they are: the work item's workspace, project and id; the caller, who
// writes them, at the write's one moment.
type stamp struct {
	workspace, project, issue uuid.UUID
	by                        uuid.UUID
	at                        time.Time
}

// links are new rows under the work item of each of targets, in their
// order, each with a fresh id.
func (s stamp) links(targets []uuid.UUID) LinkRows {
	r := LinkRows{WorkspaceID: s.workspace, ProjectID: s.project, IssueID: s.issue, Targets: targets, By: s.by, At: s.at}
	for range targets {
		r.IDs = append(r.IDs, uuid.NewV7())
	}
	return r
}

// unlinks are the work item's rows of each of targets that a write
// removes.
func (s stamp) unlinks(targets []uuid.UUID) UnlinkRows {
	return UnlinkRows{IssueID: s.issue, Targets: targets, By: s.by, At: s.at}
}

// activity are the rows of the work item's activity, in their order, each
// with a fresh id, so that their ids ascend in it.
func (s stamp) activity(activities []domain.Activity) []ActivityRow {
	rows := make([]ActivityRow, len(activities))
	for i, a := range activities {
		rows[i] = ActivityRow{ID: uuid.NewV7(), WorkspaceID: s.workspace, ProjectID: s.project, IssueID: s.issue, Activity: a, ActorID: s.by,
			At: s.at}
	}
	return rows
}

// assignees is the accounts ids with their display names, in their order,
// read through profiles; none for none. An account without one is an
// error: an assignee is, or was, a member of the project, and accounts are
// never deleted.
func assignees(ctx context.Context, profiles PublicProfiles, ids []uuid.UUID) ([]domain.Assignee, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	names, err := profiles.DisplayNames(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]domain.Assignee, len(ids))
	for i, id := range ids {
		name, ok := names[id]
		if !ok {
			return nil, fmt.Errorf("assignee %s has no profile", id)
		}
		out[i] = domain.Assignee{ID: id, DisplayName: name}
	}
	return out, nil
}

// readBack reads the work item id back as stored, in the transaction ctx
// carries: a write's answer. A work item not there after its write is an
// internal error, not a 404.
func readBack(ctx context.Context, issues IssueReader, id uuid.UUID) (domain.Issue, error) {
	i, found, err := issues.GetIssue(ctx, id)
	switch {
	case err != nil:
		return domain.Issue{}, err
	case !found:
		return domain.Issue{}, errors.New("work item " + id.String() + " is not there after its write")
	}
	return i, nil
}
