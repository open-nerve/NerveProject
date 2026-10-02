package bootstrap

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The two rules of M3 design 3.8 that keep an invitation from changing a
// membership, on the wired app, in the orders 9.3 gives: Codex's S2 and the
// review's spike 9d. Each starts as S2 does: acme's two admins, alice and
// bob; alice removes bob, invites him again as a guest, and
// `nerve workspaces reactivate-member` restores his membership, an
// admin's, while the invitation stays pending.

// adminsWorld is the wired app on a database of its own: acme, whose
// admins are alice, its creator, and bob, by accepting her invitation as
// an admin, and whose member is carol, by accepting hers; Web, alice's.
// Then alice has removed bob, invited him again as a guest (invitation),
// and reactivate-member has restored him.
type adminsWorld struct {
	url, base  string
	contract   *apitest.Contract
	pool       *pgxpool.Pool
	tokens     map[string]string    // access tokens, by name
	ids        map[string]uuid.UUID // accounts, by name
	web        uuid.UUID
	invitation invitationLink // the guest's invitation to bob's address
}

func newAdminsWorld(t *testing.T) adminsWorld {
	t.Helper()
	w := adminsWorld{url: pgtest.NewDatabase(t), contract: apitest.Load(t), tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	w.base, w.pool = startApp(t, testConfig(t, w.url, false), migrations.FS()), openPool(t, w.url)
	for _, name := range []string{"alice", "bob", "carol"} {
		w.tokens[name] = registerAccount(t, w.contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, w.contract, w.base, w.tokens[name])
	}
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
		`{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	for name, role := range map[string]shared.Role{"bob": shared.RoleAdmin, "carol": shared.RoleMember} {
		answerInvitation(t, w.contract, w.base, w.tokens[name], "accept",
			inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", name+"@example.com", role), http.StatusOK)
	}
	w.web = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	if status, body := w.remove(t); status != http.StatusNoContent {
		t.Fatalf("alice's removal of bob = %d %s", status, body)
	}
	w.invitation = inviteAs(t, w.contract, w.base, w.tokens["alice"], "acme", "bob@example.com", shared.RoleGuest)
	if out, _, err := runWorkspaces(t, w.url, ReactivateMember("acme", "bob@example.com")); err != nil ||
		!strings.HasPrefix(out, "reactivated bob@example.com in acme as admin;") {
		t.Fatalf("reactivate-member of bob = %q, %v", out, err)
	}
	return w
}

// remove is alice's removal of bob from acme: its status and body.
func (w adminsWorld) remove(t *testing.T) (int, string) {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids["bob"]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return call(t, w.contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+id.String(), w.tokens["alice"], "")
}

// joins has name join the project, wanting 200.
func (w adminsWorld) joins(t *testing.T, name string, project uuid.UUID) {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+project.String()+"/join", w.tokens[name],
		""); status != http.StatusOK {
		t.Fatalf("%s's joining %s = %d %s", name, project, status, body)
	}
}

// bobs is bob's membership of acme and of each project he has one of, each
// as its name, its role and whether it is active, in byte order.
func (w adminsWorld) bobs(t *testing.T) string {
	t.Helper()
	var s string
	if err := w.pool.QueryRow(context.Background(), `SELECT string_agg(r, '; ' ORDER BY r COLLATE "C") FROM (
		SELECT 'acme ' || role || ' ' || is_active AS r FROM workspace_members WHERE member_id = $1
		UNION ALL SELECT p.name || ' ' || m.role || ' ' || m.is_active FROM project_members m JOIN projects p ON p.id = m.project_id
		WHERE m.member_id = $1) s`, w.ids["bob"]).Scan(&s); err != nil {
		t.Fatal(err)
	}
	return s
}

// look is the status and body of a look at the invitation through its
// link, as anyone holding it.
func (w adminsWorld) look(t *testing.T) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodGet, w.base+"/api/v0/workspace-invitations/"+w.invitation.id.String()+"?token="+w.invitation.token, "",
		"")
}

// An invitation never changes an active membership (M3 design 3.8, 9.3),
// in Codex's S2 order: once reactivate-member has restored bob, an admin,
// he joins Web, its admin as acme's; alice leaves acme, which has bob as
// its only admin then; bob accepts the guest's invitation alice sent while
// he was removed: 200, its answer acme with his role, 20. The invitation is
// accepted and deleted; his membership of acme is an admin's, active, and
// his membership of Web an admin's: acme is not left without an admin, and
// no guest administers a project.
func TestAnInvitationNeverChangesAnActiveMembership(t *testing.T) {
	w := newAdminsWorld(t)
	w.joins(t, "bob", w.web)
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/leave", w.tokens["alice"], ""); status !=
		http.StatusNoContent {
		t.Fatalf("alice's leaving acme = %d %s", status, body)
	}
	if got, want := w.bobs(t), "Web 20 true; acme 20 true"; got != want {
		t.Fatalf("bob's memberships before his acceptance: %s, want %s", got, want)
	}

	status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspace-invitations/"+w.invitation.id.String()+"/accept",
		w.tokens["bob"], `{"token":"`+w.invitation.token+`"}`)

	var answer struct {
		Slug string      `json:"slug"`
		Role shared.Role `json:"role"`
	}
	if status != http.StatusOK {
		t.Fatalf("bob's acceptance = %d %s, want 200", status, body)
	}
	decodeAnswer(t, body, &answer)
	if answer.Slug != "acme" || answer.Role != shared.RoleAdmin {
		t.Errorf("bob's acceptance answers %s; want acme with his role, 20", body)
	}
	var accepted, deleted bool
	if err := w.pool.QueryRow(context.Background(), "SELECT accepted, deleted_at IS NOT NULL FROM workspace_member_invites WHERE id = $1",
		w.invitation.id).Scan(&accepted, &deleted); err != nil || !accepted || !deleted {
		t.Errorf("the invitation accepted %v, deleted %v (%v); want both", accepted, deleted, err)
	}
	if got, want := w.bobs(t), "Web 20 true; acme 20 true"; got != want {
		t.Errorf("bob's memberships after his acceptance: %s, want %s, as they were", got, want)
	}
}

// An ended membership leaves no invitation (M3 design 3.8, 9.3), in the
// review's spike 9d order, for alice's removal of bob and for his own
// leaving: once reactivate-member has restored him, bob is active with the
// guest's invitation pending. He administers Ops, his own, and carol is
// its member: the ending is 409 project.sole_admin, and the invitation
// stays pending, its link answering 200. Once alice has joined Ops, the
// ending is 204; then the link answers 404 workspace.invitation_not_found
// to a look and to bob's acceptance, he has not come back, and the
// invitation was deleted at the ending's moment, his membership's
// updated_at.
func TestAnEndedMembershipLeavesNoInvitation(t *testing.T) {
	for _, e := range []struct {
		name string
		end  func(w adminsWorld, t *testing.T) (int, string)
	}{
		{"removal", adminsWorld.remove},
		{"leaving", func(w adminsWorld, t *testing.T) (int, string) {
			return call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/leave", w.tokens["bob"], "")
		}},
	} {
		t.Run(e.name, func(t *testing.T) {
			w := newAdminsWorld(t)
			ops := createdProject(t, w.contract, w.base, w.tokens["bob"], "acme", "Ops", "OPS")
			w.joins(t, "carol", ops)
			if status, body := e.end(w, t); status != http.StatusConflict || problemCode(t, []byte(body)) != "project.sole_admin" {
				t.Fatalf("the ending, bob Ops's only admin = %d %s, want 409 project.sole_admin", status, body)
			}
			if status, body := w.look(t); status != http.StatusOK {
				t.Errorf("a look at the invitation after the refused ending = %d %s, want 200: still pending", status, body)
			}
			w.joins(t, "alice", ops)
			if status, body := e.end(w, t); status != http.StatusNoContent {
				t.Fatalf("the ending = %d %s, want 204", status, body)
			}

			if status, body := w.look(t); status != http.StatusNotFound || problemCode(t, []byte(body)) != "workspace.invitation_not_found" {
				t.Errorf("a look at the invitation = %d %s, want 404 workspace.invitation_not_found", status, body)
			}
			status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspace-invitations/"+w.invitation.id.String()+"/accept",
				w.tokens["bob"], `{"token":"`+w.invitation.token+`"}`)
			if status != http.StatusNotFound || problemCode(t, []byte(body)) != "workspace.invitation_not_found" {
				t.Errorf("bob's acceptance = %d %s, want 404 workspace.invitation_not_found", status, body)
			}
			if got, want := w.bobs(t), "Ops 20 false; acme 20 false"; got != want {
				t.Errorf("bob's memberships: %s, want %s: ended, and not back", got, want)
			}
			var deletedAt, endedAt time.Time
			if err := w.pool.QueryRow(context.Background(), `SELECT i.deleted_at, m.updated_at FROM workspace_member_invites i
				JOIN workspace_members m ON m.workspace_id = i.workspace_id WHERE i.id = $1 AND m.member_id = $2`, w.invitation.id,
				w.ids["bob"]).Scan(&deletedAt, &endedAt); err != nil || !deletedAt.Equal(endedAt) {
				t.Errorf("the invitation deleted at %v, bob's membership ended at %v (%v); want the one moment", deletedAt, endedAt, err)
			}
		})
	}
}
