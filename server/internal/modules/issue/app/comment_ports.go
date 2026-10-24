package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// CommentCreator writes comments and the automatic subscriptions they cause.
type CommentCreator interface {
	InsertComment(ctx context.Context, w CommentWrite) error
	InsertSubscribers(ctx context.Context, r LinkRows) error
	MarkChanged(ctx context.Context, r ChangedRows) error
}

// CommentLister reads undeleted comments after an exclusive position.
type CommentLister interface {
	Comments(ctx context.Context, issueID uuid.UUID, after *domain.CommentCursor, limit int) ([]domain.Comment, error)
}
