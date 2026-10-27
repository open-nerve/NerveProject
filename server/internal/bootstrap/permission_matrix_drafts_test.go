package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func draftMatrixRows() []matrixRow {
	own := func(method, body string) func(caller, seeded) (string, string, string) {
		return func(c caller, s seeded) (string, string, string) {
			return method, draftPath(s.drafts[workspaceOf(c)+"/"+string(c)]), body
		}
	}
	rows := []matrixRow{
		{op: "listDrafts", request: toWorkspace("GET", "/drafts", ""), cells: workspaceCells(cellOK, cellOK, cellForbidden, cellWorkspaceNotFound, cellWorkspaceNotFound, cellWorkspaceNotFound)},
		{op: "createDraft", write: true, request: toWorkspace("POST", "/drafts", `{"payload":{}}`), cells: workspaceCells(cellCreated, cellCreated, cellForbidden, cellWorkspaceNotFound, cellWorkspaceNotFound, cellWorkspaceNotFound)},
		{op: "getDraft", request: own("GET", ""), cells: workspaceCells(cellOK, cellOK, cellForbidden, cellDraftNotFound, cellDraftNotFound, cellDraftNotFound)},
		{op: "updateDraft", write: true, request: own("PATCH", `{"payload":{}}`), cells: workspaceCells(cellOK, cellOK, cellForbidden, cellDraftNotFound, cellDraftNotFound, cellDraftNotFound)},
		{op: "deleteDraft", write: true, request: own("DELETE", ""), cells: workspaceCells(cellNoContent, cellNoContent, cellForbidden, cellDraftNotFound, cellDraftNotFound, cellDraftNotFound)},
	}
	return rows
}

var cellDraftNotFound = cell{http.StatusNotFound, "issue.draft_not_found"}

func workspaceCells(cells ...cell) map[caller]cell {
	out := map[caller]cell{}
	for i, c := range workspaceColumns {
		out[c] = cells[i]
	}
	return out
}

// Every role exercises all five operations. Access to another owner's live
// row, a removed account, and the project deletion window must be concealed
// by the complete same problem as a random missing draft, never just a 404.
func TestPermissionMatrixDrafts(t *testing.T) {
	w := newIssueWorld(t)
	own := map[string]draftHTTP{}
	for _, who := range []string{"alice", "bob", "walt", "wanda"} {
		own[who] = createDraftHTTP(t, w, who, `{"project_id":"`+w.web.String()+`","payload":{"name":"private"}}`)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET role=5 WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"]); err != nil {
		t.Fatal(err)
	}
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspace_members SET deleted_at=now() WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	for _, who := range []string{"alice", "wanda", "walt", "bob", "olga"} {
		member := who == "alice" || who == "wanda"
		want := 404
		if member {
			want = 200
		}
		if who == "walt" {
			want = 403
		}
		listWant := want
		listed := draftRequest(t, w, who, "GET", draftsPath, "", listWant)
		if listWant == 404 {
			absent := draftRequest(t, w, who, "GET", "/api/v0/workspaces/absent/drafts", "", 404)
			if !reflect.DeepEqual(problemObject(t, listed), problemObject(t, absent)) {
				t.Fatal(listed, absent)
			}
		}
		createWant := want
		if member {
			createWant = 201
		}
		created := draftRequest(t, w, who, "POST", draftsPath, `{"payload":{}}`, createWant)
		if createWant == 404 {
			absent := draftRequest(t, w, who, "POST", "/api/v0/workspaces/absent/drafts", `{"payload":{}}`, 404)
			if !reflect.DeepEqual(problemObject(t, created), problemObject(t, absent)) {
				t.Fatal(created, absent)
			}
		}
		for _, other := range []bool{false, true} {
			id := own[who].ID
			if id == uuid.Nil() {
				id = own["alice"].ID
			}
			if other {
				id = own["alice"].ID
				if who == "alice" {
					id = own["wanda"].ID
				}
			}
			for _, method := range []string{"GET", "PATCH", "DELETE"} {
				body := ""
				if method == "PATCH" {
					body = `{}`
				}
				status := want
				if member && other {
					status = 404
				}
				if status == 200 && method == "DELETE" {
					status = 204
				}
				target := id
				if status == 204 {
					target = createDraftHTTP(t, w, who, `{"payload":{}}`).ID
				}
				raw := draftRequest(t, w, who, method, draftPath(target), body, status)
				if status == 404 {
					missing := draftRequest(t, w, who, method, draftPath(uuid.NewV7()), body, 404)
					if !reflect.DeepEqual(problemObject(t, raw), problemObject(t, missing)) {
						t.Fatal("hidden problem differs", raw, missing)
					}
				}
			}
		}
	}
	// Model a committed project tombstone before its dependent rows are swept.
	d := createDraftHTTP(t, w, "alice", `{"project_id":"`+w.web.String()+`","payload":{}}`)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE projects SET deleted_at=now() WHERE id=$1", w.web); err != nil {
		t.Fatal(err)
	}
	for _, who := range []string{"alice", "walt", "bob", "olga"} {
		for _, method := range []string{"GET", "PATCH", "DELETE"} {
			body := ""
			if method == "PATCH" {
				body = `{"project_id":null}`
			}
			raw := draftRequest(t, w, who, method, draftPath(d.ID), body, 404)
			missing := draftRequest(t, w, who, method, draftPath(uuid.NewV7()), body, 404)
			if !reflect.DeepEqual(problemObject(t, raw), problemObject(t, missing)) {
				t.Fatal(raw, missing)
			}
		}
	}
	// Workspace deletion hides both an attached and an unassigned draft.
	free := createDraftHTTP(t, w, "alice", `{"payload":{}}`)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE workspaces SET deleted_at=now() WHERE id=$1", w.acme); err != nil {
		t.Fatal(err)
	}
	for _, method := range []string{"GET", "PATCH", "DELETE"} {
		body := ""
		if method == "PATCH" {
			body = `{}`
		}
		raw := draftRequest(t, w, "alice", method, draftPath(free.ID), body, 404)
		missing := draftRequest(t, w, "alice", method, draftPath(uuid.NewV7()), body, 404)
		if !reflect.DeepEqual(problemObject(t, raw), problemObject(t, missing)) {
			t.Fatal(raw, missing)
		}
	}
}
func problemObject(t *testing.T, raw string) map[string]any {
	t.Helper()
	var obj map[string]any
	decodeAnswer(t, raw, &obj)
	return obj
}
