package bootstrap

import (
	"context"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The deletions of a project and of a workspace at the work-item side's
// scale target (M4 design decision point 7, 3.16: a project of some 20,000
// work items), run only with NERVE_MEASURE_SCALE set: `make test` and CI
// skip it. Their durations are measured, not asserted; the test asserts
// its data only: the seed's rows, each deletion's 204, and no row left
// undeleted. P1b measured P1a's synchronous cascade with it (M4/P1b
// spec 3); M4 design decision point 7 has P9 measure its jobs on the same
// data. Run it alone, from server/, with a long timeout:
//
//	NERVE_MEASURE_SCALE=1 go test -count=1 ./internal/bootstrap -run TestTheDeletionsAtTheScaleTarget -v -timeout 60m
//
// The data: Web, acme's project, and five projects of a workspace beta,
// each with 20,000 work items in Backlog, each assigned to alice, labelled
// with its project's Bug, with alice and bob subscribed and 20 activities:
// 500,000 rows of the work-item side a project, 3,000,000 in all, the ids
// of each table interleaved across the six projects, as concurrent writes
// leave them. Request latency and remaining background duration are measured
// separately; the task identities and final row counts are asserted.
func TestTheDeletionsAtTheScaleTarget(t *testing.T) {
	if os.Getenv("NERVE_MEASURE_SCALE") == "" {
		t.Skip("NERVE_MEASURE_SCALE is not set: the measurement at the scale target runs only when asked for")
	}
	w := newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Server.RequestTimeout, cfg.Server.WriteTimeout, cfg.Database.CommitTimeout = time.Hour, time.Hour, time.Hour
		cfg.Auth.AccessTokenTTL = time.Hour
	})
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, http.StatusCreated)
	projects := []uuid.UUID{w.web}
	for _, name := range []string{"One", "Two", "Three", "Four", "Five"} {
		project := createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", name, strings.ToUpper(name))
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/labels", `{"name":"Bug"}`, http.StatusCreated)
		projects = append(projects, project)
	}
	var beta uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM workspaces WHERE slug=$1", "beta").Scan(&beta); err != nil {
		t.Fatal(err)
	}
	start := time.Now()
	w.seedAtScale(t, projects)
	seeded := w.workItemRows(t, projects, true)
	t.Logf("seeded six projects, %d rows of the work-item side, in %v", seeded, time.Since(start))
	if want := len(projects) * 500000; seeded != want {
		t.Fatalf("the seed wrote %d rows of the work-item side, want %d", seeded, want)
	}

	t.Logf("deleteProject of Web: the request %v", w.deletion(t, "/api/v0/projects/"+w.web.String()))
	t.Logf("deleteWorkspace of beta: the request %v", w.deletion(t, "/api/v0/workspaces/beta"))
	background := time.Now()
	for _, project := range projects {
		waitDeletionJobWithin(t, w.pool, projectDeletionJob(t, w.pool, project), "completed", time.Hour)
	}
	waitDeletionJobWithin(t, w.pool, parentDeletionJob(t, w.pool, "workspace_drafts", beta), "completed", time.Hour)
	t.Logf("deletion tasks after both requests: %v", time.Since(background))
	if n := w.workItemRows(t, projects, true); n != 0 {
		t.Errorf("%d rows of the six projects' work-item side left undeleted, want none", n)
	}
}

// seedAtScale writes TestTheDeletionsAtTheScaleTarget's rows into each of
// the projects, in one transaction, and analyzes the tables.
func (w issueWorld) seedAtScale(t *testing.T, projects []uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	alice, bob := w.ids["alice"], w.ids["bob"]
	for _, statement := range []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE scale_projects ON COMMIT DROP AS
			SELECT p.id AS project_id, p.workspace_id, s.id AS state_id, l.id AS label_id
			FROM projects p JOIN states s ON s.project_id = p.id AND s.name = 'Backlog' JOIN labels l ON l.project_id = p.id AND l.name = 'Bug'
			WHERE p.id = ANY ($1)`, []any{projects}},
		{`INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, state_id, 'Item ' || n, n, $1, $1
			FROM (SELECT p.*, n FROM scale_projects p, generate_series(1, 20000) n ORDER BY random()) s`, []any{alice}},
		{`UPDATE projects SET last_issue_sequence = 20000 WHERE id = ANY ($1)`, []any{projects}},
		{`INSERT INTO issue_assignees (id, workspace_id, project_id, issue_id, assignee_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, $2, $2, $2
			FROM (SELECT * FROM issues WHERE project_id = ANY ($1) ORDER BY random()) i`, []any{projects, alice}},
		{`INSERT INTO issue_labels (id, workspace_id, project_id, issue_id, label_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, label_id, $1, $1
			FROM (SELECT i.*, p.label_id FROM issues i JOIN scale_projects p USING (project_id) ORDER BY random()) i`, []any{alice}},
		{`INSERT INTO issue_subscribers (id, workspace_id, project_id, issue_id, subscriber_id, created_by_id, updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, subscriber, $2, $2
			FROM (SELECT i.*, subscriber FROM issues i, unnest(ARRAY[$2::uuid, $3::uuid]) subscriber WHERE i.project_id = ANY ($1)
				ORDER BY random()) i`, []any{projects, alice, bob}},
		{`INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, verb, field, new_value, comment, actor_id, created_by_id,
				updated_by_id)
			SELECT uuidv7(), workspace_id, project_id, id, 'updated', 'name', 'Item ' || k, 'updated the name to', $2, $2, $2
			FROM (SELECT i.*, k FROM issues i, generate_series(1, 20) k WHERE i.project_id = ANY ($1) ORDER BY random()) i`,
			[]any{projects, alice}},
	} {
		if _, err := tx.Exec(ctx, statement.sql, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	w.vacuum(t)
}

// deletion is how long alice's DELETE of path took, through the API, with
// no deadline of the client's.
func (w issueWorld) deletion(t *testing.T, path string) time.Duration {
	t.Helper()
	req := newRequest(t, http.MethodDelete, w.base+path, w.tokens["alice"], nil)
	start := time.Now()
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	_ = res.Body.Close()
	if res.StatusCode != http.StatusNoContent {
		t.Fatalf("DELETE %s = %d, want 204", path, res.StatusCode)
	}
	return took
}

// vacuum vacuums and analyzes the work-item side's tables.
func (w issueWorld) vacuum(t *testing.T) {
	t.Helper()
	if _, err := w.pool.Exec(context.Background(),
		"VACUUM ANALYZE issues, issue_assignees, issue_labels, issue_subscribers, issue_activities"); err != nil {
		t.Fatal(err)
	}
}
