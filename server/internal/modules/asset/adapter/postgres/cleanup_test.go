package postgresadapter

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

type cleanupClock time.Time

func (c cleanupClock) Now() time.Time { return time.Time(c) }

type parentFacts map[app.ParentKey]app.DeletionStamp

func (f parentFacts) Deletions(context.Context, []app.Host) (map[app.ParentKey]app.DeletionStamp, error) {
	return f, nil
}

type deleteObjectFunc func(context.Context, domain.ObjectKey) error

func (f deleteObjectFunc) Delete(ctx context.Context, key domain.ObjectKey) error { return f(ctx, key) }

func (w assetWorld) local(t *testing.T) *local.Store {
	t.Helper()
	s, err := local.New(local.Config{Root: t.TempDir(), SizeLimit: domain.MaxObjectSize}, local.Deps{UploadMAC: storagetest.MAC("upload"), ReadMAC: storagetest.MAC("read"), RuntimeNow: func() time.Time { return w.now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s
}
func (w assetWorld) object(t *testing.T, s *local.Store, a domain.Asset, kind byte) domain.ObjectRecord {
	t.Helper()
	u, _ := domain.NewObjectKey('u', a.ID, uuid.NewV7())
	if err := s.Put(t.Context(), domain.UploadGrant{Key: u, Size: 3, ContentType: a.ContentType, ExpiresAt: w.now.Add(time.Minute)}, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	info, err := s.Inspect(t.Context(), u)
	if err != nil {
		t.Fatal(err)
	}
	if kind == 'o' {
		o, _ := domain.NewObjectKey('o', a.ID, uuid.NewV7())
		info, err = s.Seal(t.Context(), u, o, info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Delete(t.Context(), u); err != nil {
			t.Fatal(err)
		}
	}
	hash := sha256.Sum256([]byte("abc"))
	return domain.ObjectRecord{Key: info.Key, Phase: domain.ObjectActive, SafeAfter: w.now.Add(-time.Hour), Size: 3, ContentType: a.ContentType, ETag: info.ETag, SHA256: hex.EncodeToString(hash[:])}
}

type cleanupObjects interface {
	app.ObjectDeleter
	app.ObjectLister
	app.TemporaryCleaner
}

type faultingCleanupObjects struct {
	*local.Store
	deleter app.ObjectDeleter
}

func (s faultingCleanupObjects) Delete(ctx context.Context, key domain.ObjectKey) error {
	return s.deleter.Delete(ctx, key)
}

func (w assetWorld) cleanup(objects cleanupObjects, facts parentFacts, now time.Time) *app.Cleanup {
	return app.NewCleanup(w.cleanupDeps(objects, facts, now))
}
func (w assetWorld) cleanupDeps(objects cleanupObjects, facts parentFacts, now time.Time) app.CleanupDeps {
	return app.CleanupDeps{Rows: w.store, Locks: w.store, Writer: w.store, Parents: facts, Objects: objects, Inventory: objects, Temporary: objects, Tx: postgres.NewTxManager(w.pool, time.Second), Clock: cleanupClock(now), RuntimeNow: func() time.Time { return w.now }, Batch: 2, Budget: time.Second, OperationTimeout: time.Second, PurgeAfter: 60 * 24 * time.Hour}
}

func TestAssetPurgePreservesObjects(t *testing.T) {
	w := newAssetWorld(t)
	storage := w.local(t)
	a := w.asset(domain.ScopeIssue)
	at := w.now.Add(-61 * 24 * time.Hour)
	a.DeletedAt = &at
	a.DeletedByID = &w.remover
	a.UpdatedAt = at
	a.UpdatedByID = &w.remover
	a.Uploaded = true
	record := w.object(t, storage, a, 'o')
	a.Metadata.Final = &record
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	fault := errors.New("object device unavailable")
	failing := deleteObjectFunc(func(ctx context.Context, key domain.ObjectKey) error {
		if _, inside := postgres.Tx(ctx); inside || w.pool.Stat().AcquiredConns() != 0 {
			t.Fatal("external Delete holds database resources")
		}
		current, err := w.store.FindAsset(ctx, a.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Metadata.Final == nil || current.Metadata.Final.Phase != domain.ObjectReclaiming {
			t.Fatal("Delete preceded committed reclaiming claim")
		}
		return fault
	})
	result, err := w.cleanup(faultingCleanupObjects{storage, failing}, nil, w.now).Execute(t.Context(), app.SweepCursor{})
	if !errors.Is(err, fault) || result.Cursor.After != a.ID || result.Removed != 0 {
		t.Fatalf("failed object progress=%+v error=%v", result, err)
	}
	if _, err := storage.Inspect(t.Context(), record.Key); err != nil {
		t.Fatal("failed delete lost object", err)
	}
	page, err := w.store.PurgeAssets(t.Context(), w.now.Add(-60*24*time.Hour), uuid.Nil(), 100)
	if err != nil || page.Deleted != 0 {
		t.Fatalf("purged recovery record=%+v %v", page, err)
	}
	_, err = w.pool.Exec(t.Context(), "DELETE FROM issues WHERE id=$1", w.issue)
	var pe *pgconn.PgError
	if !errors.As(err, &pe) || pe.Code != "23503" {
		t.Fatalf("parent lost protection: %v", err)
	}
	result, err = w.cleanup(storage, nil, w.now).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete || result.Removed != 1 {
		t.Fatalf("recovery=%+v error=%v", result, err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || got.Metadata.Final != nil || got.DeletedAt == nil || !got.DeletedAt.Equal(at) || got.DeletedByID == nil || *got.DeletedByID != w.remover || !got.UpdatedAt.Equal(at) {
		t.Fatalf("recovered row=%+v error=%v", got, err)
	}
	if _, err := storage.Inspect(t.Context(), record.Key); !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("file still present: %v", err)
	}
	page, err = w.store.PurgeAssets(t.Context(), w.now.Add(-60*24*time.Hour), uuid.Nil(), 100)
	if err != nil || page.Deleted != 1 {
		t.Fatalf("empty tombstone not purged=%+v %v", page, err)
	}
	if _, err := w.pool.Exec(t.Context(), "DELETE FROM issues WHERE id=$1", w.issue); err != nil {
		t.Fatal("parent not released", err)
	}
}

func TestAssetCleanupRetentionAndRuntime(t *testing.T) {
	for _, test := range []struct {
		name         string
		deletedDays  int
		uploaded     bool
		safeFuture   bool
		businessDays int
		wantRemoved  bool
	}{
		{"live final", 0, true, false, 0, false}, {"59 day final", 59, true, false, 0, false}, {"exact 60 day final", 60, true, false, 0, false}, {"old final", 61, true, false, 0, true},
		{"pending staging", 0, false, false, 0, false}, {"expired staging", 0, false, false, 8, true}, {"business fast forward cannot expire upload safety", 0, false, true, 61, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			w := newAssetWorld(t)
			storage := w.local(t)
			a := w.asset(domain.ScopeIssue)
			kind := byte('u')
			if test.uploaded {
				kind = 'o'
				a.Uploaded = true
			}
			record := w.object(t, storage, a, kind)
			if test.safeFuture {
				record.SafeAfter = w.now.Add(time.Hour)
			}
			if test.uploaded {
				a.Metadata.Final = &record
			} else {
				a.Metadata.Origin = domain.OriginUpload
				a.Metadata.Staging = &record
			}
			if test.deletedDays > 0 {
				at := w.now.Add(-time.Duration(test.deletedDays) * 24 * time.Hour)
				a.DeletedAt = &at
				a.DeletedByID = &w.remover
			}
			if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
				t.Fatal(err)
			}
			result, err := w.cleanup(storage, nil, w.now.Add(time.Duration(test.businessDays)*24*time.Hour)).Execute(t.Context(), app.SweepCursor{})
			if err != nil || !result.Complete {
				t.Fatalf("cleanup=%+v %v", result, err)
			}
			_, err = storage.Inspect(t.Context(), record.Key)
			if errors.Is(err, domain.ErrObjectNotFound) != test.wantRemoved {
				t.Fatalf("removed=%t want=%t: %v", errors.Is(err, domain.ErrObjectNotFound), test.wantRemoved, err)
			}
		})
	}
}

func TestAssetParentDeletionUsesEarliestOriginalStamp(t *testing.T) {
	w := newAssetWorld(t)
	a := w.asset(domain.ScopeComment)
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	old := w.now.Add(-40 * 24 * time.Hour)
	facts := parentFacts{{Kind: domain.ScopeWorkspace, ID: w.workspace}: {At: old, By: &w.remover}, {Kind: domain.ScopeProject, ID: w.project}: {At: old.Add(time.Hour), By: &w.user}, {Kind: domain.ScopeIssue, ID: w.issue}: {At: old.Add(2 * time.Hour)}}
	result, err := w.cleanup(w.local(t), facts, w.now).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete {
		t.Fatalf("sweep=%+v %v", result, err)
	}
	got, err := w.store.FindAsset(t.Context(), a.ID)
	if err != nil || got.DeletedAt == nil || !got.DeletedAt.Equal(old) || got.DeletedByID == nil || *got.DeletedByID != w.remover {
		t.Fatalf("parent stamp=%+v %v", got, err)
	}
}
