package app

import (
	"time"
	"uuid"
)

// AssetParent is a lock-free fact, including live and soft-deleted rows.
// DeletedByID is the original deletion writer (the row's updated_by_id).
type AssetParent struct {
	ID          uuid.UUID
	DeletedAt   *time.Time
	DeletedByID *uuid.UUID
}
