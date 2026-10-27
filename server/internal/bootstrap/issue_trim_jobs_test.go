package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// One transaction publishes the full version fixture together with exclusions,
// so the real periodic worker cannot trim an intermediate set of rows.
func seedTrimVersions(t *testing.T, pool *pgxpool.Pool, item uuid.UUID, exclude string) []uuid.UUID {
	t.Helper()
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	if _, err := tx.Exec(pgtest.Soon(t), "DELETE FROM issue_description_versions WHERE issue_id=$1", item); err != nil {
		t.Fatal(err)
	}
	var ids []uuid.UUID
	for i := 0; i < 23; i++ {
		id := uuid.NewV7()
		ids = append(ids, id)
		if _, err := tx.Exec(pgtest.Soon(t), `INSERT INTO issue_description_versions(id,workspace_id,project_id,issue_id,owned_by_id,last_saved_at,created_at,updated_at,created_by_id,updated_by_id)
 SELECT $1,workspace_id,project_id,id,created_by_id,$3,$4,$4,created_by_id,created_by_id FROM issues WHERE id=$2`, id, item, time.Date(2020, 1, 1, 0, i, 0, 0, time.UTC), time.Date(2020, 1, 2, 0, 0, 0, 0, time.UTC)); err != nil {
			t.Fatal(err)
		}
	}
	var statements []string
	switch exclude {
	case "archived":
		statements = []string{"UPDATE projects SET archived_at=now() WHERE id=(SELECT project_id FROM issues WHERE id=$1)", "UPDATE issues SET archived_at=now() WHERE id=$1"}
	case "no members":
		statements = []string{"UPDATE project_members SET is_active=false WHERE project_id=(SELECT project_id FROM issues WHERE id=$1)"}
	case "deleted project":
		statements = []string{"UPDATE projects SET deleted_at=now() WHERE id=(SELECT project_id FROM issues WHERE id=$1)"}
	case "deleted issue":
		statements = []string{"UPDATE issues SET deleted_at=now() WHERE id=$1"}
	}
	for _, sql := range statements {
		if _, err := tx.Exec(pgtest.Soon(t), sql, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	return ids
}

func trimVersionIDs(t *testing.T, pool *pgxpool.Pool, item uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), "SELECT id FROM issue_description_versions WHERE issue_id=$1 ORDER BY id", item)
	if err != nil {
		t.Fatal(err)
	}
	ids, err := pgx.CollectRows(rows, pgx.RowTo[uuid.UUID])
	if err != nil {
		t.Fatal(err)
	}
	return ids
}

func awaitTrimVersions(t *testing.T, pool *pgxpool.Pool, want map[uuid.UUID][]uuid.UUID) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	tick := time.NewTicker(25 * time.Millisecond)
	defer tick.Stop()
	for {
		ready := true
		for id, ids := range want {
			ready = ready && slices.Equal(trimVersionIDs(t, pool, id), ids)
		}
		if ready {
			return
		}
		select {
		case <-ctx.Done():
			t.Fatal("periodic trim did not leave the exact version IDs", ctx.Err())
		case <-tick.C:
		}
	}
}

func TestTrimVersionsJobRuns(t *testing.T) {
	w := newIssueWorldWith(t, func(cfg *config.Config) { cfg.Issue.TrimVersionsInterval = time.Second })
	want := map[uuid.UUID][]uuid.UUID{}
	for i, exclude := range []string{"no members", "archived", "deleted project", "deleted issue"} {
		project := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", exclude, "RET"+string(rune('A'+i)))
		item := w.newWorkItemIn(t, project, `{"name":"Retain"}`)
		ids := seedTrimVersions(t, w.pool, item, exclude)
		if exclude == "no members" || exclude == "archived" {
			ids = ids[3:]
		}
		want[item] = ids
	}
	activities := queryIDs(t, w.pool, "SELECT id FROM issue_activities ORDER BY id")
	awaitTrimVersions(t, w.pool, want)
	if after := queryIDs(t, w.pool, "SELECT id FROM issue_activities ORDER BY id"); !reflect.DeepEqual(after, activities) {
		t.Fatal("retention wrote activities")
	}
	var wrong int
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE kind='issue.trim_description_versions' AND queue<>'periodic'").Scan(&wrong); err != nil || wrong != 0 {
		t.Fatal(wrong, err)
	}
	waitForCompletedKinds(t, w.pool, map[string]int{"issue.trim_description_versions": 1}, &lockedBuffer{})
	// The normal route still runs against the same live app after retention.
	w.send(t, "alice", http.MethodGet, "/api/v0/projects/"+w.web.String()+"/issues", "", 200)
}
