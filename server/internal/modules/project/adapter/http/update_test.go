package httpadapter_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The fields the body names go to the use case for the caller and the
// path's project, as written: a field left out nil; the lead and the
// default assignee set when named, to none when null, unset when left out.
// The answer is 200 with the project the use case answers.
func TestUpdateProjectPassesThePatch(t *testing.T) {
	update := &fakeUpdate{answer: web}
	h := newServer(t, fakes{update: update})
	path := "/api/v0/projects/" + webID.String()
	for _, tt := range []struct {
		token, body string
	}{
		{"alice", `{"name":"Site","identifier":"site","description":"The site","network":0,"project_lead_id":"0199a2b4-0000-7000-8000-000000000002",` +
			`"default_assignee_id":null,"cycle_view":true,"module_view":false,"issue_views_view":true,"intake_view":false,` +
			`"guest_view_all_features":true,"archive_in":13,"logo_props":{"in_use":"emoji"},"timezone":"Asia/Shanghai"}`},
		{"bob", `{}`},
		{"bob", `{"project_lead_id":null,"default_assignee_id":"0199a2b4-0000-7000-8000-000000000001"}`},
		{"bob", `{"cycle_view":false,"intake_view":true,"guest_view_all_features":true}`},
	} {
		if res, body := do(t, h, request(http.MethodPatch, path, tt.token, tt.body)); res.StatusCode != http.StatusOK || body != webJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", tt.body, res.StatusCode, body, webJSON)
		}
	}
	private := domain.NetworkPrivate
	want := []domain.ProjectPatch{
		{Name: ptr("Site"), Identifier: ptr("site"), Description: ptr("The site"), Network: &private, SetLead: true, LeadID: &bobID,
			SetDefaultAssignee: true, CycleView: ptr(true), ModuleView: ptr(false), IssueViewsView: ptr(true), IntakeView: ptr(false),
			GuestViewAllFeatures: ptr(true), ArchiveIn: ptr(13), LogoProps: &domain.LogoProps{InUse: ptr("emoji")}, Timezone: ptr("Asia/Shanghai")},
		{},
		{SetLead: true, SetDefaultAssignee: true, DefaultAssigneeID: &aliceID},
		{CycleView: ptr(false), IntakeView: ptr(true), GuestViewAllFeatures: ptr(true)},
	}
	if !reflect.DeepEqual(update.got, want) {
		t.Errorf("inputs = %+v, want %+v", update.got, want)
	}
	if want := []string{"alice " + webID.String(), "bob " + webID.String(), "bob " + webID.String(), "bob " + webID.String()}; !slices.Equal(update.calls, want) {
		t.Errorf("calls = %q, want %q", update.calls, want)
	}
}

// A field the caller may not write, a value of another type, a null where
// only the lead and the default assignee take one: refused as bad_request
// before the use case.
func TestUpdateProjectHoldsTheBodyToItsStructure(t *testing.T) {
	update := &fakeUpdate{answer: web}
	h := newServer(t, fakes{update: update})
	for _, body := range []string{`{"archived_at":null}`, `{"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`, `{"member_role":20}`,
		`{"id":"0199a2b4-0000-7000-8000-0000000000a1"}`, `{"name":null}`, `{"archive_in":"3"}`, `{"project_lead_id":"bob"}`,
		`{"logo_props":{"shape":"round"}}`, `[]`} {
		if res, got := do(t, h, request(http.MethodPatch, "/api/v0/projects/"+webID.String(), "alice", body)); res.StatusCode != http.StatusBadRequest {
			t.Errorf("PATCH %s = %d %s, want 400", body, res.StatusCode, got)
		}
	}
	if len(update.calls) != 0 {
		t.Errorf("the use case got %q, want nothing", update.calls)
	}
}

// The use case's refusals, as the contract declares them.
func TestUpdateProjectRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no project", domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
		{"a project member", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"archived", domain.ErrArchived, http.StatusConflict,
			`{"status":409,"code":"project.archived","title":"Conflict","detail":"The project is archived; unarchive it to change it."}`},
		{"a lead and a default assignee who may not be", shared.Invalid(domain.Unassignable("project_lead_id"),
			domain.Unassignable("default_assignee_id")), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"project_lead_id","code":"not_allowed","message":"must be an active member of the project who is not its guest"},` +
				`{"field":"default_assignee_id","code":"not_allowed","message":"must be an active member of the project who is not its guest"}]}`},
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict,
			`{"status":409,"code":"project.identifier_taken","title":"Conflict","detail":"A project of the workspace has this identifier."}`},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{update: &fakeUpdate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPatch, "/api/v0/projects/"+webID.String(), "alice", `{"name":"Site"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: PATCH = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}
