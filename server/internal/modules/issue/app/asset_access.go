package app

import (
	"context"
	"errors"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AssetIntent uint8

const (
	AssetRead AssetIntent = iota + 1
	AssetPrepare
	AssetRemove
)

type AssetHostKind uint8

const (
	AssetIssue AssetHostKind = iota + 1
	AssetComment
	AssetNewComment
	AssetDraft
)

type AssetRequest struct {
	ID     uuid.UUID
	Kind   AssetHostKind
	Intent AssetIntent
}
type AssetHost struct{ WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID }

var ErrAssetScopeChanged = errors.New("issue asset host scope changed")

// AssetProjectReader reads archive state under a project lock already held by
// the caller. It must not take a second project lock or enter a write gate.
type AssetProjectReader interface {
	AssetProjectArchived(context.Context, uuid.UUID) (bool, bool, error)
}
type AssetAccessDeps struct {
	Locks    Locks
	Drafts   DraftLocks
	Comments CommentReader
	Projects AssetProjectReader
}
type AssetAccess struct{ d AssetAccessDeps }

func NewAssetAccess(d AssetAccessDeps) *AssetAccess { return &AssetAccess{d: d} }
func (a *AssetAccess) Inspect(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, error) {
	h, _, err := a.read(ctx, actor, q)
	return h, err
}
func (a *AssetAccess) Check(ctx context.Context, actor shared.Actor, q AssetRequest, before AssetHost) error {
	h, item, err := a.read(ctx, actor, q)
	if err != nil {
		return err
	}
	if h != before {
		return ErrAssetScopeChanged
	}
	if q.Kind == AssetDraft || q.Intent == AssetRead {
		return nil
	}
	archived, found, err := a.d.Projects.AssetProjectArchived(ctx, h.ProjectID)
	if err != nil {
		return err
	}
	if !found {
		if q.Kind == AssetComment {
			return domain.ErrCommentNotFound
		}
		return domain.ErrIssueNotFound
	}
	if archived {
		return domain.ErrProjectArchived
	}
	if item.ArchivedAt != nil {
		return domain.ErrIssueArchived
	}
	return nil
}

func (a *AssetAccess) read(ctx context.Context, actor shared.Actor, q AssetRequest) (AssetHost, domain.Issue, error) {
	var none domain.Issue
	if q.ID == uuid.Nil() || q.Intent < AssetRead || q.Intent > AssetRemove {
		return AssetHost{}, none, errors.New("invalid issue asset request")
	}
	if q.Kind == AssetDraft {
		d, err := a.d.Drafts.read(ctx, actor, q.ID)
		if err != nil {
			return AssetHost{}, none, err
		}
		h := AssetHost{WorkspaceID: d.WorkspaceID, DraftID: d.ID}
		if d.ProjectID != nil {
			h.ProjectID = *d.ProjectID
		}
		return h, none, nil
	}
	id, action, notFound := q.ID, domain.ActionRead, error(domain.ErrIssueNotFound)
	var current domain.Comment
	var check func(context.Context, domain.Issue, shared.Grant) error
	switch q.Kind {
	case AssetIssue:
		if q.Intent != AssetRead {
			action = domain.ActionUpdate
		}
	case AssetNewComment:
		if q.Intent != AssetRead {
			action = domain.ActionCreateComment
		}
	case AssetComment:
		c, found, err := a.d.Comments.GetComment(ctx, q.ID)
		if err != nil {
			return AssetHost{}, none, err
		}
		if !found {
			return AssetHost{}, none, domain.ErrCommentNotFound
		}
		id, notFound = c.IssueID, domain.ErrCommentNotFound
		check = commentTargetCheck(a.d.Comments, q.ID, &current)
		if q.Intent != AssetRead {
			action = domain.ActionWriteComment
			check = commentCheck(a.d.Comments, q.ID, actor.UserID, q.Intent == AssetRemove, &current)
		}
	default:
		return AssetHost{}, none, errors.New("invalid issue asset host kind")
	}
	i, grant, err := a.d.Locks.seeAs(ctx, actor, id, action, notFound)
	if err != nil {
		return AssetHost{}, none, err
	}
	if check != nil {
		if err := check(ctx, i, grant); err != nil {
			return AssetHost{}, none, err
		}
	}
	h := AssetHost{WorkspaceID: i.WorkspaceID, ProjectID: i.ProjectID, IssueID: i.ID}
	if q.Kind == AssetComment {
		h.CommentID = current.ID
	}
	return h, i, nil
}
