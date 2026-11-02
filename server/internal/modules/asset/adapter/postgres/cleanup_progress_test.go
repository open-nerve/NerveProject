package postgresadapter

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func TestAssetCleanupBudgetResumesInsidePage(t *testing.T) {
	w := newAssetWorld(t)
	physical := w.now.Add(-25 * time.Hour)
	objects, _ := inventoryStore(t, &physical)
	old := w
	old.now = physical
	var keys []domain.ObjectKey
	for range 5 {
		keys = append(keys, old.object(t, objects, w.asset(domain.ScopeIssue), 'u').Key)
	}
	slices.Sort(keys)
	physical = w.now
	deleted := 0
	faulting := faultingCleanupObjects{objects, deleteObjectFunc(func(ctx context.Context, key domain.ObjectKey) error {
		if _, inside := postgres.Tx(ctx); inside || w.pool.Stat().AcquiredConns() != 0 {
			t.Fatal("inventory Delete holds database resources")
		}
		if deleted == 1 {
			<-ctx.Done()
			return ctx.Err()
		}
		deleted++
		return objects.Delete(ctx, key)
	})}
	d := w.cleanupDeps(faulting, nil, w.now)
	d.Batch, d.Budget = 3, time.Second
	first, err := app.NewCleanup(d).Execute(t.Context(), app.SweepCursor{})
	if err != nil || first.Complete || first.Removed != 1 || first.Failed != 0 || first.Cursor.Stage != "objects" || first.Cursor.Token != "" || first.Cursor.ObjectAfter != keys[0] {
		t.Fatalf("budget lost partial page: %+v %v", first, err)
	}
	assertInventoryObject(t, objects, keys[1], true)
	second, err := w.cleanup(objects, nil, w.now).Execute(t.Context(), first.Cursor)
	if err != nil || !second.Complete || second.Removed != 4 {
		t.Fatalf("resume: %+v %v", second, err)
	}
	for _, key := range keys {
		assertInventoryObject(t, objects, key, false)
	}
}

func TestAssetCleanupLateBytesAfterAcknowledgement(t *testing.T) {
	w := newAssetWorld(t)
	physical := w.now
	objects, _ := inventoryStore(t, &physical)
	a := w.asset(domain.ScopeIssue)
	a.Metadata.Origin = domain.OriginUpload
	a.CreatedAt = w.now.Add(-8 * 24 * time.Hour)
	u, _ := domain.NewObjectKey('u', a.ID, uuid.NewV7())
	o, _ := domain.NewObjectKey('o', a.ID, uuid.NewV7())
	record := domain.ObjectRecord{Key: u, Phase: domain.ObjectActive, SafeAfter: w.now.Add(-time.Minute), Size: 3, ContentType: a.ContentType}
	a.Metadata.Staging = &record
	sealed := record
	sealed.Key = o
	a.Metadata.Candidates = []domain.Candidate{{ObjectRecord: sealed, LeaseUntil: w.now.Add(-2 * time.Minute)}}
	if _, err := w.store.CreateAsset(t.Context(), a); err != nil {
		t.Fatal(err)
	}
	result, err := w.cleanup(objects, nil, w.now).Execute(t.Context(), app.SweepCursor{})
	if err != nil || !result.Complete || result.Removed != 2 {
		t.Fatalf("missing object acknowledgement: %+v %v", result, err)
	}
	// Storage-boundary crash fixture: materialize those exact old keys after
	// acknowledgement. This does not claim an expired public grant is accepted.
	if err := objects.Put(t.Context(), domain.UploadGrant{Key: u, Size: 3, ContentType: a.ContentType, ExpiresAt: physical.Add(time.Minute)}, strings.NewReader("abc")); err != nil {
		t.Fatal(err)
	}
	info, err := objects.Inspect(t.Context(), u)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := objects.Seal(t.Context(), u, o, info.ETag); err != nil {
		t.Fatal(err)
	}
	for _, offset := range []time.Duration{0, 24 * time.Hour, 24*time.Hour + time.Microsecond} {
		physical = w.now.Add(offset)
		current := w
		current.now = physical
		r, err := current.cleanup(objects, nil, w.now.Add(61*24*time.Hour)).Execute(t.Context(), app.SweepCursor{})
		if err != nil || !r.Complete {
			t.Fatalf("late cycle %s: %+v %v", offset, r, err)
		}
		assertInventoryObject(t, objects, u, offset <= 24*time.Hour)
		assertInventoryObject(t, objects, o, offset <= 24*time.Hour)
		got, err := w.store.FindAsset(t.Context(), a.ID)
		if err != nil || got.Metadata.Staging != nil || got.Metadata.Final != nil || len(got.Metadata.Candidates) != 0 || got.DeletedAt == nil {
			t.Fatalf("late key was re-adopted: %+v %v", got, err)
		}
	}
}

func TestAssetCleanupTemporaryErrorPreservesProgress(t *testing.T) {
	w := newAssetWorld(t)
	physical := w.now
	objects, root := inventoryStore(t, &physical)
	first := inventoryTemporary(t, root, w.now.Add(-25*time.Hour))
	second := inventoryTemporary(t, root, w.now.Add(-25*time.Hour))
	paths := []string{first, second}
	slices.Sort(paths)
	bad := filepath.Join(paths[1], "unexpected")
	if err := os.WriteFile(bad, []byte("do not delete"), 0600); err != nil {
		t.Fatal(err)
	}
	r, err := w.cleanup(objects, nil, w.now).Execute(t.Context(), app.SweepCursor{})
	if err == nil || r.Complete || r.Failed != 1 || r.Removed != 1 || r.Cursor.Stage != "tmp" || r.Cursor.Token == "" {
		t.Fatalf("temporary failure hidden: %+v %v", r, err)
	}
	if _, err := os.Stat(paths[0]); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("completed temporary not deleted: %v", err)
	}
	if _, err := os.ReadFile(bad); err != nil {
		t.Fatal("unknown layout lost", err)
	}
	if err := os.Remove(bad); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(paths[1], w.now.Add(-25*time.Hour), w.now.Add(-25*time.Hour)); err != nil {
		t.Fatal(err)
	}
	r, err = w.cleanup(objects, nil, w.now).Execute(t.Context(), r.Cursor)
	if err != nil || !r.Complete || r.Removed != 1 {
		t.Fatalf("temporary retry: %+v %v", r, err)
	}
}
