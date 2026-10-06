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

// fakeCreateLabel is createLabel: each call is recorded as "caller id",
// with what it got; it answers answer, or err.
type fakeCreateLabel struct {
	calls  []string
	got    []domain.LabelCreate
	answer domain.Label
	err    error
}

func (f *fakeCreateLabel) Execute(ctx context.Context, projectID uuid.UUID, in domain.LabelCreate) (domain.Label, error) {
	f.calls = append(f.calls, caller(ctx)+" "+projectID.String())
	f.got = append(f.got, in)
	return f.answer, f.err
}

// web's labels as the use cases answer them: Bug at the top, with a color;
// UI under it, without one, at a sort order with a fraction.
var (
	webBugLabel = domain.Label{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1"), WorkspaceID: acmeID, ProjectID: webID, Name: "Bug",
		Color: "#FF0000", SortOrder: 65535, CreatedAt: created, UpdatedAt: created}
	webUILabel = domain.Label{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d2"), WorkspaceID: acmeID, ProjectID: webID,
		ParentID: &webBugLabel.ID, Name: "UI", SortOrder: 75535.5, CreatedAt: created, UpdatedAt: created.Add(1)}
	webBugLabelJSON = `{"color":"#FF0000","created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000d1","name":"Bug",` +
		`"parent_id":null,"project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":65535,"updated_at":"2026-10-01T10:00:00.123456Z",` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	webUILabelJSON = `{"color":"","created_at":"2026-10-01T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000d2","name":"UI",` +
		`"parent_id":"0199a2b4-0000-7000-8000-0000000000d1","project_id":"0199a2b4-0000-7000-8000-0000000000a1","sort_order":75535.5,` +
		`"updated_at":"2026-10-01T10:00:00.123456001Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// The answers of the label operations, as the contract declares them.
const (
	labelNameTakenJSON = `{"status":409,"code":"project.label_name_taken","title":"Conflict",` +
		`"detail":"A label of the project has this name, in this case or another."}`
	parentRefusedJSON = `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"parent_id","code":"not_allowed","message":"must be a label without a parent: labels have two levels"}]}`
)

// parentRefused is the use case's refusal of a parent with a parent of its
// own, as parentRefusedJSON answers it.
var parentRefused = shared.Invalid(shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed,
	Message: "must be a label without a parent: labels have two levels"})

// POST goes to the use case for the caller and the path's project, with
// the body's fields: the color empty and the parent nil when not given;
// the answer is 201 with the label the use case answers, its parent's id,
// or null at the top.
func TestCreateLabel(t *testing.T) {
	path := "/api/v0/projects/" + webID.String() + "/labels"
	for _, tt := range []struct {
		body   string
		want   domain.LabelCreate
		answer domain.Label
		json   string
	}{
		{`{"name":"UI","color":"#F59E0B","parent_id":"0199a2b4-0000-7000-8000-0000000000d1"}`,
			domain.LabelCreate{Name: "UI", Color: "#F59E0B", ParentID: &webBugLabel.ID}, webUILabel, webUILabelJSON},
		{`{"name":"Bug"}`, domain.LabelCreate{Name: "Bug"}, webBugLabel, webBugLabelJSON},
	} {
		create := &fakeCreateLabel{answer: tt.answer}
		h := newServer(t, fakes{createLabel: create})
		if res, body := do(t, h, request(http.MethodPost, path, "alice", tt.body)); res.StatusCode != http.StatusCreated || body != tt.json+"\n" {
			t.Errorf("POST %s = %d %s, want 201 %s", tt.body, res.StatusCode, body, tt.json)
		}
		if want := []string{"alice " + webID.String()}; !slices.Equal(create.calls, want) || !reflect.DeepEqual(create.got,
			[]domain.LabelCreate{tt.want}) {
			t.Errorf("POST %s: calls %q with %+v; want %q with %+v", tt.body, create.calls, create.got, want, tt.want)
		}
	}
}

// A body without its name, with a field it may not have (a parent named
// so, or a sort order, which the project gives: M3 design 4.10), a field
// of another type, or a parent that is null or no id: refused as
// bad_request before the use case.
func TestCreateLabelHoldsTheBodyToItsStructure(t *testing.T) {
	create := &fakeCreateLabel{}
	h := newServer(t, fakes{createLabel: create})
	for _, body := range []string{`{}`, `{"color":"#000"}`, `{"name":"A","parent":"0199a2b4-0000-7000-8000-0000000000d1"}`, `{"name":1}`,
		`{"name":"A","sort_order":1}`, `{"name":"A","parent_id":null}`, `{"name":"A","parent_id":"Bug"}`} {
		res, got := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/labels", "alice", body))
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
func TestCreateLabelRefusals(t *testing.T) {
	for _, tt := range []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"a parent refused", parentRefused, http.StatusUnprocessableEntity, parentRefusedJSON},
		{"no project", domain.ErrNotFound, http.StatusNotFound, projectNotFoundJSON},
		{"a project member", shared.Forbidden(), http.StatusForbidden, forbiddenJSON},
		{"a name taken", domain.ErrLabelNameTaken, http.StatusConflict, labelNameTakenJSON},
		{"a failure", errGone, http.StatusInternalServerError, internalErrorJSON},
	} {
		h := newServer(t, fakes{createLabel: &fakeCreateLabel{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/projects/"+webID.String()+"/labels", "alice", `{"name":"Bug"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
