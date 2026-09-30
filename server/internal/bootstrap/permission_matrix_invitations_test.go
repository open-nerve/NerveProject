package bootstrap

import (
	"maps"
	"net/http"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The invitations' part of the workspace module's rows (M3 design 9.2).

var (
	cellInvitationNotFound = cell{http.StatusNotFound, "workspace.invitation_not_found"}
	cellEmailMismatch      = cell{http.StatusForbidden, "workspace.invitation_email_mismatch"}
)

// toOwnInvitation is the request of a row whose callers each answer, with
// its token, the invitation to their own address (ownInvitation).
func toOwnInvitation(answer string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		id := s.invitation(ownInvitation(c), emailOf(c))
		return http.MethodPost, "/api/v0/workspace-invitations/" + id.String() + "/" + answer, `{"token":"` + invitationToken(s.t, id) + `"}`
	}
}

// toNewcomersInvitation is the request of a row whose callers each answer,
// with its token, acme's invitation of newcomer@example.com: another
// address than any caller's.
func toNewcomersInvitation(answer string) func(caller, seeded) (string, string, string) {
	return func(_ caller, s seeded) (string, string, string) {
		id := s.invitation("acme", "newcomer@example.com")
		return http.MethodPost, "/api/v0/workspace-invitations/" + id.String() + "/" + answer, `{"token":"` + invitationToken(s.t, id) + `"}`
	}
}

// joinsAsAMember: each caller's acceptance answers the workspace of his
// invitation, where he is now an active member as the invitation's
// member: one member more than it had (other: its admin and the removed
// member; acme: its admin, member and guest).
func joinsAsAMember(t *testing.T, c caller, _ seeded, answer string) {
	var w struct {
		Slug         string `json:"slug"`
		Role         int    `json:"role"`
		TotalMembers int    `json:"total_members"`
	}
	decodeAnswer(t, answer, &w)
	want := map[string]int{"other": 3, "acme": 4}[ownInvitation(c)]
	if w.Slug != ownInvitation(c) || w.Role != 15 || w.TotalMembers != want {
		t.Errorf("%s's acceptance answers %+v, want %s with role 15 and %d members", c, w, ownInvitation(c), want)
	}
}

// ofInvitation are the cells of a row that names an invitation: the answers
// of the workspace's admin, member and guest, and
// workspace.invitation_not_found for the callers the workspace is not
// visible to.
func ofInvitation(admin, member, guest cell) map[caller]cell {
	return map[caller]cell{callerAdmin: admin, callerMember: member, callerGuest: guest,
		callerNever: cellInvitationNotFound, callerRemoved: cellInvitationNotFound, callerDeleted: cellInvitationNotFound}
}

// toInvitation is the request of a row whose callers each send method and
// body to the invitation of email seeded in the workspace their column
// targets.
func toInvitation(method, email, body string) func(caller, seeded) (string, string, string) {
	return func(c caller, s seeded) (string, string, string) {
		return method, "/api/v0/workspace-invitations/" + s.invitation(workspaceOf(c), email).String(), body
	}
}

// promotesTheNewcomer: the admin's answer is acme's invitation of
// newcomer@example.com, the one seeded, of acme, now as an admin, with the
// token of its id.
func promotesTheNewcomer(t *testing.T, c caller, s seeded, answer string) {
	var inv struct {
		ID          uuid.UUID `json:"id"`
		WorkspaceID uuid.UUID `json:"workspace_id"`
		Email       string    `json:"email"`
		Role        int       `json:"role"`
		Token       string    `json:"token"`
	}
	decodeAnswer(t, answer, &inv)
	if inv.ID != s.invitation("acme", "newcomer@example.com") || inv.WorkspaceID != s.workspace("acme") || inv.Email != "newcomer@example.com" ||
		inv.Role != 20 || inv.Token != invitationToken(t, inv.ID) {
		t.Errorf("%s's change answers %+v, want acme's seeded invitation of newcomer@example.com as an admin, with its token", c, inv)
	}
}

// listsTheInvitations: the admin's list is acme's invitations, each with
// its seeded id, not gone's of the same address, and the token of its id.
func listsTheInvitations(t *testing.T, c caller, s seeded, answer string) {
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Email string    `json:"email"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	var got, want []string
	for _, i := range list.Data {
		got = append(got, i.Email)
		if want := s.invitation("acme", i.Email); i.ID != want {
			t.Errorf("%s: the invitation of %s is %s, want acme's seeded %s", c, i.Email, i.ID, want)
		}
		if i.Token != invitationToken(t, i.ID) {
			t.Errorf("%s: the invitation of %s has the token %s, want its id's", c, i.Email, i.Token)
		}
	}
	for _, i := range matrixInvitations {
		if i.slug == "acme" {
			want = append(want, i.email)
		}
	}
	slices.Sort(got)
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("%s lists the invitations of %q, want %q", c, got, want)
	}
}

// inviting is the body of a creation that invites email as a guest.
func inviting(email string) string {
	return `{"invitations":[{"email":"` + email + `","role":5}]}`
}

// invitesTheInvitee: the admin's answer is the one new invitation, of
// invitee@example.com as a guest, with the token of its id.
func invitesTheInvitee(t *testing.T, c caller, _ seeded, answer string) {
	var list struct {
		Data []struct {
			ID    uuid.UUID `json:"id"`
			Email string    `json:"email"`
			Role  int       `json:"role"`
			Token string    `json:"token"`
		} `json:"data"`
	}
	decodeAnswer(t, answer, &list)
	if len(list.Data) != 1 || list.Data[0].Email != "invitee@example.com" || list.Data[0].Role != 5 ||
		list.Data[0].Token != invitationToken(t, list.Data[0].ID) {
		t.Errorf("%s's invitation answers %+v, want invitee@example.com's, as a guest, with its token", c, list.Data)
	}
}

// wholeAnswer is an answer as a caller could compare two: the status, the
// body and every header but the two that differ by request (Date,
// X-Request-Id).
type wholeAnswer struct {
	status  int
	body    string
	headers string
}

// wholeAnswerOf is res, whose body is body, as a wholeAnswer.
func wholeAnswerOf(res *http.Response, body []byte) wholeAnswer {
	header := res.Header.Clone()
	header.Del("Date")
	header.Del("X-Request-Id")
	var lines []string
	for _, name := range slices.Sorted(maps.Keys(header)) {
		lines = append(lines, name+": "+strings.Join(header[name], ", "))
	}
	return wholeAnswer{res.StatusCode, string(body), strings.Join(lines, "\n")}
}

// The public getWorkspaceInvitation has no row (matrixExempt): its route
// reads no credential. This test stands for the row. Each link is asked
// with no bearer token, with one that is no credential (a malformed
// nrv_pat_, which every other route answers 401), and with each column's,
// and every caller gets the same answer: acme's invitation with its token,
// 200, without the address;
// and one 404 workspace.invitation_not_found, the same byte for byte for a
// character of the token changed, another invitation's token, gone's
// invitation, deleted with gone, with its own token, and an id no
// invitation has (M3 design 8.2): nothing tells a wrong token from a
// missing invitation.
func TestTheInvitationLinkAnswersEveryCallerAlike(t *testing.T) {
	d := prepareMatrix(t)
	contract := apitest.Load(t)
	base := startApp(t, d.config(t, pgtest.NewDatabaseFrom(t, d.url), nil), migrations.FS())
	s := d.seeded.in(t)
	acme, gone, nobodys := s.invitation("acme", "newcomer@example.com"), s.invitation("gone", "newcomer@example.com"), uuid.NewV7()
	right := invitationToken(t, acme)
	changed := []byte(right)
	changed[len("nrv_inv_")+10] = 'A'
	if string(changed) == right {
		changed[len("nrv_inv_")+10] = 'B'
	}
	links := []struct {
		name  string
		id    uuid.UUID
		token string
		shows bool // the link that shows its invitation; every other is the one 404
	}{
		{"acme's invitation", acme, right, true},
		{"a character changed", acme, string(changed), false},
		{"another invitation's token", acme, invitationToken(t, gone), false},
		{"gone's invitation", gone, invitationToken(t, gone), false},
		{"no invitation", nobodys, invitationToken(t, nobodys), false},
	}
	bearers := maps.Clone(d.tokens)
	bearers["an invalid bearer"] = "nrv_pat_malformed"
	if status, body := call(t, contract, http.MethodGet, base+"/api/v0/workspaces", bearers["an invalid bearer"], ""); status != http.StatusUnauthorized {
		t.Fatalf("the invalid bearer on listWorkspaces = %d %s, want 401", status, body)
	}
	callers := append([]caller{"nobody", "an invalid bearer"}, workspaceColumns...)
	var notFound *wholeAnswer
	for _, l := range links {
		var first wholeAnswer
		for i, c := range callers {
			req := newRequest(t, http.MethodGet, base+"/api/v0/workspace-invitations/"+l.id.String()+"?token="+l.token, bearers[c], nil)
			contract.CheckRequest(t, req)
			res, body := send(t, req)
			contract.CheckResponse(t, req, res)
			got := wholeAnswerOf(res, body)
			if i == 0 {
				first = got
			} else if got != first {
				t.Errorf("%s asked by %s = %+v, want what %s got: %+v", l.name, c, got, callers[0], first)
			}
		}
		switch {
		case l.shows:
			var p struct {
				ID            uuid.UUID `json:"id"`
				Role          int       `json:"role"`
				Declined      bool      `json:"declined"`
				WorkspaceSlug string    `json:"workspace_slug"`
			}
			decodeAnswer(t, first.body, &p)
			if first.status != http.StatusOK || p.ID != acme || p.Role != 15 || p.Declined || p.WorkspaceSlug != "acme" ||
				strings.Contains(first.body, "email") || strings.Contains(first.body, "newcomer") {
				t.Errorf("%s = %d %s; want 200, acme's pending invitation as a member, without the address", l.name, first.status, first.body)
			}
		case notFound == nil:
			if first.status != http.StatusNotFound || problemCode(t, []byte(first.body)) != "workspace.invitation_not_found" {
				t.Errorf("%s = %d %s; want 404 workspace.invitation_not_found", l.name, first.status, first.body)
			}
			notFound = &first
		case first != *notFound:
			t.Errorf("%s = %+v; want the same answer as %s: %+v", l.name, first, links[1].name, *notFound)
		}
	}
}
