package app

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

const assetSetBatch = 1000

// The owner already holds its parent locks. Collect every candidate before this
// call: ordering each page independently would allow a later, lower UUID lock.
func lockAssetSet(ctx context.Context, locks AssetLocker, ids []uuid.UUID) ([]domain.Asset, error) {
	ids = slices.Clone(ids)
	slices.SortFunc(ids, uuid.UUID.Compare)
	ids = slices.Compact(ids)
	if len(ids) == 0 {
		// Even an empty binding must require the owner's existing transaction.
		_, err := locks.LockAssets(ctx, nil)
		return nil, err
	}
	byID := make(map[uuid.UUID]domain.Asset, len(ids))
	for batch := range slices.Chunk(ids, assetSetBatch) {
		rows, err := locks.LockAssets(ctx, batch)
		if err != nil {
			return nil, err
		}
		for _, row := range rows {
			byID[row.ID] = row
		}
	}
	rows := make([]domain.Asset, 0, len(ids))
	for _, id := range ids {
		row, ok := byID[id]
		if !ok || id == uuid.Nil() {
			return nil, domain.ErrAssetNotFound
		}
		rows = append(rows, row)
	}
	return rows, nil
}
