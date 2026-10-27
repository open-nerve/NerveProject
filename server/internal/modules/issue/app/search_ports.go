package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// SearchRead carries directories already authorized in the caller's snapshot.
type SearchRead struct {
	ActorID        uuid.UUID
	Projects       []VisibleProject
	Query          domain.SearchQuery
	ExactProjectID *uuid.UUID
}

// SearchRows filters and orders the complete candidate set before its limit.
type SearchRows interface {
	Search(context.Context, SearchRead) ([]domain.Issue, error)
}
