package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type GetDeps struct {
	Rows   AssetReader
	Hosts  HostAccess
	ReadTx shared.TxManager
}
type Get struct{ d GetDeps }

func NewGet(d GetDeps) *Get { return &Get{d: d} }

func (u *Get) Execute(ctx context.Context, id uuid.UUID) (domain.Asset, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Asset{}, err
	}
	var row domain.Asset
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		var err error
		row, err = ownedAsset(ctx, u.d.Rows, actor, id)
		if err != nil {
			return err
		}
		host, err := u.d.Hosts.Inspect(ctx, actor, HostRequest{Scope: row.Placement.Scope, Intent: ReadAsset})
		if err != nil {
			return err
		}
		if hostOf(row.Placement) != host {
			return domain.ErrAssetScopeChanged
		}
		return nil
	})
	if err != nil {
		return domain.Asset{}, err
	}
	return row, nil
}

type ResolveContentDeps struct {
	Rows       AssetReader
	Access     *Access
	Signer     ReadSigner
	RuntimeNow func() time.Time
}
type ResolveContent struct{ d ResolveContentDeps }

func NewResolveContent(d ResolveContentDeps) *ResolveContent { return &ResolveContent{d: d} }

func (u *ResolveContent) Execute(ctx context.Context, id uuid.UUID) (domain.SignedRequest, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.SignedRequest{}, err
	}
	before, err := ownedAsset(ctx, u.d.Rows, actor, id)
	if err != nil {
		return domain.SignedRequest{}, err
	}
	var signed domain.SignedRequest
	err = u.d.Access.Within(ctx, actor, []HostRequest{{Scope: before.Placement.Scope, Intent: ReadAsset}}, []uuid.UUID{id}, func(ctx context.Context, held AccessHeld) error {
		row := held.Assets[0]
		if err := currentAsset(before, row, held.Hosts[0]); err != nil {
			return err
		}
		if !row.Uploaded || row.Metadata.Final == nil || row.Metadata.Final.Phase != domain.ObjectActive {
			return domain.ErrAssetNotUploaded
		}
		var err error
		signed, err = u.d.Signer.SignRead(ctx, domain.ReadGrant{Key: row.Metadata.Final.Key, Method: "GET", Name: row.Name, ContentType: row.ContentType, Inline: row.Purpose.ImageRequired(), ExpiresAt: contentExpiry(u.d.RuntimeNow())})
		return err
	})
	if err != nil {
		return domain.SignedRequest{}, err
	}
	return signed, nil
}

func contentExpiry(now time.Time) time.Time {
	expires := now.UTC().Truncate(time.Hour).Add(time.Hour)
	if expires.Sub(now) < 5*time.Minute {
		expires = expires.Add(time.Hour)
	}
	return expires
}

func ownedAsset(ctx context.Context, rows AssetReader, actor shared.Actor, id uuid.UUID) (domain.Asset, error) {
	row, err := rows.FindAsset(ctx, id)
	if err != nil {
		return domain.Asset{}, err
	}
	if row.Placement.UserID != actor.UserID || row.DeletedAt != nil {
		return domain.Asset{}, domain.ErrAssetNotFound
	}
	return row, nil
}

func currentAsset(before, row domain.Asset, host Host) error {
	boundSame := before.BoundAt == nil && row.BoundAt == nil || before.BoundAt != nil && row.BoundAt != nil && before.BoundAt.Equal(*row.BoundAt)
	if row.Placement != before.Placement || hostOf(row.Placement) != host || row.Purpose != before.Purpose || !boundSame || row.StorageRevision != before.StorageRevision {
		return domain.ErrAssetScopeChanged
	}
	return nil
}
