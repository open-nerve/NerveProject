package httpadapter_test

import (
	"context"
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeUpdateState is updateState: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeUpdateState struct {
	calls  []string
	got    []domain.StatePatch
	answer domain.State
	err    error
}

func (f *fakeUpdateState) Execute(ctx context.Context, id uuid.UUID, p domain.StatePatch) (domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+id.String())
	f.got = append(f.got, p)
	return f.answer, f.err
}

// The answers of the writes on a state, as the contract declares them.
const (
	stateNotFoundJSON = `{"status":404,"code":"project.state_not_found","title":"Not Found",` +
		`"detail":"The state does not exist, cannot be changed through this API, or you cannot see its project."}`
	stateLastInGroupJSON = `{"status":409,"code":"project.state_last_in_group","title":"Conflict",` +
		`"detail":"The state is the only one of its group, and every group keeps one; add another to the group first."}`
	stateDefaultJSON = `{"status":409,"code":"project.state_default","title":"Conflict",` +
		`"detail":"The default state cannot be deleted; make another state the default first."}`
	stateNameTakenJSON = `{"status":409,"code":"project.state_name_taken","title":"Conflict",` +
		`"detail":"A state of the project has this name."}`
)

// PATCH goes to the use case for the caller and the path's state, with the
// fields the body gives, a group as given, and nil for each it does not;
// the answer is 200 with the state the use case answers.
func TestUpdateState(t *testing.T) {
	path := "/api/v0/states/" + webReview.ID.String()
	for _, tt := range []struct {
		body string
		want domain.StatePatch
	}{
		{`{}`, domain.StatePatch{}},
		{`{"name":"Review","color":"#F59E0B","group":"started","description":"Waiting for a review","sequence":70000.5}`,
			domain.StatePatch{Name: ptr("Review"), Color: ptr("#F59E0B"), Group: ptr(domain.GroupStarted), Description: ptr("Waiting for a review"),
				Sequence: ptr(70000.5)}},
		{`{"group":"triage"}`, domain.StatePatch{Group: ptr(domain.GroupTriage)}},
		{`{"sequence":-1}`, domain.StatePatch{Sequence: ptr(-1.0)}},
	} {
		update := &fakeUpdateState{answer: webReview}
		h := newServer(t, fakes{updateState: update})
		if res, body := do(t, h, request(http.MethodPatch, path, "alice", tt.body)); res.StatusCode != http.StatusOK || body != webReviewJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webReviewJSON)
		}
		if want := []string{"alice " + webReview.ID.String()}; !slices.Equal(update.calls, want) || !reflect.DeepEqual(update.got,
			[]domain.StatePatch{tt.want}) {
			t.Errorf("PATCH %s: calls %q with %+v; want %q with %+v", tt.body, update.calls, update.got, want, tt.want)
		}
	}
}

// A body with a field it may not have, a field of another type, or null:
// refused as bad_request before the use case.
func TestUpdateStateHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdateState{}
	h := newServer(t, fakes{updateState: update})
	for _, body := range []string{`{"default":true}`, `{"project_id":"0199a2b4-0000-7000-8000-0000000000a1"}`, `{"name":1}`, `{"sequence":"1"}`,
		`{"name":null}`, `{"group":null}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/states/"+webReview.ID.String(), "alice", body))
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
func TestUpdateStateRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"an empty name", shared.Invalid(shared.FieldError{Field: "name", Code: shared.FieldTooShort, Message: "must not be empty"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
				`"detail":"The request has invalid values.","errors":[{"field":"name","code":"too_short","message":"must not be empty"}]}`},
		{"no state", domain.ErrStateNotFound, http.StatusNotFound, stateNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrStateNameTaken, http.StatusConflict, stateNameTakenJSON},
		{"its group's only state moved", domain.ErrStateLastInGroup, http.StatusConflict, stateLastInGroupJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{updateState: &fakeUpdateState{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/states/"+webReview.ID.String(), "alice", `{"name":""}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// DELETE goes to the use case for the caller and the path's state, and
// answers 204 with no body; the use case's refusals, as the contract
// declares them, and its failure.
func TestDeleteState(t *testing.T) {
	remove := &fakeDelete{}
	h := newServer(t, fakes{deleteState: remove})
	path := "/api/v0/states/" + webReview.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webReview.ID.String()}; !slices.Equal(remove.calls, want) {
		t.Errorf("calls = %q, want %q", remove.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no state", domain.ErrStateNotFound, http.StatusNotFound, stateNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"the default", domain.ErrStateDefault, http.StatusConflict, stateDefaultJSON},
		{"its group's only state", domain.ErrStateLastInGroup, http.StatusConflict, stateLastInGroupJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{deleteState: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: DELETE = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// POST mark-default goes to the use case for the caller and the path's
// state, and answers 204 with no body; the use case's refusals, as the
// contract declares them, and its failure.
func TestMarkDefaultState(t *testing.T) {
	mark := &fakeDelete{}
	h := newServer(t, fakes{markDefault: mark})
	path := "/api/v0/states/" + webReview.ID.String() + "/mark-default"
	if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("POST = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"alice " + webReview.ID.String()}; !slices.Equal(mark.calls, want) {
		t.Errorf("calls = %q, want %q", mark.calls, want)
	}
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no state", domain.ErrStateNotFound, http.StatusNotFound, stateNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{markDefault: &fakeDelete{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
