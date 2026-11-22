package postgresadapter_test

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// readVisits reads each row of ids.
func readVisits(t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) []visitColumns {
	t.Helper()
	out := make([]visitColumns, len(ids))
	for i, id := range ids {
		out[i] = readVisit(t, pool, id)
	}
	return out
}

// A work item's deletion soft-deletes, in its transaction, every account's undeleted visits of it in its workspace
// (P3d spec 2.6): at the moment and by the actor it is given, nil for none, their visits kept. They are locked in id
// order (spec 2.8): written in the reverse of their id order with the middle one held, the cascade waits on it holding
// the lowest alone. One of it deleted before keeps its deletion; his visits of another work item and of its project,
// and a visit of the same id in another workspace, stay as they were.
func TestTheCascadeOfADeletedWorkItem(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newViews(t, pool)
	third, deleter := uuid.NewV7(), uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cid@corp.com', 'x', 'cid'), ($2, 'dan@corp.com', 'x', 'dan')", third, deleter)
	item, other := uuid.NewV7(), uuid.NewV7()
	made, visited := now.Add(-2*time.Hour), now.Add(-time.Hour)
	of := func(n int, workspace, project, user uuid.UUID, kind domain.RecentVisitKind, target uuid.UUID) visitRow {
		return visitRow{numberedID(n), workspace, project, user, kind, target, made, visited, nil}
	}
	before := of(4, w.workspace, w.project, w.owner, domain.RecentVisitIssue, item)
	before.deleted = ptr(now.Add(-time.Minute))
	seedVisit(t, pool, before, of(3, w.workspace, w.project, w.owner, domain.RecentVisitIssue, item),
		of(2, w.workspace, w.project, w.other, domain.RecentVisitIssue, item), of(1, w.workspace, w.project, third, domain.RecentVisitIssue, item),
		of(5, w.workspace, w.project, w.owner, domain.RecentVisitIssue, other), of(6, w.workspace, w.project, w.owner, domain.RecentVisitProject, w.project),
		of(7, w.elsewhere, w.elsewhereProject, w.owner, domain.RecentVisitIssue, item))
	mine := []uuid.UUID{numberedID(1), numberedID(2), numberedID(3)}
	kept := []uuid.UUID{numberedID(4), numberedID(5), numberedID(6), numberedID(7)}
	want, unchanged := readVisits(t, pool, mine...), readVisits(t, pool, kept...)
	at := now.Add(time.Hour)
	for i := range want {
		want[i].Deleted, want[i].Changed, want[i].Changer = &at, at, &deleter
	}

	holder, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = holder.Rollback(ctx) }()
	if _, err := holder.Exec(ctx, "SELECT 1 FROM user_recent_visits WHERE id = $1 FOR UPDATE", numberedID(2)); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(ctx, func(ctx context.Context) error {
			return s.DeleteIssueRecentVisits(ctx, w.workspace, item, at, &deleter)
		})
	}()
	pgtest.WaitForLockWaitBehind(t, pool, holder.Conn().PgConn(), 10*time.Second)
	if !pgtest.RowWaits(t, pool, "user_recent_visits", numberedID(1), "FOR UPDATE") || pgtest.RowWaits(t, pool, "user_recent_visits", numberedID(3), "FOR UPDATE") {
		t.Fatal("the cascade waits on the middle row without holding the lowest id alone; want the rows locked in id order")
	}
	if err := holder.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if got := readVisits(t, pool, mine...); !reflect.DeepEqual(got, want) {
		t.Fatalf("the work item's visits %+v; want %+v", got, want)
	}
	if got := readVisits(t, pool, kept...); !reflect.DeepEqual(got, unchanged) {
		t.Fatalf("deleted before, another work item's, the project's and another workspace's %+v; want %+v", got, unchanged)
	}

	// A deletion without an actor writes none: every account's visits of the other work item.
	seedVisit(t, pool, of(8, w.workspace, w.project, w.other, domain.RecentVisitIssue, other))
	nobody := readVisits(t, pool, numberedID(5), numberedID(8))
	later := at.Add(time.Minute)
	for i := range nobody {
		nobody[i].Deleted, nobody[i].Changed, nobody[i].Changer = &later, later, nil
	}
	if err := s.DeleteIssueRecentVisits(ctx, w.workspace, other, later, nil); err != nil {
		t.Fatal(err)
	}
	if got := readVisits(t, pool, numberedID(5), numberedID(8)); !reflect.DeepEqual(got, nobody) {
		t.Fatalf("deleted without an actor %+v; want %+v", got, nobody)
	}
}
