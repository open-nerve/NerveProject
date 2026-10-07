package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The project store's labels, as the label operations read and write them
// (M3 design 3.16).

// LabelLister is listLabels' repository.
type LabelLister interface {
	ProjectFinder
	// ListLabels lists projectID's undeleted labels, parents and children
	// alike, by sort order, then id; an archived project's too.
	ListLabels(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error)
}

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

// LabelFinder reads a label by its id: a label a write names, first
// without a lock, for its project and the project's workspace, then again
// under their locks (lockRowAndDecide); and a label a write names as the
// parent, under the project's lock (checkParent).
type LabelFinder interface {
	// LabelByID is the undeleted label id; found is false when there is
	// none.
	LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error)
}

// labelWrite is the write action on the label id, addressed by its
// resource (rowWrite): labels reads it, and project.label_not_found is its
// 404.
func labelWrite(id uuid.UUID, action shared.Action, labels LabelFinder) rowWrite[domain.Label] {
	return rowWrite[domain.Label]{id: id, action: action, find: labels.LabelByID, notFound: domain.ErrLabelNotFound}
}

// LabelCreator is createLabel's repository. GreatestSortOrder and
// CreateLabel run in the transaction ctx carries, under the project's FOR
// NO KEY UPDATE, which every write of its labels takes, so that its labels
// stay as read (M3 design 3.16); createLabel reads by LabelByID only the
// parent given, under that lock too (checkParent).
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

// LabelUpdater is updateLabel's repository. HasChildren and UpdateLabel run
// in the transaction ctx carries, under the project's FOR NO KEY UPDATE;
// LabelByID's reads are as LabelFinder says.
type LabelUpdater interface {
	LabelFinder
	// HasChildren reports whether an undeleted label has the label id as
	// its parent.
	HasChildren(ctx context.Context, id uuid.UUID) (bool, error)
	// UpdateLabel changes the fields p gives of the undeleted label id, by
	// the account by at now, and answers it as stored. A name another
	// undeleted label of the project has, in any case, is
	// domain.ErrLabelNameTaken.
	UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error)
}

// LabelDeleter is deleteLabel's repository. DeleteLabel runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE, which
// covers the labels under the label: they are of its project (M3 design
// 3.16, 3.6 convention 5); LabelByID's reads are as LabelFinder says.
type LabelDeleter interface {
	LabelFinder
	// DeleteLabel deletes the undeleted label id and the undeleted labels
	// under it, by the account by at now, in one statement.
	DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error
}
