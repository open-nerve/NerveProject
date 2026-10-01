package httpadapter_test

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	acmeID = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	webID  = uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1")
	// web is a project as its admin sees it, every field set.
	web = domain.Project{ID: webID, WorkspaceID: acmeID, Name: "Web", Description: "The app", Identifier: "WEB", Network: domain.NetworkPrivate,
		LeadID: &bobID, DefaultAssigneeID: &aliceID, CycleView: true, ModuleView: true, IssueViewsView: true, IntakeView: true,
		GuestViewAllFeatures: true, ArchiveIn: 3, ArchivedAt: &created,
		LogoProps: domain.LogoProps{InUse: ptr("icon"), Emoji: &domain.Emoji{Value: ptr("128640"), URL: ptr("https://example.com/e.png")},
			Icon: &domain.Icon{Name: ptr("home"), Color: ptr("#ffffff"), BackgroundColor: ptr("#000000")}},
		Timezone: "Asia/Shanghai", CreatedAt: created, UpdatedAt: created, MemberRole: ptr(shared.RoleAdmin), SortOrder: ptr(-9900.5),
		MemberIDs: []uuid.UUID{aliceID, bobID}}
	// bare is a project with every optional field empty, as a caller who is
	// not its member sees it.
	bare = domain.Project{ID: webID, WorkspaceID: acmeID, Name: "Web", Identifier: "WEB", Network: domain.NetworkPublic, Timezone: "UTC",
		CreatedAt: created, UpdatedAt: created}
)

const (
	webJSON = `{"archive_in":3,"archived_at":"2026-10-01T10:00:00.123456Z","cover_image_url":null,"created_at":"2026-10-01T10:00:00.123456Z",` +
		`"cycle_view":true,"default_assignee_id":"0199a2b4-0000-7000-8000-000000000001","description":"The app","guest_view_all_features":true,` +
		`"id":"0199a2b4-0000-7000-8000-0000000000a1","identifier":"WEB","intake_view":true,"issue_views_view":true,` +
		`"logo_props":{"emoji":{"url":"https://example.com/e.png","value":"128640"},"icon":{"background_color":"#000000","color":"#ffffff","name":"home"},"in_use":"icon"},` +
		`"member_ids":["0199a2b4-0000-7000-8000-000000000001","0199a2b4-0000-7000-8000-000000000002"],"member_role":20,"module_view":true,` +
		`"name":"Web","network":0,"project_lead_id":"0199a2b4-0000-7000-8000-000000000002","sort_order":-9900.5,"timezone":"Asia/Shanghai",` +
		`"updated_at":"2026-10-01T10:00:00.123456Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bareJSON = `{"archive_in":0,"archived_at":null,"cover_image_url":null,"created_at":"2026-10-01T10:00:00.123456Z","cycle_view":false,` +
		`"default_assignee_id":null,"description":"","guest_view_all_features":false,"id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"identifier":"WEB","intake_view":false,"issue_views_view":false,"logo_props":{},"member_ids":[],"member_role":null,"module_view":false,` +
		`"name":"Web","network":2,"project_lead_id":null,"sort_order":null,"timezone":"UTC","updated_at":"2026-10-01T10:00:00.123456Z",` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

func ptr[T any](v T) *T { return &v }

// The body becomes the use case's input for the caller and the path's
// workspace, each optional field nil or empty when absent; the answer is 201
// with the project the use case answers, every field shown, the empty ones
// as null, [] and {}.
func TestCreateProjectAnswers201(t *testing.T) {
	create := &fakeCreate{answer: web}
	h := newServer(t, fakes{create: create})

	res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
		`{"name":"Web","identifier":"web","description":"The app","network":0,"project_lead_id":"0199a2b4-0000-7000-8000-000000000002",`+
			`"logo_props":{"in_use":"icon","emoji":{"value":"128640","url":"https://example.com/e.png"},`+
			`"icon":{"name":"home","color":"#ffffff","background_color":"#000000"}},"timezone":"Asia/Shanghai"}`))
	if res.StatusCode != http.StatusCreated || body != webJSON+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, body, webJSON)
	}
	create.answer = bare
	res, body = do(t, h, request(http.MethodPost, "/api/v0/workspaces/beta/projects", "bob", `{"name":"Web","identifier":"WEB"}`))
	if res.StatusCode != http.StatusCreated || body != bareJSON+"\n" {
		t.Errorf("POST without the optional fields = %d %s, want 201 %s", res.StatusCode, body, bareJSON)
	}

	private := domain.NetworkPrivate
	want := []domain.NewProject{
		{Name: "Web", Identifier: "web", Description: "The app", Network: &private, LeadID: &bobID, Timezone: ptr("Asia/Shanghai"),
			LogoProps: web.LogoProps},
		{Name: "Web", Identifier: "WEB"},
	}
	if !reflect.DeepEqual(create.got, want) {
		t.Errorf("inputs = %+v, want %+v", create.got, want)
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(create.calls, want) {
		t.Errorf("calls = %q, want %q", create.calls, want)
	}
}

// The use case's refusals, as the contract declares them.
func TestCreateProjectRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{"a guest", shared.Forbidden(), http.StatusForbidden,
			`{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{"a lead who may not lead", domain.LeadNotAllowed(), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"project_lead_id","code":"not_allowed","message":"must be an active admin or member of the workspace"}]}`},
		{"a taken identifier", domain.ErrIdentifierTaken, http.StatusConflict,
			`{"status":409,"code":"project.identifier_taken","title":"Conflict","detail":"A project of the workspace has this identifier."}`},
		{"a taken name", domain.ErrNameTaken, http.StatusConflict,
			`{"status":409,"code":"project.name_taken","title":"Conflict","detail":"A project of the workspace has this name."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{create: &fakeCreate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice", `{"name":"Web","identifier":"WEB"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// logo_props is a closed structure (M3 design 5.2): a key it does not have,
// at any depth, or a value of another type is refused as bad_request before
// the use case; an in_use of the right type but no known value is the
// domain's to refuse, and reaches it.
func TestCreateProjectHoldsTheLogoToItsStructure(t *testing.T) {
	create := &fakeCreate{answer: bare}
	h := newServer(t, fakes{create: create})
	for _, logo := range []string{
		`{"shape":"round"}`, `{"emoji":{"value":"1","size":2}}`, `{"icon":{"name":"home","shape":"round"}}`,
		`{"in_use":17}`, `{"emoji":"x"}`, `{"emoji":{"value":1}}`, `{"icon":{"color":null}}`, `[]`, `"x"`,
	} {
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
			`{"name":"Web","identifier":"WEB","logo_props":`+logo+`}`))
		if res.StatusCode != http.StatusBadRequest {
			t.Errorf("logo_props %s: POST = %d %s, want 400", logo, res.StatusCode, body)
		}
	}
	if len(create.got) != 0 {
		t.Fatalf("the use case got %+v, want nothing", create.got)
	}
	res, _ := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
		`{"name":"Web","identifier":"WEB","logo_props":{"in_use":"other"}}`))
	if res.StatusCode != http.StatusCreated || len(create.got) != 1 || create.got[0].LogoProps.InUse == nil || *create.got[0].LogoProps.InUse != "other" {
		t.Errorf("in_use other: POST = %d, inputs %+v; want it passed to the use case", res.StatusCode, create.got)
	}
}
