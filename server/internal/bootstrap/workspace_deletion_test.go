package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"net/http"
	"slices"
	"strings"
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

// The cascade of deleteWorkspace against the catalog (M3 design 3.3, 3.6,
// 4): a workspace deleted through the wired app leaves no undeleted row
// under it, in any table with a foreign key to workspaces, whichever module
// owns the table; deletes each at the workspace's instant, by its deleter;
// and changes no row of another workspace. The tables come from
// pg_constraint, not from a list kept here: a phase that adds a table under
// workspaces fails this test until it seeds a row of it (seedWorkspace) and
// the cascade deletes it: the projects' tables through ProjectCascade. A
// table under workspaces only through another table, such as projects,
// fails it too. This file holds the tests and the seed; the reading of the
// catalog and of each key's rows is in workspace_deletion_catalog_test.go.

// survivesItsWorkspace are the foreign keys to workspaces, as table.column,
// whose rows must outlive the workspace's deletion, each with its reason.
// None today: profiles.last_workspace_id, which the deletion keeps (M3
// design 3.14), has no foreign key. An entry that no foreign key matches is
// reported.
var survivesItsWorkspace = map[string]string{}

// rowsUnder is what one foreign key to the parent's table (the parent row
// itself as its id) held under the test's two parent rows, a workspace or
// a project each, before and after the deletion: the deleted one's
// undeleted rows, counted, and the kept one's rows, as text.
type rowsUnder struct {
	key                         string // table.column
	deletedBefore, deletedAfter int
	keptBefore, keptAfter       string
}

// deletionViolations reports, for each foreign key to parent, the parent's
// table ("workspaces" or "projects", as keysTo takes it): no row under
// either parent row before the deletion, which would leave its checks
// nothing to see; an undeleted row left under the deleted one, or, for a
// key on exempt, a row deleted that must survive; a row of the kept one
// changed. An exempt key that no foreign key matches is reported too.
func deletionViolations(parent string, under []rowsUnder, exempt map[string]string) []string {
	var found []string
	row := rowOf(parent)
	for _, r := range under {
		reason, survives := exempt[r.key]
		switch {
		case r.deletedBefore == 0:
			found = append(found, fmt.Sprintf("%s: seed a row under the deleted %s", r.key, row))
		case survives && r.deletedAfter != r.deletedBefore:
			found = append(found, fmt.Sprintf("%s: %d of %d rows deleted with the %s, want them kept: %s",
				r.key, r.deletedBefore-r.deletedAfter, r.deletedBefore, row, reason))
		case !survives && r.deletedAfter != 0:
			found = append(found, fmt.Sprintf("%s: %d rows left undeleted under the deleted %s: the cascade misses the table",
				r.key, r.deletedAfter, row))
		}
		switch {
		case r.keptBefore == "":
			found = append(found, fmt.Sprintf("%s: seed a row under the kept %s", r.key, row))
		case r.keptAfter != r.keptBefore:
			found = append(found, fmt.Sprintf("%s: the kept %s's rows changed:\n%s\nwant\n%s", r.key, row, r.keptAfter, r.keptBefore))
		}
	}
	for _, key := range slices.Sorted(maps.Keys(exempt)) {
		if !slices.ContainsFunc(under, func(r rowsUnder) bool { return r.key == key }) {
			found = append(found, fmt.Sprintf("the exempt %s is no foreign key to %s", key, parent))
		}
	}
	return found
}

// Deleting a workspace through the API soft-deletes every row under it in
// every table the catalog ties to workspaces, and the workspace row, each
// at the workspace's deleted_at and by the account that deleted it, and
// changes nothing under another workspace; the deletion is logged once.
// Rows deleted before keep their own instant: they are not among the rows
// recorded before the deletion.
func TestDeletingAWorkspaceLeavesNoUndeletedRowUnderIt(t *testing.T) {
	contract := apitest.Load(t)
	url := pgtest.NewDatabase(t)
	var logs lockedBuffer
	base := startAppLogging(t, testConfig(t, url, false), migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	deleted, kept := seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")
	keys := keysTo(t, pool, "workspaces")
	under, recorded := make([]rowsUnder, len(keys)), make([][]uuid.UUID, len(keys))
	for i, k := range keys {
		recorded[i] = k.undeleted(t, pool, deleted)
		under[i] = rowsUnder{key: k.String(), deletedBefore: len(recorded[i]), keptBefore: k.rows(t, pool, kept)}
	}

	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/deleted", admin, ""); status != http.StatusNoContent {
		t.Fatalf("deleting the workspace = %d %s, want 204", status, body)
	}

	for i, k := range keys {
		under[i].deletedAfter, under[i].keptAfter = len(k.undeleted(t, pool, deleted)), k.rows(t, pool, kept)
	}
	for _, v := range deletionViolations("workspaces", under, survivesItsWorkspace) {
		t.Error(v)
	}
	var adminID uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT id FROM users WHERE email = 'admin@example.com'").Scan(&adminID); err != nil {
		t.Fatal(err)
	}
	for i, k := range keys {
		if _, survives := survivesItsWorkspace[k.String()]; !survives {
			if rows := k.unstamped(t, pool, recorded[i], deleted, adminID); rows != "" {
				t.Errorf("%s: rows not deleted at the workspace's deleted_at by its deleter %s:\n%s", k, adminID, rows)
			}
		}
	}
	want := `msg="workspace deleted" workspace_id=` + deleted.String() + " user_id=" + adminID.String() + "\n"
	if out := logs.String(); strings.Count(out, `msg="workspace deleted"`) != 1 || !strings.Contains(out, want) {
		t.Errorf("the logs:\n%s\nwant one line %q", out, want)
	}
}

// seedWorkspace creates the workspace slug through the API, the caller of
// token its admin, and seeds a row of each table under it: the admin's
// membership, which the creation writes; the member's, and an invitation
// the admin sent, through the workspace store; the admin's display
// settings, through the API; a project with the member's membership, his
// display settings in it and a state (seedProject). A phase that adds a
// table under workspaces seeds a row of it here. It returns the workspace's
// id.
func seedWorkspace(t *testing.T, contract *apitest.Contract, base string, pool *pgxpool.Pool, token, slug string) uuid.UUID {
	t.Helper()
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", token, `{"name":"`+slug+`","slug":"`+slug+`"}`); status != http.StatusCreated {
		t.Fatalf("creating %s = %d %s, want 201", slug, status, body)
	}
	var id, member, admin uuid.UUID
	if err := pool.QueryRow(context.Background(), `SELECT (SELECT id FROM workspaces WHERE slug = $1), (SELECT id FROM users WHERE email = $2),
		(SELECT created_by_id FROM workspaces WHERE slug = $1)`, slug, "member@example.com").Scan(&id, &member, &admin); err != nil {
		t.Fatal(err)
	}
	store := workspacepg.New(pool)
	if err := store.CreateMember(context.Background(), workspaceapp.MemberRow{
		ID: uuid.NewV7(), WorkspaceID: id, MemberID: member, Role: shared.RoleMember, CreatedBy: member, Now: time.Now(),
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateInvitations(context.Background(), []workspaceapp.InvitationRow{
		{ID: uuid.NewV7(), WorkspaceID: id, Email: "invitee@example.com", Role: shared.RoleGuest, CreatedBy: admin, Now: time.Now()},
	}); err != nil {
		t.Fatal(err)
	}
	if status, body := call(t, contract, http.MethodPatch, base+"/api/v0/me/workspaces/"+slug+"/preferences", token,
		`{"navigation_project_limit":3}`); status != http.StatusOK {
		t.Fatalf("the settings in %s = %d %s, want 200", slug, status, body)
	}
	seedProject(t, pool, id, admin, member)
	return id
}

// seedProject writes a project of the workspace id, created by admin,
// with member's membership, his display settings in it and one state,
// directly: the seed does not depend on which of the project module's
// writes exist, nor on what they write besides.
func seedProject(t *testing.T, pool *pgxpool.Pool, id, admin, member uuid.UUID) {
	t.Helper()
	ctx, project := context.Background(), uuid.NewV7()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{"INSERT INTO projects (id, workspace_id, name, identifier, created_by_id) VALUES ($1, $2, 'Web', 'WEB', $3)", []any{project, id, admin}},
		{"INSERT INTO project_members (id, workspace_id, project_id, member_id, role) VALUES ($1, $2, $3, $4, 15)",
			[]any{uuid.NewV7(), id, project, member}},
		{"INSERT INTO project_user_properties (id, workspace_id, project_id, user_id) VALUES ($1, $2, $3, $4)",
			[]any{uuid.NewV7(), id, project, member}},
		{`INSERT INTO states (id, workspace_id, project_id, name, color, "default") VALUES ($1, $2, $3, 'Backlog', '#60646C', true)`,
			[]any{uuid.NewV7(), id, project}},
	} {
		if _, err := pool.Exec(ctx, stmt.sql, stmt.args...); err != nil {
			t.Fatal(err)
		}
	}
}

// A failing projects' step fails the whole deletion (M3 design 3.3, 9.3):
// when it fails on the wired app, deleteWorkspace answers the failure and
// every row rolls back, under either workspace, the projects' and the
// workspace's own. The states table, renamed while the request runs, fails
// the last statement of the cascade. It does not show that the statements
// share the deletion's one transaction: the projects' statements in a
// transaction of their own roll back here too, with the failing one.
// TestADeletionRefusedAtItsCommitChangesNoRow shows that.
func TestAFailedProjectsStepRollsTheDeletionBack(t *testing.T) {
	contract := apitest.Load(t)
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	ids := []uuid.UUID{seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")}
	rowsOf := func() []string {
		var all []string
		for _, k := range keysTo(t, pool, "workspaces") {
			for _, id := range ids {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	exec := func(sql string) {
		t.Helper()
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	exec("ALTER TABLE states RENAME TO states_away")
	status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/deleted", admin, "")
	exec("ALTER TABLE states_away RENAME TO states")
	if status != http.StatusInternalServerError {
		t.Fatalf("deleting the workspace with the states' step failing = %d %s, want 500", status, body)
	}
	if after := rowsOf(); !slices.Equal(after, before) {
		t.Errorf("the rows after the failed deletion:\n%q\nwant\n%q", after, before)
	}
}

// Every statement of the deletion runs in its one transaction (M3 design
// 3.3, 9.3), the cascade's last one too: none outside any transaction, and
// none in a transaction of its own, which would commit before the
// deletion's. A failing step cannot show it; a deletion refused at its
// commit, after every statement ran, changes no row under either
// workspace. A deferred constraint trigger on workspaces refuses the
// commit.
func TestADeletionRefusedAtItsCommitChangesNoRow(t *testing.T) {
	contract := apitest.Load(t)
	url := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, url, false), migrations.FS())
	pool := openPool(t, url)
	admin := registerAccount(t, contract, base, "admin@example.com").AccessToken
	registerAccount(t, contract, base, "member@example.com")
	ids := []uuid.UUID{seedWorkspace(t, contract, base, pool, admin, "deleted"), seedWorkspace(t, contract, base, pool, admin, "kept")}
	rowsOf := func() []string {
		var all []string
		for _, k := range keysTo(t, pool, "workspaces") {
			for _, id := range ids {
				all = append(all, k.String()+":\n"+k.rows(t, pool, id))
			}
		}
		return all
	}
	before := rowsOf()
	for _, sql := range []string{
		`CREATE FUNCTION refuse_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'the commit is refused'; END $$`,
		`CREATE CONSTRAINT TRIGGER refuse_commit AFTER UPDATE ON workspaces DEFERRABLE INITIALLY DEFERRED
			FOR EACH ROW EXECUTE FUNCTION refuse_commit()`,
	} {
		if _, err := pool.Exec(context.Background(), sql); err != nil {
			t.Fatal(err)
		}
	}
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/workspaces/deleted", admin, ""); status != http.StatusInternalServerError {
		t.Fatalf("deleting the workspace with its commit refused = %d %s, want 500", status, body)
	}
	if after := rowsOf(); !slices.Equal(after, before) {
		t.Errorf("the rows after the refused deletion:\n%q\nwant\n%q", after, before)
	}
}

// Each check of deletionViolations fails on its counterexample.
func TestDeletionViolationsCatchesEachGap(t *testing.T) {
	// with is a complete cascade over the workspace row and one table, the
	// table's rows changed by change.
	with := func(change func(r *rowsUnder)) []rowsUnder {
		r := rowsUnder{key: "widgets.workspace_id", deletedBefore: 2, keptBefore: "(a)", keptAfter: "(a)"}
		change(&r)
		return []rowsUnder{{key: "workspaces.id", deletedBefore: 1, keptBefore: "(w)", keptAfter: "(w)"}, r}
	}
	exempt := map[string]string{"widgets.workspace_id": "kept for the audit"}
	tests := []struct {
		name   string
		under  []rowsUnder
		exempt map[string]string
		want   []string
	}{
		{"a complete cascade", with(func(*rowsUnder) {}), nil, nil},
		{"an exempt table whose rows are kept", with(func(r *rowsUnder) { r.deletedAfter = 2 }), exempt, nil},
		{"a table without a row under the deleted workspace", with(func(r *rowsUnder) { r.deletedBefore = 0 }), nil,
			[]string{"widgets.workspace_id: seed a row under the deleted workspace"}},
		{"a table without a row under the kept workspace", with(func(r *rowsUnder) { r.keptBefore, r.keptAfter = "", "" }), nil,
			[]string{"widgets.workspace_id: seed a row under the kept workspace"}},
		{"a table the cascade misses", with(func(r *rowsUnder) { r.deletedAfter = 2 }), nil,
			[]string{"widgets.workspace_id: 2 rows left undeleted under the deleted workspace: the cascade misses the table"}},
		{"a table the cascade misses in part", with(func(r *rowsUnder) { r.deletedAfter = 1 }), nil,
			[]string{"widgets.workspace_id: 1 rows left undeleted under the deleted workspace: the cascade misses the table"}},
		{"a row of the kept workspace changed", with(func(r *rowsUnder) { r.keptAfter = "(b)" }), nil,
			[]string{"widgets.workspace_id: the kept workspace's rows changed:\n(b)\nwant\n(a)"}},
		{"an exempt table whose rows are deleted", with(func(r *rowsUnder) { r.deletedAfter = 1 }), exempt,
			[]string{"widgets.workspace_id: 1 of 2 rows deleted with the workspace, want them kept: kept for the audit"}},
		{"a stale exempt entry", with(func(*rowsUnder) {}), map[string]string{"profiles.last_workspace_id": "the landing"},
			[]string{"the exempt profiles.last_workspace_id is no foreign key to workspaces"}},
	}
	for _, tt := range tests {
		if got := deletionViolations("workspaces", tt.under, tt.exempt); !slices.Equal(got, tt.want) {
			t.Errorf("%s: %q, want %q", tt.name, got, tt.want)
		}
	}
}
