package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type listScaleProject struct {
	ID             uuid.UUID
	States, Labels []uuid.UUID
	Roots          int
}
type listScaleFixture struct {
	w           issueWorld
	owner       *pgxpool.Pool
	projects    []listScaleProject
	rows        []listScaleRow
	SeedSeconds float64
}

// v1: N normal roots in Web, 200 in each of four scope neighbours. Per 100
// roots add triage/archive/deleted roots and four children (own/other/archive/deleted).
// SQL creates the data; the oracle independently enumerates rows in Go.
func seedListScale(t *testing.T, n int, roles splitRoles, pool *pgxpool.Pool) listScaleFixture {
	t.Helper()
	w := newIssueWorldOn(t, roles.serverURL, pool, func(*config.Config) {})
	f := listScaleFixture{w: w, owner: roles.owner}
	for p, name := range []string{"Web", "Visible", "Own", "Exited", "Archived"} {
		id := w.web
		if p != 0 {
			id = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, fmt.Sprintf("P%d", p))
		}
		w.makeMember(t, "walt", id, shared.RoleMember)
		if p != 0 {
			w.makeMember(t, "bob", id, shared.RoleMember)
		}
		project := listScaleProject{ID: id, Roots: 200}
		if p == 0 {
			project.Roots = n
		}
		for _, state := range []string{"Backlog", "Todo", "In Progress", "Done", "Cancelled", "Triage"} {
			project.States = append(project.States, stateID(t, pool, id, state))
		}
		for _, label := range []string{"Alpha", "Charlie"} {
			var made struct{ ID uuid.UUID }
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+id.String()+"/labels", fmt.Sprintf(`{"name":%q}`, label), 201), &made)
			project.Labels = append(project.Labels, made.ID)
		}
		f.projects = append(f.projects, project)
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	for p, project := range f.projects {
		f.seedProject(t, ctx, p, project)
	}
	// End/downgrade memberships after assignment, retaining historical assignees.
	pm := queryIDs(t, pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", f.projects[2].ID, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+pm.String(), `{"role":5}`, 200)
	w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+f.projects[3].ID.String()+"/leave", "", 204)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+f.projects[4].ID.String()+"/archive", "", 200)
	if _, err := roles.owner.Exec(ctx, "ANALYZE issues, issue_labels, issue_assignees, issue_subscribers, issue_mentions, projects, project_members, states, labels"); err != nil {
		t.Fatal(err)
	}
	f.SeedSeconds = time.Since(start).Seconds()
	f.rows = f.expectedRows()
	return f
}

func (f listScaleFixture) seedProject(t *testing.T, ctx context.Context, p int, project listScaleProject) {
	t.Helper()
	tx, err := f.owner.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	statements := []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE list_seed ON COMMIT DROP AS
 WITH numbers AS (
 SELECT n,n+1 AS seq,0 AS kind FROM generate_series(0,$1::int-1) n
 UNION ALL SELECT k*100,$1::int+k*7+kind,kind FROM generate_series(0,$1::int/100-1) k CROSS JOIN generate_series(1,7) kind)
 SELECT *,('00000000-0000-7000-8000-'||lpad(to_hex(($2::bigint+1)*1000000-seq),12,'0'))::uuid AS id,
 CASE WHEN kind>=4 THEN ('00000000-0000-7000-8000-'||lpad(to_hex(($2::bigint+1)*1000000-(n+1)),12,'0'))::uuid END AS parent
 FROM numbers`, []any{project.Roots, p}},
		{`INSERT INTO issues (id,workspace_id,project_id,state_id,name,sequence_id,parent_id,priority,created_by_id,updated_by_id,
 created_at,updated_at,start_date,target_date,sort_order,completed_at,archived_at,deleted_at)
 SELECT id,$1,$2,($3::uuid[])[CASE WHEN kind=1 THEN 6 ELSE (n/5)%5+1 END],'Scale '||seq,seq,parent,
 (ARRAY['urgent','high','medium','low','none'])[n%5+1],
 CASE WHEN kind=4 THEN $6::uuid WHEN kind=5 THEN $4::uuid ELSE (ARRAY[$4::uuid,$5::uuid,$6::uuid,NULL])[n%4+1] END,$4,
 '2026-01-01Z'::timestamptz+(n/4)*interval '1 microsecond',
 '2026-01-02Z'::timestamptz+(n%17)*interval '1 microsecond',
 CASE WHEN n%3<>0 THEN '2026-01-01'::date+n%11 END,
 CASE WHEN n%4<>0 THEN '2026-01-21'::date+n%7 END,1000+n%8,
 CASE WHEN (n/5)%5=3 AND kind<>1 THEN '2026-01-03Z'::timestamptz END,
 CASE WHEN kind IN (2,6) THEN '2026-02-01Z'::timestamptz END,
 CASE WHEN kind IN (3,7) THEN '2026-02-02Z'::timestamptz END FROM list_seed`, []any{f.w.acme, project.ID, project.States, f.w.ids["alice"], f.w.ids["bob"], f.w.ids["walt"]}},
		{`UPDATE projects SET last_issue_sequence=$2 WHERE id=$1`, []any{project.ID, project.Roots + 7*(project.Roots/100)}},
		{`INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,created_by_id)
 SELECT uuidv7(),$1,$2,id,($3::uuid[])[v],$4 FROM list_seed CROSS JOIN generate_series(1,2) v
 WHERE (n/25)%4=3 OR (n/25)%4=v`, []any{f.w.acme, project.ID, project.Labels, f.w.ids["alice"]}},
		{`INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,created_by_id)
 SELECT uuidv7(),$1,$2,id,($3::uuid[])[v],$4 FROM list_seed CROSS JOIN generate_series(1,2) v
 WHERE (n/100)%4=3 OR (n/100)%4=v`, []any{f.w.acme, project.ID, []uuid.UUID{f.w.ids["alice"], f.w.ids["bob"]}, f.w.ids["alice"]}},
		{`INSERT INTO issue_subscribers(id,workspace_id,project_id,issue_id,subscriber_id,created_by_id)
 SELECT uuidv7(),$1,$2,id,($3::uuid[])[v],$4 FROM list_seed CROSS JOIN generate_series(1,2) v
 WHERE (n/7)%4=3 OR (n/7)%4=v`, []any{f.w.acme, project.ID, []uuid.UUID{f.w.ids["alice"], f.w.ids["bob"]}, f.w.ids["alice"]}},
		{`INSERT INTO issue_mentions(id,workspace_id,project_id,issue_id,mention_id,created_by_id)
 SELECT uuidv7(),$1,$2,id,$3,$3 FROM list_seed WHERE n%13=0`, []any{f.w.acme, project.ID, f.w.ids["bob"]}},
		{`INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,deleted_at)
 SELECT uuidv7(),$1,$2,id,$3,'2026-02-02Z' FROM list_seed WHERE kind=0 AND n%100=0`, []any{f.w.acme, project.ID, project.Labels[1]}},
	}
	for _, s := range statements {
		if _, err := tx.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
}

func scaleID(p, seq int) uuid.UUID {
	return uuid.MustParse(fmt.Sprintf("00000000-0000-7000-8000-%012x", (p+1)*1000000-seq))
}

// Aggregate independent relation tables separately, avoiding join multiplication.
func (f listScaleFixture) checkDistribution(t *testing.T) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	got := map[string]map[string]int64{}
	queries := map[string]string{
		"scope":       `SELECT p.id::text||'/'||(p.archived_at IS NOT NULL)::text||'/'||pm.role::text||'/'||pm.is_active::text||'/'||p.guest_view_all_features::text,count(*) FROM projects p JOIN project_members pm ON pm.project_id=p.id WHERE pm.member_id=$1 AND pm.deleted_at IS NULL GROUP BY 1`,
		"rows":        `SELECT i.project_id::text||'/'||COALESCE(s."group",'')||'/'||priority||'/'||(parent_id IS NOT NULL)::text||'/'||(i.archived_at IS NOT NULL)::text||'/'||(i.deleted_at IS NOT NULL)::text||'/'||(start_date IS NULL)::text||'/'||(target_date IS NULL)::text,count(*) FROM issues i JOIN states s ON s.id=i.state_id GROUP BY 1`,
		"degrees":     `SELECT project_id::text||'/'||l::text||'/'||a::text,count(*) FROM (SELECT i.project_id,(SELECT count(*) FROM issue_labels l WHERE l.issue_id=i.id AND l.deleted_at IS NULL) l,(SELECT count(*) FROM issue_assignees a WHERE a.issue_id=i.id AND a.deleted_at IS NULL) a FROM issues i) q GROUP BY 1`,
		"creators":    `SELECT project_id::text||'/'||COALESCE(created_by_id::text,'none'),count(*) FROM issues GROUP BY 1`,
		"subscribers": `SELECT project_id::text||'/'||subscriber_id::text,count(*) FROM issue_subscribers WHERE deleted_at IS NULL GROUP BY 1`,
		"mentions":    `SELECT project_id::text||'/'||mention_id::text,count(*) FROM issue_mentions WHERE deleted_at IS NULL GROUP BY 1`,
		"relations":   `SELECT 'labels/'||(deleted_at IS NOT NULL)::text,count(*) FROM issue_labels GROUP BY 1 UNION ALL SELECT 'assignees/'||(deleted_at IS NOT NULL)::text,count(*) FROM issue_assignees GROUP BY 1`,
	}
	for name, sql := range queries {
		counts := map[string]int64{}
		var args []any
		if name == "scope" {
			args = []any{f.w.ids["walt"]}
		}
		rows, err := f.owner.Query(ctx, sql, args...)
		if err != nil {
			t.Fatal(err)
		}
		var key string
		var n int64
		_, err = pgx.ForEachRow(rows, []any{&key, &n}, func() error { counts[key] = n; return nil })
		if err != nil {
			t.Fatal(err)
		}
		got[name] = counts
	}
	want := f.expectedDistribution()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("seed distribution differs: got=%v want=%v", got, want)
	}
	return map[string]any{"actual": got, "expected": want}
}
