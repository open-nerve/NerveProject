package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

// H11's bounds of a work item in the intake through the app (M7 P2a spec
// 2.6, 4), on bob's Login breaks, which he submits to Web, its intake
// turned on. Each of the six writes it refuses, by alice, Web's admin,
// is refused 409 issue.in_triage, writing nothing: its change, archiving,
// unarchiving and deletion, sub-items put under it, relations from it.
// As the other end it is refused 422 with the problem the write answers a
// work item it may not take: the parent of a creation, of Ordinary's
// change and of a draft's publication, a sub-item of Ordinary, a work item
// related to Ordinary. Its collaboration goes on: a comment, a reaction, a
// link and alice's subscription are taken. A search from it is 404
// issue.not_found, as from one not there. With Web archived, its change is
// still refused 409 issue.in_triage before project.archived.
func TestAWorkItemInTheIntakeThroughTheApp(t *testing.T) {
	w := newIssueWorld(t)
	ordinary := w.newWorkItemIn(t, w.web, `{"name":"Ordinary"}`)
	w.intakeOn(t)
	item := w.submit(t, "bob", `{"name":"Login breaks"}`).ID
	path, other := "/api/v0/issues/"+item.String(), "/api/v0/issues/"+ordinary.String()

	t.Run("the writes it refuses", func(t *testing.T) {
		stamps := w.workItemStamps(t, item)
		for _, write := range []struct{ method, path, body string }{
			{http.MethodPatch, path, `{"name":"Mended"}`}, {http.MethodPost, path + "/archive", ""}, {http.MethodPost, path + "/unarchive", ""},
			{http.MethodDelete, path, ""}, {http.MethodPost, path + "/sub-issues", fmt.Sprintf(`{"sub_issue_ids":[%q]}`, ordinary)},
			{http.MethodPost, path + "/relations", fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, ordinary)},
		} {
			w.refusesAs(t, "alice", write.method, write.path, write.body, http.StatusConflict, "issue.in_triage")
		}
		if after := w.workItemStamps(t, item); !slices.Equal(after, stamps) {
			t.Errorf("the refused writes changed its rows:\n%q\nwant\n%q", after, stamps)
		}
	})
	t.Run("as the other end", func(t *testing.T) {
		parent := "validation_failed parent_id not_allowed"
		under := fmt.Sprintf(`{"name":"Under","parent_id":%q}`, item)
		w.refusesAs(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", under, http.StatusUnprocessableEntity, parent)
		w.refusesAs(t, "alice", http.MethodPatch, other, fmt.Sprintf(`{"parent_id":%q}`, item), http.StatusUnprocessableEntity, parent)
		if got := refusalOf(t, []byte(draftRequest(t, w, "alice", http.MethodPost, publishPath(w.draftIn(t, under).ID), "",
			http.StatusUnprocessableEntity)), false); got != parent {
			t.Errorf("the publication of a draft under it = %s, want %s", got, parent)
		}
		w.refusesAs(t, "alice", http.MethodPost, other+"/sub-issues", fmt.Sprintf(`{"sub_issue_ids":[%q]}`, item), http.StatusUnprocessableEntity,
			"validation_failed sub_issue_ids[0] not_allowed")
		w.refusesAs(t, "alice", http.MethodPost, other+"/relations", fmt.Sprintf(`{"relation_type":"blocking","issue_ids":[%q]}`, item),
			http.StatusUnprocessableEntity, "validation_failed issue_ids[0] not_allowed")
	})
	t.Run("its collaboration", func(t *testing.T) {
		w.send(t, "alice", http.MethodPost, path+"/comments", `{"comment_html":"<p>Seen it</p>"}`, http.StatusCreated)
		w.send(t, "alice", http.MethodPost, path+"/reactions", `{"reaction":"128077"}`, http.StatusCreated)
		w.send(t, "alice", http.MethodPost, path+"/links", `{"url":"http://169.254.169.254/intake"}`, http.StatusCreated)
		w.send(t, "alice", http.MethodPost, path+"/subscribe", "", http.StatusNoContent)
	})
	t.Run("a search from it", func(t *testing.T) {
		if got := refusalOf(t, []byte(draftRequest(t, w, "alice", http.MethodGet, searchPath+"?for=relation&issue_id="+item.String(), "",
			http.StatusNotFound)), false); got != "issue.not_found" {
			t.Errorf("a search from it = %s, want issue.not_found", got)
		}
	})
	t.Run("its project archived", func(t *testing.T) {
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", `{}`, http.StatusOK)
		w.refusesAs(t, "alice", http.MethodPatch, path, `{"name":"Mended"}`, http.StatusConflict, "issue.in_triage")
	})
}
