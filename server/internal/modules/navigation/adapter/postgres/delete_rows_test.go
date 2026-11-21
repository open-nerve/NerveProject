package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

var (
	_ app.ViewDeletions     = (*postgresadapter.Store)(nil)
	_ app.FavoriteDeletions = (*postgresadapter.Store)(nil)
)

// stamps reads each row's deletion of table as text, in the order of ids: its deleted_at, updated_at and
// updated_by_id.
func stamps(t *testing.T, pool *pgxpool.Pool, table string, ids ...uuid.UUID) []string {
	t.Helper()
	out := make([]string, len(ids))
	for i, id := range ids {
		if err := pool.QueryRow(context.Background(), "SELECT concat_ws(',', deleted_at, updated_at, updated_by_id) FROM "+table+" WHERE id = $1", id).Scan(&out[i]); err != nil {
			t.Fatal(err)
		}
	}
	return out
}

// The deletion of a deleted parent's views soft-deletes them a batch at a time after the cursor, in id order, at the
// deletion's moment by its deleter, and the batch answers their ids (P3b spec 2.7). A project's batch takes its
// project's views alone; a workspace's, the workspace's own, those without a project, and none of its projects'. A
// view deleted before, another project's and another workspace's stay as they were.
func TestDeleteViewsOfADeletedParent(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mine   bool // the deleted parent's views are its project's, not its workspace's own
		delete func(*postgresadapter.Store, context.Context, app.RowBatch) ([]uuid.UUID, error)
	}{
		{"project", true, (*postgresadapter.Store).DeleteProjectViews},
		{"workspace", false, (*postgresadapter.Store).DeleteWorkspaceViews},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newViews(t, pool)
			parent := w.project
			if !tt.mine {
				parent = w.workspace
			}
			for _, n := range []int{5, 4, 3, 2, 1} {
				viewRow(t, pool, w.world, numberedID(n), tt.mine, nil)
			}
			before := now.Add(-time.Hour)
			exec(t, pool, "UPDATE issue_views SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", numberedID(2), before, w.other)
			// The other kind under the same workspace, another project's and another workspace's of either kind.
			viewRow(t, pool, w.world, numberedID(6), !tt.mine, nil)
			viewRow(t, pool, world{owner: w.owner, workspace: w.workspace, project: w.ops}, numberedID(7), true, nil)
			elsewhere := world{owner: w.owner, workspace: w.elsewhere, project: w.elsewhereProject}
			viewRow(t, pool, elsewhere, numberedID(8), true, nil)
			viewRow(t, pool, elsewhere, numberedID(9), false, nil)
			// Each live view's last writer its owner, as its creation leaves it: the deletion must stamp its deleter.
			exec(t, pool, "UPDATE issue_views SET updated_by_id = owned_by_id WHERE deleted_at IS NULL")
			kept := []uuid.UUID{numberedID(2), numberedID(6), numberedID(7), numberedID(8), numberedID(9)}
			unchanged := stamps(t, pool, "issue_views", kept...)
			at := now.Add(2 * time.Hour)
			batch := func(after uuid.UUID, want ...uuid.UUID) {
				t.Helper()
				got, err := tt.delete(s, context.Background(), app.RowBatch{ParentID: parent, After: after, By: w.other, At: at, Limit: 2})
				slices.SortFunc(got, uuid.UUID.Compare)
				if err != nil || !slices.Equal(got, want) {
					t.Fatalf("batch after %s=%v,%v; want %v", after, got, err, want)
				}
			}
			batch(uuid.UUID{}, numberedID(1), numberedID(3))
			batch(numberedID(3), numberedID(4), numberedID(5))
			batch(numberedID(5))
			var stamped int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM issue_views WHERE id = ANY($1) AND deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3",
				[]uuid.UUID{numberedID(1), numberedID(3), numberedID(4), numberedID(5)}, at, w.other).Scan(&stamped); err != nil || stamped != 4 {
				t.Fatalf("the parent's views deleted at %s by %s: %d of 4, %v", at, w.other, stamped, err)
			}
			if got := stamps(t, pool, "issue_views", kept...); !slices.Equal(got, unchanged) {
				t.Fatalf("deleted before, the other kind's, another project's and another workspace's=%v; want %v", got, unchanged)
			}
			for _, limit := range []int{0, 2147483648} {
				if _, err := tt.delete(s, context.Background(), app.RowBatch{ParentID: parent, By: w.other, At: at, Limit: limit}); err == nil {
					t.Fatalf("accepted the batch %d", limit)
				}
			}
		})
	}
}

// liveFavorite inserts user's live favorite id in workspace, his last writing, as a creation leaves it: of kind
// "folder", his folder named for its id; of kind "module", of a new module of project; of kind "view", of a new view,
// project's or, project nil, the workspace's; in the folder parent when it is not nil.
func liveFavorite(t *testing.T, pool *pgxpool.Pool, kind string, id, workspace uuid.UUID, project *uuid.UUID, user uuid.UUID, parent *uuid.UUID) {
	t.Helper()
	var name *string
	target := ptr(uuid.NewV7())
	if kind == "folder" {
		name, target = ptr("Folder "+id.String()), nil
	}
	exec(t, pool, `INSERT INTO user_favorites (id, workspace_id, project_id, user_id, entity_type, entity_identifier, name, parent_id, sequence,
			created_by_id, updated_by_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 65535, $4, $4, $9, $9)`, id, workspace, project, user, kind, target, name, parent, now.Add(-time.Hour))
}

// The deletion of a deleted parent's favorites soft-deletes every account's a batch at a time after the cursor, in id
// order, at the deletion's moment by its deleter, and the batch answers their ids (P3c spec 2.6, R11). A project's
// batch takes the favorites of the project (of its module here), a folder's as well; a workspace's are its own, those
// without a project, its folders' batch and its other favorites' (a workspace view's here), the folders first (spec
// 2.8), and none of its projects'. Under the same workspace, each batch leaves the other two kinds, and every batch
// leaves a favorite deleted before, another project's and another workspace's as they were.
func TestDeleteFavoritesOfADeletedParent(t *testing.T) {
	for _, tt := range []struct {
		name, kind string // the deleted parent's favorites are of kind
		delete     func(*postgresadapter.Store, context.Context, app.RowBatch) ([]uuid.UUID, error)
	}{
		{"project", "module", (*postgresadapter.Store).DeleteProjectFavorites},
		{"workspace folders", "folder", (*postgresadapter.Store).DeleteWorkspaceFavoriteFolders},
		{"workspace favorite items", "view", (*postgresadapter.Store).DeleteWorkspaceFavoriteItems},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newViews(t, pool)
			deleter := uuid.NewV7()
			exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cid@corp.com', 'x', 'cid')", deleter)
			projectOf := func(kind string) *uuid.UUID {
				if kind == "module" {
					return &w.project
				}
				return nil
			}
			parent := w.workspace
			if tt.kind == "module" {
				parent = w.project
			}
			// The workspace's folder that holds the other favorites: 1, the deleted parent's first, or 11, kept.
			folder := numberedID(11)
			if tt.kind == "folder" {
				folder = numberedID(1)
			} else {
				liveFavorite(t, pool, "folder", folder, w.workspace, nil, w.owner, nil)
			}
			// The deleted parent's, written in the reverse of their id order, bob's and ann's; 1 and 3 in the folder.
			for _, n := range []int{5, 4, 3, 2, 1} {
				user, in := w.owner, &folder
				if n%2 == 0 {
					user = w.other
				}
				if tt.kind == "folder" || n == 2 || n > 3 {
					in = nil
				}
				liveFavorite(t, pool, tt.kind, numberedID(n), w.workspace, projectOf(tt.kind), user, in)
			}
			exec(t, pool, "UPDATE user_favorites SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", numberedID(2), now.Add(-time.Minute), w.other)
			// The other kinds under the same workspace, in the folder; another project's; another workspace's of each kind.
			for i, kind := range []string{"module", "view"} {
				if kind != tt.kind {
					liveFavorite(t, pool, kind, numberedID(12+i), w.workspace, projectOf(kind), w.owner, &folder)
				}
			}
			liveFavorite(t, pool, "view", numberedID(6), w.workspace, &w.ops, w.owner, &folder)
			liveFavorite(t, pool, "folder", numberedID(7), w.elsewhere, nil, w.owner, nil)
			liveFavorite(t, pool, "view", numberedID(8), w.elsewhere, nil, w.owner, ptr(numberedID(7)))
			liveFavorite(t, pool, "module", numberedID(9), w.elsewhere, &w.elsewhereProject, w.owner, nil)
			mine := []uuid.UUID{numberedID(1), numberedID(3), numberedID(4), numberedID(5)}
			var kept []uuid.UUID
			if err := pool.QueryRow(context.Background(), "SELECT array_agg(id ORDER BY id) FROM user_favorites WHERE NOT id = ANY($1)", mine).Scan(&kept); err != nil || len(kept) != 7 {
				t.Fatalf("the kept favorites %v, %v; want 2, 6 to 9 and the two other kinds under the workspace", kept, err)
			}
			unchanged := stamps(t, pool, "user_favorites", kept...)
			at := now.Add(2 * time.Hour)
			batch := func(after uuid.UUID, want ...uuid.UUID) {
				t.Helper()
				got, err := tt.delete(s, context.Background(), app.RowBatch{ParentID: parent, After: after, By: deleter, At: at, Limit: 2})
				slices.SortFunc(got, uuid.UUID.Compare)
				if err != nil || !slices.Equal(got, want) {
					t.Fatalf("batch after %s=%v,%v; want %v", after, got, err, want)
				}
			}
			batch(uuid.UUID{}, numberedID(1), numberedID(3))
			batch(numberedID(3), numberedID(4), numberedID(5))
			batch(numberedID(5))
			var stamped int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM user_favorites WHERE id = ANY($1) AND deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3",
				mine, at, deleter).Scan(&stamped); err != nil || stamped != 4 {
				t.Fatalf("the parent's favorites deleted at %s by %s: %d of 4, %v", at, deleter, stamped, err)
			}
			if got := stamps(t, pool, "user_favorites", kept...); !slices.Equal(got, unchanged) {
				t.Fatalf("deleted before, the other kinds', another project's and another workspace's=%v; want %v", got, unchanged)
			}
			for _, limit := range []int{0, 2147483648} {
				if _, err := tt.delete(s, context.Background(), app.RowBatch{ParentID: parent, By: deleter, At: at, Limit: limit}); err == nil {
					t.Fatalf("accepted the batch %d", limit)
				}
			}
		})
	}
}

// Each deletion locks its rows in id order before it writes them (M3 design 3.6, convention 5; P3c spec 2.8): with
// the rows written in the reverse of their id order and the middle one held, each waits on it holding the lower one
// and not yet the higher one. A statement that locked in its scan's order would hold the higher one.
func TestDeletionsLockInIDOrder(t *testing.T) {
	view := func(project bool) func(*testing.T, *pgxpool.Pool, world, uuid.UUID) {
		return func(t *testing.T, pool *pgxpool.Pool, w world, id uuid.UUID) { viewRow(t, pool, w, id, project, nil) }
	}
	favorite := func(kind string) func(*testing.T, *pgxpool.Pool, world, uuid.UUID) {
		return func(t *testing.T, pool *pgxpool.Pool, w world, id uuid.UUID) {
			var project *uuid.UUID
			if kind == "module" {
				project = &w.project
			}
			liveFavorite(t, pool, kind, id, w.workspace, project, w.owner, nil)
		}
	}
	for _, tt := range []struct {
		name, table string
		mine        bool // the deleted parent is the project, not the workspace
		row         func(*testing.T, *pgxpool.Pool, world, uuid.UUID)
		delete      func(*postgresadapter.Store, context.Context, app.RowBatch) ([]uuid.UUID, error)
	}{
		{"project", "issue_views", true, view(true), (*postgresadapter.Store).DeleteProjectViews},
		{"workspace", "issue_views", false, view(false), (*postgresadapter.Store).DeleteWorkspaceViews},
		{"project favorites", "user_favorites", true, favorite("module"), (*postgresadapter.Store).DeleteProjectFavorites},
		{"workspace folders", "user_favorites", false, favorite("folder"), (*postgresadapter.Store).DeleteWorkspaceFavoriteFolders},
		{"workspace favorite items", "user_favorites", false, favorite("view"), (*postgresadapter.Store).DeleteWorkspaceFavoriteItems},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newWorld(t, pool)
			parent := w.project
			if !tt.mine {
				parent = w.workspace
			}
			for _, n := range []int{3, 2, 1} {
				tt.row(t, pool, w, numberedID(n))
			}
			holder, err := pool.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = holder.Rollback(context.Background()) }()
			if _, err := holder.Exec(context.Background(), "SELECT 1 FROM "+tt.table+" WHERE id = $1 FOR UPDATE", numberedID(2)); err != nil {
				t.Fatal(err)
			}
			done := make(chan error, 1)
			go func() {
				done <- postgres.NewTxManager(pool, 10*time.Second).WithinTx(context.Background(), func(ctx context.Context) error {
					_, err := tt.delete(s, ctx, app.RowBatch{ParentID: parent, By: w.owner, At: now, Limit: 10})
					return err
				})
			}()
			pgtest.WaitForLockWaitBehind(t, pool, holder.Conn().PgConn(), 10*time.Second)
			if !pgtest.RowWaits(t, pool, tt.table, numberedID(1), "FOR UPDATE") || pgtest.RowWaits(t, pool, tt.table, numberedID(3), "FOR UPDATE") {
				t.Fatal("the deletion waits on the middle row without holding the lowest id alone; want the rows locked in id order")
			}
			if err := holder.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			var live int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM "+tt.table+" WHERE deleted_at IS NULL").Scan(&live); err != nil || live != 0 {
				t.Fatalf("live rows after the deletion=%d,%v; want none", live, err)
			}
		})
	}
}
