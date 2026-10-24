package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ActivityPage contains visible activity rows in ascending position order.
type ActivityPage struct {
	Data       []ActivityRecord
	NextCursor string
}

// ListIssueActivitiesDeps supplies authorization, historical references and a snapshot.
type ListIssueActivitiesDeps struct {
	Locks      Locks
	Activities ActivityLister
	Projects   VisibleProjects
	ReadTx     shared.TxManager
}

// ListIssueActivities reads one page of visible activity.
type ListIssueActivities struct{ d ListIssueActivitiesDeps }

func NewListIssueActivities(d ListIssueActivitiesDeps) *ListIssueActivities {
	return &ListIssueActivities{d: d}
}
func (u *ListIssueActivities) Execute(ctx context.Context, issueID uuid.UUID, limit *int, cursor *string) (ActivityPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return ActivityPage{}, err
	}
	var after *domain.ActivityCursor
	if cursor != nil {
		after = &domain.ActivityCursor{}
		if err := shared.DecodeCursor(*cursor, after); err != nil {
			return ActivityPage{}, err
		}
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return ActivityPage{}, err
	}
	page := ActivityPage{Data: []ActivityRecord{}}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		issue, _, err := u.d.Locks.see(ctx, actor, issueID, domain.ActionRead)
		if err != nil {
			return err
		}
		projects, err := u.d.Projects.Historical(ctx, issue.WorkspaceID, actor.UserID)
		if err != nil {
			return err
		}
		rows, err := u.d.Activities.Activities(ctx, issueID, actor.UserID, projects, after, size+1)
		if err != nil {
			return err
		}
		if len(rows) > size {
			rows = rows[:size]
			last := rows[len(rows)-1]
			page.NextCursor, err = shared.EncodeCursor(domain.ActivityCursor{CreatedAt: last.CreatedAt, ID: last.ID})
			if err != nil {
				return err
			}
		}
		page.Data = append(page.Data, rows...)
		return nil
	})
	if err != nil {
		return ActivityPage{}, err
	}
	return page, nil
}
