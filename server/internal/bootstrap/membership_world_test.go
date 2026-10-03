package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// memberWorld is the wired app on a database of its own, for the writes on
// one project membership: acme, alice its admin; bob, carol, dave and gina
// its members, erin its guest, each by accepting alice's invitation; beta,
// bob's, carol its member by accepting his. Web, alice's: bob and dave its
// admins, carol and gina its members, erin its guest, each by her adding;
// gina then made acme's admin by alice, so that she is a member of Web who
// is a workspace admin (PM+WA). Ops, alice's: carol its admin, bob its
// member, by her adding. Lab, beta's, bob's: carol its member by his
// adding. bob and carol are each an admin somewhere and a member elsewhere,
// and bob is a workspace admin in beta alone.
type memberWorld struct {
	contract      *apitest.Contract
	base          string
	pool          *pgxpool.Pool
	tokens        map[string]string    // access tokens, by name
	ids           map[string]uuid.UUID // accounts, by name
	web, ops, lab uuid.UUID
}

// worldStanding is memberWorld's standing as newMemberWorld makes it.
const worldStanding = "acme: alice 20, bob 15, carol 15, dave 15, erin 5, gina 20; beta: bob 20, carol 15; Lab: bob 20, carol 15; " +
	"Ops: alice 20, bob 15, carol 20; Web: alice 20, bob 20, carol 15, dave 20, erin 5, gina 15"

func newMemberWorld(t *testing.T) memberWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := memberWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin", "gina"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	for _, ws := range []struct{ slug, admin string }{{"acme", "alice"}, {"beta", "bob"}} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens[ws.admin],
			`{"name":"`+ws.slug+`","slug":"`+ws.slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", ws.slug, status, body)
		}
	}
	for _, m := range []struct {
		slug, admin, name string
		role              shared.Role
	}{{"acme", "alice", "bob", shared.RoleMember}, {"acme", "alice", "carol", shared.RoleMember}, {"acme", "alice", "dave", shared.RoleMember},
		{"acme", "alice", "gina", shared.RoleMember}, {"acme", "alice", "erin", shared.RoleGuest}, {"beta", "bob", "carol", shared.RoleMember}} {
		answerInvitation(t, contract, w.base, w.tokens[m.name], "accept",
			inviteAs(t, contract, w.base, w.tokens[m.admin], m.slug, m.name+"@example.com", m.role), http.StatusOK)
	}
	w.web = createdProject(t, contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	w.ops = createdProject(t, contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.lab = createdProject(t, contract, w.base, w.tokens["bob"], "beta", "Lab", "LAB")
	for _, add := range []struct {
		by      string
		project uuid.UUID
		members string
	}{
		{"alice", w.web, w.added("bob", 20, "dave", 20, "carol", 15, "gina", 15, "erin", 5)},
		{"alice", w.ops, w.added("carol", 20, "bob", 15)},
		{"bob", w.lab, w.added("carol", 15)},
	} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/projects/"+add.project.String()+"/members", w.tokens[add.by],
			add.members); status != http.StatusCreated {
			t.Fatalf("%s's adding %s = %d %s", add.by, add.members, status, body)
		}
	}
	gina := w.acmeMembership(t, "gina")
	if status, body := call(t, contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+gina.String(), w.tokens["alice"],
		`{"role":20}`); status != http.StatusOK {
		t.Fatalf("alice's making gina acme's admin = %d %s", status, body)
	}
	if got := w.standing(t); got != worldStanding {
		t.Fatalf("the world: %s; want %s", got, worldStanding)
	}
	return w
}

// added is the body of an addition of the named accounts, each with the
// role after its name.
func (w memberWorld) added(namesAndRoles ...any) string {
	body := `{"members":[`
	for i := 0; i < len(namesAndRoles); i += 2 {
		if i > 0 {
			body += ","
		}
		body += fmt.Sprintf(`{"member_id":"%s","role":%d}`, w.ids[namesAndRoles[i].(string)], namesAndRoles[i+1].(int))
	}
	return body + "]}"
}

// standing is every active membership of the world, the workspaces' and
// the projects', the workspaces first, then the projects, each by name with
// its members' roles in name order: "acme: alice 20, bob 15; Web: …". The
// names are ordered byte by byte (COLLATE "C"), whatever the database's
// collation.
func (w memberWorld) standing(t *testing.T) string {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT string_agg(place || ': ' || members, '; ' ORDER BY kind, place COLLATE "C") FROM (
		SELECT kind, place, string_agg(name || ' ' || role, ', ' ORDER BY name COLLATE "C") AS members FROM (
			SELECT 0 AS kind, s.slug AS place, split_part(u.email, '@', 1) AS name, m.role FROM workspace_members m
				JOIN workspaces s ON s.id = m.workspace_id JOIN users u ON u.id = m.member_id WHERE m.is_active AND m.deleted_at IS NULL
			UNION ALL SELECT 1, p.name, split_part(u.email, '@', 1), m.role FROM project_members m
				JOIN projects p ON p.id = m.project_id JOIN users u ON u.id = m.member_id WHERE m.is_active AND m.deleted_at IS NULL) a
		GROUP BY kind, place) b`).Scan(&got); err != nil {
		t.Fatal(err)
	}
	return got
}

// membership is the id of name's membership of project.
func (w memberWorld) membership(t *testing.T, project uuid.UUID, name string) uuid.UUID {
	t.Helper()
	return projectMemberships(t, w.pool, w.ids[name], project)[0]
}

// acmeMembership is the id of name's membership of acme. It fails the test
// on the test's goroutine, so call it there.
func (w memberWorld) acmeMembership(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(soon(t), `SELECT m.id FROM workspace_members m JOIN workspaces s ON s.id = m.workspace_id
		WHERE s.slug = 'acme' AND m.member_id = $1`, w.ids[name]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// change is by's change of name's role in project to role: its status and
// body.
func (w memberWorld) change(t *testing.T, by string, project uuid.UUID, name string, role int) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPatch, w.base+"/api/v0/project-members/"+w.membership(t, project, name).String(), w.tokens[by],
		fmt.Sprintf(`{"role":%d}`, role))
}
