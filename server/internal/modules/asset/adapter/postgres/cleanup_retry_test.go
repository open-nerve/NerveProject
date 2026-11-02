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

type recoveryWriter struct {
	app.MetadataWriter
	calls  int
	failAt int
	fault  error
}

func (w *recoveryWriter) SaveAsset(ctx context.Context, a domain.Asset) (domain.Asset, error) {
	w.calls++
	if w.calls == w.failAt {
		return domain.Asset{}, w.fault
	}
	return w.MetadataWriter.SaveAsset(ctx, a)
}

func TestAssetCleanupReplaysDeletedCandidateUntilAcknowledged(t *testing.T) {
	w := newAssetWorld(t)
	objects := w.local(t)
	a := w.asset(domain.ScopeIssue)
	record := w.object(t, objects, a, 'o')
	a.Metadata.Candidates = []domain.Candidate{{ObjectRecord: record, LeaseUntil: w.now.Add(-2 * time.Hour)}}
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	fault := errors.New("acknowledgement transaction failed")
	writer := &recoveryWriter{MetadataWriter: w.store, failAt: 2, fault: fault}
	uc := app.NewCleanup(app.CleanupDeps{Rows: w.store, Locks: w.store, Writer: writer, Parents: parentFacts{}, Objects: objects, Tx: postgres.NewTxManager(w.pool, time.Second), Clock: cleanupClock(w.now), RuntimeNow: func() time.Time { return w.now }, Batch: 2, Budget: time.Second, OperationTimeout: time.Second})
	progress, err := uc.Execute(t.Context(), app.SweepCursor{})
	if !errors.Is(err, fault) || progress.Removed != 0 || progress.Cursor.After != uuid.Nil() {
		t.Fatalf("failed ack advanced: %+v %v", progress, err)
	}
	if _, err := objects.Inspect(t.Context(), record.Key); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("object was not actually deleted: %v", err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || len(got.Metadata.Candidates) != 1 || got.Metadata.Candidates[0].Phase != domain.ObjectReclaiming || got.DeletedAt != nil {
		t.Fatalf("failed ack forgot live candidate: %+v %v", got, err)
	}
	progress, err = w.cleanup(objects, nil, w.now).Execute(t.Context(), progress.Cursor)
	if err != nil || !progress.Complete || progress.Removed != 1 {
		t.Fatalf("replay failed: %+v %v", progress, err)
	}
	got, err = w.store.FindAsset(t.Context(), a.ID)
	if err != nil || len(got.Metadata.Candidates) != 0 || got.DeletedAt != nil || !got.UpdatedAt.Equal(a.UpdatedAt) {
		t.Fatalf("replay changed live audit: %+v %v", got, err)
	}
}

type changingParentFacts struct {
	w assetWorld
	a domain.Asset
}

func (f changingParentFacts) Deletions(ctx context.Context, _ []app.Host) (map[app.ParentKey]app.DeletionStamp, error) {
	// A concurrent writer commits a new revision after the page was read.
	if _, err := f.w.store.SaveAsset(ctx, f.a); err != nil {
		return nil, err
	}
	return parentFacts{{Kind: domain.ScopeIssue, ID: f.w.issue}: {At: f.w.now.Add(-time.Hour), By: &f.w.remover}}, nil
}
func TestAssetCleanupSkipsChangedRevisionAfterParentFacts(t *testing.T) {
	w := newAssetWorld(t)
	objects := w.local(t)
	a := w.asset(domain.ScopeIssue)
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	uc := app.NewCleanup(app.CleanupDeps{Rows: w.store, Locks: w.store, Writer: w.store, Parents: changingParentFacts{w, a}, Objects: objects, Tx: postgres.NewTxManager(w.pool, time.Second), Clock: cleanupClock(w.now), RuntimeNow: func() time.Time { return w.now }, Batch: 2, Budget: time.Second, OperationTimeout: time.Second})
	r, err := uc.Execute(t.Context(), app.SweepCursor{})
	if err != nil || !r.Complete || r.Scanned != 1 {
		t.Fatalf("sweep=%+v %v", r, err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || got.DeletedAt != nil || got.StorageRevision != 1 {
		t.Fatalf("stale parent fact deleted revised asset: %+v %v", got, err)
	}
}
