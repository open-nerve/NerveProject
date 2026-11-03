package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func checkContentRows(q ContentBindRequest, rows []domain.Asset) error {
	images, attachments := assetIDSet(q.ImageIDs), assetIDSet(q.AttachmentIDs)
	description, attachment := domain.IssueDescription, domain.IssueAttachment
	if q.Host.CommentID != uuid.Nil() {
		description, attachment = domain.CommentDescription, domain.CommentAttachment
	}
	// Check the entire visibility boundary before disclosing publication state.
	for _, row := range rows {
		applicant := q.ActorID
		if row.BoundAt != nil && !q.NewComment {
			applicant = row.Placement.UserID
		}
		if row.DeletedAt != nil || row.Placement != contentPlacement(q, applicant) ||
			(images[row.ID] && row.Purpose != description) || (attachments[row.ID] && row.Purpose != attachment) {
			return domain.ErrAssetNotFound
		}
	}
	for _, row := range rows {
		if !isPublished(row) {
			return domain.ErrAssetNotUploaded
		}
		if q.NewComment && row.BoundAt != nil {
			return domain.ErrAssetBound
		}
	}
	return nil
}

func contentPlacement(q ContentBindRequest, applicant uuid.UUID) domain.Placement {
	p := domain.Placement{UserID: applicant, WorkspaceID: q.Host.WorkspaceID,
		ProjectID: q.Host.ProjectID, IssueID: q.Host.IssueID, CommentID: q.Host.CommentID}
	switch {
	case q.NewComment:
		p.Scope = domain.Scope{Kind: domain.ScopeNewComment, ID: q.Host.IssueID}
		p.CommentID = uuid.Nil()
	case q.Host.CommentID != uuid.Nil():
		p.Scope = domain.Scope{Kind: domain.ScopeComment, ID: q.Host.CommentID}
	default:
		p.Scope = domain.Scope{Kind: domain.ScopeIssue, ID: q.Host.IssueID}
	}
	return p
}

func assetIDSet(ids []uuid.UUID) map[uuid.UUID]bool {
	set := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		set[id] = true
	}
	return set
}
