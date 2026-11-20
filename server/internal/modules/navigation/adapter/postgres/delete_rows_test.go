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

var _ app.ViewDeletions = (*postgresadapter.Store)(nil)

// stamps reads each view's deletion as text, in the order of ids: its deleted_at, updated_at and updated_by_id.
func stamps(t *testing.T, pool *pgxpool.Pool, ids ...uuid.UUID) []string {
	t.Helper()
	out := make([]string, len(ids))
	for i, id := range ids {
		if err := pool.QueryRow(context.Background(), "SELECT concat_ws(',', deleted_at, updated_at, updated_by_id) FROM issue_views WHERE id = $1", id).Scan(&out[i]); err != nil {
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
			unchanged := stamps(t, pool, kept...)
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
			if got := stamps(t, pool, kept...); !slices.Equal(got, unchanged) {
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

// Both deletions lock their rows in id order before they write them (M3 design 3.6, convention 5): with the rows
// written in the reverse of their id order and the middle one held, each waits on it holding the lower one and not
// yet the higher one. A statement that locked in its scan's order would hold the higher one.
func TestViewDeletionsLockInIDOrder(t *testing.T) {
	for _, tt := range []struct {
		name   string
		mine   bool
		delete func(*postgresadapter.Store, context.Context, app.RowBatch) ([]uuid.UUID, error)
	}{
		{"project", true, (*postgresadapter.Store).DeleteProjectViews},
		{"workspace", false, (*postgresadapter.Store).DeleteWorkspaceViews},
	} {
		t.Run(tt.name, func(t *testing.T) {
			s, pool := newStore(t)
			w := newWorld(t, pool)
			parent := w.project
			if !tt.mine {
				parent = w.workspace
			}
			for _, n := range []int{3, 2, 1} {
				viewRow(t, pool, w, numberedID(n), tt.mine, nil)
			}
			holder, err := pool.Begin(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = holder.Rollback(context.Background()) }()
			if _, err := holder.Exec(context.Background(), "SELECT 1 FROM issue_views WHERE id = $1 FOR UPDATE", numberedID(2)); err != nil {
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
			if !pgtest.RowWaits(t, pool, "issue_views", numberedID(1), "FOR UPDATE") || pgtest.RowWaits(t, pool, "issue_views", numberedID(3), "FOR UPDATE") {
				t.Fatal("the deletion waits on the middle row without holding the lowest id alone; want the rows locked in id order")
			}
			if err := holder.Rollback(context.Background()); err != nil {
				t.Fatal(err)
			}
			if err := <-done; err != nil {
				t.Fatal(err)
			}
			var live int
			if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM issue_views WHERE deleted_at IS NULL").Scan(&live); err != nil || live != 0 {
				t.Fatalf("live views after the deletion=%d,%v; want none", live, err)
			}
		})
	}
}
