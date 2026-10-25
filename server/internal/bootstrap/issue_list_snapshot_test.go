package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type listSnapshotWrite struct {
	sql  string
	args []any
}

func newListProbe(w issueWorld, stage string) *listSnapshotProbe {
	return &listSnapshotProbe{pool: w.pool, stage: stage, entered: make(chan struct{}), release: make(chan struct{})}
}

// No sleeps and no row-lock surrogate: wait for a completed directory/empty
// query, commit on a different backend, then release the still-open read tx.
func interleaveList(t *testing.T, w issueWorld, p *listSnapshotProbe, actor string, execute func(context.Context) (issueapp.IssueListPage, error), writes []listSnapshotWrite) issueapp.IssueListPage {
	t.Helper()
	ctx, cancel := context.WithTimeout(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids[actor]}), 10*time.Second)
	type result struct {
		page issueapp.IssueListPage
		err  error
	}
	resultCh := make(chan result, 1)
	collected := false
	t.Cleanup(func() {
		p.resume()
		cancel()
		if !collected {
			<-resultCh
		}
	})
	go func() {
		page, err := execute(ctx)
		resultCh <- result{page, err}
	}()
	select {
	case <-p.entered:
	case r := <-resultCh:
		collected = true
		t.Fatalf("reader finished before %s barrier: %+v / %v", p.stage, r.page, r.err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	writerCtx, writerCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer writerCancel()
	tx, err := w.pool.Begin(writerCtx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	var pid int32
	if err := tx.QueryRow(writerCtx, "SELECT pg_backend_pid()").Scan(&pid); err != nil {
		t.Fatal(err)
	}
	p.mu.Lock()
	readerPID := p.pid
	p.mu.Unlock()
	if pid == readerPID || readerPID == 0 {
		t.Fatal("writer reused the reader connection", pid)
	}
	for _, write := range writes {
		if _, err := tx.Exec(writerCtx, write.sql, write.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(writerCtx); err != nil {
		t.Fatal(err)
	}
	p.resume() // Only after COMMIT has returned.
	r := <-resultCh
	collected = true
	if r.err != nil {
		t.Fatal(r.err)
	}
	return r.page
}

func checkSnapshotIDs(t *testing.T, p issueapp.IssueListPage, total int64, ids ...uuid.UUID) {
	t.Helper()
	got := make([]uuid.UUID, len(p.Data))
	for n, row := range p.Data {
		got[n] = row.ID
	}
	if p.Total != total || !slices.Equal(got, ids) || p.NextCursor != "" {
		t.Fatalf("snapshot %+v; want IDs %v total %d", p, ids, total)
	}
}

// Dropping ReadTx or escaping its ctx must expose a mixed directory/row view
// or fail the actual RO/RR/connection observations, including the fallback.
func TestIssueListSnapshot(t *testing.T) {
	t.Run("state-and-parents", func(t *testing.T) {
		w := newIssueWorld(t)
		w.makeMember(t, "walt", w.web, shared.RoleMember)
		parent := w.newWorkItem(t)
		first := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"First","state_id":%q,"parent_id":%q}`, w.todo, parent))
		second := w.newWorkItemBy(t, "walt", w.web, fmt.Sprintf(`{"name":"Second","state_id":%q}`, w.todo))
		listSeedSQL(t, w, "UPDATE project_members SET role=5 WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])
		p := newListProbe(w, "states")
		u, _ := snapshotListUseCases(w, p)
		raw := map[string][]string{"order_by": {"sort_order"}}
		nextState := uuid.NewV7()
		old := interleaveList(t, w, p, "walt", func(ctx context.Context) (issueapp.IssueListPage, error) { return u.Execute(ctx, w.web, raw) }, []listSnapshotWrite{
			{`INSERT INTO states(id,workspace_id,project_id,name,color,"group") VALUES($1,$2,$3,'New state','#000000','started')`, []any{nextState, w.acme, w.web}},
			{"UPDATE issues SET state_id=$2 WHERE id=$1", []any{first, nextState}},
		})
		checkSnapshotIDs(t, old, 2, first, second)
		if old.Data[0].StateID != w.todo || old.Data[0].ParentID != nil || !slices.Contains(p.steps, "parents") {
			t.Fatal("old state or hidden parent escaped snapshot", old, p.steps)
		}
		next, _ := snapshotListUseCases(w, newListProbe(w, ""))
		page, err := next.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), w.web, raw)
		if err != nil {
			t.Fatal(err)
		}
		checkSnapshotIDs(t, page, 2, first, second)
		if page.Data[0].StateID != nextState || page.Data[0].ParentID != nil {
			t.Fatal("next request did not see committed state", page)
		}
	})
	for _, field := range []string{"label", "assignee"} {
		t.Run(field+"-name", func(t *testing.T) { snapshotListNames(t, field) })
	}
	t.Run("caller-exits", func(t *testing.T) {
		w := newIssueWorld(t)
		w.makeMember(t, "walt", w.web, shared.RoleMember)
		other := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
		w.makeMember(t, "walt", other, shared.RoleMember)
		w.makeMember(t, "bob", other, shared.RoleMember)
		first := w.newWorkItemBy(t, "bob", w.web, `{"name":"First"}`)
		second := w.newWorkItemBy(t, "bob", other, `{"name":"Second"}`)
		p := newListProbe(w, "visible")
		_, u := snapshotListUseCases(w, p)
		raw := map[string][]string{"order_by": {"sort_order"}, "relation": {"created"}}
		old := interleaveList(t, w, p, "walt", func(ctx context.Context) (issueapp.IssueListPage, error) {
			return u.Execute(ctx, "acme", w.ids["bob"], raw)
		}, []listSnapshotWrite{
			{"UPDATE project_members SET is_active=false WHERE project_id=$1 AND member_id=$2", []any{other, w.ids["walt"]}},
			{"UPDATE issues SET name='After exit' WHERE id=$1", []any{second}},
		})
		checkSnapshotIDs(t, old, 2, first, second)
		if old.Data[1].Name != "Second" {
			t.Fatal("old project scope combined with new issue", old)
		}
		if !slices.Equal(p.steps, []string{"workspace", "auth", "member", "visible", "states", "labels", "assignees", "names", "rows"}) {
			t.Fatal("incomplete member snapshot chain", p.steps)
		}
		_, next := snapshotListUseCases(w, newListProbe(w, ""))
		page, err := next.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), "acme", w.ids["bob"], raw)
		if err != nil {
			t.Fatal(err)
		}
		checkSnapshotIDs(t, page, 1, first)
	})
	for _, shape := range []string{"flat", "groups", "subgroups", "member"} {
		t.Run("fallback/"+shape, func(t *testing.T) { snapshotListFallback(t, shape) })
	}
}

func snapshotListNames(t *testing.T, field string) {
	t.Helper()
	w := newIssueWorld(t)
	var c struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"Beta"}`, 201), &c)
	listSeedSQL(t, w, "UPDATE labels SET name='Alpha' WHERE id=$1", w.bug)
	listSeedSQL(t, w, "UPDATE users SET display_name=CASE WHEN id=$1 THEN 'Alpha' ELSE 'Beta' END WHERE id=ANY($2::uuid[])", w.ids["bob"], []uuid.UUID{w.ids["bob"], w.ids["walt"]})
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	first := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"First","state_id":%q,"label_ids":%s,"assignee_ids":%s}`, w.todo, idArray(w.bug), idArray(w.ids["bob"])))
	second := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Second","state_id":%q,"label_ids":%s,"assignee_ids":%s}`, w.todo, idArray(c.ID), idArray(w.ids["walt"])))
	third := uuid.NewV7()
	stage := "labels"
	writes := []listSnapshotWrite{{"UPDATE labels SET name='Zulu' WHERE id=$1", []any{w.bug}}}
	if field == "assignee" {
		stage = "names"
		writes = []listSnapshotWrite{{"UPDATE users SET display_name='Zulu' WHERE id=$1", []any{w.ids["bob"]}}}
	}
	writes = append(writes,
		listSnapshotWrite{`INSERT INTO issues(id,workspace_id,project_id,state_id,sequence_id,name,created_by_id) VALUES($1,$2,$3,$4,7000,'Third',$5)`, []any{third, w.acme, w.web, w.todo, w.ids["alice"]}},
		listSnapshotWrite{"INSERT INTO issue_labels(id,workspace_id,project_id,issue_id,label_id) VALUES($1,$2,$3,$4,$5)", []any{uuid.NewV7(), w.acme, w.web, third, w.bug}},
		listSnapshotWrite{"INSERT INTO issue_assignees(id,workspace_id,project_id,issue_id,assignee_id) VALUES($1,$2,$3,$4,$5)", []any{uuid.NewV7(), w.acme, w.web, third, w.ids["bob"]}})
	p := newListProbe(w, stage)
	u, _ := snapshotListUseCases(w, p)
	raw := map[string][]string{"order_by": {field + "_name"}}
	old := interleaveList(t, w, p, "alice", func(ctx context.Context) (issueapp.IssueListPage, error) { return u.Execute(ctx, w.web, raw) }, writes)
	checkSnapshotIDs(t, old, 2, first, second)
	next, _ := snapshotListUseCases(w, newListProbe(w, ""))
	page, err := next.Execute(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["alice"]}), w.web, raw)
	if err != nil {
		t.Fatal(err)
	}
	checkSnapshotIDs(t, page, 3, second, first, third)
}

func snapshotListFallback(t *testing.T, shape string) {
	t.Helper()
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	var c struct{ ID uuid.UUID }
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"C"}`, 201), &c)
	body := fmt.Sprintf(`{"name":"Match","label_ids":%s,"assignee_ids":%s}`, idArray(w.bug, c.ID), idArray(w.ids["bob"]))
	first := w.newWorkItemBy(t, "bob", w.web, body)
	w.newWorkItemBy(t, "bob", w.web, body)
	w.newWorkItemBy(t, "bob", w.web, fmt.Sprintf(`{"name":"Only C","label_ids":%s,"assignee_ids":%s}`, idArray(c.ID), idArray(w.ids["bob"])))
	cursor, err := domain.EncodeListCursor(domain.ListCursor{Order: "sort_order", ID: first, Key: []byte("9999999999")})
	if err != nil {
		t.Fatal(err)
	}
	raw := map[string][]string{"order_by": {"sort_order"}, "limit": {"2"}, "cursor": {cursor}, "label_id": {w.bug.String()}}
	if shape != "flat" {
		raw["group_by"], raw["group_key"] = []string{"label_id"}, []string{c.ID.String()}
	}
	if shape == "subgroups" {
		raw["sub_group_by"], raw["sub_group_key"] = []string{"assignee_id"}, []string{w.ids["bob"].String()}
	}
	if shape == "member" {
		raw["relation"] = []string{"assigned"}
	}
	execute := func(p *listSnapshotProbe) func(context.Context) (issueapp.IssueListPage, error) {
		u, m := snapshotListUseCases(w, p)
		return func(ctx context.Context) (issueapp.IssueListPage, error) {
			if shape == "member" {
				return m.Execute(ctx, "acme", w.ids["bob"], raw)
			}
			return u.Execute(ctx, w.web, raw)
		}
	}
	p := newListProbe(w, "empty")
	old := interleaveList(t, w, p, "walt", execute(p), []listSnapshotWrite{
		{"UPDATE issues SET deleted_at=now() WHERE id=$1", []any{first}},
		{"UPDATE labels SET name='Renamed C' WHERE id=$1", []any{c.ID}},
	})
	checkSnapshotIDs(t, old, 2)
	if len(p.reads) != 2 || p.reads[0].Query.Limit != 2 || p.reads[1].Query.Limit != 1 {
		t.Fatal("missing real fallback", p.reads)
	}
	page, err := execute(newListProbe(w, ""))(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}))
	if err != nil {
		t.Fatal(err)
	}
	checkSnapshotIDs(t, page, 1)
}
