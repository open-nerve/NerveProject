package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CreateIssueLinkUseCase creates a link on a visible work item.
type CreateIssueLinkUseCase interface {
	Execute(context.Context, uuid.UUID, domain.NewLink) (domain.Link, error)
}

// ListIssueLinksUseCase reads a visible work item's links.
type ListIssueLinksUseCase interface {
	Execute(context.Context, uuid.UUID) ([]domain.Link, error)
}
