package bootstrap

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
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
// marked when ended, and whether his memberships of acme and of acme's
// project were both last written by the account by, at one moment.
func rolesOf(t *testing.T, pool *pgxpool.Pool, member, by uuid.UUID) string {
	t.Helper()
	var roles string
	if err := pool.QueryRow(context.Background(), `
SELECT string_agg(w.slug || ' ' || wm.role || CASE WHEN wm.is_active THEN '' ELSE ' ended' END
                  || ', project ' || pm.role || CASE WHEN pm.is_active THEN '' ELSE ' ended' END, '; ' ORDER BY w.slug)
       || ' | ' || bool_and(w.slug <> 'acme'
                            OR (wm.updated_by_id = $2 AND pm.updated_by_id = $2 AND pm.updated_at = wm.updated_at))
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

// refusingCommits makes the commit of every transaction that inserted or
// updated a row of table fail, until restore runs, or the test ends: a
// deferred constraint trigger, which runs at the commit, after every
// statement.
func refusingCommits(t *testing.T, pool *pgxpool.Pool, table string) (restore func()) {
	t.Helper()
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	exec(`CREATE FUNCTION refuse_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'the commit is refused'; END $$`)
	exec(`CREATE CONSTRAINT TRIGGER refuse_commit AFTER INSERT OR UPDATE ON ` + table + ` DEFERRABLE INITIALLY DEFERRED
		FOR EACH ROW EXECUTE FUNCTION refuse_commit()`)
	restore = func() {
		exec("DROP TRIGGER IF EXISTS refuse_commit ON " + table)
		exec("DROP FUNCTION IF EXISTS refuse_commit()")
	}
	t.Cleanup(restore)
	return restore
}

// sendHoldingTheMemberships sends req, which makes member a guest in
// acme's projects, while other transactions hold his membership of acme
// FOR SHARE, as P4b's add and join will, and then his membership of acme's
// project FOR UPDATE, and answers what req got once both have rolled back.
// req takes M3 design 3.6's global order, workspace_members before
// projects: while it waits for his membership of acme, it has locked none
// of his projects in acme yet, so a FOR SHARE NOWAIT of each succeeds,
// where a demotion run before the membership's write would hold them FOR
// NO KEY UPDATE; nor has it written his project membership, which is held
// NOWAIT. Then, as req's write of his project membership waits for
// that row, the project must be held FOR NO KEY UPDATE, so that a FOR SHARE
// of it NOWAIT fails with lock_not_available (55P03): a lock taken outside
// req's transaction would no longer be held.
func sendHoldingTheMemberships(t *testing.T, contract *apitest.Contract, pool *pgxpool.Pool, req *http.Request, member uuid.UUID) (int, string) {
	t.Helper()
	ctx := context.Background()
	hold := func(sql string) pgx.Tx {
		t.Helper()
		tx, err := pool.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = tx.Rollback(ctx) })
		if _, err := tx.Exec(ctx, sql, member); err != nil {
			t.Fatal(err)
		}
		return tx
	}
	release := func(tx pgx.Tx) {
		t.Helper()
		if err := tx.Rollback(ctx); err != nil {
			t.Fatal(err)
		}
	}
	membership := hold(`SELECT 1 FROM workspace_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 FOR SHARE OF m`)
	contract.CheckRequest(t, req)
	answered := sendInBackground(req)
	pgtest.WaitForLockWaitOn(t, pool, "workspace_members", 10*time.Second)
	var free int
	err := pool.QueryRow(ctx, `SELECT count(*) FROM (SELECT p.id FROM projects p JOIN workspaces w ON w.id = p.workspace_id
		WHERE w.slug = 'acme' AND EXISTS (SELECT 1 FROM project_members m WHERE m.project_id = p.id AND m.member_id = $1)
		FOR SHARE OF p NOWAIT) s`, member).Scan(&free)
	if err != nil || free == 0 {
		t.Errorf("a FOR SHARE NOWAIT of his projects in acme while %s %s waits for his membership of acme = %d, %v; "+
			"want each locked at once, and one at least", req.Method, req.URL.Path, free, err)
	}
	projectMembership := hold(`SELECT 1 FROM project_members m JOIN workspaces w ON w.id = m.workspace_id
		WHERE w.slug = 'acme' AND m.member_id = $1 FOR UPDATE OF m NOWAIT`)
	release(membership)
	pgtest.WaitForLockWaitOn(t, pool, "project_members", 10*time.Second)
	_, err = pool.Exec(ctx, `SELECT 1 FROM projects p JOIN workspaces w ON w.id = p.workspace_id
		WHERE w.slug = 'acme' FOR SHARE OF p NOWAIT`)
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "55P03" {
		t.Errorf("a FOR SHARE of acme's project while %s %s waits = %v; want lock_not_available (55P03)", req.Method, req.URL.Path, err)
	}
	release(projectMembership)
	a := receiveWithin(t, answered, 10*time.Second, "answer to "+req.Method+" "+req.URL.Path)
	if a.err != nil {
		t.Fatal(a.err)
	}
	contract.CheckResponse(t, req, a.res)
	return a.res.StatusCode, string(a.body)
}

// A change of a member's role to guest makes him a guest in the
// workspace's projects in the same transaction (M3 design 3.3, 9.3), on the
// wired app: bob, a member of acme and of beta, leads a project in each,
// and so is its admin. While the projects' step fails, alice's change of
// his role in acme answers 500 and changes nothing. A failing step cannot
// show that the step shares the change's transaction; a change refused at
// its commit, after every statement ran, also changes nothing, which a step
// that wrote in a transaction of its own would have outlived. Then the
// change runs while another transaction holds bob's membership of acme FOR
// SHARE: as the change waits for that row, it has locked no project yet (M3
// design 3.6's order). Once that row is free, while another transaction
// holds his membership of acme's project: as the change's write waits for
// that row, acme's project is held FOR NO KEY UPDATE, so a FOR SHARE of it
// waits, which a lock taken outside the change's transaction would no
// longer be. Once that row is free too, he is acme's guest and a guest in
// acme's project, by alice at the time of his workspace role's change, and
// still beta's member and the admin of its project.
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

	restore = refusingCommits(t, pool, "workspace_members")
	status, body = patch()
	restore()
	if got := rolesOf(t, pool, bobID, aliceID); status != http.StatusInternalServerError || got != before {
		t.Errorf("the change refused at its commit = %d %s, roles %s; want 500 and %s", status, body, got, before)
	}

	status, body = sendHoldingTheMemberships(t, contract, pool,
		newRequest(t, http.MethodPatch, base+"/api/v0/workspace-members/"+membership.String(), alice, []byte(`{"role":5}`)), bobID)
	if got, want := rolesOf(t, pool, bobID, aliceID), "acme 5, project 5; beta 15, project 20 | true"; status != http.StatusOK || got != want {
		t.Errorf("the change = %d %s, roles %s; want 200 and %s", status, body, got, want)
	}
}

// Accepting an invitation as a guest that restores an ended membership
// makes its member a guest in the workspace's projects in the same
// transaction (M3 design 3.8, 9.3), on the wired app: bob led acme's
// project Web, so was its admin, beside alice, its creator. He is the last
// writer of both his memberships, of acme and of Web (SQL makes him so),
// and alice has removed him from acme, which ended both, at one moment, by
// her. She invites him again, as a guest. While the projects' step fails,
// his acceptance answers 500 and changes nothing, the invitation still
// pending. So does an acceptance refused at its commit, after every
// statement ran (its restoring updates his membership of acme), which a
// step that wrote in a transaction of its own would have outlived. Then he
// accepts while another transaction holds his membership of acme FOR
// SHARE: as the restoring waits for that row, no project is locked yet (M3
// design 3.6's order). Once that row is free, while another transaction
// holds his membership of Web: as the step's write waits for that row, Web
// is held FOR NO KEY UPDATE, so a FOR SHARE of it waits, which a lock taken
// outside the acceptance's transaction would no longer be. Once that row
// is free too, he is acme's guest again and a guest in Web, still ended
// there, both his memberships by himself at the time of the restoring, and
// the invitation is consumed.
func TestAcceptingAsAGuestAgainDemotesInTheWorkspacesProjects(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	bob := registerAccount(t, contract, base, "bob@example.com").AccessToken
	aliceID, bobID := accountID(t, contract, base, alice), accountID(t, contract, base, bob)
	withBobLeadingWeb(t, contract, base, alice, bob, bobID, "acme")
	// bob is made the last writer of both his memberships, so that "by
	// alice" after her removal can fail for either: his acceptance wrote his
	// membership of acme, but her creation of Web wrote his membership of it.
	for _, table := range []string{"workspace_members", "project_members"} {
		if tag, err := pool.Exec(context.Background(), "UPDATE "+table+" SET updated_by_id = $1 WHERE member_id = $1", bobID); err != nil ||
			tag.RowsAffected() != 1 {
			t.Fatalf("bob's row of %s last written by him: %v, %v", table, tag, err)
		}
	}
	var membership uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT id FROM workspace_members WHERE member_id = $1", bobID).Scan(&membership); err != nil {
		t.Fatal(err)
	}
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspace-members/"+membership.String(), alice, ""); status !=
		http.StatusNoContent {
		t.Fatalf("alice's removal of bob = %d %s", status, body)
	}
	link := inviteAs(t, contract, base, alice, "acme", "bob@example.com", shared.RoleGuest)
	url, reqBody := base+"/api/v0/workspace-invitations/"+link.id.String()+"/accept", `{"token":"`+link.token+`"}`
	accept := func() (int, string) {
		return call(t, contract, http.MethodPost, url, bob, reqBody)
	}
	pending := func() bool {
		t.Helper()
		var pending bool
		if err := pool.QueryRow(context.Background(),
			"SELECT EXISTS (SELECT 1 FROM workspace_member_invites WHERE id = $1 AND deleted_at IS NULL AND NOT accepted)", link.id).
			Scan(&pending); err != nil {
			t.Fatal(err)
		}
		return pending
	}
	// alice's removal wrote both his rows, at one moment, so neither is his.
	before := rolesOf(t, pool, bobID, aliceID)
	if want := "acme 15 ended, project 20 ended | true"; before != want {
		t.Fatalf("bob's roles before, by alice = %s, want %s", before, want)
	}

	restore := failingDemotions(t, pool)
	status, body := accept()
	restore()
	if got := rolesOf(t, pool, bobID, aliceID); status != http.StatusInternalServerError || got != before || !pending() {
		t.Errorf("the acceptance with the projects' step failing = %d %s, roles %s; want 500, %s and the invitation pending", status, body, got,
			before)
	}

	restore = refusingCommits(t, pool, "workspace_members")
	status, body = accept()
	restore()
	if got := rolesOf(t, pool, bobID, aliceID); status != http.StatusInternalServerError || got != before || !pending() {
		t.Errorf("the acceptance refused at its commit = %d %s, roles %s; want 500, %s and the invitation pending", status, body, got, before)
	}

	status, body = sendHoldingTheMemberships(t, contract, pool, newRequest(t, http.MethodPost, url, bob, []byte(reqBody)), bobID)
	if got, want := rolesOf(t, pool, bobID, bobID), "acme 5, project 5 ended | true"; status != http.StatusOK || got != want || pending() {
		t.Errorf("the acceptance = %d %s, roles %s; want 200, %s and the invitation consumed", status, body, got, want)
	}
}
