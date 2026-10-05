package httpadapter_test

import (
	"encoding/json"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
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
		Timezone: "Asia/Shanghai", CreatedAt: created, UpdatedAt: created.Add(time.Hour), MemberRole: ptr(shared.RoleAdmin), SortOrder: ptr(-9900.5),
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
		`"updated_at":"2026-10-01T11:00:00.123456Z","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bareJSON = `{"archive_in":0,"archived_at":null,"cover_image_url":null,"created_at":"2026-10-01T10:00:00.123456Z","cycle_view":false,` +
		`"default_assignee_id":null,"description":"","guest_view_all_features":false,"id":"0199a2b4-0000-7000-8000-0000000000a1",` +
		`"identifier":"WEB","intake_view":false,"issue_views_view":false,"logo_props":{},"member_ids":[],"member_role":null,"module_view":false,` +
		`"name":"Web","network":2,"project_lead_id":null,"sort_order":null,"timezone":"UTC","updated_at":"2026-10-01T10:00:00.123456Z",` +
		`"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	// workspaceNotFoundJSON is the answer for a workspace that is not
	// there, or of which the caller is not an active member.
	workspaceNotFoundJSON = `{"status":404,"code":"workspace.not_found","title":"Not Found",` +
		`"detail":"The workspace does not exist, or you are not a member of it."}`
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
	if res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice",
		`{"name":"Web","identifier":"WEB","network":2}`)); res.StatusCode != http.StatusCreated {
		t.Errorf("POST with network 2 = %d %s, want 201", res.StatusCode, body)
	}

	private, public := domain.NetworkPrivate, domain.NetworkPublic
	want := []domain.NewProject{
		{Name: "Web", Identifier: "web", Description: "The app", Network: &private, LeadID: &bobID, Timezone: ptr("Asia/Shanghai"),
			LogoProps: web.LogoProps},
		{Name: "Web", Identifier: "WEB"},
		{Name: "Web", Identifier: "WEB", Network: &public},
	}
	if !reflect.DeepEqual(create.got, want) {
		t.Errorf("inputs = %+v, want %+v", create.got, want)
	}
	if want := []string{"alice acme", "bob beta", "alice acme"}; !slices.Equal(create.calls, want) {
		t.Errorf("calls = %q, want %q", create.calls, want)
	}
}

// Each switch of the answer is its own field's, and member_role the
// caller's role as it is: bare with one switch on answers that key alone
// true, and each role answers its own number.
func TestCreateProjectAnswersEachSwitchAndRoleAsItIs(t *testing.T) {
	answer := func(p domain.Project) map[string]any {
		t.Helper()
		h := newServer(t, fakes{create: &fakeCreate{answer: p}})
		_, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/projects", "alice", `{"name":"Web","identifier":"WEB"}`))
		var got map[string]any
		if err := json.Unmarshal([]byte(body), &got); err != nil {
			t.Fatalf("POST answered %s: %v", body, err)
		}
		return got
	}
	switches := []string{"cycle_view", "module_view", "issue_views_view", "intake_view", "guest_view_all_features"}
	for i, on := range []func(*domain.Project){
		func(p *domain.Project) { p.CycleView = true },
		func(p *domain.Project) { p.ModuleView = true },
		func(p *domain.Project) { p.IssueViewsView = true },
		func(p *domain.Project) { p.IntakeView = true },
		func(p *domain.Project) { p.GuestViewAllFeatures = true },
	} {
		p := bare
		on(&p)
		got := answer(p)
		for j, key := range switches {
			if got[key] != (i == j) {
				t.Errorf("with %s alone on, the answer's %s is %v", switches[i], key, got[key])
			}
		}
	}
	for _, role := range []shared.Role{shared.RoleGuest, shared.RoleMember, shared.RoleAdmin} {
		p := bare
		p.MemberRole = &role
		if got := answer(p)["member_role"]; got != float64(role) {
			t.Errorf("with the role %d, the answer's member_role is %v", role, got)
		}
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
		{"no workspace", domain.ErrWorkspaceNotFound, http.StatusNotFound, workspaceNotFoundJSON},
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

// The path's id goes to the use case for the caller; a project it does not
// find, or the caller does not see, is project.not_found.
func TestGetProject(t *testing.T) {
	get := &fakeGet{projects: map[string]domain.Project{"alice " + webID.String(): web, "bob " + webID.String(): bare}}
	h := newServer(t, fakes{get: get})
	tests := []struct {
		token, id string
		status    int
		want      string
	}{
		{"alice", webID.String(), http.StatusOK, webJSON},
		{"bob", webID.String(), http.StatusOK, bareJSON},
		{"alice", acmeID.String(), http.StatusNotFound,
			`{"status":404,"code":"project.not_found","title":"Not Found","detail":"The project does not exist, or you cannot see it."}`},
	}
	for _, tt := range tests {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/projects/"+tt.id, tt.token, ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s GET %s = %d %s, want %d %s", tt.token, tt.id, res.StatusCode, body, tt.status, tt.want)
		}
	}
	if want := []string{"alice " + webID.String(), "bob " + webID.String(), "alice " + acmeID.String()}; !slices.Equal(get.calls, want) {
		t.Errorf("calls = %q, want %q", get.calls, want)
	}
}

// The path's workspace and identifier, decoded, go to the use case for the
// caller, the identifier as it is written: its case and its spaces, outer
// and inner, are the use case's to judge. The answer is its availability,
// and its refusals as the contract declares them.
func TestCheckProjectIdentifier(t *testing.T) {
	check := &fakeCheck{available: map[string]bool{"NEW": true, "ÇAY": true}}
	h := newServer(t, fakes{check: check})
	for path, want := range map[string]string{
		"/api/v0/workspaces/acme/project-identifiers/NEW":       `{"available":true}`,
		"/api/v0/workspaces/acme/project-identifiers/WEB":       `{"available":false}`,
		"/api/v0/workspaces/acme/project-identifiers/%C3%87AY":  `{"available":true}`,
		"/api/v0/workspaces/acme/project-identifiers/%20we%20b": `{"available":false}`,
	} {
		res, body := do(t, h, request(http.MethodGet, path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", path, res.StatusCode, body, want)
		}
	}
	if want := []string{"alice acme NEW", "alice acme WEB", "alice acme ÇAY", "alice acme " + " we b"}; !slices.Equal(
		slices.Sorted(slices.Values(check.calls)), slices.Sorted(slices.Values(want))) {
		t.Errorf("calls = %q, want %q", check.calls, want)
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrWorkspaceNotFound, http.StatusNotFound, workspaceNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{check: &fakeCheck{err: tt.err}})
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/project-identifiers/NEW", "bob", ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("GET = %d %s, want %d %s", res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// The caller, the path's workspace and archived, false when absent, go to
// the use case; the answer is its list, empty as [].
func TestListProjects(t *testing.T) {
	list := &fakeList{lists: map[string][]domain.Project{"alice": {web, bare}}}
	h := newServer(t, fakes{list: list})
	for _, tt := range []struct {
		token, query string
		want         string
	}{
		{"alice", "", `{"data":[` + webJSON + `,` + bareJSON + `]}`},
		{"bob", "?archived=true", `{"data":[]}`},
		{"bob", "?archived=false", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/projects"+tt.query, tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s = %d %s, want 200 %s", tt.token, tt.query, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice acme false", "bob acme true", "bob acme false"}; !slices.Equal(list.calls, want) {
		t.Errorf("calls = %q, want %q", list.calls, want)
	}
	h = newServer(t, fakes{list: &fakeList{err: domain.ErrWorkspaceNotFound}})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/projects", "alice", ""))
	if res.StatusCode != http.StatusNotFound || body != workspaceNotFoundJSON+"\n" {
		t.Errorf("GET = %d %s, want 404 %s", res.StatusCode, body, workspaceNotFoundJSON)
	}
}
