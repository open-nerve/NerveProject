package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// refusal is a use case's error and the answer it is.
type refusal struct {
	err    error
	status int
	want   string
}

// refusals are every refusal the module's operations declare, by name, and
// a failure.
func refusals(t *testing.T) map[string]refusal {
	t.Helper()
	_, problems := domain.CheckNewView(domain.NewView{Name: " "})
	if len(problems) == 0 {
		t.Fatal("a blank name was not refused")
	}
	return map[string]refusal{
		"invalid": {shared.Invalid(problems...), http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
			`"detail":"The request has invalid values.","errors":[{"field":"name","code":"too_short","message":"must not be empty"}]}`},
		"no project": {domain.ErrProjectNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		"no workspace": {domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		"no view": {domain.ErrViewNotFound, http.StatusNotFound,
			`{"status":404,"code":"navigation.view_not_found","title":"Not Found","detail":"The view does not exist, or you cannot see it."}`},
		"forbidden": {shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		"views off": {domain.ErrViewsDisabled, http.StatusConflict, `{"status":409,"code":"navigation.views_disabled","title":"Conflict",` +
			`"detail":"Views are turned off in this project; turn them on to change them."}`},
		"failure": {errGone, http.StatusInternalServerError, `{"status":500,"code":"internal_error","title":"Internal Server Error"}`},
	}
}

// Each operation answers each refusal its x-problem-codes declare, and a
// failure, as the use case gives it: the problem, its status and its code.
func TestTheRefusalsOfEachOperation(t *testing.T) {
	all := refusals(t)
	for _, op := range []struct {
		method, path, body string
		refusals           []string
	}{
		{http.MethodGet, projectPath, "", []string{"no project", "forbidden", "failure"}},
		{http.MethodPost, projectPath, `{"name":" "}`, []string{"invalid", "no project", "forbidden", "views off", "failure"}},
		{http.MethodGet, workspacePath, "", []string{"no workspace", "forbidden", "failure"}},
		{http.MethodPost, workspacePath, `{"name":" "}`, []string{"invalid", "no workspace", "forbidden", "failure"}},
		{http.MethodGet, viewPath, "", []string{"no view", "forbidden", "failure"}},
		{http.MethodPatch, viewPath, `{"name":" "}`, []string{"invalid", "no view", "forbidden", "views off", "failure"}},
		{http.MethodDelete, viewPath, "", []string{"no view", "forbidden", "views off", "failure"}},
	} {
		for _, name := range op.refusals {
			tt := all[name]
			f := &fakes{err: tt.err}
			res, body := do(t, newServer(t, f.useCases()), request(op.method, op.path, op.body))
			if res.StatusCode != tt.status || body != tt.want+"\n" {
				t.Errorf("%s %s, %s: %d %s, want %d %s", op.method, op.path, name, res.StatusCode, body, tt.status, tt.want)
			}
			if len(f.calls) != 1 {
				t.Errorf("%s %s, %s: the use cases got %+v, want one call", op.method, op.path, name, f.calls)
			}
		}
	}
}

// problemCode is the code of the problem body.
func problemCode(t *testing.T, body string) string {
	t.Helper()
	var p struct{ Code string }
	if err := json.Unmarshal([]byte(body), &p); err != nil {
		t.Fatalf("not a problem: %s", body)
	}
	return p.Code
}

// A body that is not the creation's or the change's structure answers 400
// bad_request before any use case: not an object, a field it does not have
// (sub_issue among the filters, or a field of the view the body does not
// write), null for any field, a value of another type, a display setting
// without each of its keys, and a creation without a name. The values
// (the name's length, the icon's in_use, the words) are the use case's.
func TestABodyHeldToItsStructure(t *testing.T) {
	const display = `"display_filters":{"layout":"list","group_by":null,"order_by":"-created_at","sub_issue":true,"show_empty_groups":true}`
	broken := []string{`[]`, `"Board"`, `{"name":null}`, `{"name":1}`, `{"filters":null}`, `{"filters":[]}`, `{"filters":{"sub_issue":[]}}`,
		`{"filters":{"archived":[]}}`, `{"filters":{"state_id":"started"}}`, `{"filters":{"state_id":[1]}}`, `{"filters":{"state_id":[null]}}`,
		`{"description":null}`, `{"logo_props":null}`, `{"logo_props":{"in_use":"icon","colour":"red"}}`, `{"logo_props":{"icon":{"name":1}}}`,
		`{"display_filters":null}`, `{` + display + `}`, `{"display_properties":{"assignee":true}}`, `{"display_properties":null}`,
		`{"owned_by_id":"0199a2b4-0000-7000-8000-000000000001"}`, `{"project_id":null}`, `{"is_favorite":true}`}
	f := &fakes{}
	h := newServer(t, f.useCases())
	for _, path := range []string{projectPath, workspacePath} {
		for _, body := range append(broken, `{}`, `{"description":"Open bugs"}`) {
			if res, got := do(t, h, request(http.MethodPost, path, body)); res.StatusCode != http.StatusBadRequest || problemCode(t, got) != "bad_request" {
				t.Errorf("POST %s %s = %d %s, want 400 bad_request", path, body, res.StatusCode, got)
			}
		}
	}
	for _, body := range broken {
		if res, got := do(t, h, request(http.MethodPatch, viewPath, body)); res.StatusCode != http.StatusBadRequest || problemCode(t, got) != "bad_request" {
			t.Errorf("PATCH %s = %d %s, want 400 bad_request", body, res.StatusCode, got)
		}
	}
	if len(f.calls) != 0 {
		t.Errorf("the use cases got %+v, want none", f.calls)
	}
}
