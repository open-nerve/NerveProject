package bootstrap

import (
	"net/http"
	"strconv"
	"testing"
	"uuid"
)

// subItems is the number of sub-items an answer shows of the work item it
// is.
func subItems(t *testing.T, answer string) int {
	t.Helper()
	var i struct {
		SubIssuesCount *int `json:"sub_issues_count"`
	}
	decodeAnswer(t, answer, &i)
	if i.SubIssuesCount == nil {
		t.Fatalf("the answer %s has no sub_issues_count", answer)
	}
	return *i.SubIssuesCount
}

// Each answer of a work item shows the number of its sub-items the caller
// sees (M4 design 3.4, 3.9), as bootstrap wires it: its undeleted,
// unarchived direct sub-items, those he created when the guest rule hides
// others' from him. walt, acme's admin and Web's guest (adminGuest), makes
// His; alice puts hers, Part, under it, walt his, Mine, and alice an
// archived one and a deleted one; Low, under Part, is not His. At each
// step walt, then alice, reads His, by its id and by its identifier,
// changes it, changes it in nothing, moves it into Done, archives it twice
// and unarchives it twice, the second of a step finding it done: each
// answer shows walt 1 sub-item, Mine, and alice, who sees them all, 2. A
// new work item has none.
func TestEachAnswerCountsTheSubItemsTheCallerSees(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	var created struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
	}
	decodeAnswer(t, w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"His"}`, http.StatusCreated),
		&created)
	his := created.ID
	part := w.newWorkItemIn(t, w.web, underParent("Part", his))
	w.newWorkItemIn(t, w.web, underParent("Low", part))
	w.newWorkItemBy(t, "walt", w.web, underParent("Mine", his))
	shelved := w.newWorkItemIn(t, w.web, `{"name":"Shelved","state_id":"`+w.done.String()+`","parent_id":"`+his.String()+`"}`)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newWorkItemIn(t, w.web, underParent("Gone", his))
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)

	path := "/api/v0/issues/" + his.String()
	identifier := "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(created.SequenceID)
	for _, step := range []struct{ name, method, path, body string }{
		{"read", http.MethodGet, path, ""},
		{"read by its identifier", http.MethodGet, identifier, ""},
		{"changed", http.MethodPatch, path, `{"name":"Mend"}`},
		{"changed in nothing", http.MethodPatch, path, `{"name":"Mend"}`},
		{"moved into Done", http.MethodPatch, path, `{"state_id":"` + w.done.String() + `"}`},
		{"archived", http.MethodPost, path + "/archive", ""},
		{"archived again", http.MethodPost, path + "/archive", ""},
		{"unarchived", http.MethodPost, path + "/unarchive", ""},
		{"unarchived again", http.MethodPost, path + "/unarchive", ""},
	} {
		for _, by := range []struct {
			name string
			want int
		}{{"walt", 1}, {"alice", 2}} {
			if got := subItems(t, w.send(t, by.name, step.method, step.path, step.body, http.StatusOK)); got != by.want {
				t.Errorf("%s, by %s: %d sub-items, want %d", step.name, by.name, got, by.want)
			}
		}
	}
	answer := w.send(t, "walt", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"New"}`, http.StatusCreated)
	if got := subItems(t, answer); got != 0 {
		t.Errorf("a new work item has %d sub-items, want 0", got)
	}
}
