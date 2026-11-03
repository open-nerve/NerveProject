package app

import (
	"context"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type ImageReadDeps struct {
	Locks      AssetLocker
	Signer     ReadSigner
	RuntimeNow func() time.Time
}
type BoundImageReader struct{ d ImageReadDeps }

func NewImageReader(d ImageReadDeps) *BoundImageReader { return &BoundImageReader{d: d} }
func (u *BoundImageReader) ReadBoundImages(ctx context.Context, references []ImageReference) (map[uuid.UUID]ImageURL, error) {
	refs := make(map[uuid.UUID]ImageReference, len(references))
	for _, ref := range references {
		if ref.AssetID == uuid.Nil() {
			continue
		}
		if previous, ok := refs[ref.AssetID]; ok && previous != ref {
			return nil, domain.ErrInvalidAsset
		}
		refs[ref.AssetID] = ref
	}
	ids := make([]uuid.UUID, 0, len(refs))
	for id := range refs {
		ids = append(ids, id)
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	var eligible []domain.Asset
	for start := 0; start < len(ids); start += 1000 {
		rows, err := u.d.Locks.LockAssets(ctx, ids[start:min(start+1000, len(ids))])
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			if readableImage(row, refs[row.ID]) {
				eligible = append(eligible, row)
			}
		}
	}
	result := make(map[uuid.UUID]ImageURL, len(eligible))
	if len(eligible) == 0 {
		return result, nil
	}
	// Every SQL block has completed before any capability can be issued.
	expires := contentExpiry(u.d.RuntimeNow())
	for _, row := range eligible {
		signed, err := u.d.Signer.SignRead(ctx, domain.ReadGrant{
			Key: row.Metadata.Final.Key, Method: "GET", Name: row.Name,
			ContentType: row.ContentType, Inline: true, ExpiresAt: expires,
		})
		if err != nil {
			return nil, err
		}
		result[row.ID] = ImageURL{URL: signed.URL, ExpiresAt: signed.ExpiresAt}
	}
	return result, nil
}

func readableImage(row domain.Asset, ref ImageReference) bool {
	if row.ID != ref.AssetID || row.Purpose != ref.Purpose || row.BoundAt == nil ||
		row.DeletedAt != nil || !row.Uploaded || row.Metadata.Final == nil || row.Metadata.Final.Phase != domain.ObjectActive {
		return false
	}
	expected := domain.Placement{UserID: row.Placement.UserID, WorkspaceID: ref.WorkspaceID, ProjectID: ref.ProjectID}
	switch ref.Purpose {
	case domain.UserAvatar, domain.UserCover:
		expected.UserID = ref.UserID
		expected.Scope = domain.Scope{Kind: domain.ScopeUser}
	case domain.WorkspaceLogo:
		if ref.UserID != uuid.Nil() {
			return false
		}
		expected.Scope = domain.Scope{Kind: domain.ScopeWorkspace, ID: ref.WorkspaceID}
	case domain.ProjectCover:
		if ref.UserID != uuid.Nil() {
			return false
		}
		expected.Scope = domain.Scope{Kind: domain.ScopeProject, ID: ref.ProjectID}
	default:
		return false
	}
	return row.Placement == expected
}
