package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type DeleteDeps struct {
	Events shared.EventPublisher
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
	before, err := accessibleAsset(ctx, u.d.Rows, actor, id, RemoveAsset)
	if err != nil {
		return err
	}
	request := HostRequest{Scope: before.Placement.Scope, Intent: RemoveAsset}
	return u.d.Access.Within(ctx, actor, []HostRequest{request}, []AssetAccessRequest{{ID: id, Host: request}}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[id]
		if err := currentPlacement(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if row.DeletedAt != nil {
			return nil
		}
		if row.BoundAt != nil && !row.IsOrdinaryAttachment() {
			return domain.ErrAssetBound
		}
		row.DeletedAt, row.DeletedByID = &held.At, &actor.UserID
		row.UpdatedAt, row.UpdatedByID = held.At, &actor.UserID
		deleted, err := u.d.Writer.SaveAsset(ctx, row)
		if err != nil {
			return err
		}
		return publishAttachmentChange(ctx, u.d.Events, deleted, actor.UserID, held.At, false)
	})
}
