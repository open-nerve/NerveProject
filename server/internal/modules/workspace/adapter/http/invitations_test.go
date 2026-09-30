package httpadapter_test

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeInvitations is the invitations' use cases: each records its call as
// "<operation> <caller> <arguments>" and answers what it is given.
type fakeInvitations struct {
	calls   []string
	lists   map[string][]domain.InvitationWithToken // by slug
	one     domain.InvitationWithToken              // the answer of an update
	preview domain.InvitationPreview                // the answer of a get
	joined  domain.Workspace                        // the answer of an acceptance
	err     error
}

type fakeAcceptInvitation struct{ *fakeInvitations }

func (f fakeAcceptInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.Workspace, error) {
	f.calls = append(f.calls, fmt.Sprintf("accept %s %s %s", caller(ctx), id, token))
	return f.joined, f.err
}

type fakeDeclineInvitation struct{ *fakeInvitations }

func (f fakeDeclineInvitation) Execute(ctx context.Context, id uuid.UUID, token string) error {
	f.calls = append(f.calls, fmt.Sprintf("decline %s %s %s", caller(ctx), id, token))
	return f.err
}

type fakeGetInvitation struct{ *fakeInvitations }

func (f fakeGetInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error) {
	f.calls = append(f.calls, fmt.Sprintf("get %s %s %s", caller(ctx), id, token))
	return f.preview, f.err
}

type fakeCreateInvitations struct{ *fakeInvitations }

func (f fakeCreateInvitations) Execute(ctx context.Context, slug string, batch []domain.NewInvitation) ([]domain.InvitationWithToken, error) {
	f.calls = append(f.calls, fmt.Sprintf("create %s %s %+v", caller(ctx), slug, batch))
	return f.lists[slug], f.err
}

type fakeListInvitations struct{ *fakeInvitations }

func (f fakeListInvitations) Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error) {
	f.calls = append(f.calls, "list "+caller(ctx)+" "+slug)
	return f.lists[slug], f.err
}

type fakeUpdateInvitation struct{ *fakeInvitations }

func (f fakeUpdateInvitation) Execute(ctx context.Context, id uuid.UUID, role shared.Role) (domain.InvitationWithToken, error) {
	f.calls = append(f.calls, fmt.Sprintf("update %s %s %d", caller(ctx), id, role))
	return f.one, f.err
}

type fakeDeleteInvitation struct{ *fakeInvitations }

func (f fakeDeleteInvitation) Execute(ctx context.Context, id uuid.UUID) error {
	f.calls = append(f.calls, fmt.Sprintf("delete %s %s", caller(ctx), id))
	return f.err
}

var (
	declinedAt = time.Date(2026, 9, 30, 8, 0, 0, 0, time.UTC)
	// carolInvited is pending, alice's; daveDeclined declined, its inviter
	// gone.
	carolInvited = domain.InvitationWithToken{Invitation: domain.Invitation{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000c1"),
		WorkspaceID: acmeID, Email: "carol@corp.com", Role: shared.RoleMember, CreatedAt: created, CreatedByID: &aliceID},
		Token: "nrv_inv_kvqyKBh-bANMT6JAIYzolA"}
	daveDeclined = domain.InvitationWithToken{Invitation: domain.Invitation{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000d1"),
		WorkspaceID: acmeID, Email: "dave@corp.com", Role: shared.RoleGuest, RespondedAt: &declinedAt, CreatedAt: created},
		Token: "nrv_inv_AAAAAAAAAAAAAAAAAAAAAA"}
)

const (
	carolInvitedJSON = `{"accepted":false,"created_at":"2026-09-29T10:00:00.123456Z","created_by_id":"0199a2b4-0000-7000-8000-000000000001",` +
		`"email":"carol@corp.com","id":"0199a2b4-0000-7000-8000-0000000000c1","responded_at":null,"role":15,` +
		`"token":"nrv_inv_kvqyKBh-bANMT6JAIYzolA","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	daveDeclinedJSON = `{"accepted":false,"created_at":"2026-09-29T10:00:00.123456Z","created_by_id":null,"email":"dave@corp.com",` +
		`"id":"0199a2b4-0000-7000-8000-0000000000d1","responded_at":"2026-09-30T08:00:00Z","role":5,` +
		`"token":"nrv_inv_AAAAAAAAAAAAAAAAAAAAAA","workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// The list is the use case's for the caller and the slug of the path, each
// invitation with its token: a pending one and a declined one whose inviter
// is gone.
func TestListWorkspaceInvitations(t *testing.T) {
	inv := &fakeInvitations{lists: map[string][]domain.InvitationWithToken{"acme": {carolInvited, daveDeclined}}}
	h := newServer(t, fakes{invitations: inv})
	for _, tt := range []struct{ token, slug, want string }{
		{"alice", "acme", `{"data":[` + carolInvitedJSON + `,` + daveDeclinedJSON + `]}`},
		{"bob", "beta", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/"+tt.slug+"/invitations", tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s's invitations = %d %s, want 200 %s", tt.token, tt.slug, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"list alice acme", "list bob beta"}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusals, as the contract declares them.
func TestListWorkspaceInvitationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/invitations", "bob", "")); res.StatusCode != tt.status ||
			body != tt.want+"\n" {
			t.Errorf("GET refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
}

// POST hands the batch to the use case as sent, the addresses as they are
// and every role, for the caller and the slug of the path, and answers the
// invitations it creates with 201. Two callers, two workspaces.
func TestCreateWorkspaceInvitations(t *testing.T) {
	inv := &fakeInvitations{lists: map[string][]domain.InvitationWithToken{"acme": {carolInvited}, "beta": {daveDeclined}}}
	h := newServer(t, fakes{invitations: inv})
	for _, tt := range []struct{ token, slug, body, want string }{
		{"alice", "acme", `{"invitations":[{"email":" Carol@corp.com ","role":15},{"email":"dave","role":10}]}`, `{"data":[` + carolInvitedJSON + `]}`},
		{"bob", "beta", `{"invitations":[{"email":"dave@corp.com","role":5}]}`, `{"data":[` + daveDeclinedJSON + `]}`},
	} {
		res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/"+tt.slug+"/invitations", tt.token, tt.body))
		if res.StatusCode != http.StatusCreated || body != tt.want+"\n" {
			t.Errorf("%s POST to %s = %d %s, want 201 %s", tt.token, tt.slug, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"create alice acme [{Email: Carol@corp.com  Role:15} {Email:dave Role:10}]",
		"create bob beta [{Email:dave@corp.com Role:5}]"}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusals, as the contract declares them; a body whose
// invitation lacks a role is refused before it.
func TestCreateWorkspaceInvitationsRefusals(t *testing.T) {
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{shared.Invalid(shared.FieldError{Field: "invitations[1].email", Code: shared.FieldDuplicate, Message: "has an invitation to the workspace already"}),
			http.StatusUnprocessableEntity, `{"status":422,"code":"validation_failed","title":"Unprocessable Entity",` +
				`"detail":"The request has invalid values.","errors":[{"field":"invitations[1].email","code":"duplicate",` +
				`"message":"has an invitation to the workspace already"}]}`},
		{domain.ErrNotFound, http.StatusNotFound,
			`{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/invitations", "bob",
			`{"invitations":[{"email":"carol@corp.com","role":5}]}`)); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("POST refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	inv := &fakeInvitations{}
	h := newServer(t, fakes{invitations: inv})
	if res, body := do(t, h, request(http.MethodPost, "/api/v0/workspaces/acme/invitations", "alice",
		`{"invitations":[{"email":"carol@corp.com"}]}`)); res.StatusCode != http.StatusBadRequest || len(inv.calls) != 0 ||
		!strings.Contains(body, `"field":"invitations[0].role","code":"required"`) {
		t.Errorf("POST without a role = %d %s, calls %q; want 400 on invitations[0].role and no call", res.StatusCode, body, inv.calls)
	}
}

// PATCH hands the invitation of the path and the role as sent to the use
// case, for the caller, and answers the invitation it gives.
func TestUpdateWorkspaceInvitation(t *testing.T) {
	inv := &fakeInvitations{one: carolInvited}
	h := newServer(t, fakes{invitations: inv})
	for _, body := range []string{`{"role":20}`, `{"role":10}`} {
		res, got := do(t, h, request(http.MethodPatch, "/api/v0/workspace-invitations/"+carolInvited.ID.String(), "alice", body))
		if res.StatusCode != http.StatusOK || got != carolInvitedJSON+"\n" {
			t.Errorf("PATCH %s = %d %s, want 200 %s", body, res.StatusCode, got, carolInvitedJSON)
		}
	}
	id := carolInvited.ID.String()
	if want := []string{"update alice " + id + " 20", "update alice " + id + " 10"}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusals, as the contract declares them; a body without a
// role and an id that is no UUID are refused before it.
func TestUpdateWorkspaceInvitationRefusals(t *testing.T) {
	path := "/api/v0/workspace-invitations/" + carolInvited.ID.String()
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{shared.Invalid(shared.FieldError{Field: "role", Code: shared.FieldInvalidFormat, Message: "is not 5, 15 or 20"}), http.StatusUnprocessableEntity,
			`{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
				`"errors":[{"field":"role","code":"invalid_format","message":"is not 5, 15 or 20"}]}`},
		{domain.ErrInvitationNotFound, http.StatusNotFound, invitationNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
		{domain.ErrInvitationResponded, http.StatusConflict,
			`{"status":409,"code":"workspace.invitation_responded","title":"Conflict","detail":"The invitation has been answered already."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodPatch, path, "bob", `{"role":5}`)); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("PATCH refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	inv := &fakeInvitations{}
	h := newServer(t, fakes{invitations: inv})
	for _, req := range []*http.Request{request(http.MethodPatch, path, "alice", `{}`),
		request(http.MethodPatch, "/api/v0/workspace-invitations/carol", "alice", `{"role":5}`)} {
		if res, _ := do(t, h, req); res.StatusCode != http.StatusBadRequest || len(inv.calls) != 0 {
			t.Errorf("PATCH %s = %d, calls %q; want 400 and no call", req.URL.Path, res.StatusCode, inv.calls)
		}
	}
}

// invitationNotFoundJSON is workspace.invitation_not_found's problem.
const invitationNotFoundJSON = `{"status":404,"code":"workspace.invitation_not_found","title":"Not Found",` +
	`"detail":"The invitation does not exist, or its link is not valid."}`

// DELETE hands the invitation of the path to the use case, for the caller,
// and answers 204 without a body; its refusals are the contract's, and an
// id that is no UUID is refused before it.
func TestDeleteWorkspaceInvitation(t *testing.T) {
	inv := &fakeInvitations{}
	h := newServer(t, fakes{invitations: inv})
	path := "/api/v0/workspace-invitations/" + daveDeclined.ID.String()
	if res, body := do(t, h, request(http.MethodDelete, path, "alice", "")); res.StatusCode != http.StatusNoContent || body != "" {
		t.Errorf("DELETE = %d %q, want 204 and no body", res.StatusCode, body)
	}
	if want := []string{"delete alice " + daveDeclined.ID.String()}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
	for _, tt := range []struct {
		err    error
		status int
		want   string
	}{
		{domain.ErrInvitationNotFound, http.StatusNotFound, invitationNotFoundJSON},
		{shared.Forbidden(), http.StatusForbidden, `{"status":403,"code":"forbidden","title":"Forbidden","detail":"Your role does not allow this."}`},
	} {
		h := newServer(t, fakes{invitations: &fakeInvitations{err: tt.err}})
		if res, body := do(t, h, request(http.MethodDelete, path, "bob", "")); res.StatusCode != tt.status || body != tt.want+"\n" {
			t.Errorf("DELETE refused with %v = %d %s, want %d %s", tt.err, res.StatusCode, body, tt.status, tt.want)
		}
	}
	idle := &fakeInvitations{}
	h = newServer(t, fakes{invitations: idle})
	if res, _ := do(t, h, request(http.MethodDelete, "/api/v0/workspace-invitations/carol", "alice", "")); res.StatusCode != http.StatusBadRequest ||
		len(idle.calls) != 0 {
		t.Errorf("DELETE /api/v0/workspace-invitations/carol = %d, calls %q; want 400 and no call", res.StatusCode, idle.calls)
	}
}

// carolsLink is the path of carol's invitation's link, with its token.
var carolsLink = "/api/v0/workspace-invitations/" + carolInvited.ID.String() + "?token=" + carolInvited.Token

// The public GET hands the invitation of the path and the token of the
// query to the use case and answers what it shows, without the address.
// The route needs no bearer token and reads none: with one, even one that
// is not valid, the use case is called by nobody all the same.
func TestGetWorkspaceInvitation(t *testing.T) {
	inv := &fakeInvitations{preview: domain.InvitationPreview{ID: carolInvited.ID, Role: shared.RoleMember, Declined: true,
		WorkspaceName: "Acme", WorkspaceSlug: "acme"}}
	h := newServer(t, fakes{invitations: inv})
	want := `{"declined":true,"id":"0199a2b4-0000-7000-8000-0000000000c1","role":15,"workspace_logo_url":null,` +
		`"workspace_name":"Acme","workspace_slug":"acme"}`
	for _, token := range []string{"", "alice", "mallory"} {
		if res, body := do(t, h, request(http.MethodGet, carolsLink, token, "")); res.StatusCode != http.StatusOK || body != want+"\n" {
			t.Errorf("GET with the bearer %q = %d %s, want 200 %s", token, res.StatusCode, body, want)
		}
	}
	call := "get nobody " + carolInvited.ID.String() + " " + carolInvited.Token
	if want := []string{call, call, call}; !slices.Equal(inv.calls, want) {
		t.Errorf("calls = %q, want %q", inv.calls, want)
	}
}

// The use case's refusal, as the contract declares it; a link without its
// token and an id that is no UUID are refused before it. No answer repeats
// the token.
func TestGetWorkspaceInvitationRefusals(t *testing.T) {
	h := newServer(t, fakes{invitations: &fakeInvitations{err: domain.ErrInvitationNotFound}})
	if res, body := do(t, h, request(http.MethodGet, carolsLink, "", "")); res.StatusCode != http.StatusNotFound || body != invitationNotFoundJSON+"\n" {
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, invitationNotFoundJSON)
	}
	inv := &fakeInvitations{}
	h = newServer(t, fakes{invitations: inv})
	for _, tt := range []struct{ path, field string }{
		{"/api/v0/workspace-invitations/" + carolInvited.ID.String(), `"field":"token","code":"required"`},
		{"/api/v0/workspace-invitations/carol?token=" + carolInvited.Token, `"field":"invitation_id","code":"invalid_format"`},
	} {
		res, body := do(t, h, request(http.MethodGet, tt.path, "", ""))
		if res.StatusCode != http.StatusBadRequest || !strings.Contains(body, tt.field) || strings.Contains(body, carolInvited.Token) || len(inv.calls) != 0 {
			t.Errorf("GET %s = %d %s, calls %q; want 400 on %s without the token, and no call", tt.path, res.StatusCode, body, inv.calls, tt.field)
		}
	}
}

// The two answers to an invitation hand the invitation of the path and the
// token of the body to their use case, for the caller: accepting answers
// the workspace it gives, declining 204 without a body. Each refuses as
// the contract declares; a body without its token, or with a field it does
// not have, and an id that is no UUID are refused before the use case, and
// no answer repeats the token.
func TestAnsweringAWorkspaceInvitation(t *testing.T) {
	path := "/api/v0/workspace-invitations/" + carolInvited.ID.String()
	body := `{"token":"` + carolInvited.Token + `"}`
	for _, tt := range []struct {
		name, path string
		status     int
		answer     string
	}{{"accept", path + "/accept", http.StatusOK, acmeJSON + "\n"}, {"decline", path + "/decline", http.StatusNoContent, ""}} {
		inv := &fakeInvitations{joined: acme}
		h := newServer(t, fakes{invitations: inv})
		if res, got := do(t, h, request(http.MethodPost, tt.path, "bob", body)); res.StatusCode != tt.status || got != tt.answer {
			t.Errorf("POST %s = %d %q, want %d %q", tt.path, res.StatusCode, got, tt.status, tt.answer)
		}
		if want := []string{tt.name + " bob " + carolInvited.ID.String() + " " + carolInvited.Token}; !slices.Equal(inv.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, inv.calls, want)
		}
		for _, refusal := range []struct {
			err    error
			status int
			want   string
		}{
			{domain.ErrInvitationNotFound, http.StatusNotFound, invitationNotFoundJSON},
			{domain.ErrInvitationEmailMismatch, http.StatusForbidden, `{"status":403,"code":"workspace.invitation_email_mismatch",` +
				`"title":"Forbidden","detail":"The invitation was sent to another e-mail address."}`},
			{domain.ErrInvitationResponded, http.StatusConflict,
				`{"status":409,"code":"workspace.invitation_responded","title":"Conflict","detail":"The invitation has been answered already."}`},
		} {
			h := newServer(t, fakes{invitations: &fakeInvitations{err: refusal.err}})
			if res, got := do(t, h, request(http.MethodPost, tt.path, "bob", body)); res.StatusCode != refusal.status || got != refusal.want+"\n" {
				t.Errorf("%s refused with %v = %d %s, want %d %s", tt.name, refusal.err, res.StatusCode, got, refusal.status, refusal.want)
			}
		}
		idle := &fakeInvitations{}
		h = newServer(t, fakes{invitations: idle})
		for _, bad := range []struct{ path, body string }{
			{tt.path, `{}`},
			{tt.path, `{"token":"` + carolInvited.Token + `","email":"carol@corp.com"}`},
			{"/api/v0/workspace-invitations/carol/" + tt.name, body},
		} {
			res, got := do(t, h, request(http.MethodPost, bad.path, "bob", bad.body))
			if res.StatusCode != http.StatusBadRequest || strings.Contains(got, carolInvited.Token) || len(idle.calls) != 0 {
				t.Errorf("POST %s with %s = %d %s, calls %q; want 400 without the token, and no call", bad.path, bad.body, res.StatusCode, got,
					idle.calls)
			}
		}
	}
}
