package bootstrap

import (
	"context"
	"encoding/json"
	"maps"
	"net/http"
	"slices"
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
	if err := w.pool.QueryRow(context.Background(), `SELECT concat_ws(', ',
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
	if err := w.pool.QueryRow(context.Background(), `SELECT string_agg(p.name || ' ' || u.email || ' ' || m.role::text ||
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
	if err := w.pool.QueryRow(context.Background(), "SELECT id FROM workspaces WHERE slug = $1 AND deleted_at IS NULL", slug).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// membership is the id of name's membership of acme.
func (w endingWorld) membership(t *testing.T, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 AND m.deleted_at IS NULL`, w.ids[name]).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// leave is name's leaving of the workspace slug: its status and body.
func (w endingWorld) leave(t *testing.T, slug, name string) (int, string) {
	t.Helper()
	return call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces/"+slug+"/leave", w.tokens[name], "")
}

// rowJSON is the row id of table as JSON, its columns by name; nil when
// there is none.
func rowJSON(t *testing.T, pool *pgxpool.Pool, table string, id uuid.UUID) map[string]any {
	t.Helper()
	var text []byte
	if err := pool.QueryRow(context.Background(), "SELECT coalesce((SELECT row_to_json(r) FROM "+table+" r WHERE r.id = $1)::text, 'null')", id).
		Scan(&text); err != nil {
		t.Fatal(err)
	}
	var row map[string]any
	if err := json.Unmarshal(text, &row); err != nil {
		t.Fatal(err)
	}
	return row
}

// rowsBut is every table's rows as tableRows has them, River's left out,
// but the rows whose ids are in ids.
func rowsBut(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) map[string]string {
	t.Helper()
	all := map[string]string{}
	for table := range tableRows(t, pool, riversOwn) {
		var text string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(string_agg(t, E'\n' ORDER BY t), '') FROM (SELECT row_to_json(r)::text AS t
			FROM `+table+` r WHERE NOT to_jsonb(r) ? 'id' OR (to_jsonb(r)->>'id') <> ALL ($1::text[])) s`, uuidTexts(ids)).Scan(&text); err != nil {
			t.Fatalf("%s: %v", table, err)
		}
		all[table] = text
	}
	return all
}

// uuidTexts are ids as text.
func uuidTexts(ids []uuid.UUID) []string {
	out := make([]string, len(ids))
	for i, id := range ids {
		out[i] = id.String()
	}
	return out
}

// projectMemberships are the ids of user's memberships of the projects, in
// the order of projects.
func projectMemberships(t *testing.T, pool *pgxpool.Pool, user uuid.UUID, projects ...uuid.UUID) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, len(projects))
	for i, p := range projects {
		if err := pool.QueryRow(context.Background(), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
			p, user).Scan(&ids[i]); err != nil {
			t.Fatal(err)
		}
	}
	return ids
}

// An ending of name's membership of acme on the wired app: alice's removal
// of him, or his leaving.
type ending struct {
	name string
	// request is the ending's method, its path, and the access token it is
	// sent with.
	request func(w endingWorld, t *testing.T, name string) (method, path, token string)
	by      func(name string) string // the account it writes as
}

// end sends e's request on w's app: its status and body.
func (e ending) end(w endingWorld, t *testing.T, name string) (int, string) {
	t.Helper()
	method, path, token := e.request(w, t, name)
	return call(t, w.contract, method, w.base+path, token, "")
}

var endings = []ending{
	{name: "removal", request: func(w endingWorld, t *testing.T, name string) (string, string, string) {
		return http.MethodDelete, "/api/v0/workspace-members/" + w.membership(t, name).String(), w.tokens["alice"]
	}, by: func(string) string { return "alice" }},
	{name: "leaving", request: func(w endingWorld, _ *testing.T, name string) (string, string, string) {
		return http.MethodPost, "/api/v0/workspaces/acme/leave", w.tokens[name]
	}, by: func(name string) string { return name }},
}

// A removal and a leaving each end a membership and leave no invitation,
// in one transaction at one moment (M3 design 3.6, 3.7 rule 2, 3.8, 9.3),
// on the wired app.
//   - bob is Ops's only active admin and carol its member, erin's ended
//     membership as its admin counting for nothing: the ending of his
//     membership is 409 project.sole_admin, and no row of any table
//     changes, the pending invitation to his address, which the ending
//     would have deleted, still pending.
//   - Once alice, acme's admin, has joined Ops, as its admin, the ending
//     refused at its commit, after every statement ran, for each table it
//     writes, is 500, and no row changes: no step wrote in a transaction of
//     its own.
//   - Then it is 204, Solo, of which he is the only active member, refusing
//     nothing, nor Lab, beta's, of which he is the only admin: bob's
//     membership of acme, of Web, of Ops and of Solo ended, each row kept
//     with its role; the pending invitation to his address in acme deleted;
//     each by the ender, at one moment no earlier than the request. Each of
//     those rows was last written by dave before, so that the claim of the
//     ender's writing can fail. His membership of beta and of Lab, the
//     invitation to him there, and every other row of every table are as
//     they were.
//   - Ending dave's membership leaves his declined invitation as it was.
func TestAnEndingEndsTheMembershipsAndLeavesNoInvitation(t *testing.T) {
	for _, e := range endings {
		t.Run(e.name, func(t *testing.T) {
			w := newEndingWorld(t)
			before := tableRows(t, w.pool, riversOwn)
			if status, body := e.end(w, t, "bob"); status != http.StatusConflict || problemCode(t, []byte(body)) != "project.sole_admin" {
				t.Fatalf("ending bob's membership, Ops's only admin = %d %s, want 409 project.sole_admin", status, body)
			}
			if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
				t.Errorf("the tables after the refused ending changed:\n%v\nwant them as they were:\n%v", after, before)
			}
			if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/projects/"+w.ops.String()+"/join", w.tokens["alice"],
				""); status != http.StatusOK {
				t.Fatalf("alice's joining Ops = %d %s", status, body)
			}
			before = tableRows(t, w.pool, riversOwn)
			for _, table := range []string{"workspace_member_invites", "workspace_members", "project_members"} {
				restore := refusingCommits(t, w.pool, table)
				status, body := e.end(w, t, "bob")
				restore()
				if after := tableRows(t, w.pool, riversOwn); status != http.StatusInternalServerError || !maps.Equal(after, before) {
					t.Errorf("the ending refused at its commit for %s = %d %s; want 500 and every table as it was", table, status, body)
				}
			}
			written := slices.Concat([]uuid.UUID{w.membership(t, "bob"), w.bobsInvitation},
				projectMemberships(t, w.pool, w.ids["bob"], w.web, w.ops, w.solo))
			tables := []string{"workspace_members", "workspace_member_invites", "project_members", "project_members", "project_members"}
			rowsBefore := make([]map[string]any, len(written))
			for i, id := range written {
				if tag, err := w.pool.Exec(context.Background(), "UPDATE "+tables[i]+" SET updated_by_id = $2 WHERE id = $1", id,
					w.ids["dave"]); err != nil || tag.RowsAffected() != 1 {
					t.Fatalf("%s %s last written by dave: %v, %v", tables[i], id, tag, err)
				}
				rowsBefore[i] = rowJSON(t, w.pool, tables[i], id)
			}
			others := rowsBut(t, w.pool, written)
			started := time.Now()

			if status, body := e.end(w, t, "bob"); status != http.StatusNoContent {
				t.Fatalf("ending bob's membership = %d %s, want 204", status, body)
			}

			moment, _ := rowJSON(t, w.pool, "workspace_member_invites", w.bobsInvitation)["deleted_at"].(string)
			if at, err := time.Parse(time.RFC3339Nano, moment); err != nil || at.Before(started.Truncate(time.Microsecond)) {
				t.Errorf("the invitation deleted at %q (%v); want a moment no earlier than the request, %v", moment, err, started)
			}
			for i, id := range written {
				after := rowJSON(t, w.pool, tables[i], id)
				want := maps.Clone(rowsBefore[i])
				want["updated_at"], want["updated_by_id"] = moment, w.ids[e.by("bob")].String()
				if tables[i] == "workspace_member_invites" {
					want["deleted_at"] = moment
				} else {
					want["is_active"] = false
				}
				if !maps.Equal(after, want) {
					t.Errorf("%s %s after the ending:\n%v\nwant\n%v", tables[i], id, after, want)
				}
			}
			if after := rowsBut(t, w.pool, written); !maps.Equal(after, others) {
				t.Errorf("every other row after the ending:\n%v\nwant them as they were:\n%v", after, others)
			}
			declined := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined)
			if status, body := e.end(w, t, "dave"); status != http.StatusNoContent {
				t.Fatalf("ending dave's membership = %d %s, want 204", status, body)
			}
			if after := rowJSON(t, w.pool, "workspace_member_invites", w.davesDeclined); !maps.Equal(after, declined) {
				t.Errorf("dave's declined invitation after the ending:\n%v\nwant it as it was:\n%v", after, declined)
			}
		})
	}
}

// A workspace's only active admin cannot leave it (M3 design 3.7 rule 1,
// 9.3): alice is refused 409 workspace.sole_admin in solo, where she is
// alone, and in acme, beside its members, and no row of any table changes;
// once carol is acme's admin too, alice leaves it.
func TestTheOnlyAdminCannotLeave(t *testing.T) {
	w := newEndingWorld(t)
	if status, body := call(t, w.contract, http.MethodPost, w.base+"/api/v0/workspaces", w.tokens["alice"],
		`{"name":"solo","slug":"solo"}`); status != http.StatusCreated {
		t.Fatalf("creating solo = %d %s", status, body)
	}
	before := tableRows(t, w.pool, riversOwn)
	for _, slug := range []string{"solo", "acme"} {
		if status, body := w.leave(t, slug, "alice"); status != http.StatusConflict || problemCode(t, []byte(body)) != "workspace.sole_admin" {
			t.Errorf("alice's leaving %s, its only admin = %d %s, want 409 workspace.sole_admin", slug, status, body)
		}
	}
	if after := tableRows(t, w.pool, riversOwn); !maps.Equal(after, before) {
		t.Errorf("the tables after the refused leavings changed:\n%v\nwant them as they were:\n%v", after, before)
	}
	if status, body := call(t, w.contract, http.MethodPatch, w.base+"/api/v0/workspace-members/"+w.membership(t, "carol").String(),
		w.tokens["alice"], `{"role":20}`); status != http.StatusOK {
		t.Fatalf("carol made acme's admin = %d %s", status, body)
	}
	if status, body := w.leave(t, "acme", "alice"); status != http.StatusNoContent {
		t.Errorf("alice's leaving acme, carol its admin too = %d %s, want 204", status, body)
	}
}
