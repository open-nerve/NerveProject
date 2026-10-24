package bootstrap

import (
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"uuid"
)

func TestIssueDescriptionAnswers(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	html := "<p>Saved description</p>"
	var created struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
		HTML       string    `json:"description_html"`
	}
	decodeAnswer(t, w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", fmt.Sprintf(`{"name":"Description","description_html":%q}`, html), 201), &created)
	if created.HTML != html {
		t.Fatal("create lost description", created)
	}
	up := w.newWorkItemIn(t, w.web, `{"name":"Hidden parent"}`)
	path := "/api/v0/issues/" + created.ID.String()
	w.send(t, "alice", http.MethodPatch, path, fmt.Sprintf(`{"parent_id":%q}`, up), 200)
	w.newWorkItemBy(t, "walt", w.web, underParent("Visible child", created.ID))
	w.newWorkItemIn(t, w.web, underParent("Hidden child", created.ID))
	w.send(t, "alice", http.MethodPost, path+"/relations", fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, up), 201)
	for _, step := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""},
		{http.MethodGet, "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(created.SequenceID), ""},
		{http.MethodPatch, path, `{"description_html":"<p>Changed description</p>"}`},
		{http.MethodPatch, path, `{"state_id":"` + w.done.String() + `"}`},
		{http.MethodPost, path + "/archive", ""},
		{http.MethodPost, path + "/unarchive", ""},
	} {
		if step.body == `{"description_html":"<p>Changed description</p>"}` {
			html = "<p>Changed description</p>"
		}
		var got struct {
			HTML   string     `json:"description_html"`
			Parent *uuid.UUID `json:"parent_id"`
			Count  int        `json:"sub_issues_count"`
		}
		decodeAnswer(t, w.send(t, "walt", step.method, step.path, step.body, 200), &got)
		if got.HTML != html || got.Parent != nil || got.Count != 1 {
			t.Fatalf("detail answer bypassed description or seen: %+v", got)
		}
	}
	// Lists keep the smaller Issue shape even when their source carries a description.
	for _, suffix := range []string{"/sub-issues", "/relations"} {
		var page struct {
			Data []map[string]any `json:"data"`
		}
		decodeAnswer(t, w.send(t, "alice", http.MethodGet, path+suffix, "", 200), &page)
		if len(page.Data) == 0 {
			t.Fatal("empty list fixture", suffix)
		}
		for _, row := range page.Data {
			if suffix == "/relations" {
				row = row["issue"].(map[string]any)
			}
			if _, ok := row["description_html"]; ok {
				t.Fatal("description leaked into Issue list", suffix)
			}
		}
	}
}
