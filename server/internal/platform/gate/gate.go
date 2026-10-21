// Package gate bounds how many callers hold a key at once, within this
// process (M4 design 3.5, 11.11): a caller enters a key's gate, or several
// keys' one after another, waits in each key's queue, first come first
// served, while as many others hold it as its bound allows, and leaves
// them once. A key nobody holds or waits for takes no memory. It uses only
// the standard library.
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

// Enter returns once the caller holds each of keys, which it enters one
// after another, in their order, with the function that leaves them all,
// which the caller calls once on every path, and which does nothing when
// called again. While PerKey callers hold a key, or others wait for it
// already, the caller waits behind them, holding only the keys it entered
// before it; it is refused at once with ErrFull when MaxQueue others wait,
// and gives up with ErrTimeout once it has waited MaxWait in all, or with
// ctx's error once ctx is done, before or while it waits: then it leaves
// the keys it entered, and holds nothing. Callers that enter several keys
// enter them in one order, each once (the write gate: by project id, M4
// design 11.11), so that none waits for a key whose holder waits for one
// it holds.
func (g *Keyed[K]) Enter(ctx context.Context, keys ...K) (leave func(), err error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	keys = slices.Clone(keys)
	deadline := time.Now().Add(g.limits.MaxWait)
	for n, key := range keys {
		if err := g.enter(ctx, key, deadline); err != nil {
			g.leave(keys[:n])
			return nil, err
		}
	}
	var once sync.Once
	return func() { once.Do(func() { g.leave(keys) }) }, nil
}

// enter takes key for the caller: at once when fewer than PerKey hold it
// and no one waits for it; else it waits behind those who do until it is
// handed the key, or gives up, as Enter says, at deadline or once ctx is
// done, holding nothing of key and out of its queue.
func (g *Keyed[K]) enter(ctx context.Context, key K, deadline time.Time) error {
	g.mu.Lock()
	q := g.keys[key]
	if q == nil {
		q = &queue{}
		g.keys[key] = q
	}
	if q.holders < g.limits.PerKey && len(q.waiting) == 0 {
		q.holders++
		g.mu.Unlock()
		return nil
	}
	if len(q.waiting) >= g.limits.MaxQueue {
		g.mu.Unlock()
		return ErrFull
	}
	w := &waiter{ready: make(chan struct{})}
	q.waiting = append(q.waiting, w)
	g.mu.Unlock()

	timer := time.NewTimer(time.Until(deadline))
	defer timer.Stop()
	var err error
	select {
	case <-w.ready:
		return nil
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
		return err
	}
	q.waiting = slices.DeleteFunc(q.waiting, func(other *waiter) bool { return other == w })
	g.mu.Unlock()
	return err
}

// leave gives up each of keys (release).
func (g *Keyed[K]) leave(keys []K) {
	for _, key := range keys {
		g.release(key)
	}
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
