package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Real dispatcher, original transaction context, real River InsertTx: failure
// at either the first or last event must undo every business/activity/job row.
func TestIssueEventThreeSubscribersAndBatchRollback(t *testing.T) {
	for _, failAt := range []int{1, 3, 0} {
		t.Run(fmt.Sprintf("fail_at_%d", failAt), func(t *testing.T) {
			w := newIssueWorld(t)
			at := time.Date(2044, 3, 31, 10, 0, 0, 123456000, time.UTC)
			ids := []uuid.UUID{archiveRaceSeed(t, w, at, 40*24*time.Hour), archiveRaceSeed(t, w, at, 40*24*time.Hour), archiveRaceSeed(t, w, at, 40*24*time.Hour)}
			before := make(map[uuid.UUID]string)
			for _, id := range ids {
				before[id] = issueSnapshot(t, w, id)
			}
			client, err := jobs.NewInserter(w.pool)
			if err != nil {
				t.Fatal(err)
			}
			sentinel := errors.New("second subscriber rejects event")
			type contextKey struct{}
			ctx := context.WithValue(pgtest.Soon(t), contextKey{}, sentinel)
			var received []shared.Event
			var calls []string
			var transaction any
			checkContext := func(ctx context.Context) {
				t.Helper()
				tx, ok := postgres.Tx(ctx)
				if !ok || ctx.Value(contextKey{}) != sentinel {
					t.Fatal("subscriber lost original context or transaction")
				}
				if transaction == nil {
					transaction = tx
				} else if transaction != tx {
					t.Fatal("subscriber changed transaction")
				}
			}
			d, err := newDispatcher(issue.EventTypes(), []shared.EventSubscriber{
				{Name: "first inserts", Types: []string{"issue.archived"}, Handle: func(ctx context.Context, e shared.Event) error {
					checkContext(ctx)
					received = append(received, e)
					calls = append(calls, "first")
					var archived, activities int
					err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT count(*) FROM issues WHERE id=ANY($1::uuid[]) AND archived_at=$2 AND updated_at=$2 AND updated_by_id IS NULL`, ids, at).Scan(&archived)
					if err != nil {
						return err
					}
					err = postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT count(*) FROM issue_activities WHERE issue_id=ANY($1::uuid[]) AND field='archived_at' AND new_value='archive' AND actor_id IS NULL AND created_at=$2`, ids, at).Scan(&activities)
					if err != nil {
						return err
					}
					if archived != 3 || activities != 3 {
						t.Errorf("publication preceded complete business write: issues=%d activities=%d", archived, activities)
					}
					if issueEventCount(t, w) != 0 {
						t.Error("uncommitted jobs visible outside original transaction")
					}
					return (issueInserter{client: client}).InsertTx(ctx, issueEventProbeArgs{Event: e}, &river.InsertOpts{Queue: "issue_events"})
				}},
				{Name: "second may fail", Types: []string{"issue.archived"}, Handle: func(ctx context.Context, e shared.Event) error {
					checkContext(ctx)
					calls = append(calls, "second")
					if e.ID != received[len(received)-1].ID {
						t.Error("subscriber event identity changed")
					}
					var count int
					if err := postgres.DB(ctx, w.pool).QueryRow(ctx, `SELECT count(*) FROM river_job WHERE kind='issue_test.event_probe'`).Scan(&count); err != nil {
						return err
					}
					if count != len(received) {
						t.Errorf("prior InsertTx not in original transaction: jobs=%d events=%d", count, len(received))
					}
					if len(received) == failAt {
						return sentinel
					}
					return nil
				}},
				{Name: "third observes", Types: []string{"issue.archived"}, Handle: func(ctx context.Context, e shared.Event) error {
					checkContext(ctx)
					calls = append(calls, "third")
					if e.ID != received[len(received)-1].ID {
						t.Error("third event identity changed")
					}
					return nil
				}},
			})
			if err != nil {
				t.Fatal(err)
			}
			n, err := autoArchiveOn(w, clockAt(at), d).Execute(ctx)
			expectedEvents, wantJobs := 3, 3
			if failAt != 0 {
				expectedEvents, wantJobs = failAt, 0
				if n != 0 || !errors.Is(err, sentinel) {
					t.Fatalf("failure=%d,%v", n, err)
				}
				for _, id := range ids {
					if issueSnapshot(t, w, id) != before[id] {
						t.Errorf("batch rollback changed issue/activity %s", id)
					}
				}
			} else if n != 3 || err != nil {
				t.Fatalf("success=%d,%v", n, err)
			}
			if len(received) != expectedEvents || issueEventCount(t, w) != wantJobs {
				t.Fatalf("events=%d jobs=%d want=%d/%d", len(received), issueEventCount(t, w), expectedEvents, wantJobs)
			}
			var wantCalls []string
			seen := map[uuid.UUID]bool{}
			for i, e := range received {
				wantCalls = append(wantCalls, "first", "second")
				if i+1 != failAt {
					wantCalls = append(wantCalls, "third")
				}
				if e.ID == (uuid.UUID{}) || e.ID == e.Entity.ID || seen[e.ID] {
					t.Errorf("non-independent event ID: %+v", e)
				}
				seen[e.ID] = true
			}
			if !reflect.DeepEqual(calls, wantCalls) {
				t.Fatalf("subscriber order=%v want=%v", calls, wantCalls)
			}
		})
	}
}
