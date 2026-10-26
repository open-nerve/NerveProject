package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Every request publisher stays silent for hidden resources. Check the complete
// response against a nonexistent target, including the child-resource routes.
func TestIssueEventRefusalsAreUnobservable(t *testing.T) {
	w := newIssueWorld(t)
	id, other := w.newWorkItem(t), w.newWorkItem(t)
	comment := w.newComment(t, "alice", id)
	w.send(t, "alice", http.MethodPost, relationsOf(id), relating("relates_to", other), 201)
	relation := queryIDs(t, w.pool, "SELECT id FROM issue_relations WHERE issue_id=$1 OR related_issue_id=$1", id)[0]
	var events []shared.Event
	var failure error
	clock := &eventCountingClock{base: clockAt(time.Date(2044, 3, 31, 10, 0, 0, 0, time.UTC))}
	route := issueEventRouteFor(t, w, clock, &events, &failure, issue.EventTypes(), nil)
	cases := []struct {
		name, method, body string
		id                 uuid.UUID
		path               func(uuid.UUID) string
	}{
		{"create", http.MethodPost, `{"name":"Hidden"}`, w.web, func(id uuid.UUID) string { return "/api/v0/projects/" + id.String() + "/issues" }},
		{"update", http.MethodPatch, `{"name":"Hidden"}`, id, func(id uuid.UUID) string { return "/api/v0/issues/" + id.String() }},
		{"delete", http.MethodDelete, "", id, func(id uuid.UUID) string { return "/api/v0/issues/" + id.String() }},
		{"archive", http.MethodPost, "", id, func(id uuid.UUID) string { return "/api/v0/issues/" + id.String() + "/archive" }},
		{"unarchive", http.MethodPost, "", id, func(id uuid.UUID) string { return "/api/v0/issues/" + id.String() + "/unarchive" }},
		{"subissues", http.MethodPost, subIssueIDs(other), id, subIssues},
		{"relation add", http.MethodPost, relating("relates_to", other), id, relationsOf},
		{"relation remove", http.MethodDelete, "", relation, relationPath},
		{"comment create", http.MethodPost, `{"comment_html":"<p>Hidden</p>"}`, id, commentsOf},
		{"comment update", http.MethodPatch, `{"comment_html":"<p>Hidden</p>"}`, comment.ID, commentPath},
		{"comment delete", http.MethodDelete, "", comment.ID, commentPath},
		{"issue reaction add", http.MethodPost, `{"reaction":"128077"}`, id, reactionsOf},
		{"issue reaction remove", http.MethodDelete, "", id, func(id uuid.UUID) string { return reactionsOf(id) + "/128077" }},
		{"comment reaction add", http.MethodPost, `{"reaction":"128077"}`, comment.ID, commentReactionsOf},
		{"comment reaction remove", http.MethodDelete, "", comment.ID, func(id uuid.UUID) string { return commentReactionsOf(id) + "/128077" }},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			req, hidden := route.send(c.method, c.path(c.id), w.ids["olga"], c.body)
			w.contract.CheckResponse(t, req, hidden.Result())
			req, missing := route.send(c.method, c.path(uuid.NewV7()), w.ids["olga"], c.body)
			w.contract.CheckResponse(t, req, missing.Result())
			if hidden.Code != 404 || missing.Code != 404 || hidden.Body.String() != missing.Body.String() || !reflect.DeepEqual(hidden.Header(), missing.Header()) {
				t.Errorf("hidden differs from missing: %d %s %v / %d %s %v", hidden.Code, hidden.Body, hidden.Header(), missing.Code, missing.Body, missing.Header())
			}
			if len(events) != 0 || issueEventCount(t, w) != 0 || clock.calls.Load() != 0 {
				t.Fatal("refusal published, inserted a job or read clock")
			}
		})
	}
}
