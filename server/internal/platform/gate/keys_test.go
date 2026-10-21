package gate

import (
	"context"
	"errors"
	"testing"
	"time"
)

// A caller who enters several keys holds each: while he holds api and web,
// a caller of either waits, and once he leaves, each holds the key he
// waits for. One who finds his second key held waits for it holding his
// first, before which a caller of it then waits too; once the second is
// free, he holds both. Once every caller has left, the gate keeps nothing.
func TestACallerHoldsEachKeyHeEnters(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	both := enter(g, ctx, "api", "web").held(t)
	api, web := enter(g, ctx, "api"), enter(g, ctx, "web")
	api.waits(t)
	web.waits(t)
	both.leave()
	api.held(t).leave()
	web.held(t)
	second := enter(g, ctx, "api", "web")
	queued(t, g, "web", 1)
	behind := enter(g, ctx, "api")
	behind.waits(t)
	web.leave()
	second.held(t).leave()
	behind.held(t).leave()
	forgotten(t, g)
}

// A caller leaves the keys he entered, whatever he does with his slice of
// them afterwards: one who reuses it for ops and hr, which others hold,
// before he leaves, leaves api and web, which a caller of both then holds
// at once; ops and hr stay held, and a caller of either waits until its
// holder leaves. Once every caller has left, the gate keeps nothing.
func TestACallerLeavesTheKeysHeEntered(t *testing.T) {
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: long})
	ctx := context.Background()
	ops, hr := enter(g, ctx, "ops").held(t), enter(g, ctx, "hr").held(t)
	keys := []string{"api", "web"}
	both := enter(g, ctx, keys...).held(t)
	keys[0], keys[1] = "ops", "hr"
	both.leave()
	nextOps, nextHR := enter(g, ctx, "ops"), enter(g, ctx, "hr")
	nextOps.waits(t)
	nextHR.waits(t)
	enter(g, ctx, "api", "web").held(t).leave()
	ops.leave()
	nextOps.held(t).leave()
	hr.leave()
	nextHR.held(t).leave()
	forgotten(t, g)
}

// A caller of api and web who does not get web, its queue full, his wait
// at MaxWait, or his context cancelled while he waits, leaves api, which a
// caller of it then holds at once, and holds nothing; once web's holder
// leaves, the gate keeps nothing.
func TestACallerWhoDoesNotGetEachKeyHoldsNone(t *testing.T) {
	for _, tt := range []struct {
		name     string
		maxQueue int
		maxWait  time.Duration
		cancels  bool // whether the test cancels his context once he waits
		want     error
	}{
		{"its queue full", 0, long, false, ErrFull},
		{"waited MaxWait", 8, 30 * time.Millisecond, false, ErrTimeout},
		{"cancelled while he waits", 8, long, true, context.Canceled},
	} {
		t.Run(tt.name, func(t *testing.T) {
			g := New[string](Limits{PerKey: 1, MaxQueue: tt.maxQueue, MaxWait: tt.maxWait})
			web := enter(g, context.Background(), "web").held(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			gaveUp := enter(g, ctx, "api", "web")
			if tt.cancels {
				queued(t, g, "web", 1)
				cancel()
			}
			if gaveUp.returned(t); !errors.Is(gaveUp.err, tt.want) || gaveUp.leave != nil {
				t.Errorf("Enter() = %v, want %v and nothing to leave", gaveUp.err, tt.want)
			}
			enter(g, context.Background(), "api").held(t).leave()
			web.leave()
			forgotten(t, g)
		})
	}
}

// A caller waits MaxWait in all for his keys, not MaxWait for each: one
// who waited for api three quarters of MaxWait gives up on web at MaxWait
// from his start, where a wait of MaxWait for web alone would last until
// one and three quarters. MaxWait is a second, so the two are three
// quarters of a second apart: the answer must come at MaxWait at the
// earliest and before halfway between them.
func TestACallerWaitsMaxWaitInAll(t *testing.T) {
	const maxWait = time.Second
	g := New[string](Limits{PerKey: 1, MaxQueue: 8, MaxWait: maxWait})
	api, web := enter(g, context.Background(), "api").held(t), enter(g, context.Background(), "web").held(t)
	start := time.Now()
	both := enter(g, context.Background(), "api", "web")
	queued(t, g, "api", 1)
	time.Sleep(maxWait * 3 / 4)
	api.leave()
	both.returned(t)
	if took := time.Since(start); !errors.Is(both.err, ErrTimeout) || took < maxWait || took >= maxWait*11/8 {
		t.Errorf("Enter() = %v after %v; want ErrTimeout at %v, before %v", both.err, took, maxWait, maxWait*11/8)
	}
	web.leave()
	forgotten(t, g)
}
