package jobs

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// logBuffer collects the log lines that River's goroutines and the runner
// write concurrently.
type logBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *logBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// waitFor fails the test unless the logs hold want within limit.
func (b *logBuffer) waitFor(t *testing.T, want string, limit time.Duration) {
	t.Helper()
	for deadline := time.Now().Add(limit); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		if strings.Contains(b.String(), want) {
			return
		}
	}
	t.Fatalf("logs lack %q after %s:\n%s", want, limit, b.String())
}

func newLogger(b *logBuffer) *slog.Logger { return slog.New(slog.NewTextHandler(b, nil)) }

// newPool connects to a database of its own, migrated, River's tables too.
func newPool(t *testing.T, url string) *pgxpool.Pool {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: url, MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

type probeArgs struct{}

func (probeArgs) Kind() string { return "jobs_test.probe" }

// probeWorker runs work for every probe job.
type probeWorker struct {
	river.WorkerDefaults[probeArgs]
	work func(ctx context.Context) error
}

func (w *probeWorker) Work(ctx context.Context, _ *river.Job[probeArgs]) error { return w.work(ctx) }

// probeJob is a periodic job every interval, the first at once.
func probeJob(interval time.Duration, work func(ctx context.Context) error) Job {
	return Job{
		Add: func(w *river.Workers) error { return river.AddWorkerSafely(w, &probeWorker{work: work}) },
		Periodic: river.NewPeriodicJob(river.PeriodicInterval(interval),
			func() (river.JobArgs, *river.InsertOpts) { return probeArgs{}, nil },
			&river.PeriodicJobOpts{ID: "jobs_test.probe", RunOnStart: true}),
	}
}

// receive fails the test unless ch yields within limit.
func receive[T any](t *testing.T, ch <-chan T, limit time.Duration, what string) T {
	t.Helper()
	select {
	case v := <-ch:
		return v
	case <-time.After(limit):
		t.Fatalf("no %s within %s", what, limit)
	}
	var zero T
	return zero
}

// start runs r.Start(ctx) and fails the test unless it returns at once:
// Start must not wait for River, which may not start for a long while.
func start(t *testing.T, r *Runner, ctx context.Context) {
	t.Helper()
	returned := make(chan struct{})
	go func() {
		r.Start(ctx)
		close(returned)
	}()
	receive(t, returned, 100*time.Millisecond, "return from Start")
}

// The periodic job runs at once, then every interval, until Stop; Stop
// returns once the client has stopped, and nothing runs after it.
func TestRunnerWorksAPeriodicJobUntilStopped(t *testing.T) {
	t.Parallel()
	runs := make(chan time.Time, 100)
	var logs logBuffer
	r, err := New(newPool(t, pgtest.NewDatabase(t)), Config{ShutdownTimeout: 5 * time.Second, Logger: newLogger(&logs)},
		[]Job{probeJob(time.Second, func(context.Context) error { runs <- time.Now(); return nil })})
	if err != nil {
		t.Fatal(err)
	}
	start(t, r, context.Background())

	first := receive(t, runs, 15*time.Second, "first run")
	second := receive(t, runs, 10*time.Second, "second run")
	if gap := second.Sub(first); gap < 500*time.Millisecond {
		t.Errorf("runs %s apart, want about the 1s interval", gap)
	}
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	for len(runs) > 0 { // a run that was fetched before Stop
		<-runs
	}
	select {
	case at := <-runs:
		t.Errorf("a run at %v after Stop returned", at)
	case <-time.After(1500 * time.Millisecond):
	}
	out := logs.String()
	if !strings.Contains(out, `msg="jobs started"`) || !strings.Contains(out, `msg="jobs stopped"`) {
		t.Errorf("logs lack the start and the stop:\n%s", out)
	}
}

// Stop lets a running job finish for jobs.shutdown_timeout, then cancels
// its context and returns once it has returned. Two settings, so the value
// reaches River whichever it is.
func TestStopCancelsARunningJobAfterTheShutdownTimeout(t *testing.T) {
	t.Parallel()
	for _, timeout := range []time.Duration{time.Second, 3 * time.Second} {
		t.Run(timeout.String(), func(t *testing.T) {
			t.Parallel()
			running, cancelled := make(chan struct{}, 1), make(chan time.Time, 1)
			var logs logBuffer
			r, err := New(newPool(t, pgtest.NewDatabase(t)), Config{ShutdownTimeout: timeout, Logger: newLogger(&logs)},
				[]Job{probeJob(time.Hour, func(ctx context.Context) error {
					running <- struct{}{}
					select {
					case <-ctx.Done():
						cancelled <- time.Now()
						return ctx.Err()
					case <-time.After(15 * time.Second): // never cancelled: end, so the test fails instead of hanging
						return errors.New("the job's context was never cancelled")
					}
				})})
			if err != nil {
				t.Fatal(err)
			}
			start(t, r, context.Background())
			receive(t, running, 15*time.Second, "running job")

			stopping := time.Now()
			err = r.Stop(context.Background())
			took := time.Since(stopping)

			at := receive(t, cancelled, time.Second, "cancellation")
			if err != nil || took < timeout || took > timeout+cancelGrace {
				t.Errorf("Stop() = %v after %s, want nil after %s to %s", err, took, timeout, timeout+cancelGrace)
			}
			if waited := at.Sub(stopping); waited < timeout || waited > timeout+500*time.Millisecond {
				t.Errorf("the job's context was cancelled %s after Stop began, want %s", waited, timeout)
			}
		})
	}
}

// A job that ignores its cancelled context does not hold nerve's shutdown:
// Stop gives up cancelGrace after the timeout.
func TestStopGivesUpOnAJobThatIgnoresCancellation(t *testing.T) {
	t.Parallel()
	running, release := make(chan struct{}, 1), make(chan struct{})
	var logs logBuffer
	r, err := New(newPool(t, pgtest.NewDatabase(t)), Config{ShutdownTimeout: time.Second, Logger: newLogger(&logs)},
		[]Job{probeJob(time.Hour, func(context.Context) error {
			running <- struct{}{}
			<-release
			return nil
		})})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { close(release) }) // before the pool closes: cleanups run last first
	start(t, r, context.Background())
	receive(t, running, 15*time.Second, "running job")

	stopping := time.Now()
	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	err = receive(t, stopped, time.Second+cancelGrace+2*time.Second, "return from Stop")
	took := time.Since(stopping)

	if !errors.Is(err, context.DeadlineExceeded) || !strings.Contains(err.Error(), "after jobs.shutdown_timeout (1s)") ||
		took < time.Second+cancelGrace || took > time.Second+cancelGrace+500*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want the deadline after %s", err, took, time.Second+cancelGrace)
	}
}

// River's Start fails while the database cannot be reached: the runner
// keeps trying in the background, and Stop ends that at once.
func TestStartWithoutADatabase(t *testing.T) {
	var logs logBuffer
	r, err := New(newPool(t, "postgres://nobody@127.0.0.1:1/nowhere"), Config{ShutdownTimeout: 5 * time.Second, Logger: newLogger(&logs)},
		[]Job{probeJob(time.Hour, func(context.Context) error { return nil })})
	if err != nil {
		t.Fatal(err)
	}
	start(t, r, context.Background())
	logs.waitFor(t, `msg="jobs did not start; trying again"`, 5*time.Second)

	stopping := time.Now()
	if err := r.Stop(context.Background()); err != nil || time.Since(stopping) > 500*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want nil at once", err, time.Since(stopping))
	}
	if strings.Contains(logs.String(), `msg="jobs started"`) {
		t.Errorf("logs claim the jobs started:\n%s", logs.String())
	}
}

func TestNewRejectsTwoWorkersOfOneKind(t *testing.T) {
	job := probeJob(time.Hour, func(context.Context) error { return nil })
	_, err := New(newPool(t, "postgres://nobody@127.0.0.1:1/nowhere"), Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)},
		[]Job{job, {Add: job.Add}})
	if err == nil || !strings.Contains(err.Error(), "register a job") {
		t.Errorf("New() = %v, want the second worker of jobs_test.probe refused", err)
	}
}

// fakeClient fails Start until fails runs out; it records the contexts
// that Start and Stop get.
type fakeClient struct {
	mu       sync.Mutex
	fails    int
	starts   []context.Context
	stops    []context.Context
	attempts chan struct{}
}

func (f *fakeClient) Start(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.starts = append(f.starts, ctx)
	f.attempts <- struct{}{}
	if f.fails > 0 {
		f.fails--
		return errors.New("dial tcp 127.0.0.1:1: connection refused")
	}
	return nil
}

func (f *fakeClient) Stop(ctx context.Context) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.stops = append(f.stops, ctx)
	return nil
}

// The runner tries again, waiting twice as long each time up to its
// longest wait, until River starts; the client runs on, whatever happens to
// the caller's context, until Stop stops it within the shutdown budget.
func TestStartTriesAgainUntilTheClientStarts(t *testing.T) {
	fake := &fakeClient{fails: 3, attempts: make(chan struct{}, 10)}
	var logs logBuffer
	r := newRunner(fake, Config{ShutdownTimeout: 7 * time.Second, Logger: newLogger(&logs)})
	r.firstRetry, r.lastRetry = 10*time.Millisecond, 25*time.Millisecond
	ctx, cancel := context.WithCancel(context.Background())

	start(t, r, ctx)
	for range 4 {
		receive(t, fake.attempts, 5*time.Second, "attempt to start")
	}
	logs.waitFor(t, `msg="jobs started" shutdown_timeout=7s`, 5*time.Second)
	cancel() // the server's context ends at shutdown, before the jobs stop

	fake.mu.Lock()
	running := fake.starts[3]
	fake.mu.Unlock()
	if running.Err() != nil {
		t.Errorf("the client's context ended with the caller's: %v", running.Err())
	}
	out := logs.String()
	for _, wait := range []string{"retry_in=10ms", "retry_in=20ms", "retry_in=25ms"} {
		if !strings.Contains(out, `msg="jobs did not start; trying again" error="dial tcp 127.0.0.1:1: connection refused" `+wait) {
			t.Errorf("logs lack the retry with %s:\n%s", wait, out)
		}
	}

	stopping := time.Now()
	if err := r.Stop(context.Background()); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.stops) != 1 {
		t.Fatalf("Stop() called the client's Stop %d times, want once", len(fake.stops))
	}
	deadline, ok := fake.stops[0].Deadline()
	if budget := deadline.Sub(stopping); !ok || budget < 7*time.Second+cancelGrace || budget > 7*time.Second+cancelGrace+100*time.Millisecond {
		t.Errorf("the client's Stop got %s to stop (deadline set: %v), want 7s and cancelGrace", budget, ok)
	}
}

// Stop ends the attempts to start without waiting out the pause between
// them, and has nothing to stop.
func TestStopEndsTheAttemptsToStart(t *testing.T) {
	fake := &fakeClient{fails: 1000, attempts: make(chan struct{}, 1000)}
	r := newRunner(fake, Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)})
	start(t, r, context.Background())
	receive(t, fake.attempts, 5*time.Second, "attempt to start")

	stopping := time.Now()
	if err := r.Stop(context.Background()); err != nil || time.Since(stopping) > 100*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want nil at once, not after the %s pause", err, time.Since(stopping), firstRetry)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.stops) != 0 || len(fake.starts) != 1 {
		t.Errorf("client started %d times, stopped %d times; want one attempt and no stop", len(fake.starts), len(fake.stops))
	}
}
