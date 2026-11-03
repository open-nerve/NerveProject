package postgresadapter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"slices"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	assetriver "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	assets3 "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/s3"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

type s3RecoveryProcessInput struct {
	Database          string
	Storage           assets3.Config
	Business, Runtime time.Time
	JobID             int64
}

func startS3RecoveryRunner(t *testing.T, pool *pgxpool.Pool, s *assets3.Store, d app.ObjectDeleter, in s3RecoveryProcessInput) *jobs.Runner {
	t.Helper()
	rows := New(pool)
	uc := app.NewCleanup(app.CleanupDeps{Rows: rows, Locks: rows, Writer: rows, Parents: parentFacts{}, Objects: d, Inventory: s, Temporary: s, Tx: postgres.NewTxManager(pool, time.Second), Clock: cleanupClock(in.Business), RuntimeNow: func() time.Time { return in.Runtime }, Batch: 2, Budget: time.Second, OperationTimeout: time.Second, PurgeAfter: 60 * 24 * time.Hour})
	r, err := jobs.New(pool, jobs.Config{DefaultWorkers: 1, PeriodicWorkers: 1, ShutdownTimeout: 50 * time.Millisecond, Logger: slog.New(slog.NewTextHandler(io.Discard, nil))}, []jobs.Job{{Worker: jobs.Register(assetriver.NewCleanupWorker(uc, time.Second))}}, []string{assetriver.CleanupKind})
	if err != nil {
		t.Fatal(err)
	}
	r.Start(t.Context())
	t.Cleanup(func() {
		if err := r.Stop(context.Background()); err != nil {
			t.Error(err)
		}
	})
	return r
}
func readS3Sweep(t *testing.T, pool *pgxpool.Pool, id int64) (string, app.SweepResult) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Second)
	defer cancel()
	var state string
	var raw []byte
	if err := pool.QueryRow(ctx, "SELECT state,metadata FROM river_job WHERE id=$1", id).Scan(&state, &raw); err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Output app.SweepResult `json:"output"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatal(err)
	}
	return state, envelope.Output
}
func awaitS3Sweep(t *testing.T, ready func() bool) {
	t.Helper()
	timer := time.NewTimer(30 * time.Second)
	defer timer.Stop()
	tick := time.NewTicker(20 * time.Millisecond)
	defer tick.Stop()
	for {
		if ready() {
			return
		}
		select {
		case <-tick.C:
		case <-timer.C:
			t.Fatal("durable S3 sweep progress timeout")
		}
	}
}
func insertS3Sweep(t *testing.T, pool *pgxpool.Pool) int64 {
	t.Helper()
	inserter, err := jobs.NewInserter(pool)
	if err != nil {
		t.Fatal(err)
	}
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

func s3ProcessRecovery(t *testing.T, b *s3test.Bucket) {
	w := newAssetWorld(t)
	cfg := s3RecoveryConfig(b)
	s := openRecoveryS3(t, cfg)
	keys := make([]domain.ObjectKey, 5)
	for i := range keys {
		keys[i] = s3Key(t, 'u', uuid.NewV7())
		putS3Recovery(t, b, keys[i], "text/plain")
	}
	slices.Sort(keys)
	neighbor := "neighbor/" + string(keys[0])
	_, err := b.Client().PutObject(t.Context(), &awss3.PutObjectInput{Bucket: &b.Name, Key: &neighbor, Body: bytes.NewReader([]byte("abc")), ContentType: aws.String("text/plain")})
	if err != nil {
		t.Fatal(err)
	}
	firstInfo, err := s.Inspect(t.Context(), keys[0])
	if err != nil {
		t.Fatal(err)
	}
	in := s3RecoveryProcessInput{Database: w.pool.Config().ConnString(), Storage: cfg, Business: firstInfo.ModifiedAt, Runtime: firstInfo.ModifiedAt.Add(25 * time.Hour), JobID: insertS3Sweep(t, w.pool)}
	var calls atomic.Int32
	blocking := deleteObjectFunc(func(ctx context.Context, key domain.ObjectKey) error {
		if calls.Add(1) > 3 {
			<-ctx.Done()
			return ctx.Err()
		}
		return s.Delete(ctx, key)
	})
	first := startS3RecoveryRunner(t, w.pool, s, blocking, in)
	var position app.SweepResult
	awaitS3Sweep(t, func() bool {
		_, position = readS3Sweep(t, w.pool, in.JobID)
		return position.Cursor.Stage == "objects" && position.Cursor.Token != "" && position.Cursor.ObjectAfter == keys[2]
	})
	if position.Complete || !position.Cursor.Cutoff.Equal(in.Business) || !position.Cursor.RuntimeCutoff.Equal(in.Runtime.Add(-24*time.Hour)) {
		t.Fatalf("partial page cutoffs lost: %+v", position)
	}
	if err := first.Stop(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	// No live first-run client, pool or runner participates in the next run.
	w.pool.Close()
	lowID, err := uuid.Parse("00000000-0000-4000-8000-000000000001")
	if err != nil {
		t.Fatal(err)
	}
	low := s3Key(t, 'u', lowID)
	putS3Recovery(t, b, low, "text/plain")
	child := in
	child.Business = child.Business.Add(61 * 24 * time.Hour)
	child.Runtime = child.Runtime.Add(24 * time.Hour)
	runS3RecoveryProcess(t, child)
	pool, err := pgxpool.New(t.Context(), in.Database)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	state, done := readS3Sweep(t, pool, in.JobID)
	if state != "completed" || !done.Complete || done.Cursor.Cutoff != position.Cursor.Cutoff || done.Cursor.RuntimeCutoff != position.Cursor.RuntimeCutoff {
		t.Fatalf("new process changed sweep facts: %s %+v", state, done)
	}
	s = openRecoveryS3(t, cfg)
	for _, key := range keys {
		expectS3Recovery(t, s, key, false)
	}
	expectS3Recovery(t, s, low, true)
	// The lower key was inserted behind this traversal. A fresh cycle must
	// discover it without inheriting the prior token or ObjectAfter.
	child.JobID = insertS3Sweep(t, pool)
	runS3RecoveryProcess(t, child)
	expectS3Recovery(t, s, low, false)
	out, err := b.Client().GetObject(t.Context(), &awss3.GetObjectInput{Bucket: &b.Name, Key: &neighbor})
	if err != nil {
		t.Fatal("neighbor lost", err)
	}
	body, err := io.ReadAll(out.Body)
	_ = out.Body.Close()
	if err != nil || string(body) != "abc" {
		t.Fatalf("neighbor bytes=%q %v", body, err)
	}
}

func runS3RecoveryProcess(t *testing.T, in s3RecoveryProcessInput) {
	t.Helper()
	raw, err := json.Marshal(in)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 45*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestS3RecoveryProcess$", "-test.count=1", "-test.timeout=40s")
	cmd.Env = append(os.Environ(), "NERVE_TEST_S3_RECOVERY=1")
	cmd.Stdin = bytes.NewReader(raw)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("fresh recovery process failed: %v\n%s", err, out)
	}
	if !bytes.Contains(out, []byte("PASS")) {
		t.Fatalf("fresh process did not run recovery: %s", out)
	}
}

// Invoked only by this test's parent with its generated DB/fixture credentials
// on stdin. It owns no container and never consults developer credentials.
func TestS3RecoveryProcess(t *testing.T) {
	if os.Getenv("NERVE_TEST_S3_RECOVERY") != "1" {
		t.Skip("subprocess entrypoint")
	}
	var in s3RecoveryProcessInput
	decoder := json.NewDecoder(io.LimitReader(os.Stdin, 1<<20))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&in); err != nil {
		t.Fatal("invalid recovery process fixture")
	}
	pool, err := pgxpool.New(t.Context(), in.Database)
	if err != nil {
		t.Fatal("open owned recovery database")
	}
	t.Cleanup(pool.Close)
	s := openRecoveryS3(t, in.Storage)
	startS3RecoveryRunner(t, pool, s, s, in)
	awaitS3Sweep(t, func() bool { state, _ := readS3Sweep(t, pool, in.JobID); return state == "completed" })
}
