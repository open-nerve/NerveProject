package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// IssueReactionWriter reads and changes the caller's live reaction.
type IssueReactionWriter interface {
	IssueReaction(context.Context, ReactionKey) (domain.IssueReaction, bool, error)
	InsertIssueReaction(context.Context, IssueReactionWrite) error
	DeleteIssueReaction(context.Context, ReactionDeletion) error
	MarkChanged(context.Context, ChangedRows) error
}

// IssueReactionLister reads live reactions in creation order.
type IssueReactionLister interface {
	IssueReactions(context.Context, uuid.UUID) ([]domain.IssueReaction, error)
}
