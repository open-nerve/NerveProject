package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ActivityRecord is an existing activity, including its nullable actor and timestamps.
type ActivityRecord struct {
	ID, WorkspaceID, ProjectID, IssueID uuid.UUID
	ActorID                             *uuid.UUID
	Activity                            domain.Activity
	CreatedAt, UpdatedAt                time.Time
}

// DescriptionActivityMerge advances an existing description activity without replacing its origin.
type DescriptionActivityMerge struct {
	ID, By  uuid.UUID
	NewHTML string
	At      time.Time
}

// DescriptionActivityStore reads the latest live activity and writes description activity in the caller's transaction.
type DescriptionActivityStore interface {
	InsertActivities(ctx context.Context, rows []ActivityRow) error
	LatestActivity(ctx context.Context, issueID uuid.UUID) (ActivityRecord, bool, error)
	MergeDescriptionActivity(ctx context.Context, m DescriptionActivityMerge) error
}

// ActivityLister filters historical references before selecting a keyset page.
type ActivityLister interface {
	Activities(context.Context, uuid.UUID, uuid.UUID, []VisibleProject, *domain.ActivityCursor, int) ([]ActivityRecord, error)
}
