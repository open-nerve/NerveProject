package bootstrap

import (
	"context"
	"log/slog"
	"os"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	assetdomain "github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// The wrapper observes real asset pages; it cannot fabricate row deletion or
// cursor progress. It uses the runtime clock or the real work-context deadline
// to pause at a committed page, a failed page, or a cancelled initial attempt.
type imageProcessPurger struct {
	Purger
	pool        *pgxpool.Pool
	mode        string
	paused      atomic.Bool
	elapsed     *atomic.Int64
	expireFirst atomic.Bool
}

func (p *imageProcessPurger) Page(ctx context.Context, cutoff time.Time, batch int, cursor purgePageCursor) (purgePageProgress, error) {
	var n int64
	if err := p.pool.QueryRow(ctx, "INSERT INTO image_purge_calls(process,mode,after_id,cutoff) VALUES($1,$2,$3,$4) RETURNING n", os.Getpid(), p.mode, cursor.After, cutoff).Scan(&n); err != nil {
		return purgePageProgress{Cursor: cursor}, err
	}
	if p.expireFirst.Swap(false) || p.paused.Load() {
		<-ctx.Done()
		return purgePageProgress{Cursor: cursor}, ctx.Err()
	}
	page, err := p.Purger.Page(ctx, cutoff, batch, cursor)
	if _, recordErr := p.pool.Exec(ctx, "UPDATE image_purge_calls SET committed_after=$2,failed=$3 WHERE n=$1", n, page.Cursor.After, err != nil); recordErr != nil {
		return page, recordErr
	}
	// A cancelled attempt has not reached the observed commit/failure. It must
	// leave the real page eligible for River's next attempt.
	if p.mode == "page_committed" && err == nil {
		p.paused.Store(true)
		p.elapsed.Add(int64(5 * time.Second))
	}
	if p.mode == "page_failed" && err != nil && ctx.Err() == nil {
		p.paused.Store(true)
	}
	return page, err
}

// The API fixture can leave unrelated periodic jobs in this same database.
// Defer those through River so only the parent's selected job can reach the
// process-local page observer; the selected job uses the production worker.
type imageProcessWorker struct {
	*purgeWorker
	id int64
}

func (w *imageProcessWorker) Work(ctx context.Context, job *river.Job[purgeArgs]) error {
	if job.ID != w.id {
		return river.JobSnooze(time.Hour)
	}
	return w.purgeWorker.Work(ctx, job)
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
	pages := &imageProcessPurger{Purger: modules[0].rows, pool: pool, mode: mode, elapsed: &elapsed}
	pages.expireFirst.Store(os.Getenv(imageProcessExpireFirst) == "true")
	modules[0].rows = pages
	uc := newPurgeDeleted(modules, clockAt(now), runtimeNow, 60*24*time.Hour, 1, 5*time.Second)
	var logs lockedBuffer
	runner, err := jobs.New(pool, jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, ShutdownTimeout: 100 * time.Millisecond, Logger: slog.New(slog.NewTextHandler(&logs, nil))},
		[]jobs.Job{{Worker: jobs.Register(&imageProcessWorker{purgeWorker: &purgeWorker{uc: uc}, id: id})}}, []string{purgeArgs{}.Kind()})
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
	var lastOutput string
	awaitPurge(t, ctx, &logs, "durable image page progress", func() bool {
		var state string
		var failures int
		var metadata []byte
		if err := pool.QueryRow(ctx, "SELECT state,metadata,coalesce(cardinality(errors),0) FROM river_job WHERE id=$1", id).Scan(&state, &metadata, &failures); err != nil {
			t.Fatalf("reading purge progress: %v; logs:\n%s", err, logs.String())
		}
		if current := state + string(metadata); current != lastOutput {
			t.Logf("purge state=%s failures=%d metadata=%s", state, failures, metadata)
			lastOutput = current
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
		if result.Complete || result.Cursor.Module != "asset" {
			t.Fatalf("first process advanced beyond its page: %+v state=%s", result, state)
		}
		if result.Cursor.Table == "" && result.Cursor.After == uuid.Nil() && result.Scanned == 0 && result.Deleted == 0 && result.Blocked == 0 {
			return false // A budget can expire before the first page commits.
		}
		if result.Cursor.Table != "file_assets" || result.Cursor.After == uuid.Nil() {
			t.Fatalf("first process has no committed asset position: %+v state=%s", result, state)
		}
		// Counts belong to an attempt, whereas the cursor survives subsequent
		// budget-only attempts. The parent verifies its exact first ID and the
		// real rows/references on both sides of that committed position.
		// River represents short retries as available and longer ones as
		// retryable. The persisted failure, not that scheduling choice, proves
		// that the real worker returned its second-page error.
		return mode == "page_committed" || failures > 0 && (state == "available" || state == "retryable" || state == "running")
	})
	t.Logf("real child pid=%d mode=%s River job=%d persisted output", os.Getpid(), mode, id)
}
