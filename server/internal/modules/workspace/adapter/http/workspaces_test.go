package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	acmeID = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000a")
	betaID = uuid.MustParse("0199a2b4-0000-7000-8000-00000000000b")
	size   = "2-10"
	acme   = domain.Workspace{ID: acmeID, Name: "Acme", Slug: "acme", OrganizationSize: &size, Timezone: "Asia/Shanghai",
		Role: shared.RoleAdmin, TotalMembers: 3, CreatedAt: created, UpdatedAt: created.Add(time.Hour)}
	beta = domain.Workspace{ID: betaID, Name: "Beta", Slug: "beta", Timezone: "UTC", Role: shared.RoleGuest, TotalMembers: 1,
		CreatedAt: created, UpdatedAt: created}
)

const (
	acmeJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-00000000000a","logo_url":null,` +
		`"name":"Acme","organization_size":"2-10","role":20,"slug":"acme","timezone":"Asia/Shanghai","total_members":3,` +
		`"updated_at":"2026-09-29T11:00:00.123456Z"}`
	betaJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-00000000000b","logo_url":null,` +
		`"name":"Beta","organization_size":null,"role":5,"slug":"beta","timezone":"UTC","total_members":1,` +
		`"updated_at":"2026-09-29T10:00:00.123456Z"}`
)

// The list is the caller's, as the use case answers it: two callers.
func TestListWorkspacesAnswersTheCallersList(t *testing.T) {
	list := &fakeList{lists: map[string][]domain.Workspace{"alice": {acme, beta}, "bob": nil}}
	h := newServer(t, fakes{list: list})
	for token, want := range map[string]string{
		"alice": `{"data":[` + acmeJSON + `,` + betaJSON + `]}` + "\n",
		"bob":   `{"data":[]}` + "\n",
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces", token, ""))
		if res.StatusCode != http.StatusOK || body != want {
			t.Errorf("%s: GET /api/v0/workspaces = %d %s, want 200 %s", token, res.StatusCode, body, want)
		}
	}
	if want := []string{"alice", "bob"}; !slices.Equal(slices.Sorted(slices.Values(list.calls)), want) {
		t.Errorf("callers = %q, want %q", list.calls, want)
	}
}

// The body becomes the use case's input, the optional fields nil when
// absent; the answer is 201 with the workspace.
func TestCreateWorkspaceAnswers201(t *testing.T) {
	create := &fakeCreate{answer: acme}
	h := newServer(t, fakes{create: create})

	res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces", "bob",
		`{"name":"Acme","slug":"acme","organization_size":"2-10","timezone":"Asia/Shanghai"}`))
	if res.StatusCode != http.StatusCreated || body != acmeJSON+"\n" {
		t.Errorf("POST = %d %s, want 201 %s", res.StatusCode, body, acmeJSON)
	}
	res, _ = do(t, h, request(http.MethodPost, "/api/v0/workspaces", "alice", `{"name":"Beta","slug":"beta"}`))
	if res.StatusCode != http.StatusCreated {
		t.Errorf("POST without the optional fields = %d, want 201", res.StatusCode)
	}

	zone := "Asia/Shanghai"
	want := []domain.NewWorkspace{{Name: "Acme", Slug: "acme", OrganizationSize: &size, Timezone: &zone}, {Name: "Beta", Slug: "beta"}}
	if len(create.got) != 2 || !sameInput(create.got[0], want[0]) || !sameInput(create.got[1], want[1]) {
		t.Errorf("inputs = %+v, want %+v", create.got, want)
	}
	if !slices.Equal(create.callers, []string{"bob", "alice"}) {
		t.Errorf("callers = %q, want bob, then alice", create.callers)
	}
}

func sameInput(a, b domain.NewWorkspace) bool {
	same := func(x, y *string) bool { return (x == nil) == (y == nil) && (x == nil || *x == *y) }
	return a.Name == b.Name && a.Slug == b.Slug && same(a.OrganizationSize, b.OrganizationSize) && same(a.Timezone, b.Timezone)
}

// The use case's refusals, as the contract declares them.
func TestCreateWorkspaceRefusals(t *testing.T) {
	tests := []struct {
		name   string
		err    error
		status int
		want   string
	}{
		{"creation disabled", domain.ErrCreationDisabled, http.StatusForbidden,
			`{"status":403,"code":"workspace.creation_disabled","title":"Forbidden","detail":"Creating workspaces is disabled on this instance."}`},
		{"a reserved slug", shared.Invalid(shared.FieldError{Field: "slug", Code: shared.FieldNotAllowed, Message: "is reserved"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"slug","code":"not_allowed","message":"is reserved"}]}`},
		{"a taken slug", domain.ErrSlugTaken, http.StatusConflict,
			`{"status":409,"code":"workspace.slug_taken","title":"Conflict","detail":"A workspace with this slug exists."}`},
		{"an account deactivated meanwhile", shared.Unauthenticated(), http.StatusUnauthorized,
			`{"status":401,"code":"unauthorized","title":"Unauthorized","detail":"Authentication is required."}`},
	}
	for _, tt := range tests {
		h := newServer(t, fakes{create: &fakeCreate{err: tt.err}})
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces", "alice", `{"name":"Acme","slug":"acme"}`))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s: POST = %d %s, want %d %s", tt.name, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// An unknown organization size is the domain's to refuse: the body's
// structure is fine, and the use case gets it.
func TestCreateWorkspacePassesAnUnknownSizeToTheUseCase(t *testing.T) {
	create := &fakeCreate{err: shared.Invalid(shared.FieldError{Field: "organization_size", Code: shared.FieldInvalidFormat, Message: "is not a known organization size"})}
	h := newServer(t, fakes{create: create})
	res, _ := do(t, h, request(http.MethodPost, "/api/v0/workspaces", "alice", `{"name":"Acme","slug":"acme","organization_size":"1000+"}`))
	if res.StatusCode != http.StatusUnprocessableEntity || len(create.got) != 1 || *create.got[0].OrganizationSize != "1000+" {
		t.Errorf("POST = %d, inputs %+v; want 422 from the use case", res.StatusCode, create.got)
	}
}

// The slug of the path goes to the use case for the caller; a workspace it
// does not find is workspace.not_found.
func TestGetWorkspace(t *testing.T) {
	get := &fakeGet{workspaces: map[string]domain.Workspace{"alice acme": acme, "bob beta": beta}}
	h := newServer(t, fakes{get: get})
	tests := []struct {
		token, path string
		status      int
		want        string
	}{
		{"alice", "/api/v0/workspaces/acme", http.StatusOK, acmeJSON},
		{"bob", "/api/v0/workspaces/beta", http.StatusOK, betaJSON},
		{"bob", "/api/v0/workspaces/acme", http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist."}`},
	}
	for _, tt := range tests {
		res, body := do(t, h, request(http.MethodGet, tt.path, tt.token, ""))
		if res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("%s GET %s = %d %s, want %d %s", tt.token, tt.path, res.StatusCode, body, tt.status, tt.want)
		}
	}
	if want := []string{"alice acme", "bob beta", "bob acme"}; !slices.Equal(get.calls, want) {
		t.Errorf("calls = %q, want %q", get.calls, want)
	}
}

// The availability, and the reason when the slug is not available; the slug
// of the path arrives unescaped.
func TestCheckWorkspaceSlug(t *testing.T) {
	check := &fakeCheck{reasons: map[string]domain.SlugReason{
		"acme": domain.SlugTaken, "settings": domain.SlugReserved, "my team": domain.SlugInvalid,
	}}
	h := newServer(t, fakes{check: check})
	tests := []struct{ path, want string }{
		{"/api/v0/workspace-slugs/free", `{"available":true}`},
		{"/api/v0/workspace-slugs/acme", `{"available":false,"reason":"taken"}`},
		{"/api/v0/workspace-slugs/settings", `{"available":false,"reason":"reserved"}`},
		{"/api/v0/workspace-slugs/my%20team", `{"available":false,"reason":"invalid"}`},
	}
	for _, tt := range tests {
		res, body := do(t, h, request(http.MethodGet, tt.path, "alice", ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("GET %s = %d %s, want 200 %s", tt.path, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice free", "alice acme", "alice settings", "alice my team"}; !slices.Equal(check.calls, want) {
		t.Errorf("calls = %q, want %q", check.calls, want)
	}
}
