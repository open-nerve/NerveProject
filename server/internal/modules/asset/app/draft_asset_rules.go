package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func validDraftAssetRequest(q DraftAssetRequest) bool {
	if q.ActorID == uuid.Nil() || q.WorkspaceID == uuid.Nil() || q.DraftID == uuid.Nil() ||
		(q.OldProjectID != nil && *q.OldProjectID == uuid.Nil()) || (q.NewProjectID != nil && *q.NewProjectID == uuid.Nil()) {
		return false
	}
	return !q.Publish || (q.OldProjectID != nil && q.NewProjectID != nil && *q.OldProjectID == *q.NewProjectID)
}

func checkDraftAssetRows(q DraftAssetRequest, rows []domain.Asset) error {
	images := assetIDSet(q.ImageIDs)
	want := domain.Placement{UserID: q.ActorID, WorkspaceID: q.WorkspaceID, ProjectID: contentProjectID(q.OldProjectID),
		DraftID: q.DraftID, Scope: domain.Scope{Kind: domain.ScopeDraft, ID: q.DraftID}}
	for _, row := range rows {
		if row.DeletedAt != nil || row.Placement != want ||
			(row.Purpose != domain.DraftIssueDescription && row.Purpose != domain.DraftIssueAttachment) ||
			(images[row.ID] && row.Purpose != domain.DraftIssueDescription) {
			return domain.ErrAssetNotFound
		}
	}
	for _, row := range rows {
		if (q.Publish || images[row.ID]) && !isPublished(row) {
			return domain.ErrAssetNotUploaded
		}
	}
	return nil
}

func contentProjectID(id *uuid.UUID) uuid.UUID {
	if id != nil {
		return *id
	}
	return uuid.Nil()
}
