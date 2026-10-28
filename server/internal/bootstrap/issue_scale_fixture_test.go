package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type scaleProject struct{ ID, Workspace uuid.UUID }
type scaleRows struct {
	Project                           uuid.UUID
	Table                             string
	Total, Live, MinFanout, MaxFanout int
	MinID, MaxID                      string
	Runs                              int
}
type scalePayload struct {
	Project                                                        uuid.UUID
	Activities, Descriptions, MinBytes, MaxBytes, DistinctPayloads int
}
type scaleMark struct {
	Table       string
	Project, ID uuid.UUID
}

func (w issueWorld) scaleFramework(t *testing.T) ([]scaleProject, uuid.UUID, uuid.UUID) {
	t.Helper()
	w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Beta","slug":"beta"}`, 201)
	for _, name := range []string{"bob", "walt"} {
		answerInvitation(t, w.contract, w.base, w.tokens[name], "accept", invite(t, w.contract, w.base, w.tokens["alice"], "beta", name+"@example.com"), 200)
	}
	var beta uuid.UUID
	if err := w.pool.QueryRow(t.Context(), "SELECT id FROM workspaces WHERE slug='beta'").Scan(&beta); err != nil {
		t.Fatal(err)
	}
	projects := []scaleProject{{w.web, w.acme}}
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	for _, name := range []string{"One", "Two", "Three", "Four", "Five"} {
		id := createdProject(t, w.contract, w.base, w.tokens["alice"], "beta", name, strings.ToUpper(name))
		for _, member := range []string{"bob", "walt"} {
			w.makeMember(t, member, id, shared.RoleMember)
		}
		w.send(t, "alice", "POST", "/api/v0/projects/"+id.String()+"/labels", `{"name":"Bug"}`, 201)
		projects = append(projects, scaleProject{id, beta})
	}
	for _, p := range projects {
		w.send(t, "alice", "POST", "/api/v0/projects/"+p.ID.String()+"/labels", `{"name":"Feature"}`, 201)
	}
	neighbor := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Neighbor", "NEIGHBOR")
	probe := w.newWorkItemIn(t, neighbor, `{"name":"Scale probe"}`)
	return projects, beta, probe
}

// The sorted, materialized input precedes volatile UUID generation. Every
// table interleaves all six projects rather than inserting six project blocks.
func (w issueWorld) seedAtScale(t *testing.T, projects []uuid.UUID) (time.Duration, time.Duration) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Minute)
	defer cancel()
	start := time.Now()
	tx, err := w.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	statements := []struct {
		sql  string
		args []any
	}{
		{`CREATE TEMP TABLE scale_projects ON COMMIT DROP AS SELECT p.id project_id,p.workspace_id,s.id state_id,
   ARRAY(SELECT l.id FROM labels l WHERE l.project_id=p.id ORDER BY l.id) labels
   FROM projects p JOIN states s ON s.project_id=p.id AND s.name='Backlog' WHERE p.id=ANY($1)`, []any{projects}},
		{`WITH s AS MATERIALIZED (SELECT p.*,n FROM scale_projects p,generate_series(1,20000) n ORDER BY n,p.project_id)
   INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,state_id,'Item '||n,n,$1,$1 FROM s`, []any{alice}},
		{`UPDATE projects SET last_issue_sequence=20000 WHERE id=ANY($1)`, []any{projects}},
		{`WITH s AS MATERIALIZED (SELECT i.*,a FROM issues i,unnest(ARRAY[$2::uuid,$3::uuid]) a WHERE project_id=ANY($1) ORDER BY sequence_id,a,project_id)
   INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,a,$2,$2 FROM s`, []any{projects, alice, bob}},
		{`WITH s AS MATERIALIZED (SELECT i.*,l FROM issues i JOIN scale_projects p USING(project_id),unnest(p.labels) l ORDER BY sequence_id,l,project_id)
   INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,l,$1,$1 FROM s`, []any{alice}},
		{`WITH s AS MATERIALIZED (SELECT i.*,a FROM issues i,unnest(ARRAY[$2::uuid,$3::uuid,$4::uuid]) a WHERE project_id=ANY($1) ORDER BY sequence_id,a,project_id)
   INSERT INTO issue_subscribers(id,workspace_id,project_id,issue_id,subscriber_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,a,$2,$2 FROM s`, []any{projects, alice, bob, walt}},
		{`WITH s AS MATERIALIZED (SELECT i.*,k FROM issues i,generate_series(1,20) k WHERE project_id=ANY($1) ORDER BY sequence_id,k,project_id)
   INSERT INTO issue_activities(id,workspace_id,project_id,issue_id,verb,field,new_value,comment,actor_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,'updated',CASE WHEN k%5=0 THEN 'description' ELSE 'name' END,
   CASE WHEN k%5=0 THEN (SELECT string_agg(md5(s.id::text||':'||k||':'||g),'' ORDER BY g) FROM generate_series(1,32) g) ELSE 'Item '||k END,
   'changed the value',$2,$2,$2 FROM s`, []any{projects, alice}},
		{`WITH s AS MATERIALIZED (SELECT i.*,k FROM issues i,generate_series(1,2) k WHERE project_id=ANY($1) ORDER BY sequence_id,k,project_id)
   INSERT INTO issue_comments(id,workspace_id,project_id,issue_id,actor_id,comment_html,comment_stripped,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,$2,'<p>Comment '||k||' on '||id||'</p>','Comment '||k||' on '||id,$2,$2 FROM s`, []any{projects, alice}},
		{`WITH s AS MATERIALIZED (SELECT i.*,k FROM issues i,generate_series(1,3) k WHERE project_id=ANY($1) ORDER BY sequence_id,k,project_id)
   INSERT INTO issue_description_versions(id,workspace_id,project_id,issue_id,description_html,description_stripped,last_saved_at,owned_by_id,created_by_id,updated_by_id)
   SELECT uuidv7(),workspace_id,project_id,id,'<p>Version '||k||' of '||id||'</p>','Version '||k||' of '||id,now()+k*interval '1 second',$2,$2,$2 FROM s`, []any{projects, alice}},
	}
	for _, s := range statements {
		if _, err = tx.Exec(ctx, s.sql, s.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	seeded := time.Since(start)
	start = time.Now()
	if _, err = w.pool.Exec(ctx, `ANALYZE issues,issue_assignees,issue_labels,issue_subscribers,issue_activities,issue_comments,issue_description_versions`); err != nil {
		t.Fatal(err)
	}
	return seeded, time.Since(start)
}

// Expectations are deliberately independent of the insertion SQL.
func (w issueWorld) checkScaleRows(t *testing.T, projects []scaleProject, deleted bool, dir, name string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	start := time.Now()
	var evidence []scaleRows
	var payloads []scalePayload
	totals := map[uuid.UUID]int{}
	workspaceTotals := map[uuid.UUID]int{}
	for _, table := range []struct {
		name          string
		count, fanout int
	}{
		{"issues", 20000, 1}, {"issue_activities", 400000, 20}, {"issue_assignees", 40000, 2}, {"issue_labels", 40000, 2},
		{"issue_subscribers", 60000, 3}, {"issue_comments", 40000, 2}, {"issue_description_versions", 60000, 3},
	} {
		for _, p := range projects {
			r := scaleRows{Project: p.ID, Table: table.name}
			if err := w.pool.QueryRow(ctx, "SELECT count(*),count(*) FILTER(WHERE deleted_at IS NULL),coalesce(min(id::text),''),coalesce(max(id::text),'') FROM "+table.name+" WHERE project_id=$1", p.ID).Scan(&r.Total, &r.Live, &r.MinID, &r.MaxID); err != nil {
				t.Fatal(err)
			}
			evidence = append(evidence, r)
			listEvidence(t, dir, name+"-counts", evidence)
			wantLive := table.count
			if deleted {
				wantLive = 0
			}
			if r.Total != table.count || r.Live != wantLive {
				t.Fatalf("SP2 %s project=%s total/live=%d/%d want %d/%d", table.name, p.ID, r.Total, r.Live, table.count, wantLive)
			}
			if !deleted {
				key := "issue_id"
				if table.name == "issues" {
					key = "id"
				}
				if err := w.pool.QueryRow(ctx, "SELECT min(n),max(n) FROM (SELECT count(*) n FROM "+table.name+" WHERE project_id=$1 GROUP BY "+key+") s", p.ID).Scan(&r.MinFanout, &r.MaxFanout); err != nil {
					t.Fatal(err)
				}
				if r.MinFanout != table.fanout || r.MaxFanout != table.fanout {
					t.Fatalf("SP2 %s fanout=%d..%d want %d", table.name, r.MinFanout, r.MaxFanout, table.fanout)
				}
				evidence[len(evidence)-1] = r
			}
			totals[p.ID] += r.Total
			workspaceTotals[p.Workspace] += r.Total
		}
		if !deleted {
			ids := make([]uuid.UUID, 0, len(projects))
			for _, p := range projects {
				ids = append(ids, p.ID)
			}
			for i := range evidence {
				if evidence[i].Table != table.name {
					continue
				}
				r := &evidence[i]
				if err := w.pool.QueryRow(ctx, "WITH ordered AS (SELECT project_id,lag(project_id) OVER(ORDER BY id) prev FROM "+table.name+" WHERE project_id=ANY($1)) SELECT count(*) FROM ordered WHERE project_id=$2 AND project_id IS DISTINCT FROM prev", ids, r.Project).Scan(&r.Runs); err != nil {
					t.Fatal(err)
				}
				if r.Runs < 2 {
					t.Fatalf("SP2 %s project %s is not interleaved: %d runs", table.name, r.Project, r.Runs)
				}
			}
		}
	}
	for _, p := range projects {
		if totals[p.ID] != 660000 {
			t.Fatalf("SP2 project total=%d want660000", totals[p.ID])
		}
		if !deleted {
			r := scalePayload{Project: p.ID}
			if err := w.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE field='description'),
    coalesce(min(octet_length(new_value)) FILTER(WHERE field='description'),0),coalesce(max(octet_length(new_value)) FILTER(WHERE field='description'),0),
    count(DISTINCT new_value) FILTER(WHERE field='description') FROM issue_activities WHERE project_id=$1`, p.ID).Scan(&r.Activities, &r.Descriptions, &r.MinBytes, &r.MaxBytes, &r.DistinctPayloads); err != nil {
				t.Fatal(err)
			}
			payloads = append(payloads, r)
			listEvidence(t, dir, name+"-payloads", payloads)
			if r.Activities != 400000 || r.Descriptions != 80000 || r.MinBytes != 1024 || r.MaxBytes != 1024 || r.DistinctPayloads != 80000 {
				t.Fatalf("SP2 description payload=%+v want400000/80000/1024/1024/80000", r)
			}
		}
	}
	if len(projects) == 6 {
		if workspaceTotals[projects[1].Workspace] != 3300000 {
			t.Fatal("SP2 workspace total", workspaceTotals)
		}
	}
	listEvidence(t, dir, name+"-counts", evidence)
	listEvidence(t, dir, name+"-summary", map[string]any{"count_seconds": time.Since(start).Seconds(), "projects": totals, "workspaces": workspaceTotals})
}

func (w issueWorld) scaleMarkers(t *testing.T, projects []scaleProject) []scaleMark {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	var out []scaleMark
	for _, table := range []string{"issues", "issue_activities", "issue_description_versions"} {
		for _, p := range projects {
			rows, err := w.pool.Query(ctx, fmt.Sprintf(`SELECT id FROM (SELECT id,row_number() OVER(ORDER BY id) n FROM %s WHERE project_id=$1) s WHERE n%%500=1 OR n=(SELECT count(*) FROM %s WHERE project_id=$1) ORDER BY id`, table, table), p.ID)
			if err != nil {
				t.Fatal(err)
			}
			for rows.Next() {
				var id uuid.UUID
				if err = rows.Scan(&id); err != nil {
					t.Fatal(err)
				}
				out = append(out, scaleMark{table, p.ID, id})
			}
			if err = rows.Err(); err != nil {
				t.Fatal(err)
			}
			rows.Close()
		}
	}
	return out
}
