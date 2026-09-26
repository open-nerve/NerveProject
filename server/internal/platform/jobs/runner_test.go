package jobs

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"
	"time"
)

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
	if err := stop(t, r, context.Background(), time.Second); err != nil {
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
	if err := stop(t, r, context.Background(), time.Second); err != nil || time.Since(stopping) > 100*time.Millisecond {
		t.Errorf("Stop() = %v after %s, want nil at once, not after the %s pause", err, time.Since(stopping), firstRetry)
	}
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if len(fake.stops) != 0 || len(fake.starts) != 1 {
		t.Errorf("client started %d times, stopped %d times; want one attempt and no stop", len(fake.starts), len(fake.stops))
	}
}

// slowStart is a client whose Start takes until release is closed.
type slowStart struct {
	fakeClient
	underWay, release chan struct{}
}

func (s *slowStart) Start(ctx context.Context) error {
	close(s.underWay)
	<-s.release
	return s.fakeClient.Start(ctx)
}

// Stop waits for an attempt to start that is under way, and stops the client
// that the attempt started.
func TestStopWaitsForAnAttemptUnderWay(t *testing.T) {
	slow := &slowStart{fakeClient: fakeClient{attempts: make(chan struct{}, 1)}, underWay: make(chan struct{}), release: make(chan struct{})}
	r := newRunner(slow, Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)})
	start(t, r, context.Background())
	receive(t, slow.underWay, 5*time.Second, "attempt to start")

	stopped := make(chan error, 1)
	go func() { stopped <- r.Stop(context.Background()) }()
	select {
	case err := <-stopped:
		t.Fatalf("Stop() = %v while an attempt to start was under way", err)
	case <-time.After(100 * time.Millisecond):
	}
	close(slow.release)
	if err := receive(t, stopped, time.Second, "return from Stop"); err != nil {
		t.Errorf("Stop() = %v", err)
	}
	slow.mu.Lock()
	defer slow.mu.Unlock()
	if len(slow.stops) != 1 {
		t.Errorf("the client was stopped %d times, want once: the attempt under way started it", len(slow.stops))
	}
}

// hungStart is a client whose Start waits for its context, as River's does
// on a SELECT 1 that the database never answers.
type hungStart struct {
	fakeClient
	underWay chan struct{}
}

func (h *hungStart) Start(ctx context.Context) error {
	close(h.underWay)
	<-ctx.Done()
	return ctx.Err()
}

// Stop cancels an attempt to start that is under way, so that a database
// that never answers cannot hold the shutdown; nothing started, so there is
// no client to stop.
func TestStopCancelsAnAttemptUnderWay(t *testing.T) {
	hung := &hungStart{underWay: make(chan struct{})}
	r := newRunner(hung, Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)})
	start(t, r, context.Background())
	receive(t, hung.underWay, 5*time.Second, "attempt to start")

	if err := stop(t, r, context.Background(), time.Second); err != nil {
		t.Errorf("Stop() = %v", err)
	}
	hung.mu.Lock()
	defer hung.mu.Unlock()
	if len(hung.stops) != 0 {
		t.Errorf("the client was stopped %d times, want none: it never started", len(hung.stops))
	}
}

// stopWatcher records whether the context the client started with was
// still live when the runner stopped the client.
type stopWatcher struct {
	fakeClient
	liveAtStop chan bool
}

func (s *stopWatcher) Stop(ctx context.Context) error {
	s.mu.Lock()
	started := s.starts[len(s.starts)-1]
	s.mu.Unlock()
	s.liveAtStop <- started.Err() == nil
	return s.fakeClient.Stop(ctx)
}

// Once the client has started, Stop stops it with the client's Stop alone:
// River then cancels its own contexts with its stop cause, on which its
// reindexer drops an interrupted index build, rather than with the plain
// cancellation of the context it started with, which skips that cleanup.
// That context is released once the client has stopped.
func TestStopLeavesTheStartedClientToItsOwnStop(t *testing.T) {
	w := &stopWatcher{fakeClient: fakeClient{attempts: make(chan struct{}, 1)}, liveAtStop: make(chan bool, 1)}
	r := newRunner(w, Config{ShutdownTimeout: time.Second, Logger: slog.New(slog.DiscardHandler)})
	start(t, r, context.Background())
	receive(t, w.attempts, 5*time.Second, "attempt to start")

	if err := stop(t, r, context.Background(), time.Second); err != nil {
		t.Fatalf("Stop() = %v", err)
	}
	if !receive(t, w.liveAtStop, time.Second, "stop of the client") {
		t.Error("Stop cancelled the context the client started with before calling the client's Stop")
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.starts[0].Err() == nil {
		t.Error("the context the client started with is still live after Stop: it is never released")
	}
}
