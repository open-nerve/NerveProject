package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// UpdateIssueCommentDeps are the ports a comment edit consumes.
type UpdateIssueCommentDeps struct {
	Locks     Locks
	Comments  CommentUpdater
	Members   ProjectMembers
	Sanitizer HTMLSanitizer
	Clock     Clock
}

// UpdateIssueComment edits only the caller's own visible comment.
type UpdateIssueComment struct{ d UpdateIssueCommentDeps }

func NewUpdateIssueComment(d UpdateIssueCommentDeps) *UpdateIssueComment {
	return &UpdateIssueComment{d: d}
}
func (u *UpdateIssueComment) Execute(ctx context.Context, id uuid.UUID, html string) (domain.Comment, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Comment{}, err
	}
	if err := domain.CheckCommentHTML(html); err != nil {
		return domain.Comment{}, err
	}
	clean := u.d.Sanitizer.Clean(html)
	if clean.EmptyComment() {
		return domain.Comment{}, domain.RequiredComment()
	}
	current, found, err := u.d.Comments.GetComment(ctx, id)
	if err != nil {
		return domain.Comment{}, err
	}
	if !found {
		return domain.Comment{}, domain.ErrCommentNotFound
	}
	w := write{issue: &current.IssueID, action: domain.ActionWriteComment, notFound: domain.ErrCommentNotFound}
	w.check = commentCheck(u.d.Comments, id, actor.UserID, false, &current)
	err = u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		if current.HTML == clean.HTML {
			return embedCommentReactions(ctx, u.d.Comments, []domain.Comment{current}, func(rows []domain.Comment) { current = rows[0] })
		}
		targets, err := visibleMentionSubscribers(ctx, u.d.Members, h.issue, actor.UserID, clean.Identity.MentionIDs)
		if err != nil {
			return err
		}
		at := u.d.Clock.Now()
		current.HTML, current.Text, current.EditedAt, current.UpdatedAt = clean.HTML, clean.Text, &at, at
		if err := u.d.Comments.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{h.issue.ID}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		stamp := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: h.issue.ID, by: actor.UserID, at: at}
		if err := u.d.Comments.InsertSubscribers(ctx, stamp.links(targets)); err != nil {
			return err
		}
		if err := u.d.Comments.UpdateComment(ctx, CommentWrite{Comment: current, By: actor.UserID, At: at}); err != nil {
			return err
		}
		return embedCommentReactions(ctx, u.d.Comments, []domain.Comment{current}, func(rows []domain.Comment) { current = rows[0] })
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return current, nil
}

// DeleteIssueCommentDeps are the ports a comment deletion consumes.
type DeleteIssueCommentDeps struct {
	Locks    Locks
	Comments CommentDeleter
	Clock    Clock
}

// DeleteIssueComment soft-deletes an author's or administrator's comment.
type DeleteIssueComment struct{ d DeleteIssueCommentDeps }

func NewDeleteIssueComment(d DeleteIssueCommentDeps) *DeleteIssueComment {
	return &DeleteIssueComment{d: d}
}
func (u *DeleteIssueComment) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	current, found, err := u.d.Comments.GetComment(ctx, id)
	if err != nil {
		return err
	}
	if !found {
		return domain.ErrCommentNotFound
	}
	w := write{issue: &current.IssueID, action: domain.ActionWriteComment, notFound: domain.ErrCommentNotFound}
	w.check = commentCheck(u.d.Comments, id, actor.UserID, true, &current)
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		at := u.d.Clock.Now()
		if err := u.d.Comments.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{h.issue.ID}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		d := CommentDeletion{ID: id, By: actor.UserID, At: at}
		if err := u.d.Comments.DeleteComment(ctx, d); err != nil {
			return err
		}
		return u.d.Comments.DeleteCommentReactionRows(ctx, d)
	})
}

// commentCheck adds the author constraint to the shared live target check.
func commentCheck(comments CommentReader, id, actorID uuid.UUID, allowAdmin bool, current *domain.Comment) func(context.Context, domain.Issue, shared.Grant) error {
	check := commentTargetCheck(comments, id, current)
	return func(ctx context.Context, i domain.Issue, g shared.Grant) error {
		if err := check(ctx, i, g); err != nil {
			return err
		}
		if current.ActorID != actorID && (!allowAdmin || !g.ProjectAdmin) {
			return domain.ErrNotAuthor
		}
		return nil
	}
}

// commentTargetCheck validates existence and parent membership before and under the lock.
func commentTargetCheck(comments CommentReader, id uuid.UUID, current *domain.Comment) func(context.Context, domain.Issue, shared.Grant) error {
	return func(ctx context.Context, i domain.Issue, _ shared.Grant) error {
		c, found, err := comments.GetComment(ctx, id)
		if err != nil {
			return err
		}
		if !found || c.IssueID != i.ID {
			return domain.ErrCommentNotFound
		}
		*current = c
		return nil
	}
}
