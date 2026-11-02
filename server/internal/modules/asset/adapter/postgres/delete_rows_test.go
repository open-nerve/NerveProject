package postgresadapter

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type failingAssetWriter struct {
	store         *Store
	failAt, calls int
	err           error
}

func (f *failingAssetWriter) SaveAsset(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	f.calls++
	out, err := f.store.SaveAsset(ctx, a)
	if err != nil {
		return out, err
	}
	if f.calls == f.failAt {
		return out, f.err
	}
	return out, nil
}
func TestAssetParentDeletionRollback(t *testing.T) {
	w := newAssetWorld(t)
	var ids []uuid.UUID
	for range 4 {
		a := w.asset(domain.ScopeIssue)
		if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, a.ID)
	}
	neighbour := w.asset(domain.ScopeUser)
	if _, err := w.store.CreateAsset(t.Context(), neighbour); err != nil {
		t.Fatal(err)
	}
	fault := errors.New("failed after asset write")
	writer := &failingAssetWriter{store: w.store, failAt: 3, err: fault}
	deps := app.DeleteScopeDeps{Rows: w.store, Locks: w.store, Writer: writer, Tx: postgres.NewTxManager(w.pool, time.Second), Batch: 2, Budget: time.Second}
	stamp := app.DeletionStamp{At: w.now.Add(time.Hour), By: &w.remover}
	result, err := app.NewDeleteScope(deps).Execute(t.Context(), domain.ScopeProject, w.project, stamp, app.SweepCursor{})
	if !errors.Is(err, fault) || result.Scanned != 2 || result.Cursor.After != ids[1] || result.Complete {
		t.Fatalf("partial progress=%+v %v", result, err)
	}
	for i, id := range ids {
		a, err := w.store.FindAsset(t.Context(), id)
		if err != nil || (a.DeletedAt != nil) != (i < 2) {
			t.Fatalf("row %d escaped page rollback: %+v %v", i, a, err)
		}
	}
	deps.Writer = w.store
	result, err = app.NewDeleteScope(deps).Execute(t.Context(), domain.ScopeProject, w.project, stamp, result.Cursor)
	if err != nil || !result.Complete || result.Scanned != 2 {
		t.Fatalf("resumed=%+v %v", result, err)
	}
	for _, id := range ids {
		a, err := w.store.FindAsset(t.Context(), id)
		if err != nil || a.DeletedAt == nil || !a.DeletedAt.Equal(stamp.At) || a.DeletedByID == nil || *a.DeletedByID != w.remover {
			t.Fatalf("lost original stamp=%+v %v", a, err)
		}
	}
	a, err := w.store.FindAsset(t.Context(), neighbour.ID)
	if err != nil || a.DeletedAt != nil {
		t.Fatalf("deleted neighbour=%+v %v", a, err)
	}
}
