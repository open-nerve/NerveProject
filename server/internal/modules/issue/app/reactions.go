package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AddIssueReactionDeps are the ports used to add reactions.
type AddIssueReactionDeps struct {
	Events    shared.EventPublisher
	Locks     Locks
	Reactions IssueReactionWriter
	Clock     Clock
}

// AddIssueReaction adds the caller's reaction under the project lock.
type AddIssueReaction struct{ d AddIssueReactionDeps }

func NewAddIssueReaction(d AddIssueReactionDeps) *AddIssueReaction { return &AddIssueReaction{d} }
func (u *AddIssueReaction) Execute(ctx context.Context, id uuid.UUID, reaction string) (domain.IssueReaction, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.IssueReaction{}, err
	}
	if err = domain.CheckReaction(reaction); err != nil {
		return domain.IssueReaction{}, err
	}
	var created domain.IssueReaction
	err = u.d.Locks.run(ctx, actor, write{issue: &id, action: domain.ActionWriteReaction, notFound: domain.ErrIssueNotFound}, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		key := ReactionKey{ResourceID: id, ActorID: actor.UserID, Reaction: reaction}
		if _, found, err := u.d.Reactions.IssueReaction(ctx, key); err != nil {
			return err
		} else if found {
			return domain.ErrReactionExists
		}
		at := u.d.Clock.Now()
		created = domain.IssueReaction{ID: uuid.NewV7(), WorkspaceID: h.issue.WorkspaceID, ProjectID: h.issue.ProjectID, IssueID: id, ActorID: actor.UserID, Reaction: reaction, CreatedAt: at}
		if err := u.d.Reactions.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{id}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		if err := u.d.Reactions.InsertIssueReaction(ctx, IssueReactionWrite{Reaction: created, By: actor.UserID, At: at}); err != nil {
			return err
		}
		stamp := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: h.issue.ID, by: actor.UserID, at: at}
		return u.d.Events.Publish(ctx, stamp.reactionEvent(domain.EventReactionAdded, "issue_reaction", created.ID, created.Reaction, "issue", created.IssueID))
	})
	if err != nil {
		return domain.IssueReaction{}, err
	}
	return created, nil
}

// RemoveIssueReactionDeps are the ports used to remove reactions.
type RemoveIssueReactionDeps = AddIssueReactionDeps

// RemoveIssueReaction soft-deletes only the caller's live reaction.
type RemoveIssueReaction struct{ d RemoveIssueReactionDeps }

func NewRemoveIssueReaction(d RemoveIssueReactionDeps) *RemoveIssueReaction {
	return &RemoveIssueReaction{d}
}
func (u *RemoveIssueReaction) Execute(ctx context.Context, id uuid.UUID, reaction string) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	if err = domain.CheckReaction(reaction); err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, write{issue: &id, action: domain.ActionWriteReaction, notFound: domain.ErrIssueNotFound}, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		key := ReactionKey{ResourceID: id, ActorID: actor.UserID, Reaction: reaction}
		current, found, err := u.d.Reactions.IssueReaction(ctx, key)
		if err != nil {
			return err
		}
		if !found {
			return domain.ErrReactionNotFound
		}
		at := u.d.Clock.Now()
		if err := u.d.Reactions.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{id}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		if err := u.d.Reactions.DeleteIssueReaction(ctx, ReactionDeletion{Key: key, By: actor.UserID, At: at}); err != nil {
			return err
		}
		stamp := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: h.issue.ID, by: actor.UserID, at: at}
		return u.d.Events.Publish(ctx, stamp.reactionEvent(domain.EventReactionRemoved, "issue_reaction", current.ID, current.Reaction, "issue", current.IssueID))
	})
}

// ListIssueReactionsDeps include an independent read snapshot.
type ListIssueReactionsDeps struct {
	Locks     Locks
	Reactions IssueReactionLister
	ReadTx    shared.TxManager
}

// ListIssueReactions reads a visible issue's reactions in one snapshot.
type ListIssueReactions struct{ d ListIssueReactionsDeps }

func NewListIssueReactions(d ListIssueReactionsDeps) *ListIssueReactions {
	return &ListIssueReactions{d}
}
func (u *ListIssueReactions) Execute(ctx context.Context, id uuid.UUID) ([]domain.IssueReaction, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	out := []domain.IssueReaction{}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		if _, _, err := u.d.Locks.see(ctx, actor, id, domain.ActionRead); err != nil {
			return err
		}
		rows, err := u.d.Reactions.IssueReactions(ctx, id)
		if err != nil {
			return err
		}
		out = append(out, rows...)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}
