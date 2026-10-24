package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateIssueCommentDeps are the ports a comment creation consumes.
type CreateIssueCommentDeps struct {
	Locks     Locks
	Comments  CommentCreator
	Members   ProjectMembers
	Sanitizer HTMLSanitizer
	Clock     Clock
}

// CreateIssueComment creates a work item's comment under its project's lock.
type CreateIssueComment struct{ d CreateIssueCommentDeps }

func NewCreateIssueComment(d CreateIssueCommentDeps) *CreateIssueComment {
	return &CreateIssueComment{d: d}
}
func (u *CreateIssueComment) Execute(ctx context.Context, id uuid.UUID, html string) (domain.Comment, error) {
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
	var created domain.Comment
	err = u.d.Locks.run(ctx, actor, write{issue: &id, action: domain.ActionCreateComment, notFound: domain.ErrIssueNotFound}, func(ctx context.Context, h held) error {
		if h.project.Archived {
			return domain.ErrProjectArchived
		}
		if h.issue.ArchivedAt != nil {
			return domain.ErrIssueArchived
		}
		targets, err := visibleMentionSubscribers(ctx, u.d.Members, h.issue, actor.UserID, clean.Identity.MentionIDs)
		if err != nil {
			return err
		}
		at := u.d.Clock.Now()
		created = domain.Comment{ID: uuid.NewV7(), WorkspaceID: h.issue.WorkspaceID, ProjectID: h.issue.ProjectID, IssueID: id, ActorID: actor.UserID, HTML: clean.HTML, Text: clean.Text, CreatedAt: at, UpdatedAt: at, Reactions: []domain.CommentReaction{}}
		if err := u.d.Comments.MarkChanged(ctx, ChangedRows{IDs: []uuid.UUID{id}, By: actor.UserID, At: at}); err != nil {
			return err
		}
		stamp := stamp{workspace: h.issue.WorkspaceID, project: h.issue.ProjectID, issue: id, by: actor.UserID, at: at}
		if err := u.d.Comments.InsertSubscribers(ctx, stamp.links(targets)); err != nil {
			return err
		}
		return u.d.Comments.InsertComment(ctx, CommentWrite{Comment: created, By: actor.UserID, At: at})
	})
	if err != nil {
		return domain.Comment{}, err
	}
	return created, nil
}

// ListIssueCommentsDeps are the read ports and independent snapshot manager.
type ListIssueCommentsDeps struct {
	Locks    Locks
	Comments CommentLister
	ReadTx   shared.TxManager
}

// ListIssueComments lists a work item's comments in a consistent snapshot.
type ListIssueComments struct{ d ListIssueCommentsDeps }

func NewListIssueComments(d ListIssueCommentsDeps) *ListIssueComments {
	return &ListIssueComments{d: d}
}
func (u *ListIssueComments) Execute(ctx context.Context, id uuid.UUID, limit *int, cursor *string) (CommentPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return CommentPage{}, err
	}
	var after *domain.CommentCursor
	if cursor != nil {
		after = &domain.CommentCursor{}
		if err := shared.DecodeCursor(*cursor, after); err != nil {
			return CommentPage{}, err
		}
	}
	size, err := shared.PageSize(limit)
	if err != nil {
		return CommentPage{}, err
	}
	page := CommentPage{Data: []domain.Comment{}}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		if _, _, err := u.d.Locks.see(ctx, actor, id, domain.ActionRead); err != nil {
			return err
		}
		rows, err := u.d.Comments.Comments(ctx, id, after, size+1)
		if err != nil {
			return err
		}
		if len(rows) > size {
			rows = rows[:size]
			last := rows[len(rows)-1]
			page.NextCursor, err = shared.EncodeCursor(domain.CommentCursor{CreatedAt: last.CreatedAt, ID: last.ID})
			if err != nil {
				return err
			}
		}
		page.Data = append(page.Data, rows...)
		return nil
	})
	if err != nil {
		return CommentPage{}, err
	}
	return page, nil
}
