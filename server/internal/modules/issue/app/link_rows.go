package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// LinkWrite is a link and the account and moment writing it.
type LinkWrite struct {
	Link domain.Link
	By   uuid.UUID
	At   time.Time
}

// LinkDeletion names a link and its deletion audit stamp.
type LinkDeletion struct {
	ID, By uuid.UUID
	At     time.Time
}
