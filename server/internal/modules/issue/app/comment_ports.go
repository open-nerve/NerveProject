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

// CommentUpdater reads and updates a comment and its new subscriptions.
type CommentUpdater interface {
	GetComment(context.Context, uuid.UUID) (domain.Comment, bool, error)
	UpdateComment(context.Context, CommentWrite) error
	InsertSubscribers(context.Context, LinkRows) error
	MarkChanged(context.Context, ChangedRows) error
}

// CommentDeleter reads and soft-deletes a comment.
type CommentDeleter interface {
	GetComment(context.Context, uuid.UUID) (domain.Comment, bool, error)
	DeleteComment(context.Context, CommentDeletion) error
	MarkChanged(context.Context, ChangedRows) error
}
