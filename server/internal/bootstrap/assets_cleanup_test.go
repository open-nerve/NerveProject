package bootstrap

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	assetpg "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/postgres"
	assetriver "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	assetapp "github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

type cleanupInventoryFunc func(context.Context, string, int) (assetapp.ObjectPage, error)

func (f cleanupInventoryFunc) List(ctx context.Context, token string, n int) (assetapp.ObjectPage, error) {
	return f(ctx, token, n)
}

type noCleanupParents struct{}

func (noCleanupParents) Deletions(context.Context, []assetapp.Host) (map[assetapp.ParentKey]assetapp.DeletionStamp, error) {
	return nil, errors.New("orphan fixture unexpectedly read owner facts")
}

// Actual River persists a failed physical deletion and a snoozed inventory
// position. A newly constructed runner/store resumes the same job and cutoff.
func TestAssetSweepRestarts(t *testing.T) {
	pool := openPool(t, pgtest.NewDatabase(t))
	rows := assetpg.New(pool)
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	var physical atomic.Int64
	physical.Store(now.Add(-25 * time.Hour).UnixNano())
	runtimeNow := func() time.Time { return time.Unix(0, physical.Load()).UTC() }
	root := t.TempDir()
	open := func() *local.Store {
		t.Helper()
		s, err := local.New(local.Config{Root: root, SizeLimit: assetdomain.MaxObjectSize}, local.Deps{UploadMAC: storagetest.MAC("upload"), ReadMAC: storagetest.MAC("read"), RuntimeNow: runtimeNow})
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
	store := open()
	put := func(s *local.Store, n int) assetdomain.ObjectKey {
		t.Helper()
		id, err := uuid.Parse(fmt.Sprintf("00000000-0000-4000-8000-%012x", n))
		if err != nil {
			t.Fatal(err)
		}
		key, err := assetdomain.NewObjectKey('u', id, uuid.NewV7())
		if err != nil {
			t.Fatal(err)
		}
		if err := s.Put(t.Context(), assetdomain.UploadGrant{Key: key, Size: 3, ContentType: "text/plain", ExpiresAt: runtimeNow().Add(time.Minute)}, strings.NewReader("abc")); err != nil {
			t.Fatal(err)
		}
		return key
	}
	keys := []assetdomain.ObjectKey{put(store, 10), put(store, 20), put(store, 30), put(store, 40)}
	physical.Store(now.UnixNano())
	fresh := put(store, 50)
	// This is an actual filesystem fault, not an ObjectDeleter that returns a
	// fabricated error. Restore permissions before any fixture cleanup.
	blockedPath := filepath.Join(root, "objects", string(keys[0]))
	if err := os.Chmod(blockedPath, 0500); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(blockedPath, 0700) }()
	inserter, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
	insert := func() int64 {
		t.Helper()
		tx, err := pool.Begin(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = tx.Rollback(context.Background()) }()
		if err := inserter.InsertTx(t.Context(), tx, assetriver.CleanupArgs{}, &river.InsertOpts{Queue: "periodic"}); err != nil {
			t.Fatal(err)
		}
		if err := tx.Commit(t.Context()); err != nil {
			t.Fatal(err)
		}
		var id int64
		if err := pool.QueryRow(t.Context(), "SELECT max(id) FROM river_job WHERE kind=$1", assetriver.CleanupKind).Scan(&id); err != nil {
			t.Fatal(err)
		}
		return id
	}
	jobID := insert()
	var logs lockedBuffer
	start := func(s *local.Store, listing assetapp.ObjectLister, business time.Time) *jobs.Runner {
		t.Helper()
		uc := assetapp.NewCleanup(assetapp.CleanupDeps{Rows: rows, Locks: rows, Writer: rows, Parents: noCleanupParents{}, Objects: s, Inventory: listing, Temporary: s, Tx: postgres.NewTxManager(pool, time.Second), Clock: clockAt(business), RuntimeNow: runtimeNow, Batch: 2, Budget: time.Second, OperationTimeout: time.Second, PurgeAfter: 60 * 24 * time.Hour})
		r, err := jobs.New(pool, jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, ShutdownTimeout: 100 * time.Millisecond, Logger: slog.New(slog.NewTextHandler(&logs, nil))}, []jobs.Job{{Worker: jobs.Register(assetriver.NewCleanupWorker(uc, time.Second))}}, []string{assetriver.CleanupKind})
		if err != nil {
			t.Fatal(err)
		}
		r.Start(t.Context())
		t.Cleanup(func() { _ = r.Stop(context.Background()) })
		return r
	}
	var calls atomic.Int32
	first := start(store, cleanupInventoryFunc(func(ctx context.Context, token string, n int) (assetapp.ObjectPage, error) {
		if calls.Add(1) >= 3 {
			<-ctx.Done()
			return assetapp.ObjectPage{}, ctx.Err()
		}
		return store.List(ctx, token, n)
	}), now)
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	var state string
	var metadata []byte
	var failures int
	read := func(id int64) {
		t.Helper()
		if err := pool.QueryRow(ctx, "SELECT state,metadata,coalesce(cardinality(errors),0) FROM river_job WHERE id=$1", id).Scan(&state, &metadata, &failures); err != nil {
			t.Fatal(err)
		}
	}
	awaitPurge(t, ctx, &logs, "retry then durable snooze", func() bool { read(jobID); return calls.Load() >= 4 || state == "completed" || state == "discarded" })
	var envelope struct {
		Output assetapp.SweepResult `json:"output"`
	}
	if err := json.Unmarshal(metadata, &envelope); err != nil {
		t.Fatal(err)
	}
	position := envelope.Output.Cursor
	if calls.Load() < 4 || failures != 1 || envelope.Output.Failed != 1 || envelope.Output.Complete || position.Stage != "objects" || position.Token == "" || !position.Cutoff.Equal(now) || !position.RuntimeCutoff.Equal(now.Add(-24*time.Hour)) {
		t.Fatalf("lost failure/snoozed position: calls=%d errors=%d state=%s metadata=%s", calls.Load(), failures, state, metadata)
	}
	if _, err := store.Inspect(ctx, keys[0]); err != nil {
		t.Fatal("failed Delete lost object", err)
	}
	if _, err := store.Inspect(ctx, keys[1]); !errors.Is(err, assetdomain.ErrObjectNotFound) {
		t.Fatalf("neighbour was not deleted: %v", err)
	}
	if err := first.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(blockedPath, 0700); err != nil {
		t.Fatal(err)
	}
	physical.Store(now.Add(49 * time.Hour).UnixNano())
	reopened := open()
	second := start(reopened, reopened, now.Add(61*24*time.Hour))
	awaitPurge(t, ctx, &logs, "same sweep completed after restart", func() bool { read(jobID); return state == "completed" })
	if err := json.Unmarshal(metadata, &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.Output.Complete || envelope.Output.Failed != 1 || !envelope.Output.Cursor.Cutoff.Equal(now) || !envelope.Output.Cursor.RuntimeCutoff.Equal(position.RuntimeCutoff) || failures < 1 {
		t.Fatalf("completion erased failure or changed cutoff: %s errors=%d", metadata, failures)
	}
	for _, key := range keys[1:] {
		if _, err := reopened.Inspect(ctx, key); !errors.Is(err, assetdomain.ErrObjectNotFound) {
			t.Fatalf("missed later page %s: %v", key, err)
		}
	}
	for _, key := range []assetdomain.ObjectKey{keys[0], fresh} {
		if _, err := reopened.Inspect(ctx, key); err != nil {
			t.Fatalf("failed or young key lost: %s %v", key, err)
		}
	}
	if err := second.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	// A new cycle starts at the beginning, finding both an old blocked key and
	// a newly discovered low ID, without inheriting the old traversal position.
	physical.Store(now.UnixNano())
	low := put(reopened, 1)
	physical.Store(now.Add(49 * time.Hour).UnixNano())
	newID := insert()
	third := start(reopened, reopened, now.Add(61*24*time.Hour))
	awaitPurge(t, ctx, &logs, "fresh cycle completed", func() bool { read(newID); return state == "completed" })
	for _, key := range []assetdomain.ObjectKey{keys[0], fresh, low} {
		if _, err := reopened.Inspect(ctx, key); !errors.Is(err, assetdomain.ErrObjectNotFound) {
			t.Fatalf("new cycle missed %s: %v", key, err)
		}
	}
	if err := third.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
}
