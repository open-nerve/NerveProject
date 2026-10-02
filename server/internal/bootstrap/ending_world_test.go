package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// endingWorld is the wired app on a database of its own: acme, alice its
// admin; bob, carol, dave and erin its members, each by accepting alice's
// invitation; Web, alice's, led by bob, so both are its admins, and carol
// its member by joining; Ops, bob's, his alone to administer, carol its
// member by his adding, and erin, whom he added as its admin; Solo, bob's,
// erin its member by his adding; erin's memberships of Ops and Solo ended
// by alice's removal of her from acme, so that bob is Solo's only active
// member; beta, alice its admin, bob and carol its members; Lab, beta's,
// bob's, his alone to administer, carol its member by joining.
// Then, through the workspace store, the invitations no operation makes,
// 3.8 refusing to invite an active member: one pending to bob's address in
// acme, and one in beta; one pending to carol's in acme; one to dave's in
// acme that he declined.
type endingWorld struct {
	contract       *apitest.Contract
	base           string
	pool           *pgxpool.Pool
	tokens         map[string]string    // access tokens, by name
	ids            map[string]uuid.UUID // accounts, by name
	web, ops, solo uuid.UUID
	lab            uuid.UUID // beta's
	bobsInvitation uuid.UUID // the pending one to bob's address in acme
	davesDeclined  uuid.UUID
}

func newEndingWorld(t *testing.T) endingWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := endingWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "carol", "dave", "erin"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	for _, slug := range []string{"acme", "beta"} {
		if status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
			`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
			t.Fatalf("creating %s = %d %s", slug, status, body)
		}
	}
	for _, m := range []struct{ slug, name string }{{"acme", "bob"}, {"acme", "carol"}, {"acme", "dave"}, {"acme", "erin"},
		{"beta", "bob"}, {"beta", "carol"}} {
		answerInvitation(t, contract, w.base, w.tokens[m.name], "accept", invite(t, contract, w.base, w.tokens["alice"], m.slug, m.name+"@example.com"),
			http.StatusOK)
	}
	status, body := call(t, contract, http.MethodPost, w.base+"/api/v0/workspaces/acme/projects", w.tokens["alice"],
		`{"name":"Web","identifier":"WEB","project_lead_id":"`+w.ids["bob"].String()+`"}`)
	var web struct {
		ID uuid.UUID `json:"id"`
	}
	if status != http.StatusCreated {
		t.Fatalf("creating Web = %d %s", status, body)
	}
	decodeAnswer(t, body, &web)
	w.web, w.ops = web.ID, createdProject(t, contract, w.base, w.tokens["bob"], "acme", "Ops", "OPS")
	w.solo = createdProject(t, contract, w.base, w.tokens["bob"], "acme", "Solo", "SOLO")
	w.lab = createdProject(t, contract, w.base, w.tokens["bob"], "beta", "Lab", "LAB")
	for _, step := range []struct{ token, path, body string }{
		{w.tokens["carol"], "/api/v0/projects/" + w.web.String() + "/join", ""},
		{w.tokens["carol"], "/api/v0/projects/" + w.lab.String() + "/join", ""},
		{w.tokens["bob"], "/api/v0/projects/" + w.ops.String() + "/members", `{"members":[{"member_id":"` + w.ids["carol"].String() +
			`","role":15},{"member_id":"` + w.ids["erin"].String() + `","role":20}]}`},
		{w.tokens["bob"], "/api/v0/projects/" + w.solo.String() + "/members", `{"members":[{"member_id":"` + w.ids["erin"].String() +
			`","role":15}]}`},
	} {
		if status, body := call(t, contract, http.MethodPost, w.base+step.path, step.token, step.body); status != http.StatusOK &&
			status != http.StatusCreated {
			t.Fatalf("POST %s = %d %s", step.path, status, body)
		}
	}
	if status, body := call(t, contract, http.MethodDelete, w.base+"/api/v0/workspace-members/"+w.membership(t, "erin").String(),
		w.tokens["alice"], ""); status != http.StatusNoContent {
		t.Fatalf("alice's removal of erin = %d %s", status, body)
	}
	store, ctx := workspacepg.New(w.pool), context.Background()
	for _, inv := range []struct {
		slug, name string
		id         *uuid.UUID
	}{{"acme", "bob", &w.bobsInvitation}, {"beta", "bob", nil}, {"acme", "carol", nil}, {"acme", "dave", &w.davesDeclined}} {
		id := uuid.NewV7()
		if _, err := store.CreateInvitations(ctx, []workspaceapp.InvitationRow{{ID: id, WorkspaceID: w.workspace(t, inv.slug),
			Email: inv.name + "@example.com", Role: shared.RoleGuest, CreatedBy: w.ids["alice"], Now: time.Now()}}); err != nil {
			t.Fatal(err)
		}
		if inv.id != nil {
			*inv.id = id
		}
	}
	if err := store.DeclineInvitation(ctx, w.davesDeclined, w.ids["dave"], time.Now()); err != nil {
		t.Fatal(err)
	}
	w.bystanders(t)
	w.soleAdmins(t)
	return w
}

// bystanders checks the rows an ending of bob's membership of acme must
// leave as they were, each read by what makes it the one it is: his active
// membership of beta; the pending invitations to his address in beta and
// to carol's in acme; the one to dave's in acme, declined. Were one
// missing, an ending that also wrote it would pass.
func (w endingWorld) bystanders(t *testing.T) {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT concat_ws(', ',
		(SELECT 'bob in beta' FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
			WHERE w.slug = 'beta' AND m.member_id = $1 AND m.is_active AND m.deleted_at IS NULL),
		(SELECT string_agg(w.slug || ' ' || i.email || CASE WHEN i.responded_at IS NULL THEN ' pending' WHEN i.accepted THEN ' accepted'
			ELSE ' declined' END, ', ' ORDER BY w.slug, i.email)
			FROM workspace_member_invites i JOIN workspaces w ON w.id = i.workspace_id WHERE i.deleted_at IS NULL))`, w.ids["bob"]).
		Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "bob in beta, acme bob@example.com pending, acme carol@example.com pending, acme dave@example.com declined, " +
		"beta bob@example.com pending"; got != want {
		t.Fatalf("the rows an ending leaves: %s; want %s", got, want)
	}
}

// soleAdmins checks the project memberships that decide 3.7 rule 2 for
// bob: in Ops he is the only active admin beside carol, a member, and erin,
// an admin whose membership has ended; in Solo he is the only active
// member, erin's membership as its member ended; in Lab, beta's, the only
// admin beside carol, a member. Were one missing, a rule 2 that counted an
// ended admin as another, counted an ended member as another, refused a
// project with no other member, or judged another workspace's projects,
// would pass.
func (w endingWorld) soleAdmins(t *testing.T) {
	t.Helper()
	var got string
	if err := w.pool.QueryRow(soon(t), `SELECT string_agg(p.name || ' ' || u.email || ' ' || m.role::text ||
		CASE WHEN m.is_active THEN ' active' ELSE ' ended' END, ', ' ORDER BY p.name, u.email)
		FROM project_members m JOIN projects p ON p.id = m.project_id JOIN users u ON u.id = m.member_id
		WHERE m.project_id IN ($1, $2, $3) AND m.deleted_at IS NULL`, w.ops, w.solo, w.lab).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if want := "Lab bob@example.com 20 active, Lab carol@example.com 15 active, Ops bob@example.com 20 active, " +
		"Ops carol@example.com 15 active, Ops erin@example.com 20 ended, Solo bob@example.com 20 active, " +
		"Solo erin@example.com 15 ended"; got != want {
		t.Fatalf("the memberships of Ops, Solo and Lab: %s; want %s", got, want)
	}
}

// workspace is the id of the workspace slug.
func (w endingWorld) workspace(t *testing.T, slug string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(soon(t), "SELECT id FROM workspaces WHERE slug = $1 AND deleted_at IS NULL", slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// membership is the id of name's membership of acme. It is read for the
// removal's request while TestEachLockOfAnEndingIsItsStrength's holders
// are open: its wait for a connection ends soon.
func (w endingWorld) membership(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(soon(t), `SELECT m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 AND m.deleted_at IS NULL`, w.ids[name]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// joinsOps has alice join Ops, as its admin, so that bob is no longer its
// only admin and the ending of his membership can go through.
func (w endingWorld) joinsOps(t *testing.T) {
	t.Helper()
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.ops.String()+"/join", w.tokens["alice"],
		""); status != http.StatusOK {
		t.Fatalf("alice's joining Ops = %d %s", status, body)
	}
}

// leave is name's leaving of the workspace slug: its status and body.
func (w endingWorld) leave(t *testing.T, slug, name string) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/"+slug+"/leave", w.tokens[name], "")
}
