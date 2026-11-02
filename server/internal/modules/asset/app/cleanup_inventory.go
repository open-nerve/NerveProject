package app

import (
	"context"
	"errors"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (u *Cleanup) cleanInventoryPage(ctx context.Context, result *SweepResult) error {
	cursor := &result.Cursor
	operation, cancel := context.WithTimeout(ctx, u.d.OperationTimeout)
	page, err := u.d.Inventory.List(operation, cursor.Token, u.d.Batch)
	cancel()
	if err != nil {
		return err
	}
	if len(page.Objects) > u.d.Batch || !validSweepToken(page.Next) || (page.Next != "" && (page.Next == cursor.Token || len(page.Objects) == 0)) {
		return errors.New("invalid asset inventory page")
	}
	var last domain.ObjectKey
	ids := make([]uuid.UUID, 0, len(page.Objects))
	for _, object := range page.Objects {
		if object.Key.Kind() == 0 || object.Key <= last || object.ModifiedAt.IsZero() {
			return errors.New("invalid asset inventory object")
		}
		last = object.Key
		if object.Key > cursor.ObjectAfter && object.ModifiedAt.Before(cursor.RuntimeCutoff) {
			ids = append(ids, object.Key.AssetID())
		}
	}
	slices.SortFunc(ids, func(a, b uuid.UUID) int { return a.Compare(b) })
	references, err := u.inventoryReferences(ctx, slices.Compact(ids))
	if err != nil {
		return err
	}
	for _, object := range page.Objects {
		if object.Key <= cursor.ObjectAfter {
			continue
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if object.ModifiedAt.Before(cursor.RuntimeCutoff) && !references[object.Key] {
			operation, cancel := context.WithTimeout(ctx, u.d.OperationTimeout)
			err := u.d.Objects.Delete(operation, object.Key)
			cancel()
			// Budget expiration replays this item. A storage failure records an
			// attempted key so retries can reach its neighbours; next cycle retries it.
			if ctx.Err() != nil {
				return ctx.Err()
			}
			cursor.ObjectAfter = object.Key
			result.Scanned++
			if err != nil {
				return err
			}
			result.Removed++
		} else {
			cursor.ObjectAfter = object.Key
			result.Scanned++
		}
	}
	cursor.Token, cursor.ObjectAfter = page.Next, ""
	if page.Next == "" {
		cursor.Stage = "tmp"
	}
	return nil
}

func (u *Cleanup) inventoryReferences(ctx context.Context, ids []uuid.UUID) (map[domain.ObjectKey]bool, error) {
	references := make(map[domain.ObjectKey]bool)
	if len(ids) == 0 {
		return references, nil
	}
	// This transaction reads only asset-owned rows. It never goes back to an
	// owner lock. Every phase, including reclaiming, protects its exact key.
	err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		rows, err := u.d.Locks.LockAssets(ctx, ids)
		if err != nil {
			return err
		}
		for _, a := range rows {
			for _, record := range []*domain.ObjectRecord{a.Metadata.Staging, a.Metadata.Final} {
				if record != nil {
					references[record.Key] = true
				}
			}
			for _, candidate := range a.Metadata.Candidates {
				references[candidate.Key] = true
			}
		}
		return nil
	})
	return references, err
}

func (u *Cleanup) cleanTemporaryPage(ctx context.Context, result *SweepResult) error {
	cursor := &result.Cursor
	operation, cancel := context.WithTimeout(ctx, u.d.OperationTimeout)
	next, removed, err := u.d.Temporary.SweepTemporary(operation, cursor.RuntimeCutoff, u.d.Batch, cursor.Token)
	cancel()
	if !validSweepToken(next) || removed < 0 || removed > u.d.Batch || (err == nil && next != "" && next == cursor.Token) {
		return errors.New("invalid asset temporary progress")
	}
	cursor.Token = next
	result.Removed += removed
	if err != nil {
		return err
	}
	if next == "" {
		cursor.Stage = "done"
		result.Complete = true
	}
	return nil
}
