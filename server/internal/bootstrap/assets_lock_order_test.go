package bootstrap

import (
	"fmt"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Every wait is the actual HTTP query behind one identified row fixture.
// Earlier lock strength is attributed to this request's transaction ID, not
// to a worker, the fixture itself, or the tuple queue in front of the request.
func TestAssetContentLockOrder(t *testing.T) {
	w := newObservedIssueWorld(t)
	d := createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Lock order"}}`, w.web))
	file := uploadedAssetHTTP(t, w, "alice", "DRAFT_ISSUE_ATTACHMENT", assetScopeBody("draft", d.ID), "order.txt", "text/plain", []byte("order"))
	rows := []struct {
		table, fixture, want string
		id                   uuid.UUID
	}{
		{"users", "FOR SHARE", "FOR NO KEY UPDATE", w.ids["alice"]},
		{"workspaces", "FOR NO KEY UPDATE", "FOR SHARE", w.acme},
		{"projects", "FOR SHARE", "FOR NO KEY UPDATE", w.web},
		{"draft_issues", "FOR SHARE", "FOR UPDATE", d.ID},
		{"file_assets", "FOR SHARE", "FOR UPDATE", file.Asset.ID},
	}
	holders := make([]pgx.Tx, len(rows))
	for n, row := range rows {
		holders[n] = holding(t, w.pool, "SELECT id FROM "+row.table+" WHERE id=$1 "+row.fixture, row.id)
		defer requestRelease(t, holders[n])
	}
	req := newRequest(t, "GET", w.base+"/api/v0/assets/"+file.Asset.ID.String()+"/content", w.tokens["alice"], nil)
	probe := w.observe(req)
	result := sendInBackground(req)
	for n, row := range rows {
		pid := requestBlocked(t, w.pool, probe, holders[n].Conn().PgConn().PID(), row.table)
		if got := requestWrittenTables(t, w.pool, pid); len(got) != 0 {
			t.Fatalf("content signing wrote before final lock: %v", got)
		}
		for _, earlier := range rows[:n] {
			if got := requestRowLock(t, w.pool, w.pool, probe, earlier.table+" WHERE id=$1", earlier.id); got != earlier.want {
				t.Fatalf("while waiting on %s, earlier %s=%s want %s", row.table, earlier.table, got, earlier.want)
			}
		}
		requestRelease(t, holders[n])
	}
	got := receiveWithin(t, result, 5*time.Second, "content after every lock boundary")
	if got.err != nil || got.res.StatusCode != 200 {
		t.Fatalf("content signing=%+v", got)
	}
	w.contract.CheckResponse(t, req, got.res)
}

func TestAssetCopyProjectLockOrder(t *testing.T) {
	for _, order := range []string{"ascending", "descending", "same"} {
		t.Run(order, func(t *testing.T) {
			url := pgtest.NewDatabase(t)
			w := newIssueWorldOnRequests(t, url, openPool(t, url), func(c *config.Config) {
				c.Issue.WriteGate = config.WriteGateConfig{PerProject: 1, MaxQueue: 0, MaxWait: time.Second}
			}, &requestQueries{})
			low := w.web
			high := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Other", "OTHER")
			if low.Compare(high) >= 0 {
				t.Fatal("fixture did not produce distinct ordered project IDs")
			}
			sourceProject, targetProject := low, high
			if order == "descending" {
				sourceProject, targetProject = high, low
			}
			if order == "same" {
				targetProject = low
			}
			sourceIssue := w.newWorkItemIn(t, sourceProject, `{"name":"Source"}`)
			targetIssue := w.newWorkItemIn(t, targetProject, `{"name":"Target"}`)
			file := uploadedAssetHTTP(t, w, "alice", "ISSUE_ATTACHMENT", assetScopeBody("issue", sourceIssue), "source.txt", "text/plain", []byte("copy"))
			lower := holding(t, w.pool, "SELECT id FROM projects WHERE id=$1 FOR SHARE", low)
			defer requestRelease(t, lower)
			var upper pgx.Tx
			if order != "same" {
				upper = holding(t, w.pool, "SELECT id FROM projects WHERE id=$1 FOR SHARE", high)
				defer requestRelease(t, upper)
			}
			asset := holding(t, w.pool, "SELECT id FROM file_assets WHERE id=$1 FOR SHARE", file.Asset.ID)
			defer requestRelease(t, asset)
			body := fmt.Sprintf(`{"purpose":"ISSUE_ATTACHMENT","scope":%s}`, assetScopeBody("issue", targetIssue))
			req := newRequest(t, "POST", w.base+duplicateAssetPath(file.Asset.ID), w.tokens["alice"], []byte(body))
			probe := w.observe(req)
			result := sendInBackground(req)
			requestBlocked(t, w.pool, probe, lower.Conn().PgConn().PID(), "projects")
			requestRelease(t, lower)
			if upper != nil {
				requestBlocked(t, w.pool, probe, upper.Conn().PgConn().PID(), "projects")
				if got := requestRowLock(t, w.pool, w.pool, probe, "projects WHERE id=$1", low); got != "FOR NO KEY UPDATE" {
					t.Fatalf("low project before high=%s", got)
				}
				requestRelease(t, upper)
			}
			pid := requestBlocked(t, w.pool, probe, asset.Conn().PgConn().PID(), "file_assets")
			if got := requestWrittenTables(t, w.pool, pid); len(got) != 0 {
				t.Fatalf("copy wrote target before locking source: %v", got)
			}
			for _, id := range []uuid.UUID{sourceProject, targetProject} {
				if got := requestRowLock(t, w.pool, w.pool, probe, "projects WHERE id=$1", id); got != "FOR NO KEY UPDATE" {
					t.Fatalf("project %s=%s before source lock", id, got)
				}
			}
			requestRelease(t, asset)
			got := receiveWithin(t, result, 5*time.Second, "copy with sorted project locks")
			if got.err != nil || got.res.StatusCode != 201 {
				t.Fatalf("copy=%+v", got)
			}
			w.contract.CheckResponse(t, req, got.res)
		})
	}
}
