package httpadapter_test

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	aliceEmail  = "alice@corp.com"
	aliceMember = domain.Member{
		Membership: domain.Membership{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000a1"), WorkspaceID: acmeID, MemberID: aliceID,
			Role: shared.RoleAdmin, IsActive: true, CreatedAt: created},
		User: domain.MemberUser{ID: aliceID, DisplayName: "al", FirstName: "Alice", LastName: "Liddell", Email: &aliceEmail},
	}
	bobMember = domain.Member{
		Membership: domain.Membership{ID: uuid.MustParse("0199a2b4-0000-7000-8000-0000000000b1"), WorkspaceID: acmeID, MemberID: bobID,
			Role: shared.RoleGuest, CreatedAt: created},
		User: domain.MemberUser{ID: bobID, DisplayName: "bob"},
	}
)

const (
	aliceMemberJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000a1","is_active":true,` +
		`"member":{"avatar_url":null,"display_name":"al","email":"alice@corp.com","first_name":"Alice",` +
		`"id":"0199a2b4-0000-7000-8000-000000000001","last_name":"Liddell"},"role":20,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
	bobMemberJSON = `{"created_at":"2026-09-29T10:00:00.123456Z","id":"0199a2b4-0000-7000-8000-0000000000b1","is_active":false,` +
		`"member":{"avatar_url":null,"display_name":"bob","email":null,"first_name":"","id":"0199a2b4-0000-7000-8000-000000000002",` +
		`"last_name":""},"role":5,"workspace_id":"0199a2b4-0000-7000-8000-00000000000a"}`
)

// The list is the use case's for the caller and the slug of the path: an
// address shown, one null, the avatar null, an ended membership as it is.
func TestListWorkspaceMembers(t *testing.T) {
	members := &fakeMembers{lists: map[string][]domain.Member{"acme": {aliceMember, bobMember}}}
	h := newServer(t, fakes{member: members})
	for _, tt := range []struct{ token, slug, want string }{
		{"alice", "acme", `{"data":[` + aliceMemberJSON + `,` + bobMemberJSON + `]}`},
		{"bob", "beta", `{"data":[]}`},
	} {
		res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/"+tt.slug+"/members", tt.token, ""))
		if res.StatusCode != http.StatusOK || body != tt.want+"\n" {
			t.Errorf("%s GET %s's members = %d %s, want 200 %s", tt.token, tt.slug, res.StatusCode, body, tt.want)
		}
	}
	if want := []string{"alice acme", "bob beta"}; !slices.Equal(members.calls, want) {
		t.Errorf("calls = %q, want %q", members.calls, want)
	}
	h = newServer(t, fakes{member: &fakeMembers{err: domain.ErrNotFound}})
	res, body := do(t, h, request(http.MethodGet, "/api/v0/workspaces/acme/members", "bob", ""))
	if want := `{"status":404,"code":"workspace.not_found","title":"Not Found","detail":"The workspace does not exist, or you are not a member of it."}`; res.StatusCode != http.StatusNotFound || body != want+"\n" {
		t.Errorf("GET refused = %d %s, want 404 %s", res.StatusCode, body, want)
	}
}
