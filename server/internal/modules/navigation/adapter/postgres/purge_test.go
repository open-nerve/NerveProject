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

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

var now = time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)

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

func numberedID(n int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("0199a2b4-0000-7000-8000-%012x", n))
}

// world is an account, a workspace and its project, written directly: identity, workspace and project own them.
type world struct{ owner, workspace, project uuid.UUID }

func newWorld(t *testing.T, pool *pgxpool.Pool) world {
	t.Helper()
	w := world{uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'bob@corp.com', 'x', 'bob')", w.owner)
	exec(t, pool, "INSERT INTO workspaces (id, name, slug) VALUES ($1, 'Acme', 'acme')", w.workspace)
	exec(t, pool, "INSERT INTO projects (id, workspace_id, name, identifier) VALUES ($1, $2, 'Web', 'WEB')", w.project, w.workspace)
	return w
}

// viewRow inserts the view id of the world's owner, of its project or, when project is false, of its workspace,
// deleted at deleted when it is not nil.
func viewRow(t *testing.T, pool *pgxpool.Pool, w world, id uuid.UUID, project bool, deleted *time.Time) {
	t.Helper()
	var of *uuid.UUID
	if project {
		of = &w.project
	}
	exec(t, pool, `INSERT INTO issue_views (id, workspace_id, project_id, name, filters, display_filters, display_properties, owned_by_id, created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, 'Mine', '{}', '{"layout":"list","group_by":null,"sub_group_by":null,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true,"calendar":{"layout":"month","show_weekends":false}}',
			'{"assignee":true,"start_date":true,"due_date":true,"labels":true,"key":true,"priority":true,"state":true,"sub_issue_count":true,"link":true,"attachment_count":true,"created_on":true,"updated_on":true,"cycle":true,"modules":true}',
			$4, $5, $5, $6)`,
		id, w.workspace, of, w.owner, now.Add(-90*24*time.Hour), deleted)
}

// The purge deletes, a bounded page at a time after its cursor, the views of either kind deleted before the cutoff,
// each page committed on the pool whatever the caller's transaction does; the view deleted at the cutoff, a recent one
// and a live one stay. The module's purger binds the table. Dropping the cursor, the limit, the pool or using <=
// fails here.
func TestPurgeIssueViews(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	cutoff := now.Add(-60 * 24 * time.Hour)
	old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
	for _, row := range []struct {
		n       int
		project bool
		at      *time.Time
	}{{30, false, &older}, {10, true, &old}, {20, true, &cutoff}, {40, false, &recent}, {50, true, nil}, {60, false, &old}} {
		viewRow(t, pool, w, numberedID(row.n), row.project, row.at)
	}
	first, err := s.PurgeIssueViews(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
	if err != nil || first.Scanned != 1 || first.Deleted != 1 || first.After != numberedID(10) {
		t.Fatalf("first batch=%+v,%v; want the lowest eligible id only", first, err)
	}
	skipped, err := s.PurgeIssueViews(ctx, app.PurgeBatch{Cutoff: cutoff, After: numberedID(60), Limit: 1})
	if err != nil || skipped.Scanned != 0 || skipped.Deleted != 0 || skipped.After != numberedID(60) {
		t.Fatalf("after the greatest=%+v,%v; want empty", skipped, err)
	}
	rollback := errors.New("rollback the caller")
	err = postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		page, err := s.PurgeIssueViews(txctx, app.PurgeBatch{Cutoff: cutoff, After: numberedID(10), Limit: 1})
		if err != nil || page.Scanned != 1 || page.Deleted != 1 || page.After != numberedID(30) {
			t.Fatalf("page in a caller's transaction=%+v,%v; want 30", page, err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	purger := navigation.Provide(pool).Purger
	if !slices.Equal(purger.Tables(), []string{"issue_views", "user_favorites"}) || len(purger.ProtectedTables()) != 0 {
		t.Fatalf("directory=%v protected=%v; want issue_views then user_favorites, unprotected", purger.Tables(), purger.ProtectedTables())
	}
	deleted := 0
	cursor := navigation.PurgeCursor{}
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
	if err := pool.QueryRow(ctx, "SELECT coalesce(array_agg(id ORDER BY id),'{}') FROM issue_views").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []uuid.UUID{numberedID(20), numberedID(40), numberedID(50)}) {
		t.Fatalf("boundary/recent/live=%v; want 20,40,50", left)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := postgresadapter.New(nil).PurgeIssueViews(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
			t.Fatalf("accepted invalid batch %d", limit)
		}
	}
}

// favoriteRow inserts the world owner's favorite id: his folder named name when name is not empty, else his
// favorite of a module of the world's project in the folder parent when it is not nil; deleted at deleted when it is
// not nil.
func favoriteRow(t *testing.T, pool *pgxpool.Pool, w world, id uuid.UUID, name string, parent *uuid.UUID, deleted *time.Time) {
	t.Helper()
	created := now.Add(-90 * 24 * time.Hour)
	if name != "" {
		exec(t, pool, `INSERT INTO user_favorites (id, workspace_id, user_id, entity_type, name, sequence, created_at, updated_at, deleted_at)
			VALUES ($1, $2, $3, 'folder', $4, 65535, $5, $5, $6)`, id, w.workspace, w.owner, name, created, deleted)
		return
	}
	exec(t, pool, `INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, parent_id, sequence,
			created_at, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, 'module', gen_random_uuid(), $5, 65535, $6, $6, $7)`, id, w.workspace, w.project, w.owner, parent, created, deleted)
}

// The favorites' purge deletes, a bounded page at a time after its cursor, the favorites deleted before the cutoff,
// each page committed on the pool whatever the caller's transaction does; a folder's deletion takes the favorites in
// it through parent_id's CASCADE, uncounted, whichever comes first by id; the favorite deleted at the cutoff, a recent
// one and a live one stay. The module's purger runs it after the views. Dropping the cursor, the limit, the pool, the
// step or using <= fails here.
func TestPurgeUserFavorites(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	w := newWorld(t, pool)
	cutoff := now.Add(-60 * 24 * time.Hour)
	old, older, recent := now.Add(-61*24*time.Hour), now.Add(-62*24*time.Hour), now.Add(-59*24*time.Hour)
	// 30 and 60 are folders: 35 is in 30 after it by id, 55 in 60 before it.
	folder30, folder60 := numberedID(30), numberedID(60)
	for _, row := range []struct {
		n      int
		name   string
		parent *uuid.UUID
		at     *time.Time
	}{{30, "Thirty", nil, &older}, {10, "", nil, &old}, {20, "", nil, &cutoff}, {35, "", &folder30, &older}, {40, "", nil, &recent},
		{50, "", nil, nil}, {60, "Sixty", nil, &old}, {55, "", &folder60, &old}} {
		favoriteRow(t, pool, w, numberedID(row.n), row.name, row.parent, row.at)
	}
	first, err := s.PurgeUserFavorites(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: 1})
	if err != nil || first.Scanned != 1 || first.Deleted != 1 || first.After != numberedID(10) {
		t.Fatalf("first batch=%+v,%v; want the lowest eligible id only", first, err)
	}
	skipped, err := s.PurgeUserFavorites(ctx, app.PurgeBatch{Cutoff: cutoff, After: numberedID(60), Limit: 1})
	if err != nil || skipped.Scanned != 0 || skipped.Deleted != 0 || skipped.After != numberedID(60) {
		t.Fatalf("after the greatest=%+v,%v; want empty", skipped, err)
	}
	rollback := errors.New("rollback the caller")
	err = postgres.NewTxManager(pool, time.Second).WithinTx(ctx, func(txctx context.Context) error {
		page, err := s.PurgeUserFavorites(txctx, app.PurgeBatch{Cutoff: cutoff, After: numberedID(10), Limit: 1})
		if err != nil || page.Scanned != 1 || page.Deleted != 1 || page.After != folder30 {
			t.Fatalf("page in a caller's transaction=%+v,%v; want the folder 30", page, err)
		}
		return rollback
	})
	if !errors.Is(err, rollback) {
		t.Fatal(err)
	}
	purger := navigation.Provide(pool).Purger
	deleted := 0
	cursor := navigation.PurgeCursor{}
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
	if deleted != 2 {
		t.Fatalf("module purge deleted %d; want 55 then the folder 60, the folder 30 and 35 in it gone with the page the caller's rollback did not undo", deleted)
	}
	var left []uuid.UUID
	if err := pool.QueryRow(ctx, "SELECT coalesce(array_agg(id ORDER BY id),'{}') FROM user_favorites").Scan(&left); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(left, []uuid.UUID{numberedID(20), numberedID(40), numberedID(50)}) {
		t.Fatalf("boundary/recent/live=%v; want 20,40,50", left)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := postgresadapter.New(nil).PurgeUserFavorites(ctx, app.PurgeBatch{Cutoff: cutoff, Limit: limit}); err == nil {
			t.Fatalf("accepted invalid batch %d", limit)
		}
	}
}
