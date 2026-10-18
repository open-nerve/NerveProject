package gate

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// long is a wait no test reaches: a caller that waits it has waited in
// vain.
const long = time.Minute

// entry is a caller's Enter, run on a goroutine of its own: done is closed
// when it returns, with its leave and its error.
type entry struct {
	done  chan struct{}
	leave func()
	err   error
}

// enter calls g.Enter(ctx, key) on a goroutine of its own.
func enter(g *Keyed[string], ctx context.Context, key string) *entry {
	e := &entry{done: make(chan struct{})}
	go func() {
		defer close(e.done)
		e.leave, e.err = g.Enter(ctx, key)
	}()
	return e
}

// returned waits a second at most for e to return, and fails the test when
// it does not.
func (e *entry) returned(t *testing.T) *entry {
	t.Helper()
	select {
	case <-e.done:
	case <-time.After(time.Second):
		t.Fatal("Enter did not return within 1s")
	}
	return e
}

// waits checks that e has not returned a little while after it began.
func (e *entry) waits(t *testing.T) {
	t.Helper()
	select {
	case <-e.done:
		t.Fatalf("Enter returned %v, want it waiting", e.err)
	case <-time.After(20 * time.Millisecond):
	}
}

// waiting is how many callers wait for key, under g's lock; queued waits a
// second at most until n do.
func (g *Keyed[K]) waiting(key K) int {
	g.mu.Lock()
	defer g.mu.Unlock()
	if q := g.keys[key]; q != nil {
		return len(q.waiting)
	}
	return 0
}

func queued(t *testing.T, g *Keyed[string], key string, n int) {
	t.Helper()
	for deadline := time.Now().Add(time.Second); g.waiting(key) != n; time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatalf("%d waiting for %s, want %d", g.waiting(key), key, n)
		}
	}
}

// forgotten checks that g keeps no key: none held, none waited for.
func forgotten(t *testing.T, g *Keyed[string]) {
	t.Helper()
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.keys) != 0 {
		t.Errorf("the gate keeps %d keys, want none", len(g.keys))
	}
}

// idle is how many places of g's keys are free while callers wait for them,
// under g's lock: none, since a holder who leaves hands his place to the
// first caller waiting.
func (g *Keyed[K]) idle() int {
	g.mu.Lock()
	defer g.mu.Unlock()
	n := 0
	for _, q := range g.keys {
		if len(q.waiting) > 0 {
			n += g.limits.PerKey - q.holders
		}
	}
	return n
}

// PerKey callers hold a key at once, each entering at once; the next waits
// until one of them leaves, then holds it. Another key's holders do not
// count. Once every caller has left, the gate keeps nothing.
func TestAKeyAdmitsPerKeyCallersAtOnce(t *testing.T) {
	g := New[string](Limits{PerKey: 2, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	a, b := enter(g, ctx, "web").returned(t), enter(g, ctx, "web").returned(t)
	ops := enter(g, ctx, "ops").returned(t)
	if a.err != nil || b.err != nil || ops.err != nil {
		t.Fatalf("Enter() = %v, %v, %v; want web twice and ops at once", a.err, b.err, ops.err)
	}
	c := enter(g, ctx, "web")
	c.waits(t)
	a.leave()
	if c.returned(t); c.err != nil {
		t.Fatalf("Enter() after a leaves = %v, want web", c.err)
	}
	b.leave()
	c.leave()
	ops.leave()
	forgotten(t, g)
}

// The callers waiting for a key are served in the order they came, each
// when the one before it leaves.
func TestWaitersAreServedInTheOrderTheyCame(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	holder := enter(g, ctx, "web").returned(t)
	var waiters []*entry
	for n := range 3 {
		waiters = append(waiters, enter(g, ctx, "web"))
		queued(t, g, "web", n+1)
	}
	leave := holder.leave
	for i, w := range waiters {
		for _, later := range waiters[i+1:] {
			later.waits(t)
		}
		leave()
		if w.returned(t); w.err != nil {
			t.Fatalf("waiter %d: Enter() = %v, want the key", i, w.err)
		}
		leave = w.leave
	}
	leave()
	forgotten(t, g)
}

// A caller who finds MaxQueue others waiting is refused at once with
// ErrFull, holding nothing; with MaxQueue 0, none waits. The key is not
// forgotten while it is held.
func TestAFullQueueRefusesAtOnce(t *testing.T) {
	for _, maxQueue := range []int{0, 2} {
		g := New[string](Limits{PerKey: 1, MaxQueue: maxQueue, MaxWait: long})
		ctx := context.Background()
		holder := enter(g, ctx, "web").returned(t)
		var waiters []*entry
		for n := range maxQueue {
			waiters = append(waiters, enter(g, ctx, "web"))
			queued(t, g, "web", n+1)
		}
		if refused := enter(g, ctx, "web").returned(t); !errors.Is(refused.err, ErrFull) || refused.leave != nil {
			t.Errorf("MaxQueue %d: Enter() with the queue full = %v, want ErrFull and nothing to leave", maxQueue, refused.err)
		}
		leave := holder.leave
		for _, w := range waiters {
			leave()
			leave = w.returned(t).leave
		}
		leave()
		forgotten(t, g)
	}
}

// A waiting caller gives up after MaxWait with ErrTimeout, or with its
// context's error once the context is cancelled or past its deadline, and
// one whose context is done already does not take even a free key. Each
// holds nothing and leaves the queue: the holder's leaving forgets the key,
// and the next caller holds it at once.
func TestAWaiterGivesUp(t *testing.T) {
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, tt := range []struct {
		name    string
		maxWait time.Duration
		ctx     func() (context.Context, context.CancelFunc)
		held    bool // whether the key is held when it enters
		want    error
	}{
		{"waited MaxWait", 30 * time.Millisecond, func() (context.Context, context.CancelFunc) { return context.Background(), func() {} }, true,
			ErrTimeout},
		{"cancelled while it waits", long, func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			time.AfterFunc(30*time.Millisecond, cancel)
			return ctx, cancel
		}, true, context.Canceled},
		{"past its deadline while it waits", long, func() (context.Context, context.CancelFunc) {
			return context.WithTimeout(context.Background(), 30*time.Millisecond)
		}, true, context.DeadlineExceeded},
		{"cancelled before", long, func() (context.Context, context.CancelFunc) { return cancelled, func() {} }, false, context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: tt.maxWait})
			var holder *entry
			if tt.held {
				holder = enter(g, context.Background(), "web").returned(t)
			}
			ctx, cancel := tt.ctx()
			defer cancel()
			if gaveUp := enter(g, ctx, "web").returned(t); !errors.Is(gaveUp.err, tt.want) || gaveUp.leave != nil {
				t.Errorf("Enter() = %v, want %v and nothing to leave", gaveUp.err, tt.want)
			}
			if g.waiting("web") != 0 {
				t.Errorf("%d still waiting, want none", g.waiting("web"))
			}
			if holder != nil {
				holder.leave()
			}
			forgotten(t, g)
			next := enter(g, context.Background(), "web").returned(t)
			if next.err != nil {
				t.Fatalf("Enter() after = %v, want the key at once", next.err)
			}
			next.leave()
		})
	}
}

// Leaving twice leaves once: the second call does not give up the key of
// the caller who holds it next.
func TestLeavingTwiceLeavesOnce(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	first := enter(g, ctx, "web").returned(t)
	first.leave()
	second := enter(g, ctx, "web").returned(t)
	first.leave()
	third := enter(g, ctx, "web")
	third.waits(t)
	second.leave()
	third.returned(t).leave()
	forgotten(t, g)
}

// Under load, on a few keys, with waits that time out and contexts
// cancelled at random, so that a key is handed to a caller as he gives up:
// never more than PerKey callers hold a key at once, no key has a free
// place while callers wait for it, every caller that enters leaves, and
// once all have, the gate keeps nothing (run with and without -race).
func TestUnderLoadAKeyIsNeverHeldByMoreThanPerKey(t *testing.T) {
	const perKey = 2
	g := New[string](Limits{PerKey: perKey, MaxQueue: 16, MaxWait: 2 * time.Millisecond})
	keys := []string{"web", "ops", "api"}
	var holders [3]atomic.Int32
	var most atomic.Int32
	var entered, refused, idle atomic.Int64
	var wg sync.WaitGroup
	for range 32 {
		wg.Go(func() {
			for range 1000 {
				k := rand.IntN(len(keys))
				ctx, cancel := context.WithTimeout(context.Background(), time.Duration(rand.IntN(3000))*time.Microsecond)
				leave, err := g.Enter(ctx, keys[k])
				idle.Add(int64(g.idle()))
				if err != nil {
					refused.Add(1)
					cancel()
					continue
				}
				entered.Add(1)
				n := holders[k].Add(1)
				for m := most.Load(); n > m && !most.CompareAndSwap(m, n); m = most.Load() {
				}
				time.Sleep(time.Duration(rand.IntN(200)) * time.Microsecond)
				holders[k].Add(-1)
				leave()
				cancel()
			}
		})
	}
	wg.Wait()
	if idle.Load() != 0 {
		t.Errorf("keys with callers waiting had %d free places, summed over the checks; want none", idle.Load())
	}
	if most.Load() > perKey || entered.Load() == 0 || refused.Load() == 0 {
		t.Errorf("at most %d held a key at once, %d entered, %d refused; want at most %d, some of each", most.Load(), entered.Load(),
			refused.Load(), perKey)
	}
	forgotten(t, g)
}
