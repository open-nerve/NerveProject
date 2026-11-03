package app

import (
	"context"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (u *ImageBinder) lock(ctx context.Context, request ImageBindingRequest) (map[uuid.UUID]domain.Asset, error) {
	if !validImageBindingRequest(request) {
		return nil, domain.ErrInvalidAsset
	}
	ids := make([]uuid.UUID, 0, len(request.Changes)*2)
	for _, change := range request.Changes {
		for _, id := range []*uuid.UUID{change.CurrentID, change.NextID} {
			if id != nil {
				ids = append(ids, *id)
			}
		}
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	rows, err := u.d.Locks.LockAssets(ctx, slices.Compact(ids))
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Asset, len(rows))
	for _, row := range rows {
		byID[row.ID] = row
	}
	return byID, nil
}

// Visibility is checked for the whole batch before revealing purpose or state.
// Current references come from the locked owner, never from a client old-ID.
func checkImageBindings(request ImageBindingRequest, rows map[uuid.UUID]domain.Asset, at time.Time) error {
	for _, change := range request.Changes {
		if change.NextID == nil || sameImageID(change.CurrentID, change.NextID) {
			continue
		}
		row, ok := rows[*change.NextID]
		if !ok || row.DeletedAt != nil || row.Placement != imagePlacement(request.Host, request.Host.ApplicantID) ||
			row.CreatedAt.Before(at.Add(-7*24*time.Hour)) {
			return domain.ErrAssetNotFound
		}
	}
	for _, change := range request.Changes {
		if change.CurrentID != nil {
			row, ok := rows[*change.CurrentID]
			if !ok {
				return domain.ErrAssetNotFound
			}
			applicant := row.Placement.UserID
			if request.Host.WorkspaceID == uuid.Nil() {
				applicant = request.Host.ApplicantID
			}
			if row.Purpose != change.Purpose || row.Placement != imagePlacement(request.Host, applicant) {
				return domain.ErrAssetScopeChanged
			}
		}
	}
	var fields []shared.FieldError
	for _, change := range request.Changes {
		if change.NextID != nil && !sameImageID(change.CurrentID, change.NextID) && rows[*change.NextID].Purpose != change.Purpose {
			fields = append(fields, shared.FieldError{Field: change.Field, Code: shared.FieldNotAllowed, Message: "file purpose does not match this image field"})
		}
	}
	if len(fields) > 0 {
		return shared.Invalid(fields...)
	}
	for _, change := range request.Changes {
		if change.NextID != nil && !sameImageID(change.CurrentID, change.NextID) {
			row := rows[*change.NextID]
			if !row.Uploaded || row.Metadata.Final == nil || row.Metadata.Final.Phase != domain.ObjectActive {
				return domain.ErrAssetNotUploaded
			}
		}
	}
	for _, change := range request.Changes {
		if change.NextID != nil && !sameImageID(change.CurrentID, change.NextID) && rows[*change.NextID].BoundAt != nil {
			return domain.ErrAssetBound
		}
	}
	return nil
}

func imagePlacement(host ImageBindingHost, applicant uuid.UUID) domain.Placement {
	p := domain.Placement{UserID: applicant, WorkspaceID: host.WorkspaceID, ProjectID: host.ProjectID}
	switch {
	case host.NewProject:
		p.Scope = domain.Scope{Kind: domain.ScopeNewProject, ID: host.WorkspaceID}
		p.ProjectID = uuid.Nil()
	case host.ProjectID != uuid.Nil():
		p.Scope = domain.Scope{Kind: domain.ScopeProject, ID: host.ProjectID}
	case host.WorkspaceID != uuid.Nil():
		p.Scope = domain.Scope{Kind: domain.ScopeWorkspace, ID: host.WorkspaceID}
	default:
		p.Scope = domain.Scope{Kind: domain.ScopeUser}
	}
	return p
}

func sameImageID(a, b *uuid.UUID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func validImageBindingRequest(r ImageBindingRequest) bool {
	if r.Host.ApplicantID == uuid.Nil() || len(r.Changes) > 2 ||
		(r.Host.ProjectID != uuid.Nil() && r.Host.WorkspaceID == uuid.Nil()) ||
		(r.Host.NewProject && r.Host.ProjectID == uuid.Nil()) {
		return false
	}
	seen := make(map[string]bool, len(r.Changes))
	for _, change := range r.Changes {
		if seen[change.Field] || (r.Host.NewProject && change.CurrentID != nil) {
			return false
		}
		seen[change.Field] = true
		switch {
		case r.Host.ProjectID != uuid.Nil():
			if change.Field != "cover_image_asset_id" || change.Purpose != domain.ProjectCover {
				return false
			}
		case r.Host.WorkspaceID != uuid.Nil():
			if change.Field != "logo_asset_id" || change.Purpose != domain.WorkspaceLogo {
				return false
			}
		default:
			if (change.Field != "avatar_asset_id" || change.Purpose != domain.UserAvatar) &&
				(change.Field != "cover_image_asset_id" || change.Purpose != domain.UserCover) {
				return false
			}
		}
	}
	return true
}
