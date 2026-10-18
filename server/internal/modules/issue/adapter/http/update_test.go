package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// PATCH goes to the use case for the caller and the path's work item, with
// the body's fields, each nil when not given, a list given empty given; the
// answer is 200 with the work item the use case answers, as POST shows it.
func TestUpdateIssue(t *testing.T) {
	path := "/api/v0/issues/" + webSeven.ID.String()
	name, high, none := " Mend ", domain.PriorityHigh, []uuid.UUID{}
	for _, tt := range []struct {
		body string
		want domain.IssuePatch
	}{
		{`{"name":" Mend ","state_id":"0199a2b4-0000-7000-8000-0000000000c1","priority":"high",` +
			`"assignee_ids":["0199a2b4-0000-7000-8000-000000000002"],"label_ids":["0199a2b4-0000-7000-8000-0000000000d1"]}`,
			domain.IssuePatch{Name: &name, StateID: &stateID, Priority: &high, AssigneeIDs: &[]uuid.UUID{bobID}, LabelIDs: &[]uuid.UUID{bugID}}},
		{`{}`, domain.IssuePatch{}},
		{`{"assignee_ids":[],"label_ids":[]}`, domain.IssuePatch{AssigneeIDs: &none, LabelIDs: &none}},
	} {
		update := &fakeUpdate{answer: webSeven}
		res, body := do(t, newServer(t, httpadapter.UseCases{UpdateIssue: update}), request(http.MethodPatch, path, "alice", tt.body))
		if res.StatusCode != http.StatusOK || body != webSevenJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webSevenJSON)
		}
		if want := []string{aliceID.String() + " " + webSeven.ID.String()}; !slices.Equal(update.calls, want) ||
			!reflect.DeepEqual(update.got, []domain.IssuePatch{tt.want}) {
			t.Errorf("PATCH %s: calls %q with %+v; want %q with %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have (a number, which the project gives;
// a sort order, a parent, a description, a date, which IssueUpdate has not
// in P1b: M4/P1b spec 3), null for a field that is not nullable, a field of
// another type, or an id that is no id: refused as bad_request before the
// use case.
func TestUpdateIssueHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdate{}
	h := newServer(t, httpadapter.UseCases{UpdateIssue: update})
	for _, body := range []string{`{"sequence_id":3}`, `{"sort_order":1}`, `{"parent_id":null}`, `{"description_html":"<p>x</p>"}`,
		`{"start_date":"2026-10-18"}`, `{"name":null}`, `{"state_id":null}`, `{"assignee_ids":null}`, `{"name":1}`, `{"state_id":"Backlog"}`,
		`{"assignee_ids":"bob"}`, `{"label_ids":["Bug"]}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/issues/"+webSeven.ID.String(), "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestUpdateIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a label refused", shared.Invalid(domain.UnknownLabel(0)), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"label_ids[0]","code":"not_allowed","message":"must be a label of the project"}]}`},
		{"no work item", domain.ErrIssueNotFound, http.StatusNotFound,
			`{"status":404,"code":"issue.not_found","title":"Not Found","detail":"The work item does not exist, or you cannot see it."}`},
		{"a project guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"an archived project", domain.ErrProjectArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a failure", errGone, http.StatusInternalServerError,
			`{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	} {
		h := newServer(t, httpadapter.UseCases{UpdateIssue: &fakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/issues/"+webSeven.ID.String(), "alice", `{"name":"Mend"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
