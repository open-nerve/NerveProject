// Package jobs runs the modules' background jobs on River (M2 design 3.15):
// the server's client, which works the jobs and enqueues the periodic ones.
// It imports River and pgx, and no other platform package.
package jobs

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
)

// Job is one kind of job a module works: its worker and, when the job is
// periodic, its schedule. A module's river adapter builds it.
type Job struct {
	// Add registers the job's worker, e.g. with river.AddWorkerSafely.
	Add func(*river.Workers) error
	// Periodic, when set, enqueues the job on its schedule. River's leader
	// enqueues it, so one runs per schedule however many servers there are.
	Periodic *river.PeriodicJob
}

// Config is the runner's settings.
type Config struct {
	// ShutdownTimeout is how long Stop lets the running jobs finish before
	// it cancels their contexts: jobs.shutdown_timeout.
	ShutdownTimeout time.Duration
	Logger          *slog.Logger
}

const (
	// maxWorkers is how many jobs run at once. M2 has one periodic job.
	maxWorkers = 2
	// cancelGrace is how long Stop waits, once ShutdownTimeout has passed and
	// the jobs' contexts are cancelled, for the jobs to return.
	cancelGrace = time.Second
	// The first and the longest wait between two attempts to start.
	firstRetry = time.Second
	lastRetry  = 30 * time.Second
)

// client is what the runner uses of *river.Client.
type client interface {
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

// Runner is the server's River client.
type Runner struct {
	client          client
	logger          *slog.Logger
	shutdownTimeout time.Duration
	firstRetry      time.Duration
	lastRetry       time.Duration

	cancel  context.CancelFunc // ends the attempts to start
	done    chan struct{}      // closed once the attempts have ended
	started bool               // written before done is closed
}

// New builds the client on pool for jobs; Start runs it.
func New(pool *pgxpool.Pool, cfg Config, jobs []Job) (*Runner, error) {
	workers := river.NewWorkers()
	var periodic []*river.PeriodicJob
	for _, j := range jobs {
		if err := j.Add(workers); err != nil {
			return nil, fmt.Errorf("register a job: %w", err)
		}
		if j.Periodic != nil {
			periodic = append(periodic, j.Periodic)
		}
	}
	c, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Logger:       cfg.Logger,
		Queues:       map[string]river.QueueConfig{river.QueueDefault: {MaxWorkers: maxWorkers}},
		Workers:      workers,
		PeriodicJobs: periodic,
		// Stopping lets the running jobs finish for this long, then cancels
		// their contexts.
		SoftStopTimeout: cfg.ShutdownTimeout,
	})
	if err != nil {
		return nil, fmt.Errorf("create the jobs client: %w", err)
	}
	return newRunner(c, cfg), nil
}

func newRunner(c client, cfg Config) *Runner {
	return &Runner{client: c, logger: cfg.Logger, shutdownTimeout: cfg.ShutdownTimeout, firstRetry: firstRetry, lastRetry: lastRetry}
}

// Start starts the client in the background and returns at once. River's
// Start fails while the database cannot be reached; the runner then tries
// again, waiting from firstRetry up to lastRetry between attempts, so that
// nerve serves meanwhile and /readyz reports the database, as without jobs.
// Call Start once, and Stop after it.
func (r *Runner) Start(ctx context.Context) {
	ctx, r.cancel = context.WithCancel(context.WithoutCancel(ctx))
	r.done = make(chan struct{})
	go func() {
		defer close(r.done)
		for wait := r.firstRetry; ; wait = min(2*wait, r.lastRetry) {
			err := r.client.Start(ctx)
			if err == nil {
				r.started = true
				r.logger.InfoContext(ctx, "jobs started", slog.Duration("shutdown_timeout", r.shutdownTimeout))
				return
			}
			if ctx.Err() != nil {
				return
			}
			r.logger.WarnContext(ctx, "jobs did not start; trying again", slog.Any("error", err), slog.Duration("retry_in", wait))
			select {
			case <-ctx.Done():
				return
			case <-time.After(wait):
			}
		}
	}()
}

// Stop ends the attempts to start and stops the client: no job is fetched
// any more, the running ones get ShutdownTimeout to finish and then their
// contexts are cancelled. It returns once they have returned, or with an
// error when they still run cancelGrace after that.
func (r *Runner) Stop(ctx context.Context) error {
	r.cancel()
	<-r.done
	if !r.started {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), r.shutdownTimeout+cancelGrace)
	defer cancel()
	if err := r.client.Stop(ctx); err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			err = fmt.Errorf("jobs still running %s after jobs.shutdown_timeout (%s): %w", cancelGrace, r.shutdownTimeout, err)
		}
		return err
	}
	r.logger.InfoContext(ctx, "jobs stopped")
	return nil
}
