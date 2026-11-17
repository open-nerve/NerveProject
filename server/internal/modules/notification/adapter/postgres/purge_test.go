package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

var now = time.Date(2026, 11, 17, 9, 0, 0, 0, time.UTC)

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func exec(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func purgeID(n int) uuid.UUID { return uuid.MustParse(fmt.Sprintf("0199a2b4-0000-7000-8000-%012x", n)) }

// world is an account, a workspace and its project, written directly: identity, workspace and project own them.
type world struct{ receiver, workspace, project uuid.UUID }

func newWorld(t *testing.T, pool *pgxpool.Pool) world {
	t.Helper()
	w := world{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'bob@corp.com', 'x', 'bob')", w.receiver)
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, 'Acme', 'acme')", w.workspace)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Web', 'WEB')", w.project, w.workspace)
	return w
}

// notificationRow inserts the notification id of the world's project for its receiver, of a change of an event of
// its own, deleted at deleted when it is not nil.
func notificationRow(t *testing.T, pool *pgxpool.Pool, w world, id uuid.UUID, deleted *time.Time) {
	t.Helper()
	exec(t, pool, `INSERT INTO notifications (id, workspace_id, project_id, receiver_id, issue_id, sender, data, event_id, change_seq, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, gen_random_uuid(), 'subscribed', '{"issue": {}, "change": {}}', gen_random_uuid(), 0, $5, $5, $6)`,
		id, w.workspace, w.project, w.receiver, now.Add(-90*24*time.Hour), deleted)
}

// The purge deletes, a bounded page at a time after its cursor, the notifications deleted before the cutoff, each
// page committed on the pool whatever the caller's transaction does; the notification deleted at the cutoff, a recent
// one and a live one stay. The module's purger binds the table. Dropping the cursor, the limit, the pool or using <=
// fails here.
func TestPurgeNotifications(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	cutoff := now.Add(-60 * 24 * time.Hour)
	old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
	for _, row := range []struct {
		n  int
		at *time.Time
	}{{30, &older}, {10, &old}, {20, &cutoff}, {40, &recent}, {50, nil}, {60, &old}} {
		notificationRow(t, pool, w, purgeID(row.n), row.at)
	}
	first, err := s.PurgeNotifications(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
	if err != nil || first.Scanned != 1 || first.Deleted != 1 || first.After != purgeID(10) {
		t.Fatalf("first batch=%+v,%v; want the lowest eligible id only", first, err)
	}
	skipped, err := s.PurgeNotifications(ctx, app.PurgeBatch{Cutoff: cutoff, After: purgeID(60), Limit: 1})
	if err != nil || skipped.Scanned != 0 || skipped.Deleted != 0 || skipped.After != purgeID(60) {
		t.Fatalf("after the greatest=%+v,%v; want empty", skipped, err)
	}
	rollback := errors.New("rollback the caller")
	err = postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		page, err := s.PurgeNotifications(txctx, app.PurgeBatch{Cutoff: cutoff, After: purgeID(10), Limit: 1})
		if err != nil || page.Scanned != 1 || page.Deleted != 1 || page.After != purgeID(30) {
			t.Fatalf("page in a caller's transaction=%+v,%v; want 30", page, err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	purger := notification.Provide(pool).Purger
	if !slices.Equal(purger.Tables(), []string{"notifications"}) || len(purger.ProtectedTables()) != 0 {
		t.Fatalf("directory=%v protected=%v; want notifications alone, unprotected", purger.Tables(), purger.ProtectedTables())
	}
	deleted := 0
	cursor := notification.PurgeCursor{}
	for {
		p, err := purger.Page(ctx, cutoff, 1, cursor)
		if err != nil {
			t.Fatal(err)
		}
		deleted += p.Deleted
		if p.Done {
			break
		}
		cursor = p.Cursor
	}
	if deleted != 1 {
		t.Fatalf("module purge deleted %d; want the remaining one, the caller's rollback undoing no page", deleted)
	}
	var left []uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT coalesce(array_agg(id ORDER BY id),'{}') FROM notifications").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []uuid.UUID{purgeID(20), purgeID(40), purgeID(50)}) {
		t.Fatalf("boundary/recent/live=%v; want 20,40,50", left)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := postgresadapter.New(nil).PurgeNotifications(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
			t.Fatalf("accepted invalid batch %d", limit)
		}
	}
}
