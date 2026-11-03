package bootstrap

import (
	"fmt"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type attachmentCapacityFixture struct {
	w                    issueWorld
	trace                *attachmentCapacityTrace
	projects             []uuid.UUID
	counts               map[uuid.UUID]int64
	projectRows, ownRows map[uuid.UUID][]uuid.UUID
}

func newAttachmentCapacity(t *testing.T) attachmentCapacityFixture {
	t.Helper()
	requests := &requestQueries{}
	trace := &attachmentCapacityTrace{requestQueryTracer: requestQueryTracer{requests: requests}, names: map[string][]string{}}
	cfg := testConfig(t, pgtest.NewDatabase(t), false)
	w, _ := startImageAPIWorld(t, cfg, appFactories{pool: trace.pool, assets: newAssets})
	w.requests = requests
	for _, name := range []string{"bob", "walt"} {
		email := name + "@example.com"
		w.tokens[name] = registerAccount(t, w.contract, w.base, email).AccessToken
		w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
		link := inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", email, shared.RoleMember)
		answerInvitation(t, w.contract, w.base, w.tokens[name], "accept", link, 200)
		w.makeMember(t, name, w.web, shared.RoleMember)
	}
	// Only this disposable database: prevent automatic ANALYZE from turning a
	// cold-statistics fixture into a warm-statistics benchmark during the run.
	for _, table := range []string{"issues", "file_assets", "issue_labels", "issue_assignees", "projects", "project_members", "states", "labels"} {
		if _, err := w.pool.Exec(t.Context(), "ALTER TABLE "+table+" SET (autovacuum_enabled=false)"); err != nil {
			t.Fatal(err)
		}
	}
	return attachmentCapacityFixture{w: w, trace: trace, counts: map[uuid.UUID]int64{}, projectRows: map[uuid.UUID][]uuid.UUID{}, ownRows: map[uuid.UUID][]uuid.UUID{}}
}

// Explicit internal capacity fixture, following listScale's SQL/Go-oracle
// separation. These rows prove counting, not physical uploads or capabilities.
// The accounts/workspace/two ordinary projects above still use real APIs.
func (f *attachmentCapacityFixture) seedProject(t *testing.T, p int, project uuid.UUID, n int) {
	t.Helper()
	w := f.w
	f.projects = append(f.projects, project)
	var labels []uuid.UUID
	for _, name := range []string{"Capacity Alpha", "Capacity Beta"} {
		var label struct{ ID uuid.UUID }
		decodeAnswer(t, w.send(t, "alice", "POST", "/api/v0/projects/"+project.String()+"/labels", fmt.Sprintf(`{"name":%q}`, name), 201), &label)
		labels = append(labels, label.ID)
	}
	tx, err := w.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	capacityExec(t, tx, `CREATE TEMP TABLE capacity_seed ON COMMIT DROP AS
 SELECT ('00000000-0000-7000-8000-'||lpad(to_hex(($2::bigint+1)*1000000-seq),12,'0'))::uuid AS id,
 $3::uuid AS project_id,$4::uuid AS state_id,seq,
 CASE WHEN seq>$1 THEN 0 ELSE 1+(seq-1)%4 END AS amount,
 CASE WHEN seq%2=0 OR seq>$1 THEN $6::uuid ELSE $5::uuid END AS creator
 FROM generate_series(1,$1::int+1) seq`, n, p, project, stateID(t, w.pool, project, "Todo"), w.ids["bob"], w.ids["walt"])
	f.insertCapacityRows(t, tx)
	capacityExec(t, tx, `INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id,created_by_id)
 SELECT uuidv7(),$1,project_id,id,label,$2 FROM capacity_seed CROSS JOIN unnest($3::uuid[]) label`, w.acme, w.ids["alice"], labels)
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	for seq := 1; seq <= n+1; seq++ {
		id := scaleID(p, seq)
		count := int64(1 + (seq-1)%4)
		if seq == n+1 {
			count = 0
		}
		f.counts[id] = count
		f.projectRows[project] = append(f.projectRows[project], id)
		if seq%2 == 0 || seq == n+1 {
			f.ownRows[project] = append(f.ownRows[project], id)
		}
	}
}

func capacityExec(t *testing.T, tx pgx.Tx, sql string, args ...any) {
	t.Helper()
	if _, err := tx.Exec(t.Context(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func (f *attachmentCapacityFixture) insertCapacityRows(t *testing.T, tx pgx.Tx) {
	t.Helper()
	w := f.w
	capacityExec(t, tx, `INSERT INTO issues(id,workspace_id,project_id,state_id,name,sequence_id,created_by_id,updated_by_id)
 SELECT id,$1,project_id,state_id,'Attachment capacity '||seq,seq,creator,creator FROM capacity_seed`, w.acme)
	capacityExec(t, tx, `UPDATE projects p SET last_issue_sequence=s.seq FROM
 (SELECT project_id,max(seq) AS seq FROM capacity_seed GROUP BY project_id) s WHERE p.id=s.project_id`)
	capacityExec(t, tx, `INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id,created_by_id)
 SELECT uuidv7(),$1,project_id,id,member,$2 FROM capacity_seed CROSS JOIN unnest($3::uuid[]) member`, w.acme, w.ids["alice"], []uuid.UUID{w.ids["alice"], w.ids["bob"]})
	capacityExec(t, tx, `INSERT INTO file_assets(id,user_id,workspace_id,project_id,issue_id,entity_type,scope_kind,attributes,size,is_uploaded,bound_at,storage_metadata,created_by_id)
 SELECT uuidv7(),$2,$1,project_id,id,'ISSUE_ATTACHMENT','issue','{"name":"capacity.txt","content_type":"text/plain"}',1,true,now(),
 '{"schema_version":1,"origin":"copy","staging":null,"final":null,"candidates":[]}',$2
 FROM capacity_seed CROSS JOIN LATERAL generate_series(1,amount) a`, w.acme, w.ids["bob"])
}

// 1001 projects with interleaved issue UUIDs, rather than one contiguous
// interval per project. Both project batches must participate in every page.
func (f *attachmentCapacityFixture) seedManyProjects(t *testing.T) {
	t.Helper()
	w := f.w
	tx, err := w.pool.Begin(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(t.Context()) }()
	capacityExec(t, tx, `CREATE TEMP TABLE capacity_projects ON COMMIT DROP AS
 SELECT p,CASE WHEN p=0 THEN $1::uuid ELSE uuidv7() END AS id,uuidv7() AS state_id FROM generate_series(0,1000) p`, w.web)
	capacityExec(t, tx, `INSERT INTO projects(id,workspace_id,name,identifier)
 SELECT id,$1,'Capacity '||p,'C'||p FROM capacity_projects WHERE p>0`, w.acme)
	capacityExec(t, tx, `INSERT INTO project_members(id,workspace_id,project_id,member_id,role)
 SELECT uuidv7(),$1,id,$2,15 FROM capacity_projects WHERE p>0`, w.acme, w.ids["bob"])
	capacityExec(t, tx, `INSERT INTO states(id,workspace_id,project_id,name,color,"group")
 SELECT state_id,$1,id,'Capacity Todo','#ffffff','unstarted' FROM capacity_projects`, w.acme)
	capacityExec(t, tx, `CREATE TEMP TABLE capacity_seed ON COMMIT DROP AS
 SELECT ('00000000-0000-7000-8000-'||lpad(to_hex(5000000-(p+1+1001*(seq-1))),12,'0'))::uuid AS id,
 id AS project_id,state_id,seq,seq AS amount,$1::uuid AS creator
 FROM capacity_projects CROSS JOIN generate_series(1,2) seq`, w.ids["bob"])
	f.insertCapacityRows(t, tx)
	if err := tx.Commit(t.Context()); err != nil {
		t.Fatal(err)
	}
	f.projects = queryIDs(t, w.pool, "SELECT id FROM projects WHERE workspace_id=$1 ORDER BY id", w.acme)
	if len(f.projects) != 1001 {
		t.Fatal("wrong project fixture", len(f.projects))
	}
	for seq := 1; seq <= 2002; seq++ {
		id := scaleID(4, seq)
		f.counts[id] = int64(1 + (seq-1)/1001)
	}
}

func (f attachmentCapacityFixture) assertColdStatistics(t *testing.T) {
	t.Helper()
	var analyzed int
	err := f.w.pool.QueryRow(t.Context(), `SELECT count(*) FROM pg_stat_all_tables WHERE schemaname='public'
 AND relname=ANY($1::text[]) AND (last_analyze IS NOT NULL OR last_autoanalyze IS NOT NULL)`,
		[]string{"issues", "file_assets", "issue_assignees", "issue_labels", "projects", "project_members", "states", "labels"}).Scan(&analyzed)
	if err != nil || analyzed != 0 {
		t.Fatal("capacity statistics were warmed", analyzed, err)
	}
}
