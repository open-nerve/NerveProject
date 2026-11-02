package bootstrap

import (
	"context"
	"errors"
	"time"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type issueDraftAssets struct{ assets asset.DraftAssets }

func (d issueDraftAssets) DeleteDraftAssets(ctx context.Context, row issueapp.DraftDeletion) error {
	return d.assets.DeleteDraftAssets(ctx, row.ID, row.At, row.By)
}

type assetInserter struct{ client *jobs.Inserter }

func (i assetInserter) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	tx, ok := postgres.Tx(ctx)
	if !ok {
		return errors.New("asset job insert requires a transaction")
	}
	return i.client.InsertTx(ctx, tx, args, opts)
}
func protectedAssetReferences() []postgres.ProtectedReference {
	var refs []postgres.ProtectedReference
	for _, r := range asset.ProtectedParentReferences() {
		refs = append(refs, postgres.ProtectedReference{Schema: r.Schema, Table: r.Table, Constraint: r.Constraint})
	}
	return refs
}

type assetPurgeRows interface {
	Page(context.Context, time.Time, int, asset.PurgeCursor) (asset.PurgeProgress, error)
}
type assetPurge struct{ rows assetPurgeRows }

func forAssetPurge(rows assetPurgeRows) Purger { return assetPurge{rows: rows} }
func (p assetPurge) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	r, err := p.rows.Page(ctx, cutoff, batch, asset.PurgeCursor{Table: cursor.Table, After: cursor.After})
	return purgePageProgress{Cursor: purgePageCursor{Table: r.Cursor.Table, After: r.Cursor.After}, Scanned: r.Scanned, Deleted: r.Deleted, Blocked: r.Blocked, Done: r.Done}, err
}
