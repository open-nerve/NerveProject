// Package gate bounds how many callers hold a key at once, within this
// process (M4 design 3.5, 11.11): a caller enters a key's gate, waits in
// the key's queue, first come first served, while as many others hold it
// as its bound allows, and leaves it once. A key nobody holds or waits for
// takes no memory. It uses only the standard library.
package gate

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"
)

var (
	// ErrFull answers a caller who finds as many callers waiting for the
	// key as the queue holds.
	ErrFull = errors.New("gate: the queue is full")
	// ErrTimeout answers a caller who waited for the key the longest wait.
	ErrTimeout = errors.New("gate: waited too long")
)

// Limits are a gate's bounds: at most PerKey callers hold a key at once,
// at least 1; at most MaxQueue others wait for it, 0 for none; each waits
// at most MaxWait (the configuration validates them).
type Limits struct {
	PerKey   int
	MaxQueue int
	MaxWait  time.Duration
}

// Keyed is a gate per key of type K, under one lock.
type Keyed[K comparable] struct {
	limits Limits
	mu     sync.Mutex
	keys   map[K]*queue
}

// queue is a key's state: how many callers hold it, and those waiting for
// it, in the order they came.
type queue struct {
	holders int
	waiting []*waiter
}

// waiter is a caller waiting for a key: ready is closed when it is handed
// the key, and granted set, under the gate's lock.
type waiter struct {
	ready   chan struct{}
	granted bool
}

// New returns a gate with the limits l.
func New[K comparable](l Limits) *Keyed[K] {
	return &Keyed[K]{limits: l, keys: map[K]*queue{}}
}

// Enter returns once the caller holds key, with the function that leaves
// it, which the caller calls once on every path, and which does nothing
// when called again. While PerKey callers hold key, or others wait for it
// already, the caller waits behind them, holding nothing; it is refused at
// once with ErrFull when MaxQueue others wait, and gives up with
// ErrTimeout after MaxWait, or with ctx's error once ctx is done, before
// or while it waits: then it holds nothing and leaves the queue.
func (g *Keyed[K]) Enter(ctx context.Context, key K) (leave func(), err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	g.mu.Lock()
	q := g.keys[key]
	if q == nil {
		q = &queue{}
		g.keys[key] = q
	}
	if q.holders < g.limits.PerKey && len(q.waiting) == 0 {
		q.holders++
		g.mu.Unlock()
		return g.leaver(key), nil
	}
	if len(q.waiting) >= g.limits.MaxQueue {
		g.mu.Unlock()
		return nil, ErrFull
	}
	w := &waiter{ready: make(chan struct{})}
	q.waiting = append(q.waiting, w)
	g.mu.Unlock()

	timer := time.NewTimer(g.limits.MaxWait)
	defer timer.Stop()
	select {
	case <-w.ready:
		return g.leaver(key), nil
	case <-timer.C:
		err = ErrTimeout
	case <-ctx.Done():
		err = ctx.Err()
	}
	g.mu.Lock()
	if w.granted {
		// Handed the key as it gave up: it passes the key on.
		g.mu.Unlock()
		g.release(key)
		return nil, err
	}
	q.waiting = slices.DeleteFunc(q.waiting, func(other *waiter) bool { return other == w })
	g.mu.Unlock()
	return nil, err
}

// leaver is the function that leaves key once.
func (g *Keyed[K]) leaver(key K) func() {
	var once sync.Once
	return func() { once.Do(func() { g.release(key) }) }
}

// release gives key up: to the first caller waiting for it, who then holds
// it; when none waits, one holder fewer, and a key nobody holds is
// forgotten.
func (g *Keyed[K]) release(key K) {
	g.mu.Lock()
	defer g.mu.Unlock()
	q := g.keys[key]
	if len(q.waiting) > 0 {
		w := q.waiting[0]
		q.waiting = q.waiting[1:]
		w.granted = true
		close(w.ready)
		return
	}
	q.holders--
	if q.holders == 0 {
		delete(g.keys, key)
	}
}
