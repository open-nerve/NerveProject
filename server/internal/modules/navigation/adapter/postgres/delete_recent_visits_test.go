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
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

var _ app.RecentVisitDeletions = (*postgresadapter.Store)(nil)

// liveVisit inserts user's live recent visit id in workspace, his last writing, as a recording leaves it: of kind
// "project", of project itself; of kind "issue", of a new work item of project.
func liveVisit(t *testing.T, pool *pgxpool.Pool, kind string, id, workspace, project, user uuid.UUID) {
	t.Helper()
	target := uuid.NewV7()
	if kind == "project" {
		target = project
	}
	visited := now.Add(-time.Hour)
	seedVisit(t, pool, visitRow{id, workspace, project, user, domain.RecentVisitKind(kind), target, visited, visited, nil})
}

// The deletion of a deleted project's recent visits soft-deletes every account's under it, of the project and of its
// work items, a batch at a time after the cursor, in id order, at the deletion's moment by its deleter, and the batch
// answers their ids (P3d spec 2.6). A visit deleted before, as a work item's deletion deletes its visits at its own
// moment by its own deleter, keeps that deletion (re-92 I3 ⑤), and another project's and another workspace's stay as
// they were.
func TestDeleteRecentVisitsOfADeletedProject(t *testing.T) {
	s, pool := newStore(t)
	w := newViews(t, pool)
	deleter := uuid.NewV7()
	exec(t, pool, "INSERT INTO users (id, email, password, display_name) VALUES ($1, 'cid@corp.com', 'x', 'cid')", deleter)
	// The deleted project's, written in the reverse of their id order, bob's and ann's, of the project (1, 3) and of
	// its work items (2, 4, 5); 2 deleted before, by the deletion of its work item.
	for _, row := range []struct {
		n    int
		kind string
		user uuid.UUID
	}{{5, "issue", w.owner}, {4, "issue", w.other}, {3, "project", w.owner}, {2, "issue", w.other}, {1, "project", w.other}} {
		liveVisit(t, pool, row.kind, numberedID(row.n), w.workspace, w.project, row.user)
	}
	exec(t, pool, "UPDATE user_recent_visits SET deleted_at = $2, updated_at = $2, updated_by_id = $3 WHERE id = $1", numberedID(2), now.Add(-time.Minute), w.owner)
	// Another project's of either kind, another workspace's of either kind.
	liveVisit(t, pool, "issue", numberedID(6), w.workspace, w.ops, w.owner)
	liveVisit(t, pool, "project", numberedID(7), w.workspace, w.ops, w.owner)
	liveVisit(t, pool, "issue", numberedID(8), w.elsewhere, w.elsewhereProject, w.owner)
	liveVisit(t, pool, "project", numberedID(9), w.elsewhere, w.elsewhereProject, w.owner)
	mine := []uuid.UUID{numberedID(1), numberedID(3), numberedID(4), numberedID(5)}
	kept := []uuid.UUID{numberedID(2), numberedID(6), numberedID(7), numberedID(8), numberedID(9)}
	unchanged := stamps(t, pool, "user_recent_visits", kept...)
	at := now.Add(2 * time.Hour)
	batch := func(after uuid.UUID, want ...uuid.UUID) {
		t.Helper()
		got, err := s.DeleteProjectRecentVisits(context.Background(), app.RowBatch{ParentID: w.project, After: after, By: deleter, At: at, Limit: 2})
		slices.SortFunc(got, uuid.UUID.Compare)
		if err != nil || !slices.Equal(got, want) {
			t.Fatalf("batch after %s=%v,%v; want %v", after, got, err, want)
		}
	}
	batch(uuid.UUID{}, numberedID(1), numberedID(3))
	batch(numberedID(3), numberedID(4), numberedID(5))
	batch(numberedID(5))
	var stamped int
	if err := pool.QueryRow(context.Background(), `SELECT count(*) FROM user_recent_visits
		WHERE id = ANY($1) AND deleted_at = $2 AND updated_at = $2 AND updated_by_id = $3 AND visited_at = $4`,
		mine, at, deleter, now.Add(-time.Hour)).Scan(&stamped); err != nil || stamped != 4 {
		t.Fatalf("the project's visits deleted at %s by %s, their visits kept: %d of 4, %v", at, deleter, stamped, err)
	}
	if got := stamps(t, pool, "user_recent_visits", kept...); !slices.Equal(got, unchanged) {
		t.Fatalf("deleted before, another project's and another workspace's=%v; want %v", got, unchanged)
	}
	for _, limit := range []int{0, 2147483648} {
		if _, err := s.DeleteProjectRecentVisits(context.Background(), app.RowBatch{ParentID: w.project, By: deleter, At: at, Limit: limit}); err == nil {
			t.Fatalf("accepted the batch %d", limit)
		}
	}
}
