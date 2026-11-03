package app

import (
	"context"
	"errors"
	"math"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type PurgeCursor struct {
	Table string
	After uuid.UUID
}
type PurgeProgress struct {
	Cursor                    PurgeCursor
	Scanned, Deleted, Blocked int
	Done                      bool
}
type PurgeDeps struct {
	Candidates AssetPurgeCandidates
	Deleter    AssetPurgeDeleter
	References ImageReferenceDetacher
	Tx         shared.TxManager
}
type Purger struct{ d PurgeDeps }

func NewPurger(d PurgeDeps) *Purger { return &Purger{d: d} }
func (p *Purger) Tables() []string  { return []string{"file_assets"} }
func (p *Purger) Page(ctx context.Context, cutoff time.Time, batch int, cursor PurgeCursor) (PurgeProgress, error) {
	result := PurgeProgress{Cursor: cursor}
	if batch < 1 || batch > math.MaxInt32 {
		return result, errors.New("invalid asset purge batch")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if cursor.Table == "" && cursor.After == uuid.Nil() {
		cursor.Table = "file_assets"
	}
	if cursor.Table != "file_assets" {
		return result, errors.New("unknown asset purge table")
	}
	limit := min(batch, 1000)
	ids, err := p.d.Candidates.PurgeAssetCandidates(ctx, cutoff, cursor.After, limit)
	if err != nil {
		return result, err
	}
	if len(ids) > limit {
		return result, errors.New("invalid asset purge candidates")
	}
	previous := cursor.After
	for _, id := range ids {
		if id.Compare(previous) <= 0 {
			return result, errors.New("unordered asset purge candidates")
		}
		previous = id
	}
	if len(ids) == 0 {
		result.Cursor = cursor
		result.Done = true
		return result, nil
	}
	var deleted []uuid.UUID
	err = p.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if err := p.d.References.DetachAssetReferences(ctx, ids); err != nil {
			return err
		}
		var err error
		deleted, err = p.d.Deleter.DeletePurgeAssets(ctx, cutoff, ids)
		if err != nil {
			return err
		}
		seen := make(map[uuid.UUID]bool, len(deleted))
		for _, id := range deleted {
			if _, ok := slices.BinarySearchFunc(ids, id, uuid.UUID.Compare); !ok || seen[id] {
				return errors.New("invalid asset purge result")
			}
			seen[id] = true
		}
		return nil
	})
	if err != nil {
		return result, err
	}
	result.Cursor = PurgeCursor{Table: cursor.Table, After: ids[len(ids)-1]}
	result.Scanned = len(ids)
	result.Deleted = len(deleted)
	return result, nil
}
