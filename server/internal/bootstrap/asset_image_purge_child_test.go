package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// The wrapper observes real asset pages; it cannot fabricate row deletion or
// cursor progress. Only the runtime budget clock is advanced for a deterministic
// pause, as in the existing River restart fixture.
type imageProcessPurger struct {
	Purger
	pool    *pgxpool.Pool
	mode    string
	calls   atomic.Int32
	elapsed *atomic.Int64
}

func (p *imageProcessPurger) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	var n int64
	if err := p.pool.QueryRow(ctx, "INSERT INTO image_purge_calls(process,mode,after_id,cutoff) VALUES($1,$2,$3,$4) RETURNING n", os.Getpid(), p.mode, cursor.After, cutoff).Scan(&n); err != nil {
		return purgePageProgress{Cursor: cursor}, err
	}
	calls := p.calls.Add(1)
	if (p.mode == "page_committed" && calls > 1) || (p.mode == "page_failed" && calls > 2) {
		<-ctx.Done()
		return purgePageProgress{Cursor: cursor}, ctx.Err()
	}
	page, err := p.Purger.Page(ctx, cutoff, batch, cursor)
	if _, recordErr := p.pool.Exec(ctx, "UPDATE image_purge_calls SET committed_after=$2,failed=$3 WHERE n=$1", n, page.Cursor.After, err != nil); recordErr != nil {
		return page, recordErr
	}
	if p.mode == "page_committed" {
		p.elapsed.Add(int64(5 * time.Second))
	}
	return page, err
}

func runImagePurgeChild(t *testing.T, mode string) {
	t.Helper()
	pool := openPool(t, os.Getenv("NERVE_TEST_IMAGE_PURGE_DB"))
	id, err := strconv.ParseInt(os.Getenv("NERVE_TEST_IMAGE_PURGE_JOB"), 10, 64)
	if err != nil {
		t.Fatal(err)
	}
	now, err := time.Parse(time.RFC3339Nano, os.Getenv("NERVE_TEST_IMAGE_PURGE_NOW"))
	if err != nil {
		t.Fatal(err)
	}
	if root := os.Getenv("NERVE_TEST_CONTENT_CLEANUP_ROOT"); root != "" {
		store, err := local.New(local.Config{Root: root, SizeLimit: assetdomain.MaxObjectSize}, local.Deps{UploadMAC: storagetest.MAC("upload"), ReadMAC: storagetest.MAC("read"), RuntimeNow: time.Now})
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = store.Close() }()
		w := imageRetentionWorld{issueWorld: issueWorld{pool: pool}, objects: store, now: now}
		if failed := cleanImageObjects(t, w, store); failed != 0 {
			t.Fatal("child physical cleanup failed", failed)
		}
	}
	var elapsed atomic.Int64
	runtimeNow := func() time.Time { return now.Add(time.Duration(elapsed.Load())) }
	modules := imagePurgeModules(pool)
	modules[0].rows = &imageProcessPurger{Purger: modules[0].rows, pool: pool, mode: mode, elapsed: &elapsed}
	uc := newPurgeDeleted(modules, clockAt(now), runtimeNow, 60*24*time.Hour, 1, 5*time.Second)
	var logs lockedBuffer
	runner, err := jobs.New(pool, jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, ShutdownTimeout: 100 * time.Millisecond, Logger: slog.New(slog.NewTextHandler(&logs, nil))},
		[]jobs.Job{{Worker: jobs.Register(&purgeWorker{uc: uc})}}, []string{purgeArgs{}.Kind()})
	if err != nil {
		t.Fatal(err)
	}
	runner.Start(t.Context())
	t.Cleanup(func() {
		if err := runner.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	defer cancel()
	awaitPurge(t, ctx, &logs, "durable image page progress", func() bool {
		var state string
		var failures int
		var metadata []byte
		if err := pool.QueryRow(ctx, "SELECT state,metadata,coalesce(cardinality(errors),0) FROM river_job WHERE id=$1", id).Scan(&state, &metadata, &failures); err != nil {
			t.Fatal(err)
		}
		result, present, err := decodePurgeOutput(metadata)
		if err != nil {
			t.Fatal(err)
		}
		if !present {
			return false
		}
		if mode == "resume" {
			return result.Complete && state == "completed"
		}
		if result.Complete || result.Deleted != 1 || result.Cursor.Module != "asset" {
			t.Fatalf("first process advanced beyond its page: %+v state=%s", result, state)
		}
		// River represents short retries as available and longer ones as
		// retryable. The persisted failure, not that scheduling choice, proves
		// that the real worker returned its second-page error.
		return mode == "page_committed" || failures > 0 && (state == "available" || state == "retryable" || state == "running")
	})
	t.Logf("real child pid=%d mode=%s River job=%d persisted output", os.Getpid(), mode, id)
}
