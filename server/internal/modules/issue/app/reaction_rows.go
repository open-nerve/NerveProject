package app

import (
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ReactionKey identifies an actor's reaction on a resource.
type ReactionKey struct {
	ResourceID, ActorID uuid.UUID
	Reaction            string
}

// IssueReactionWrite carries a reaction and its audit stamp.
type IssueReactionWrite struct {
	Reaction domain.IssueReaction
	By       uuid.UUID
	At       time.Time
}

// ReactionDeletion identifies a reaction and its deletion stamp.
type ReactionDeletion struct {
	Key ReactionKey
	By  uuid.UUID
	At  time.Time
}
