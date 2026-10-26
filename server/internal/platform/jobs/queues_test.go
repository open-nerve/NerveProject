package jobs

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Both queues are held beyond capacity. Returning one job advances only
// that queue, even while the other remains full. A shared limit or incorrect
// queue capacity makes these observations fail.
func TestJobQueues(t *testing.T) {
	for _, limits := range []struct{ defaultN, periodicN int }{{10, 2}, {3, 1}} {
		t.Run(fmt.Sprintf("%d_%d", limits.defaultN, limits.periodicN), func(t *testing.T) {
			pool := newPool(t, pgtest.NewDatabase(t))
			running := map[string]chan struct{}{river.QueueDefault: make(chan struct{}, 20), "periodic": make(chan struct{}, 20)}
			releases := map[string]chan struct{}{river.QueueDefault: make(chan struct{}), "periodic": make(chan struct{})}
			worker := &probeWorker{work: func(ctx context.Context, j *river.Job[probeArgs]) error {
				running[j.Queue] <- struct{}{}
				select {
				case <-releases[j.Queue]:
					return nil
				case <-ctx.Done():
					return ctx.Err()
				}
			}}
			cfg := runnerConfig()
			cfg.DefaultWorkers, cfg.PeriodicWorkers = limits.defaultN, limits.periodicN
			r, err := New(pool, cfg, []Job{{Worker: Register(worker)}}, []string{"jobs_test.probe"})
			if err != nil {
				t.Fatal(err)
			}
			var once sync.Once
			releaseAll := func() { once.Do(func() { close(releases[river.QueueDefault]); close(releases["periodic"]) }) }
			t.Cleanup(releaseAll)
			for queue, n := range map[string]int{river.QueueDefault: limits.defaultN, "periodic": limits.periodicN} {
				for range n + 2 {
					if _, err := r.client.(*river.Client[pgx.Tx]).Insert(context.Background(), probeArgs{}, &river.InsertOpts{Queue: queue}); err != nil {
						t.Fatal(err)
					}
				}
			}
			start(t, r, context.Background())
			t.Cleanup(func() {
				releaseAll()
				if err := stop(t, r, context.Background(), 5*time.Second); err != nil {
					t.Error(err)
				}
			})
			for queue, n := range map[string]int{river.QueueDefault: limits.defaultN, "periodic": limits.periodicN} {
				for range n {
					receive(t, running[queue], 15*time.Second, "held "+queue+" job")
				}
			}
			assertHeld := func(queue string) {
				t.Helper()
				select {
				case <-running[queue]:
					t.Fatalf("%s exceeded its capacity", queue)
				case <-time.After(1500 * time.Millisecond):
				}
			}
			assertHeld(river.QueueDefault)
			assertHeld("periodic")
			releases[river.QueueDefault] <- struct{}{}
			receive(t, running[river.QueueDefault], 5*time.Second, "next default job")
			assertHeld("periodic")
			assertHeld(river.QueueDefault)
			releases["periodic"] <- struct{}{}
			receive(t, running["periodic"], 5*time.Second, "next periodic job")
			assertHeld(river.QueueDefault)
			assertHeld("periodic")
		})
	}
}
