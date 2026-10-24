package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AddCommentReactionDeps are the ports used to add reactions.
type AddCommentReactionDeps struct {
	Locks     Locks
	Comments  CommentReader
	Reactions CommentReactionWriter
	Clock     Clock
}

// AddCommentReaction adds the caller's reaction under the project lock.
type AddCommentReaction struct{ d AddCommentReactionDeps }

func NewAddCommentReaction(d AddCommentReactionDeps) *AddCommentReaction {
	return &AddCommentReaction{d}
}
func (u *AddCommentReaction) Execute(ctx context.Context, id uuid.UUID, reaction string) (domain.CommentReaction, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.CommentReaction{}, err
	}
	if err = domain.CheckReaction(reaction); err != nil {
		return domain.CommentReaction{}, err
	}
	current, found, err := u.d.Comments.GetComment(ctx, id)
	if err != nil {
		return domain.CommentReaction{}, err
	}
	if !found {
		return domain.CommentReaction{}, domain.ErrCommentNotFound
	}
	w := write{issue: &current.IssueID, action: domain.ActionWriteReaction, notFound: domain.ErrCommentNotFound, check: commentTargetCheck(u.d.Comments, id, &current)}
	var created domain.CommentReaction
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		key := ReactionKey{ResourceID: id, ActorID: actor.UserID, Reaction: reaction}
		if _, found, err := u.d.Reactions.CommentReaction(ctx, key); err != nil {
			return err
		} else if found {
			return domain.ErrReactionExists
		}
		at := u.d.Clock.Now()
		created = domain.CommentReaction{ID: uuid.NewV7(), WorkspaceID: h.issue.WorkspaceID, ProjectID: h.issue.ProjectID, CommentID: id, ActorID: actor.UserID, Reaction: reaction, CreatedAt: at}
		return u.d.Reactions.InsertCommentReaction(ctx, CommentReactionWrite{Reaction: created, By: actor.UserID, At: at})
	})
	if err != nil {
		return domain.CommentReaction{}, err
	}
	return created, nil
}

// RemoveCommentReactionDeps are the ports used to remove reactions.
type RemoveCommentReactionDeps = AddCommentReactionDeps

// RemoveCommentReaction soft-deletes only the caller's live reaction.
type RemoveCommentReaction struct{ d RemoveCommentReactionDeps }

func NewRemoveCommentReaction(d RemoveCommentReactionDeps) *RemoveCommentReaction {
	return &RemoveCommentReaction{d}
}
func (u *RemoveCommentReaction) Execute(ctx context.Context, id uuid.UUID, reaction string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	if err = domain.CheckReaction(reaction); err != nil {
		return err
	}
	current, found, err := u.d.Comments.GetComment(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrCommentNotFound
	}
	w := write{issue: &current.IssueID, action: domain.ActionWriteReaction, notFound: domain.ErrCommentNotFound, check: commentTargetCheck(u.d.Comments, id, &current)}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		key := ReactionKey{ResourceID: id, ActorID: actor.UserID, Reaction: reaction}
		if _, found, err := u.d.Reactions.CommentReaction(ctx, key); err != nil {
			return err
		} else if !found {
			return domain.ErrReactionNotFound
		}
		at := u.d.Clock.Now()
		return u.d.Reactions.DeleteCommentReaction(ctx, ReactionDeletion{Key: key, By: actor.UserID, At: at})
	})
}

// CommentReactionWrite carries the immutable value and its audit stamp.
type CommentReactionWrite struct {
	Reaction domain.CommentReaction
	By       uuid.UUID
	At       time.Time
}

// CommentReactionWriter changes only the caller's reaction on a comment.
type CommentReactionWriter interface {
	CommentReaction(context.Context, ReactionKey) (domain.CommentReaction, bool, error)
	InsertCommentReaction(context.Context, CommentReactionWrite) error
	DeleteCommentReaction(context.Context, ReactionDeletion) error
}

// embedCommentReactions uses the caller's transaction for a single page-sized read.
func embedCommentReactions(ctx context.Context, r CommentReactionReader, comments []domain.Comment, accept func([]domain.Comment)) error {
	if len(comments) == 0 {
		accept(comments)
		return nil
	}
	ids := make([]uuid.UUID, len(comments))
	for n, c := range comments {
		ids[n] = c.ID
	}
	rows, err := r.CommentReactions(ctx, ids)
	if err != nil {
		return err
	}
	grouped := make(map[uuid.UUID][]domain.CommentReaction, len(comments))
	for _, r := range rows {
		grouped[r.CommentID] = append(grouped[r.CommentID], r)
	}
	for n := range comments {
		comments[n].Reactions = append([]domain.CommentReaction{}, grouped[comments[n].ID]...)
	}
	accept(comments)
	return nil
}
