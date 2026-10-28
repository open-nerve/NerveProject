package bootstrap

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func draftTerminalRows(t *testing.T, db postgres.Querier) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, table := range []string{"draft_issues", "projects", "workspaces", "workspace_members"} {
		var rows string
		if err := db.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+table+" r").Scan(&rows); err != nil {
			t.Fatal(err)
		}
		out[table] = rows
	}
	return out
}

// W is held while both resource and role changes are uncommitted. The request
// passes the old precheck, then reads the committed terminal state after W.
// Resource deletion must win over guest, while a live (even moved) draft gives
// forbidden without acquiring old/new P or D. The trace belongs to this request.
func testDraftWorkspaceTerminal(t *testing.T) {
	for _, method := range []string{"PATCH", "DELETE", "POST"} {
		for _, terminal := range []string{"guest", "draft deleted and guest", "project deleted and guest", "workspace deleted and guest", "moved and guest"} {
			t.Run(method+"/"+terminal, func(t *testing.T) {
				w := newIssueWorld(t)
				next := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
				d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Unchanged"}}`)
				trace := &listTrace{}
				pool := listTracePool(t, w.pool.Config().ConnString(), trace)
				tx := requestTx(pool)
				route := newIssueRoute(t, pool, tx)
				holder := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id=$1 FOR NO KEY UPDATE", w.acme)
				draftExec(t, holder, "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["alice"])
				want, code := 403, "forbidden"
				switch terminal {
				case "draft deleted and guest":
					draftExec(t, holder, "UPDATE draft_issues SET deleted_at=now() WHERE id=$1", d.ID)
					want, code = 404, "issue.draft_not_found"
				case "project deleted and guest":
					draftExec(t, holder, "UPDATE projects SET deleted_at=now() WHERE id=$1", w.web)
					want, code = 404, "issue.draft_not_found"
				case "workspace deleted and guest":
					draftExec(t, holder, "UPDATE workspaces SET deleted_at=now() WHERE id=$1", w.acme)
					want, code = 404, "issue.draft_not_found"
				case "moved and guest":
					draftExec(t, holder, "UPDATE draft_issues SET project_id=$2 WHERE id=$1", d.ID, next)
				}
				expected := draftTerminalRows(t, holder)
				issues := collaborationState(t, w)
				path, body := draftPath(d.ID), ""
				if method == "PATCH" {
					body = `{"payload":{"name":"Forbidden"}}`
				}
				if method == "POST" {
					path = publishPath(d.ID)
				}
				var e exchange
				send := func() error {
					ctx := context.WithValue(context.Background(), listRequestKey{}, "terminal")
					e.req, e.rec = route.sendContext(ctx, method, path, w.ids["alice"], body)
					return nil
				}
				collect := requestBackground(t, send, func() { requestRelease(t, holder) })
				requestBlocked(t, w.pool, tx, holder.Conn().PgConn().PID(), "workspaces")
				if pgtest.RowWaits(t, w.pool, "projects", next, "FOR NO KEY UPDATE") {
					t.Fatal("request took a new project lock while waiting W")
				}
				draftCommit(t, holder)
				collect()
				w.refused(t, e, want, code)
				if !reflect.DeepEqual(expected, draftTerminalRows(t, w.pool)) || !reflect.DeepEqual(issues, collaborationState(t, w)) {
					t.Fatal("request changed terminal resource/role/counter/issue state")
				}
				records := trace.take("terminal")
				afterW := false
				resourceRead := false
				firstAfterW := true
				for _, r := range records {
					q := strings.ToLower(r.SQL)
					if strings.Contains(q, "for share") && strings.Contains(q, "workspaces") {
						afterW = true
						continue
					}
					if !afterW {
						continue
					}
					if firstAfterW && terminal != "workspace deleted and guest" && !strings.Contains(q, "from draft_issues") {
						t.Fatalf("W must be followed by unlocked resource read before authorization: %s", r.SQL)
					}
					firstAfterW = false
					if strings.Contains(q, "from draft_issues") && !strings.Contains(q, "for update") {
						resourceRead = true
					}
					if strings.Contains(q, "for update") || strings.Contains(q, "for no key update") || strings.Contains(q, "for share") {
						t.Fatalf("unauthorized request acquired extra lock after W: %s", r.SQL)
					}
				}
				if !afterW || (terminal != "workspace deleted and guest" && !resourceRead) {
					t.Fatalf("missing W then unlocked resource read: %+v", records)
				}
			})
		}
	}
	// A nil project has exactly W/D semantics, including the same terminal
	// precedence; there is no synthetic zero-UUID or newly followed project lock.
	for _, deleted := range []bool{false, true} {
		t.Run(map[bool]string{false: "nil guest", true: "nil deleted guest"}[deleted], func(t *testing.T) {
			w := newIssueWorld(t)
			d := createDraftHTTP(t, w, "alice", `{"payload":{"name":"Nil"}}`)
			holder := holding(t, w.pool, "SELECT 1 FROM workspaces WHERE id=$1 FOR NO KEY UPDATE", w.acme)
			draftExec(t, holder, "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["alice"])
			if deleted {
				draftExec(t, holder, "UPDATE draft_issues SET deleted_at=now() WHERE id=$1", d.ID)
			}
			expected := draftTerminalRows(t, holder)
			tx := requestTx(w.pool)
			var e exchange
			collect := requestBackground(t, draftWrite(t, w, tx, &e, "PATCH", draftPath(d.ID), `{"payload":{}}`), func() { requestRelease(t, holder) })
			requestBlocked(t, w.pool, tx, holder.Conn().PgConn().PID(), "workspaces")
			draftNoLock(t, w, "projects", w.web)
			draftCommit(t, holder)
			collect()
			want, code := 403, "forbidden"
			if deleted {
				want, code = 404, "issue.draft_not_found"
			}
			w.refused(t, e, want, code)
			if !reflect.DeepEqual(expected, draftTerminalRows(t, w.pool)) {
				t.Fatal("nil draft/role terminal state changed")
			}
		})
	}
}

// A real P or D holder commits deletion or a changed association after the
// request's precheck. After either wait, writes must use the final row, reject
// deletion404/change409 and never lock the newly observed project under D.
func testDraftPDTerminal(t *testing.T) {
	for _, method := range []string{"PATCH", "DELETE", "POST"} {
		for _, stage := range []string{"P", "D"} {
			for _, change := range []string{"deleted", "moved"} {
				t.Run(method+"/"+stage+"/"+change, func(t *testing.T) {
					w := newIssueWorld(t)
					next := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
					d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{"name":"Terminal"}}`)
					table, id := "projects", w.web
					if stage == "D" {
						table, id = "draft_issues", d.ID
					}
					holder := holding(t, w.pool, "SELECT 1 FROM "+table+" WHERE id=$1 FOR UPDATE", id)
					tx := requestTx(w.pool)
					path, body := draftPath(d.ID), ""
					if method == "PATCH" {
						body = `{"payload":{"name":"Must not write"}}`
					}
					if method == "POST" {
						path = publishPath(d.ID)
					}
					var e exchange
					collect := requestBackground(t, draftWrite(t, w, tx, &e, method, path, body), func() { requestRelease(t, holder) })
					requestBlocked(t, w.pool, tx, holder.Conn().PgConn().PID(), table)
					// Keep P2 locked until the request answers. Chasing it after D would
					// fail this bounded request instead of returning the required conflict.
					newP := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", next)
					t.Cleanup(func() { requestRelease(t, newP) })
					want, code := 404, "issue.draft_not_found"
					if change == "moved" {
						draftExec(t, holder, "UPDATE draft_issues SET project_id=$2 WHERE id=$1", d.ID, next)
						want, code = 409, "issue.draft_changed"
					} else if stage == "P" {
						draftExec(t, holder, "UPDATE projects SET deleted_at=now() WHERE id=$1", w.web)
					} else {
						draftExec(t, holder, "UPDATE draft_issues SET deleted_at=now() WHERE id=$1", d.ID)
					}
					expected := draftTerminalRows(t, holder)
					effects := collaborationState(t, w)
					draftCommit(t, holder)
					collect()
					w.refused(t, e, want, code)
					requestRelease(t, newP)
					if !reflect.DeepEqual(expected, draftTerminalRows(t, w.pool)) || !reflect.DeepEqual(effects, collaborationState(t, w)) {
						t.Fatal("P/D terminal refusal wrote state")
					}
					if change == "moved" {
						assertDraftLive(t, w, d.ID, next)
					}
				})
			}
		}
	}
}
