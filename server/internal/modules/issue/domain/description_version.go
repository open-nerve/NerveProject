package domain

import (
	"time"
	"uuid"
)

// DescriptionVersion is a saved description, owned by the account that wrote it.
type DescriptionVersion struct {
	ID, WorkspaceID, ProjectID, IssueID, OwnedByID, CreatedByID, UpdatedByID uuid.UUID
	HTML, Text                                                               string
	LastSavedAt, CreatedAt, UpdatedAt                                        time.Time
}

// CanMergeDescriptionVersion decides whether a save belongs to the latest version.
func CanMergeDescriptionVersion(v DescriptionVersion, actor uuid.UUID, now time.Time) bool {
	elapsed := now.Sub(v.LastSavedAt)
	return actor == v.OwnedByID && elapsed >= 0 && elapsed <= 600*time.Second
}
