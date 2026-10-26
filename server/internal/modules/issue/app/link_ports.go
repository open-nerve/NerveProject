package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// LinkReader locates a live link before relative permission checks.
type LinkReader interface {
	GetLink(context.Context, uuid.UUID) (domain.Link, bool, error)
}

// LinkLister reads a work item's links in descending creation and id order.
type LinkLister interface {
	Links(context.Context, uuid.UUID) ([]domain.Link, error)
}

// LinkURLChecker compares exact URLs in one issue, excluding the supplied id
// (zero for creation) and already-deleted rows.
type LinkURLChecker interface {
	LinkURLExists(context.Context, uuid.UUID, uuid.UUID, string) (bool, error)
}

// LinkCreator writes a new link and its issue's audit and activity rows.
type LinkCreator interface {
	LinkURLChecker
	InsertLink(context.Context, LinkWrite) error
	MarkChanged(context.Context, ChangedRows) error
	InsertActivities(context.Context, []ActivityRow) error
}

// LinkUpdater returns the database's current link after its write.
type LinkUpdater interface {
	LinkReader
	LinkURLChecker
	UpdateLink(context.Context, LinkWrite) (domain.Link, error)
	MarkChanged(context.Context, ChangedRows) error
	InsertActivities(context.Context, []ActivityRow) error
}

// LinkDeleter reads and soft-deletes a link with its issue's audit and activity.
type LinkDeleter interface {
	LinkReader
	DeleteLink(context.Context, LinkDeletion) error
	MarkChanged(context.Context, ChangedRows) error
	InsertActivities(context.Context, []ActivityRow) error
}
