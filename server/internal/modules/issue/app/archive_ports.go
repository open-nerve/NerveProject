package app

import (
	"context"
	"time"
	"uuid"
)

// ArchivePolicies discovers undeleted, unarchived project IDs with policies
// of 1–12 months, in ID order. The policy must be read again under
// ProjectGate's lock before archiving.
type ArchivePolicies interface {
	ArchiveProjects(context.Context) ([]uuid.UUID, error)
}

// AutoArchiveRows selects issues in a locked project whose updated_at is
// at or before Cutoff. StateIDs are its terminal states, read from
// StateCatalog under that lock; an empty set matches nothing. OpenCycles
// and OpenModules are the project's cycles and modules that keep their
// work items (OpenPlanning, read under that lock on At's day): one in any
// of them by an undeleted row of its own is not selected; none, nil or
// empty, keeps none. At stamps archived_at and updated_at, with no account
// as the system writer.
type AutoArchiveRows struct {
	ProjectID   uuid.UUID
	StateIDs    []uuid.UUID
	OpenCycles  []uuid.UUID
	OpenModules []uuid.UUID
	Cutoff      time.Time
	At          time.Time
}

// AutoArchiveStore writes system archival and its activities in the caller's transaction.
type AutoArchiveStore interface {
	AutoArchive(context.Context, AutoArchiveRows) ([]uuid.UUID, error)
	InsertActivities(context.Context, []ActivityRow) error
}
