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

// fakeListStates is listStates: each call is recorded as "caller id"; it
// answers list, or err.
type fakeListStates struct {
	calls []string
	list  []domain.State
	err   error
}

func (f *fakeListStates) Execute(ctx context.Context, projectID uuid.UUID) ([]domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	return f.list, f.err
}

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
	webBacklog = domain.State{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"), WorkspaceID: acmeID, ProjectID: webID, Name: "Backlog",
		Color: "#60646C", Group: domain.GroupBacklog, Default: true, Sequence: 15000, CreatedAt: created, UpdatedAt: created}
	webReview = domain.State{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c2"), WorkspaceID: acmeID, ProjectID: webID, Name: "Review",
		Description: "Waiting for a review", Color: "#F59E0B", Group: domain.GroupStarted, Sequence: 70000.5, CreatedAt: created,
		UpdatedAt: created.Add(1)}
	webBacklogJSON = `{"color":"#60646C","created_at":"2026-10-01T10:00:00.123456Z","default":true,"description":"","group":"backlog",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000c1","name":"Backlog","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sequence":15000,` +
		`"updated_at":"2026-10-01T10:00:00.123456Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	webReviewJSON = `{"color":"#F59E0B","created_at":"2026-10-01T10:00:00.123456Z","default":false,"description":"Waiting for a review",` +
		`"group":"started","id":"0199a2b4-0000-7000-8000-0000000000c2","name":"Review","project_id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"sequence":70000.5,"updated_at":"2026-10-01T10:00:00.123456001Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	projectNotFoundJSON = `{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`
)

// GET goes to the use case for the caller and the path's project; the
// answer is 200 with its list, in its order, every field of each state,
// and [] for none. Its refusals and its failure, as the contract declares
// them.
func TestListStates(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/states"
	for _, tt := range []struct {
		list []domain.State
		want string
	}{{[]domain.State{webReview, webBacklog}, `{"data":[` + webReviewJSON + `,` + webBacklogJSON + `]}`}, {nil, `{"data":[]}`}} {
		list := &fakeListStates{list: tt.list}
		h := newServer(t, fakes{listStates: list})
		if res, body := do(t, h, request(http.MethodGet, path, "bob", "")); res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob " + webID.String()}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{listStates: &fakeListStates{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, path, "alice", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}

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

// fakeListWorkspaceStates is listWorkspaceStates: each call is recorded
// as "caller slug"; it answers list, or err.
type fakeListWorkspaceStates struct {
	calls []string
	list  []domain.State
	err   error
}

func (f *fakeListWorkspaceStates) Execute(ctx context.Context, slug string) ([]domain.State, error) {
	f.calls = append(f.calls, caller(ctx)+" "+slug)
	return f.list, f.err
}

// GET goes to the use case for the caller and the path's workspace; the
// answer is 200 with its list, in its order, every field of each state,
// and [] for none. Its refusal and its failure, as the contract declares
// them.
func TestListWorkspaceStates(t *testing.T) {
	for _, tt := range []struct {
		list []domain.State
		want string
	}{{[]domain.State{webReview, webBacklog}, `{"data":[` + webReviewJSON + `,` + webBacklogJSON + `]}`}, {nil, `{"data":[]}`}} {
		list := &fakeListWorkspaceStates{list: tt.list}
		h := newServer(t, fakes{listWorkspaceStates: list})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/states", "bob", "")); res.StatusCode != http.StatusOK ||
			body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want 200 %s", res.StatusCode, body, tt.want)
		}
		if want := []string{"bob acme"}; !slices.Equal(list.calls, want) {
			t.Errorf("calls = %q, want %q", list.calls, want)
		}
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrWorkspaceNotFound, http.StatusNotFound, workspaceNotFoundJSON},
		{errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{listWorkspaceStates: &fakeListWorkspaceStates{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/states", "bob", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("GET answering %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
