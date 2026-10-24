package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// MentionStore synchronizes live description mentions in the caller's transaction.
type MentionStore interface {
	MentionIDs(context.Context, uuid.UUID) ([]uuid.UUID, error)
	InsertMentions(context.Context, LinkRows) error
	RemoveMentions(context.Context, UnlinkRows) error
}

// DescriptionVersions stores the latest description snapshot in the same transaction.
type DescriptionVersions interface {
	LatestDescriptionVersion(context.Context, uuid.UUID) (domain.DescriptionVersion, bool, error)
	InsertDescriptionVersion(context.Context, domain.DescriptionVersion) error
	UpdateDescriptionVersion(context.Context, domain.DescriptionVersion) error
}
