package httpadapter_test

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeInvitations is the invitations' use cases: each records its call as
// "<operation> <caller> <arguments>" and answers what it is given.
type fakeInvitations struct {
	calls []string
	lists map[string][]domain.InvitationWithToken // by slug
	err   error
}

type fakeListInvitations struct{ *fakeInvitations }

func (f fakeListInvitations) Execute(ctx context.Context, slug string) ([]domain.InvitationWithToken, error) {
	f.calls = append(f.calls, "list "+caller(ctx)+" "+slug)
	return f.lists[slug], f.err
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
