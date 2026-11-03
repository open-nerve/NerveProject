package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type DeleteScopeDeps struct {
	Rows   ScopeReader
	Locks  AssetLocker
	Writer MetadataWriter
	Tx     shared.TxManager
	Batch  int
	Budget time.Duration
}
type DeleteScope struct{ d DeleteScopeDeps }

func NewDeleteScope(d DeleteScopeDeps) *DeleteScope { return &DeleteScope{d: d} }
func (u *DeleteScope) Execute(ctx context.Context, kind domain.ScopeKind, id uuid.UUID, stamp DeletionStamp, cursor SweepCursor) (SweepResult, error) {
	result := SweepResult{Cursor: cursor}
	if (kind != domain.ScopeProject && kind != domain.ScopeWorkspace) || id == uuid.Nil() || stamp.At.IsZero() || (stamp.By != nil && *stamp.By == uuid.Nil()) || u.d.Batch < 1 || u.d.Batch > 1000 || u.d.Budget <= 0 {
		return result, domain.ErrInvalidAsset
	}
	if cursor.Version == 0 {
		if cursor != (SweepCursor{}) {
			return result, errors.New("uninitialized asset deletion cursor has state")
		}
		cursor = SweepCursor{Version: 1, Stage: "scope", Cutoff: stamp.At.UTC()}
	}
	if !cursor.ValidScope() || !cursor.Cutoff.Equal(stamp.At) {
		return result, errors.New("invalid asset deletion cursor")
	}
	result.Cursor = cursor
	work, cancel := context.WithTimeout(ctx, u.d.Budget)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if work.Err() != nil {
			return result, nil
		}
		rows, err := u.d.Rows.ListScope(work, kind, id, cursor.After, u.d.Batch)
		if err != nil {
			return sweepError(ctx, work, result, err)
		}
		if len(rows) == 0 {
			result.Complete = true
			return result, nil
		}
		ids := make([]uuid.UUID, 0, len(rows))
		last := cursor.After
		for _, a := range rows {
			if a.ID.Compare(last) <= 0 {
				return result, errors.New("asset deletion page is not ascending")
			}
			last = a.ID
			ids = append(ids, a.ID)
		}
		changed := 0
		err = u.d.Tx.WithinTx(work, func(ctx context.Context) error {
			held, err := u.d.Locks.LockAssets(ctx, ids)
			if err != nil {
				return err
			}
			for _, a := range held {
				inScope := (kind == domain.ScopeProject && a.Placement.ProjectID == id) || (kind == domain.ScopeWorkspace && a.Placement.WorkspaceID == id)
				if !inScope || !markDeleted(&a, stamp) {
					continue
				}
				if _, err := u.d.Writer.SaveAsset(ctx, a); err != nil {
					return err
				}
				changed++
			}
			return nil
		})
		if err != nil {
			return sweepError(ctx, work, result, err)
		}
		cursor.After = last
		result.Cursor = cursor
		result.Scanned += len(rows)
		result.Removed += changed
	}
}
