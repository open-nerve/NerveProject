package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type DeleteDeps struct {
	Rows   AssetReader
	Writer MetadataWriter
	Access *Access
}
type Delete struct{ d DeleteDeps }

func NewDelete(d DeleteDeps) *Delete { return &Delete{d: d} }

func (u *Delete) Execute(ctx context.Context, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	before, err := ownedAsset(ctx, u.d.Rows, actor, id)
	if err != nil {
		return err
	}
	return u.d.Access.Within(ctx, actor, []HostRequest{{Scope: before.Placement.Scope, Intent: RemoveAsset}}, []uuid.UUID{id}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[0]
		if err := currentPlacement(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if row.BoundAt != nil {
			return domain.ErrAssetBound
		}
		row.DeletedAt, row.DeletedByID = &held.At, &actor.UserID
		row.UpdatedAt, row.UpdatedByID = held.At, &actor.UserID
		_, err := u.d.Writer.SaveAsset(ctx, row)
		return err
	})
}
