package workspace

import (
	"context"
	"uuid"
)

type ImageReferenceDeclaration struct{ Schema, Table, Constraint, Column, Parent string }
type imageReferenceRows interface {
	DetachAssetReferences(context.Context, []uuid.UUID) error
}
type ImageReferences struct{ rows imageReferenceRows }

func (r ImageReferences) DetachAssetReferences(ctx context.Context, ids []uuid.UUID) error {
	return r.rows.DetachAssetReferences(ctx, ids)
}
func (ImageReferences) References() []ImageReferenceDeclaration {
	return []ImageReferenceDeclaration{
		{"public", "workspaces", "workspaces_logo_asset_fk", "logo_asset_id", "file_assets"},
	}
}
