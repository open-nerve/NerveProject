package app

import (
	"time"
	"uuid"
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
