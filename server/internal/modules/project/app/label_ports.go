package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// The project store's labels, as the label operations read and write them
// (M3 design 3.16).

// LabelRow is a label to insert: its values as the use case decided them,
// its parent checked and its sort order given.
type LabelRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	ParentID    *uuid.UUID
	Name        string
	Color       string
	SortOrder   float64
	CreatedBy   uuid.UUID
	Now         time.Time
}

// LabelFinder reads a label by its id: a label a write names as the
// parent, under the project's lock (checkParent).
type LabelFinder interface {
	// LabelByID is the undeleted label id; found is false when there is
	// none.
	LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error)
}

// LabelCreator is createLabel's repository. Each method runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE, which
// every write of its labels takes, so that its labels stay as read (M3
// design 3.16).
type LabelCreator interface {
	LabelFinder
	// GreatestSortOrder is the greatest sort order of projectID's undeleted
	// labels, nil when it has none.
	GreatestSortOrder(ctx context.Context, projectID uuid.UUID) (*float64, error)
	// CreateLabel inserts r and answers it as stored. A name another
	// undeleted label of the project has, in any case, is
	// domain.ErrLabelNameTaken.
	CreateLabel(ctx context.Context, r LabelRow) (domain.Label, error)
}
