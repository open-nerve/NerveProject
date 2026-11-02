package postgresadapter

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func inventoryStore(t *testing.T, now *time.Time) (*local.Store, string) {
	t.Helper()
	root := t.TempDir()
	s, err := local.New(local.Config{Root: root, SizeLimit: domain.MaxObjectSize}, local.Deps{UploadMAC: storagetest.MAC("upload"), ReadMAC: storagetest.MAC("read"), RuntimeNow: func() time.Time { return *now }})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	return s, root
}

func inventoryTemporary(t *testing.T, root string, at time.Time) string {
	t.Helper()
	path := filepath.Join(root, "tmp", uuid.NewV7().String())
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "data"), []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, at, at); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertInventoryObject(t *testing.T, s *local.Store, key domain.ObjectKey, exists bool) {
	t.Helper()
	_, err := s.Inspect(t.Context(), key)
	if exists && err != nil || !exists && !errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("object %s exists=%t: %v", key, exists, err)
	}
}

// Removing the inventory stage, using business time for age, or treating a
// referenced key as an orphan must change the physical bytes observed here.
func TestAssetCleanupClocksAndLateObjects(t *testing.T) {
	w := newAssetWorld(t)
	physical := w.now.Add(-25 * time.Hour)
	objects, root := inventoryStore(t, &physical)
	other, otherRoot := inventoryStore(t, &physical)
	old := w
	old.now = physical
	a := w.asset(domain.ScopeIssue)
	a.Uploaded = true
	a.BoundAt = &w.now
	final := old.object(t, objects, a, 'o')
	a.Metadata.Final = &final
	// Same asset ID, independent key: only the final remains referenced.
	orphan := old.object(t, objects, a, 'o')
	missingRow := w.asset(domain.ScopeIssue)
	orphanUpload := old.object(t, objects, missingRow, 'u')
	elsewhere := old.object(t, other, missingRow, 'o')
	oldTemp := inventoryTemporary(t, root, physical)
	otherTemp := inventoryTemporary(t, otherRoot, physical)
	equalTemp := inventoryTemporary(t, root, w.now.Add(-24*time.Hour))
	newTemp := inventoryTemporary(t, root, w.now)
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	// A persisted grant/lease must outlive both a +61d business clock and a
	// smaller current operation budget. Neither runtime deadline is recomputed.
	pending := w.asset(domain.ScopeIssue)
	staging := old.object(t, objects, pending, 'u')
	staging.SafeAfter = w.now.Add(time.Minute)
	candidate := old.object(t, objects, pending, 'o')
	candidate.SafeAfter = w.now.Add(time.Minute)
	pending.Metadata.Origin = domain.OriginUpload
	pending.Metadata.Staging = &staging
	pending.Metadata.Candidates = []domain.Candidate{{ObjectRecord: candidate, LeaseUntil: w.now.Add(30 * time.Second)}}
	if _, err := w.store.CreateAsset(t.Context(), pending); err != nil {
		t.Fatal(err)
	}
	physical = w.now.Add(-24 * time.Hour)
	exact := old
	exact.now = physical
	equalObject := exact.object(t, objects, missingRow, 'u')
	physical = w.now
	lateUpload := w.object(t, objects, missingRow, 'u')
	lateSeal := w.object(t, objects, missingRow, 'o')
	result, err := w.cleanup(objects, nil, w.now.Add(61*24*time.Hour)).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete || result.Removed != 3 {
		t.Fatalf("old orphans and temporary were not reconciled: %+v %v", result, err)
	}
	for _, key := range []domain.ObjectKey{orphan.Key, orphanUpload.Key} {
		assertInventoryObject(t, objects, key, false)
	}
	for _, key := range []domain.ObjectKey{final.Key, staging.Key, candidate.Key, equalObject.Key, lateUpload.Key, lateSeal.Key} {
		assertInventoryObject(t, objects, key, true)
	}
	assertInventoryObject(t, other, elsewhere.Key, true)
	if _, err := os.Stat(oldTemp); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("old temporary remains: %v", err)
	}
	for _, path := range []string{equalTemp, newTemp, otherTemp} {
		if _, err := os.Stat(path); err != nil {
			t.Fatal(err)
		}
	}
	// Recreate an object only after its recovery record was removed, as a
	// delayed writer could. A fresh physical timestamp protects it for 24h.
	physical = w.now.Add(25 * time.Hour)
	lateWorld := w
	lateWorld.now = physical
	later := lateWorld.object(t, objects, missingRow, 'u')
	result, err = lateWorld.cleanup(objects, nil, w.now.Add(61*24*time.Hour)).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete {
		t.Fatalf("next cycle: %+v %v", result, err)
	}
	for _, key := range []domain.ObjectKey{staging.Key, candidate.Key, equalObject.Key, lateUpload.Key, lateSeal.Key} {
		assertInventoryObject(t, objects, key, false)
	}
	assertInventoryObject(t, objects, later.Key, true)
	assertInventoryObject(t, objects, final.Key, true)
}

func TestAssetCleanupStrictBusinessBoundaries(t *testing.T) {
	w := newAssetWorld(t)
	objects := w.local(t)
	pending := w.asset(domain.ScopeIssue)
	pending.CreatedAt = w.now.Add(-7 * 24 * time.Hour)
	staging := w.object(t, objects, pending, 'u')
	pending.Metadata.Origin, pending.Metadata.Staging = domain.OriginUpload, &staging
	deleted := w.asset(domain.ScopeIssue)
	deleted.Uploaded = true
	stamp := w.now.Add(-60 * 24 * time.Hour)
	deleted.DeletedAt = &stamp
	final := w.object(t, objects, deleted, 'o')
	deleted.Metadata.Final = &final
	for _, a := range []domain.Asset{pending, deleted} {
		if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
			t.Fatal(err)
		}
	}
	result, err := w.cleanup(objects, nil, w.now).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete || result.Removed != 0 {
		t.Fatalf("equality: %+v %v", result, err)
	}
	got, err := w.store.FindAsset(t.Context(), pending.ID)
	if err != nil || got.DeletedAt != nil {
		t.Fatalf("exact 7d expired: %+v %v", got, err)
	}
	assertInventoryObject(t, objects, staging.Key, true)
	assertInventoryObject(t, objects, final.Key, true)
	result, err = w.cleanup(objects, nil, w.now.Add(time.Microsecond)).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete || result.Removed != 2 {
		t.Fatalf("epsilon: %+v %v", result, err)
	}
	assertInventoryObject(t, objects, staging.Key, false)
	assertInventoryObject(t, objects, final.Key, false)
}
