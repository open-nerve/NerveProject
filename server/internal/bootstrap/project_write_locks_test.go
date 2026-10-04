package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"
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

// projectWrite is a write on a project as
// TestEachWriteOnAProjectSharesItsWorkspaceFirst sends it, by its
// operationId: the request on the project, or on a membership of it, by
// alice unless by names another sender.
type projectWrite struct {
	// path: %s the project's id, or the membership's for a path of one
	// (/api/v0/project-members/); body: %s the target's id.
	op, method, path, body string
	want                   int
	// targets are the accounts, one a phase, whose membership of the
	// workspace the write locks: acme's members, whom it makes members of
	// the project, or whose role in it it changes.
	targets [2]string
	// by are the accounts, one a phase, that send the write: alice when
	// empty.
	by [2]string
	// member are the accounts, one a phase, whose membership of the project
	// the write changes: a path of a membership names it, and its row is
	// probed as the project's is.
	member [2]string
}

// param is the parameter w's path names: a membership's id for a path of
// one, else the project's.
func (w projectWrite) param() string {
	if strings.HasPrefix(w.path, "/api/v0/project-members/") {
		return "{project_member_id}"
	}
	return "{project_id}"
}

// projectWrites are the writes on a project, in the order they run on
// Web, then on Ops. deleteProject is last: it deletes Web, then Ops, and
// every other write needs its project there.
var projectWrites = []projectWrite{
	{op: "updateProject", method: http.MethodPatch, path: "/api/v0/projects/%s", body: `{"description":"Changed"}`, want: http.StatusOK},
	{op: "archiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/archive", want: http.StatusOK},
	{op: "unarchiveProject", method: http.MethodPost, path: "/api/v0/projects/%s/unarchive", want: http.StatusOK},
	{op: "updateProjectPreferences", method: http.MethodPatch, path: "/api/v0/me/projects/%s/preferences", body: `{"sort_order":1}`,
		want: http.StatusOK},
	{op: "addProjectMembers", method: http.MethodPost, path: "/api/v0/projects/%s/members", body: `{"members":[{"member_id":"%s","role":15}]}`,
		want: http.StatusCreated, targets: [2]string{"bob", "carol"}},
	{op: "joinProject", method: http.MethodPost, path: "/api/v0/projects/%s/join", want: http.StatusOK, targets: [2]string{"dave", "erin"},
		by: [2]string{"dave", "erin"}},
	// The members added before, bob in Web and carol in Ops, made guests.
	{op: "updateProjectMember", method: http.MethodPatch, path: "/api/v0/project-members/%s", body: `{"role":5}`, want: http.StatusOK,
		targets: [2]string{"bob", "carol"}, member: [2]string{"bob", "carol"}},
	// The joiners before, dave in Web and erin in Ops, removed.
	{op: "removeProjectMember", method: http.MethodDelete, path: "/api/v0/project-members/%s", want: http.StatusNoContent,
		member: [2]string{"dave", "erin"}},
	// bob and carol, guests now, leave.
	{op: "leaveProject", method: http.MethodPost, path: "/api/v0/projects/%s/leave", want: http.StatusNoContent, by: [2]string{"bob", "carol"},
		member: [2]string{"bob", "carol"}},
	// Last: it deletes the project every write before it needs.
	{op: "deleteProject", method: http.MethodDelete, path: "/api/v0/projects/%s", want: http.StatusNoContent},
}

// writesOnAProject are the writes on a project among ops, by operationId:
// every operation but GET whose path names a project by its id
// ({project_id}), or that has a row among rows with a column of the project
// level (a column of a project table, projectTables, that is no column of
// the workspace level). The second takes in a write on a project addressed
// by a row under it (P5b's /project-members/{project_member_id}, P7's
// /states/{state_id}) whose row names a project table's columns; a column
// set of a row's own counts only once it is listed in projectTables, so a
// new table of the project level goes there, one of the workspace level in
// workspaceLevelTables, and neither into matrixTables alone
// (TestEachMatrixTableIsOfOneLevel); not a workspace's write that the only
// admin's table asks (leaveWorkspace).
func writesOnAProject(ops []apitest.Operation, rows []matrixRow) []string {
	ofTheProjectLevel := func(c caller) bool {
		return !slices.Contains(workspaceColumns, c) && slices.ContainsFunc(projectTables, func(table []caller) bool { return slices.Contains(table, c) })
	}
	projectLevel := map[string]bool{}
	for _, r := range rows {
		if slices.ContainsFunc(r.columns, ofTheProjectLevel) {
			projectLevel[r.op] = true
		}
	}
	var writes []string
	for _, op := range ops {
		if op.Method != http.MethodGet && (slices.Contains(strings.Split(op.Path, "/"), "{project_id}") || projectLevel[op.ID]) {
			writes = append(writes, op.ID)
		}
	}
	return slices.Sorted(slices.Values(writes))
}

// Each shape of a write on a project is one, and nothing else is: a write
// whose path names the project, whatever its rows; one addressed by a row
// under the project whose rows name a project table's columns. Not one so
// addressed whose rows name a column set that projectTables does not list
// (a self-only one, until it is listed there), nor a read of a project,
// nor a write at the workspace level, also when the only admin's table
// asks it.
func TestWritesOnAProjectAreEachShape(t *testing.T) {
	ops := []apitest.Operation{
		{ID: "getProject", Method: http.MethodGet, Path: "/api/v0/projects/{project_id}"},
		{ID: "createProject", Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/projects"},
		{ID: "leaveProject", Method: http.MethodPost, Path: "/api/v0/projects/{project_id}/leave"},
		{ID: "updateProjectMember", Method: http.MethodPatch, Path: "/api/v0/project-members/{project_member_id}"},
		{ID: "leaveProjectSelf", Method: http.MethodPost, Path: "/api/v0/project-members/{project_member_id}/leave"},
		{ID: "leaveWorkspace", Method: http.MethodPost, Path: "/api/v0/workspaces/{slug}/leave"},
	}
	rows := []matrixRow{
		{op: "getProject", columns: projectColumns},
		{op: "createProject", write: true},
		{op: "updateProjectMember", write: true, columns: []caller{callerProjectAdmin, callerProjectMember}},
		{op: "leaveProjectSelf", write: true, columns: []caller{"a project's member, himself"}},
		{op: "leaveWorkspace", write: true, columns: soleAdminColumns},
	}
	if got, want := writesOnAProject(ops, rows), []string{"leaveProject", "updateProjectMember"}; !slices.Equal(got, want) {
		t.Errorf("writesOnAProject() = %q, want %q", got, want)
	}
}

// holding begins a transaction on pool that takes the one row lock sql
// states, with args, NOWAIT, and keeps it open until the test rolls it
// back, or ends. A lock another transaction left fails the test at once,
// and so does a pool the holders before it took (pgtest.Soon).
func holding(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) pgx.Tx {
	t.Helper()
	tx, err := pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = tx.Rollback(context.Background()) })
	if tag, err := tx.Exec(context.Background(), sql+" NOWAIT", args...); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("%s NOWAIT: %v, %v; want one row held", sql, tag, err)
	}
	return tx
}

// heldBy reports whether a transaction holds the one row the lock sql
// states, with args, NOWAIT: it answers lock_not_available (55P03) at once
// while one does. Its wait for a connection of the pool ends soon.
func heldBy(t *testing.T, pool *pgxpool.Pool, sql string, args ...any) bool {
	t.Helper()
	sql += " NOWAIT"
	tag, err := pool.Exec(pgtest.Soon(t), sql, args...)
	var pgErr *pgconn.PgError
	switch {
	case errors.As(err, &pgErr) && pgErr.Code == "55P03":
		return true
	case err != nil || tag.RowsAffected() != 1:
		t.Fatalf("%s: %v, %v; want one row", sql, tag, err)
	}
	return false
}

// lockOn is the strongest row lock other transactions hold on the one row
// of from, a table and its WHERE clause over args: FOR UPDATE when a FOR
// KEY SHARE NOWAIT of it fails (heldBy), FOR NO KEY UPDATE when a FOR
// SHARE NOWAIT does, FOR SHARE when a FOR NO KEY UPDATE NOWAIT does, FOR
// KEY SHARE when a FOR UPDATE NOWAIT does, and "no lock" when none fails.
// Each lock it takes ends with its statement.
func lockOn(t *testing.T, pool *pgxpool.Pool, from string, args ...any) string {
	t.Helper()
	modes := []string{"FOR KEY SHARE", "FOR SHARE", "FOR NO KEY UPDATE", "FOR UPDATE"}
	for i, mode := range modes {
		if heldBy(t, pool, "SELECT 1 FROM "+from+" "+mode, args...) {
			return modes[len(modes)-1-i]
		}
	}
	return "no lock"
}

// Every write on a project takes its workspace's row FOR SHARE first, in
// its transaction, before any other lock (M3 design 3.6 convention 2, the
// lock table), as bootstrap wires it: alice, acme's admin, writes on her
// projects Web and Ops, each write once on each; a write's targets are made
// members of them or their roles changed, and a write on a membership names
// it by its id, or by its project and its sender. Every write on a project
// of the contract has its row here: every operation but GET whose path
// names a project, or that a matrix row asks of a project-level column
// (writesOnAProject), is the list, which a write without a row here fails
// before any database.
//   - The workspace first: another transaction holds acme's row FOR NO KEY
//     UPDATE, as every cascade over its projects does (3.3). The write on
//     Web waits for that row, and meanwhile holds neither Web's row, nor
//     its target's membership of acme, nor the membership of Web it
//     changes: a FOR UPDATE NOWAIT of each succeeds.
//   - In its transaction, FOR SHARE, before its target and its project:
//     another transaction holds Ops's row FOR NO KEY UPDATE. The write on
//     Ops waits for it, and meanwhile holds acme's row at FOR SHARE, no
//     stronger, which another write on a project of acme shares (lockOn),
//     and its target's membership of acme (a FOR UPDATE NOWAIT fails,
//     55P03), but not the membership of Ops it changes, which comes after
//     the project.
//
// Once the other transaction ends, each write answers as it would alone.
// Each row sends its own operation's request, its method and path as the
// contract has them, so no write's row runs another write instead. Each
// write on each project is a subtest, which names it when it fails; the
// first that fails ends the test, as a write left waiting could hold what
// the next one's probes look at.
func TestEachWriteOnAProjectSharesItsWorkspaceFirst(t *testing.T) {
	contract := apitest.Load(t)
	ops := make([]string, len(projectWrites))
	for i, w := range projectWrites {
		ops[i] = w.op
		if !slices.ContainsFunc(contract.Operations(), func(o apitest.Operation) bool {
			return o.ID == w.op && o.Method == w.method && o.Path == fmt.Sprintf(w.path, w.param())
		}) {
			t.Fatalf("%s's row sends %s %s, not the contract's %s", w.op, w.method, w.path, w.op)
		}
	}
	if want := writesOnAProject(contract.Operations(), matrixRows()); !slices.Equal(slices.Sorted(slices.Values(ops)), want) {
		t.Fatalf("the writes here are %q; the writes on a project are %q: each has its row, in projectWrites", slices.Sorted(slices.Values(ops)),
			want)
	}
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	var acme struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &acme)
	projects := [2]uuid.UUID{createdProject(t, contract, base, alice, "acme", "Web", "WEB"),
		createdProject(t, contract, base, alice, "acme", "Ops", "OPS")}
	tokens, ids, aliceID := map[string]string{}, map[string]uuid.UUID{}, accountID(t, contract, base, alice)
	for _, w := range projectWrites {
		for _, name := range slices.Concat(w.targets[:], w.by[:], w.member[:]) {
			if _, registered := tokens[name]; name != "" && !registered {
				tokens[name] = registerAccount(t, contract, base, name+"@example.com").AccessToken
				ids[name] = accountID(t, contract, base, tokens[name])
				inWorkspaceOf(t, pool, projects[0], ids[name], aliceID, shared.RoleMember)
			}
		}
	}
	membership := "SELECT 1 FROM workspace_members WHERE workspace_id = $1 AND member_id = $2 AND deleted_at IS NULL FOR UPDATE"
	memberRow := "SELECT 1 FROM project_members WHERE id = $1 FOR UPDATE"
	for _, w := range projectWrites {
		for phase, project := range projects {
			on := []string{"Web", "Ops"}[phase]
			if !t.Run(w.op+" on "+on, func(t *testing.T) {
				target, member, token, body, named := w.targets[phase], w.member[phase], alice, w.body, project
				if strings.Contains(body, "%s") {
					body = fmt.Sprintf(body, ids[target])
				}
				if by := w.by[phase]; by != "" {
					token = tokens[by]
				}
				// The membership the write changes, its row probed, and named
				// by a path of one.
				var row uuid.UUID
				if member != "" {
					if err := pool.QueryRow(pgtest.Soon(t), "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND deleted_at IS NULL",
						project, ids[member]).Scan(&row); err != nil {
						t.Fatalf("%s's membership of %s: %v", member, on, err)
					}
					if w.param() == "{project_member_id}" {
						named = row
					}
				}
				req := newRequest(t, w.method, base+fmt.Sprintf(w.path, named), token, []byte(body))
				contract.CheckRequest(t, req)
				var other pgx.Tx
				if phase == 0 {
					other = holding(t, pool, "SELECT 1 FROM workspaces WHERE id = $1 FOR NO KEY UPDATE", acme.ID)
				} else {
					other = holding(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", project)
				}
				answered := sendInBackground(req)
				if phase == 0 {
					pgtest.WaitForLockWaitOn(t, pool, "workspaces", 10*time.Second)
					if heldBy(t, pool, "SELECT 1 FROM projects WHERE id = $1 FOR UPDATE", project) {
						t.Errorf("%s holds its project while it waits for its workspace", w.op)
					}
					if target != "" && heldBy(t, pool, membership, acme.ID, ids[target]) {
						t.Errorf("%s holds %s's membership of acme while it waits for its workspace", w.op, target)
					}
					if member != "" && heldBy(t, pool, memberRow, row) {
						t.Errorf("%s holds %s's membership of %s while it waits for its workspace", w.op, member, on)
					}
				} else {
					pgtest.WaitForLockWaitOn(t, pool, "projects", 10*time.Second)
					switch lock := lockOn(t, pool, "workspaces WHERE id = $1", acme.ID); lock {
					case "FOR SHARE":
					case "no lock", "FOR KEY SHARE":
						t.Errorf("%s does not hold its workspace FOR SHARE in its transaction while it waits for its project: %s", w.op, lock)
					default:
						t.Errorf("%s holds its workspace %s, stronger than FOR SHARE, while it waits for its project", w.op, lock)
					}
					if target != "" && !heldBy(t, pool, membership, acme.ID, ids[target]) {
						t.Errorf("%s does not hold %s's membership of acme while it waits for its project", w.op, target)
					}
					if member != "" && heldBy(t, pool, memberRow, row) {
						t.Errorf("%s holds %s's membership of %s before its project", w.op, member, on)
					}
				}
				if err := other.Rollback(context.Background()); err != nil {
					t.Fatal(err)
				}
				a := receiveWithin(t, answered, 10*time.Second, "the answer to "+w.op)
				if a.err != nil {
					t.Fatal(a.err)
				}
				contract.CheckResponse(t, req, a.res)
				if a.res.StatusCode != w.want {
					t.Errorf("%s on %s = %d %s, want %d", w.op, on, a.res.StatusCode, a.body, w.want)
				}
			}) {
				t.FailNow()
			}
		}
	}
}
