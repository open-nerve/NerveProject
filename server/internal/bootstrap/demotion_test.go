package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// withBobLeadingWeb has alice create the workspace slug and invite bob,
// who accepts as a member, then create its project Web with bob its lead,
// and so its admin; alice and bob are the access tokens.
func withBobLeadingWeb(t *testing.T, contract *apitest.Contract, base, alice, bob string, bobID uuid.UUID, slug string) {
	t.Helper()
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice,
		`{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s = %d %s", slug, status, body)
	}
	answerInvitation(t, contract, base, bob, "accept", invite(t, contract, base, alice, slug, "bob@example.com"), http.StatusOK)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/projects", alice,
		`{"name":"Web","identifier":"WEB","project_lead_id":"`+bobID.String()+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s's project = %d %s", slug, status, body)
	}
}

// rolesOf are member's roles in each workspace and in its project, each
// marked when ended, and whether his membership of acme's project was last
// written by the account by when his membership of acme was.
func rolesOf(t *testing.T, pool *pgxpool.Pool, member, by uuid.UUID) string {
	t.Helper()
	var roles string
	if err := pool.QueryRow(context.Background(), `
SELECT string_agg(w.slug || ' ' || wm.role || CASE WHEN wm.is_active THEN '' ELSE ' ended' END
                  || ', project ' || pm.role || CASE WHEN pm.is_active THEN '' ELSE ' ended' END, '; ' ORDER BY w.slug)
       || ' | ' || bool_and(w.slug <> 'acme' OR (pm.updated_by_id = $2 AND pm.updated_at = wm.updated_at))
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
JOIN project_members pm ON pm.workspace_id = wm.workspace_id AND pm.member_id = wm.member_id
WHERE wm.member_id = $1`, member, by).Scan(&roles); err != nil {
		t.Fatal(err)
	}
	return roles
}

// failingDemotions makes every write of a guest's project membership fail,
// the projects' step of a demotion among them, until restore runs: a CHECK
// that the rows already there need not pass.
func failingDemotions(t *testing.T, pool *pgxpool.Pool) (restore func()) {
	t.Helper()
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	exec("ALTER TABLE project_members ADD CONSTRAINT no_guests CHECK (role <> 5) NOT VALID")
	return func() { exec("ALTER TABLE project_members DROP CONSTRAINT no_guests") }
}

// refusingCommits makes the commit of every transaction that changed a
// workspace membership fail, until restore runs, or the test ends: a
// deferred constraint trigger, which runs at the commit, after every
// statement.
func refusingCommits(t *testing.T, pool *pgxpool.Pool) (restore func()) {
	t.Helper()
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	exec(`CREATE FUNCTION refuse_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'the commit is refused'; END $$`)
	exec(`CREATE CONSTRAINT TRIGGER refuse_commit AFTER UPDATE ON workspace_members DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION refuse_commit()`)
	restore = func() {
		exec("DROP TRIGGER IF EXISTS refuse_commit ON workspace_members")
		exec("DROP FUNCTION IF EXISTS refuse_commit()")
	}
	t.Cleanup(restore)
	return restore
}

// A change of a member's role to guest makes him a guest in the
// workspace's projects in the same transaction (M3 design 3.3, 9.3), on the
// wired app: bob, a member of acme and of beta, leads a project in each,
// and so is its admin. While the projects' step fails, alice's change of
// his role in acme answers 500 and changes nothing. A failing step cannot
// show that the step shares the change's transaction; a change refused at
// its commit, after every statement ran, also changes nothing, which a step
// that wrote in a transaction of its own would have outlived. Then the
// change runs while another transaction holds bob's membership of acme's
// project: as its write waits for that row, acme's project is held FOR NO
// KEY UPDATE, so a FOR SHARE of it waits, which a lock taken outside the
// change's transaction would no longer be. Once the row is free, he is
// acme's guest and a guest in acme's project, by alice at the time of his
// workspace role's change, and still beta's member and the admin of its
// project.
func TestDemotingToGuestDemotesInTheWorkspacesProjects(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	aliceID, bobID := accountID(t, contract, base, alice), accountID(t, contract, base, bob)
	withBobLeadingWeb(t, contract, base, alice, bob, bobID, "acme")
	withBobLeadingWeb(t, contract, base, alice, bob, bobID, "beta")
	var membership uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT m.id FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1`, bobID).Scan(&membership); err != nil {
		t.Fatal(err)
	}
	patch := func() (int, string) {
		return call(t, contract, http.MethodPatch, base+"/api/v0/workspace-members/"+membership.String(), alice, `{"role":5}`)
	}
	before := rolesOf(t, pool, bobID, aliceID)
	if want := "acme 15, project 20; beta 15, project 20 | false"; before != want {
		t.Fatalf("bob's roles before = %s, want %s", before, want)
	}

	restore := failingDemotions(t, pool)
	status, body := patch()
	restore()
	if got := rolesOf(t, pool, bobID, aliceID); status != http.StatusInternalServerError || got != before {
		t.Errorf("the change with the projects' step failing = %d %s, roles %s; want 500 and %s", status, body, got, before)
	}

	restore = refusingCommits(t, pool)
	status, body = patch()
	restore()
	if got := rolesOf(t, pool, bobID, aliceID); status != http.StatusInternalServerError || got != before {
		t.Errorf("the change refused at its commit = %d %s, roles %s; want 500 and %s", status, body, got, before)
	}

	hold, err := pool.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = hold.Rollback(context.Background()) })
	if _, err := hold.Exec(context.Background(), `SELECT 1 FROM project_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 FOR UPDATE OF m`, bobID); err != nil {
		t.Fatal(err)
	}
	req := newRequest(t, http.MethodPatch, base+"/api/v0/workspace-members/"+membership.String(), alice, []byte(`{"role":5}`))
	contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	pgtest.WaitForLockWaitOn(t, pool, "project_members", 10*time.Second)
	_, err = pool.Exec(context.Background(), `SELECT 1 FROM projects p JOIN workspaces w ON w.id = p.workspace_id
		WHERE w.slug = 'acme' FOR SHARE OF p NOWAIT`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("a FOR SHARE of acme's project while the change waits = %v; want lock_not_available (55P03)", err)
	}
	if err := hold.Rollback(context.Background()); err != nil {
		t.Fatal(err)
	}
	a := receiveWithin(t, answered, 10*time.Second, "answer to the change")
	if a.err != nil {
		t.Fatal(a.err)
	}
	contract.CheckResponse(t, req, a.res)

	status, body = a.res.StatusCode, string(a.body)
	if got, want := rolesOf(t, pool, bobID, aliceID), "acme 5, project 5; beta 15, project 20 | true"; status != http.StatusOK || got != want {
		t.Errorf("the change = %d %s, roles %s; want 200 and %s", status, body, got, want)
	}
}
