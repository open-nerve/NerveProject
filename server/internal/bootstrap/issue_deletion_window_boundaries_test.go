package bootstrap

import (
	"fmt"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func (f deletionWindowFixture) boundaryRequests(t *testing.T, workspace bool) []deletionWindowRequest {
	t.Helper()
	w := f.w
	var out []deletionWindowRequest
	add := func(name, actor, method, path, body string, want int, missingPath, missingBody string, check func(*testing.T, string)) {
		out = append(out, deletionWindowRequest{name, actor, method, path, body, want, missingPath, missingBody, check})
	}
	// Bob owns Q's source but is read-only there. A hidden P endpoint must
	// still produce 404, never reveal the live endpoint's ordinary 403.
	pm := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", f.q, w.ids["bob"])[0]
	w.send(t, "alice", "PATCH", "/api/v0/project-members/"+pm.String(), `{"role":5}`, 200)
	for i, id := range f.relations {
		for _, actor := range []string{"alice", "bob"} {
			add(fmt.Sprintf("relation/reverse-%d/%s", i, actor), actor, "DELETE", relationPath(id), "", 404, relationPath(uuid.NewV7()), "", nil)
		}
	}
	for _, method := range []string{"GET", "PATCH", "DELETE", "POST"} {
		path, missing := draftPath(f.noProject), draftPath(uuid.NewV7())
		body := ""
		want := 404
		var check func(*testing.T, string)
		if method == "PATCH" {
			body = `{"payload":{"name":"Unassigned"}}`
		}
		if method == "POST" {
			path, missing = publishPath(f.noProject), publishPath(uuid.NewV7())
		}
		if !workspace {
			if method == "DELETE" || method == "PATCH" {
				continue
			}
			want = 200
			missing = ""
			if method == "POST" {
				want = 422
				check = windowField("project_id required")
			}
		}
		add("unassigned/"+method, "alice", method, path, body, want, missing, body, check)
	}
	add("old-draft/clear-project", "alice", "PATCH", draftPath(f.x.draft), `{"project_id":null}`, 404, draftPath(uuid.NewV7()), `{"project_id":null}`, nil)
	want := 422
	missing := draftPath(f.noProject)
	missingBody := `{"project_id":"` + uuid.NewV7().String() + `"}`
	check := windowField("project_id not_allowed")
	if workspace {
		want = 404
		missing = draftPath(uuid.NewV7())
		check = nil
	}
	add("live-draft/explicit-deleted-project", "alice", "PATCH", draftPath(f.noProject), `{"project_id":"`+w.web.String()+`"}`, want, missing, missingBody, check)
	for _, kind := range []string{"sub", "relation"} {
		path, field := subIssues(f.kept[0]), "sub_issue_ids"
		prefix := ""
		if kind == "relation" {
			path, field = relationsOf(f.kept[0]), "issue_ids"
			prefix = `"relation_type":"relates_to",`
		}
		body := `{` + prefix + `"` + field + `":` + idArray(f.x.issue, uuid.NewV7()) + `}`
		absentBody := `{` + prefix + `"` + field + `":` + idArray(uuid.NewV7(), uuid.NewV7()) + `}`
		want := 422
		missing := path
		check := windowField(field + "[0] not_allowed; " + field + "[1] not_allowed")
		if workspace {
			want = 404
			missing = strings.Replace(path, f.kept[0].String(), uuid.NewV7().String(), 1)
			check = nil
		}
		add("live-source/hidden-"+kind, "alice", "POST", path, body, want, missing, absentBody, check)
	}
	for _, actor := range []string{"alice", "bob"} {
		path := relationsOf(f.kept[0])
		want := 200
		missing := ""
		check := func(t *testing.T, body string) {
			rows := listIn[answeredRelation](t, body)
			if len(rows) != 1 || rows[0].Issue.ID != f.kept[1] {
				t.Fatal("hidden endpoints leaked or live relation lost", body)
			}
		}
		if workspace {
			want = 404
			missing = relationsOf(uuid.NewV7())
			check = nil
		}
		add("live-relations/"+actor, actor, "GET", path, "", want, missing, "", check)
	}
	// History is authorized by historical membership, independent of deleted_at.
	path := activitiesOf(f.kept[0])
	before := deletionWindowRequest{actor: "alice", method: "GET", path: path, want: 200}.answer(t, w)
	var activity struct{ Data []any }
	decodeAnswer(t, before.body, &activity)
	if len(activity.Data) < 8 {
		t.Fatal("history has no distinguishing references", before.body)
	}
	if workspace {
		add("historical/source-hidden", "alice", "GET", path, "", 404, activitiesOf(uuid.NewV7()), "", nil)
	} else {
		add("historical/deleted-references-retained", "alice", "GET", path, "", 200, "", "", func(t *testing.T, body string) {
			if body != before.body {
				t.Fatal("Historical wrongly filtered deletion", body, before.body)
			}
		})
	}
	for _, query := range []string{"q=Window", "q=Foreign"} {
		path := "/api/v0/workspaces/acme/issue-search?" + query
		want := 200
		missing := ""
		check := windowIDs(f.kept[2], f.kept[1], f.kept[0])
		if query == "q=Foreign" {
			check = windowIDs()
		}
		if workspace {
			want = 404
			missing = strings.Replace(path, "/acme/", "/absent/", 1)
			check = nil
		}
		add("OwnOnly/search/"+query, "bob", "GET", path, "", want, missing, "", check)
	}
	for _, target := range []string{"alice", "bob"} {
		path := memberIssuesPath(w, target) + "?relation=created&order_by=-created_at"
		want, missing := 200, ""
		check := windowPageIDs(3, f.kept[2], f.kept[1], f.kept[0])
		if target == "alice" {
			check = windowPageIDs(0)
		}
		if workspace {
			want = 404
			missing = strings.Replace(path, "/acme/", "/absent/", 1)
			check = nil
		}
		add("OwnOnly/member/"+target, "bob", "GET", path, "", want, missing, "", check)
	}
	if workspace {
		add("unassigned/createDraft", "alice", "POST", draftsPath, `{"payload":{}}`, 404, "/api/v0/workspaces/absent/drafts", `{"payload":{}}`, nil)
	}
	return out
}

// Wanda's personal drafts cannot contaminate Alice's list/cursor comparisons.
// Each phase gets new successful-write resources, never a consumed draft twice.
func (f deletionWindowFixture) unassignedWrites(t *testing.T) {
	t.Helper()
	w := f.w
	d := createDraftHTTP(t, w, "wanda", `{"payload":{"name":"Independent"}}`)
	var got draftHTTP
	decodeAnswer(t, w.send(t, "wanda", "GET", draftPath(d.ID), "", 200), &got)
	if got.ID != d.ID || got.ProjectID != nil {
		t.Fatal("unassigned create/get", got)
	}
	decodeAnswer(t, w.send(t, "wanda", "PATCH", draftPath(d.ID), `{"payload":{"name":"Updated"}}`, 200), &got)
	if got.ID != d.ID || got.ProjectID != nil || got.Payload["name"] != "Updated" {
		t.Fatal("unassigned update", got)
	}
	w.send(t, "wanda", "DELETE", draftPath(d.ID), "", 204)
	var gone bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NOT NULL AND updated_by_id=$2 FROM draft_issues WHERE id=$1", d.ID, w.ids["wanda"]).Scan(&gone); err != nil || !gone {
		t.Fatal("unassigned delete", gone, err)
	}
}
