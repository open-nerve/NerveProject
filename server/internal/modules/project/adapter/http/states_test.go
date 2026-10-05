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

// fakeCreateState is createState: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeCreateState struct {
	calls  []string
	got    []domain.StateCreate
	answer domain.State
	err    error
}

func (f *fakeCreateState) Execute(ctx context.Context, projectID uuid.UUID, in domain.StateCreate) (domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

var (
	webReview = domain.State{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c2"), WorkspaceID: acmeID, ProjectID: webID, Name: "Review",
		Description: "Waiting for a review", Color: "#F59E0B", Group: domain.GroupStarted, Sequence: 70000.5, CreatedAt: created,
		UpdatedAt: created.Add(1)}
	webReviewJSON = `{"color":"#F59E0B","created_at":"2026-10-01T10:00:00.123456Z","default":false,"description":"Waiting for a review",` +
		`"group":"started","id":"0199a2b4-0000-7000-8000-0000000000c2","name":"Review","project_id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"sequence":70000.5,"updated_at":"2026-10-01T10:00:00.123456001Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	projectNotFoundJSON = `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`
)

// POST goes to the use case for the caller and the path's project, with
// the body's fields, the description empty when not given and the group as
// given, triage too, which the domain refuses; the answer is 201 with the
// state the use case answers.
func TestCreateState(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/states"
	for _, tt := range []struct {
		body string
		want domain.StateCreate
	}{
		{`{"name":"Review","color":"#F59E0B","group":"started","description":"Waiting for a review"}`,
			domain.StateCreate{Name: "Review", Color: "#F59E0B", Group: domain.GroupStarted, Description: "Waiting for a review"}},
		{`{"name":"Intake","color":"#000","group":"triage"}`, domain.StateCreate{Name: "Intake", Color: "#000", Group: domain.GroupTriage}},
	} {
		create := &fakeCreateState{answer: webReview}
		h := newServer(t, fakes{createState: create})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", tt.body)); res.StatusCode != http.StatusCreated ||
			body != webReviewJSON+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, webReviewJSON)
		}
		if want := []string{"alice " + webID.String()}; !slices.Equal(create.calls, want) || !reflect.DeepEqual(create.got,
			[]domain.StateCreate{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, create.calls, create.got, want, tt.want)
		}
	}
}

// A body without its name, its color or its group, with a field it may not
// have, or a field of another type: refused as bad_request before the use
// case.
func TestCreateStateHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreateState{}
	h := newServer(t, fakes{createState: create})
	for _, body := range []string{`{"color":"#000","group":"started"}`, `{"name":"A","group":"started"}`, `{"name":"A","color":"#000"}`,
		`{"name":"A","color":"#000","group":"started","default":true}`, `{"name":"A","color":"#000","group":"started","sequence":1}`,
		`{"name":1,"color":"#000","group":"started"}`, `{"name":"A","color":"#000","group":1}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/states", "alice", body))
		var problem struct{ Code string }
		if err := json.Unmarshal([]byte(got), &problem); err != nil || res.StatusCode != http.StatusBadRequest || problem.Code != "bad_request" {
			t.Errorf("POST %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(create.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", create.calls)
	}
}

// The use case's refusals, as the contract declares them, and its failure.
func TestCreateStateRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"the triage group", shared.Invalid(shared.FieldError{Field: "group", Code: shared.FieldNotAllowed,
			Message: "must not be triage: the intake's state is not made here"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"group","code":"not_allowed","message":"must not be triage: the intake's state is not made here"}]}`},
		{"no project", domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrStateNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.state_name_taken","title":"Conflict","detail":"A state of the project has this name."}`},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{createState: &fakeCreateState{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/states", "alice",
			`{"name":"Review","color":"#F59E0B","group":"started"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
