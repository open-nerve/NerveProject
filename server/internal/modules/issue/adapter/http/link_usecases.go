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

// UpdateIssueLinkUseCase edits a link on a visible work item.
type UpdateIssueLinkUseCase interface {
	Execute(context.Context, uuid.UUID, domain.LinkPatch) (domain.Link, error)
}

// DeleteIssueLinkUseCase deletes a link on a visible work item.
type DeleteIssueLinkUseCase interface {
	Execute(context.Context, uuid.UUID) error
}
