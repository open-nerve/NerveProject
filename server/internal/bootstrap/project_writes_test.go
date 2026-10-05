package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	workspacepg "github.com/open-nerve/NerveProject/server/internal/modules/workspace/adapter/postgres"
	workspaceapp "github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Each write on a project as bootstrap wires it stamps the rows it writes
// with the time of its request, from the clock project.New takes, and with
// its caller (M3 design 3.6): alice, acme's admin, writes on her project
// Web, one write after another; the statement of each reads the rows it
// wrote, by $1 Web's id and $2 alice's, and finds each one the write
// writes. Bob and carol, whom she adds, are acme's members; she makes bob
// an admin of Web, removes carol, creates a state, and leaves Web. Each row
// the write writes again is first made bob's, as last written by him, and
// checked so: a write that kept its row's writer would pass for alice's
// otherwise, she having made it.
func TestTheWritesOnAProjectStampTheirRequest(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID := accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	bobID := accountID(t, contract, base, registerAccount(t, contract, base, "bob@example.com").AccessToken)
	inWorkspaceOf(t, pool, web, bobID, aliceID, shared.RoleMember)
	carolID := accountID(t, contract, base, registerAccount(t, contract, base, "carol@example.com").AccessToken)
	inWorkspaceOf(t, pool, web, carolID, aliceID, shared.RoleMember)
	carol := carolID.String()
	// bobs makes bob, $3, the last writer of the one row of table that where
	// picks by $1 Web's id and $2 alice's, which alice wrote last.
	bobs := func(table, where string) string {
		return "UPDATE " + table + " SET updated_by_id = $3 WHERE " + where + " AND updated_by_id = $2"
	}
	for _, w := range []struct {
		name, method, path, body string
		of                       uuid.UUID // the account whose membership of Web the path names by its id, as %s
		status                   int
		// seed, when set, writes the rows the write writes again, by $1 Web's
		// id, $2 alice's and $3 bob's: one row, none of alice's writing.
		seed   string
		stamps string // the rows written: the time each took, and whether alice wrote it as the write does
		seeded int    // how many rows stamps reads before the write: none of them alice's
		rows   int    // how many rows stamps reads after it: each one the write writes
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1", 1, 1},
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1", 1, 1},
		// Archived again, it takes the new time.
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1", 1, 1},
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", uuid.UUID{}, http.StatusOK,
			bobs("projects", "id = $1"),
			"SELECT updated_at, updated_by_id = $2 AND archived_at IS NULL FROM projects WHERE id = $1", 1, 1},
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, uuid.UUID{}, http.StatusOK,
			bobs("project_user_properties", "project_id = $1 AND user_id = $2 AND deleted_at IS NULL"),
			"SELECT updated_at, updated_by_id = $2 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL", 1, 1},
		// Bob's membership and his display settings, made: two rows.
		{"addProjectMembers", http.MethodPost, "/api/v0/projects/" + web.String() + "/members",
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, uuid.UUID{}, http.StatusCreated, "",
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM project_members " +
				"WHERE project_id = $1 AND member_id <> $2 UNION ALL SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND " +
				"updated_at = created_at FROM project_user_properties WHERE project_id = $1 AND user_id <> $2", 0, 2},
		// Carol's ended membership, made and last written by bob an hour
		// before, without display settings, which no write leaves (a
		// removal keeps them; SQL makes it), restored, and her display
		// settings made: two rows.
		{"addProjectMembers, a membership restored", http.MethodPost, "/api/v0/projects/" + web.String() + "/members",
			`{"members":[{"member_id":"` + carol + `","role":15}]}`, uuid.UUID{}, http.StatusCreated,
			"INSERT INTO project_members (id, workspace_id, project_id, member_id, role, is_active, created_by_id, updated_by_id, created_at, " +
				"updated_at) SELECT '" + uuid.NewV7().String() + "', workspace_id, id, '" + carol + "', 15, false, $3, $3, " +
				"now() - interval '1 hour', now() - interval '1 hour' FROM projects WHERE id = $1 AND created_by_id = $2",
			"SELECT updated_at, updated_by_id = $2 AND created_at < updated_at FROM project_members WHERE project_id = $1 AND member_id = '" +
				carol + "' UNION ALL SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at " +
				"FROM project_user_properties WHERE project_id = $1 AND user_id = '" + carol + "'", 1, 2},
		// Bob's membership, a member's, made an admin's.
		{"updateProjectMember", http.MethodPatch, "/api/v0/project-members/%s", `{"role":20}`, bobID, http.StatusOK,
			bobs("project_members", "project_id = $1 AND member_id = '"+bobID.String()+"'"),
			"SELECT updated_at, updated_by_id = $2 AND role = 20 FROM project_members WHERE project_id = $1 AND member_id = '" + bobID.String() + "'",
			1, 1},
		// Carol's membership, ended.
		{"removeProjectMember", http.MethodDelete, "/api/v0/project-members/%s", "", carolID, http.StatusNoContent,
			bobs("project_members", "project_id = $1 AND member_id = '"+carol+"'"),
			"SELECT updated_at, updated_by_id = $2 AND NOT is_active FROM project_members WHERE project_id = $1 AND member_id = '" + carol + "'",
			1, 1},
		// QA, made.
		{"createState", http.MethodPost, "/api/v0/projects/" + web.String() + "/states", `{"name":"QA","color":"#0EA5E9","group":"completed"}`,
			uuid.UUID{}, http.StatusCreated, "",
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM states WHERE project_id = $1 AND name = 'QA'",
			0, 1},
		// Her own membership, ended: bob is Web's other admin.
		{"leaveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/leave", "", uuid.UUID{}, http.StatusNoContent,
			bobs("project_members", "project_id = $1 AND member_id = $2"),
			"SELECT updated_at, updated_by_id = $2 AND NOT is_active FROM project_members WHERE project_id = $1 AND member_id = $2", 1, 1},
	} {
		if w.seed != "" {
			if tag, err := pool.Exec(context.Background(), w.seed, web, aliceID, bobID); err != nil || tag.RowsAffected() != 1 {
				t.Fatalf("%s's seed: %v, %v; want one row written", w.name, tag, err)
			}
		}
		if seeded := stampsOf(t, pool, w.stamps, web, aliceID); len(seeded) != w.seeded || slices.ContainsFunc(seeded, func(s stamp) bool { return s.hers }) {
			t.Fatalf("%s's rows before it: %v; want %d, none of alice's writing", w.name, seeded, w.seeded)
		}
		path := w.path
		if w.of != (uuid.UUID{}) {
			path = fmt.Sprintf(path, projectMemberships(t, pool, w.of, web)[0])
		}
		before := time.Now().Truncate(time.Microsecond)
		status, body := call(t, contract, w.method, base+path, alice, w.body)
		after := time.Now()
		if status != w.status {
			t.Fatalf("%s = %d %s, want %d", w.name, status, body, w.status)
		}
		written := stampsOf(t, pool, w.stamps, web, aliceID)
		for _, s := range written {
			if s.at == nil || s.at.Before(before) || s.at.After(after) || !s.hers {
				t.Errorf("%s wrote a row %s; want within the request, %v to %v, by alice", w.name, s, before, after)
			}
		}
		if len(written) != w.rows {
			t.Errorf("%s: %d rows written; want %d", w.name, len(written), w.rows)
		}
	}
}

// stamp is a row a write writes, as TestTheWritesOnAProjectStampTheirRequest
// reads it: the time it took, nil for none (an archived_at before the
// archive), and whether alice wrote it as the write does.
type stamp struct {
	at   *time.Time
	hers bool
}

// String is s as a failure prints it.
func (s stamp) String() string {
	at := "no time"
	if s.at != nil {
		at = s.at.String()
	}
	return fmt.Sprintf("at %s, by alice %v", at, s.hers)
}

// stampsOf reads the stamps the statement sql selects, by $1 web and $2
// alice.
func stampsOf(t *testing.T, pool *pgxpool.Pool, sql string, web, alice uuid.UUID) []stamp {
	t.Helper()
	rows, err := pool.Query(context.Background(), sql, web, alice)
	if err != nil {
		t.Fatal(err)
	}
	stamps, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (stamp, error) {
		var s stamp
		return s, row.Scan(&s.at, &s.hers)
	})
	if err != nil {
		t.Fatal(err)
	}
	return stamps
}

// createdProject creates the project name with identifier in the workspace
// slug through the API, by the caller of token, and returns its id.
func createdProject(t *testing.T, contract *apitest.Contract, base, token, slug, name, identifier string) uuid.UUID {
	t.Helper()
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces/"+slug+"/projects", token,
		`{"name":"`+name+`","identifier":"`+identifier+`"}`)
	if status != http.StatusCreated {
		t.Fatalf("creating %s in %s = %d %s", name, slug, status, body)
	}
	var p struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, body, &p)
	return p.ID
}

// inWorkspaceOf makes user a member of the workspace of project with role,
// by the account by, through the workspace store: no story of P4b invites
// anyone. It returns the workspace's id.
func inWorkspaceOf(t *testing.T, pool *pgxpool.Pool, project, user, by uuid.UUID, role shared.Role) uuid.UUID {
	t.Helper()
	var workspace uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT workspace_id FROM projects WHERE id = $1", project).Scan(&workspace); err != nil {
		t.Fatal(err)
	}
	if err := workspacepg.New(pool).CreateMember(context.Background(), workspaceapp.MemberRow{ID: uuid.NewV7(), WorkspaceID: workspace,
		MemberID: user, Role: role, CreatedBy: by, Now: time.Now()}); err != nil {
		t.Fatal(err)
	}
	return workspace
}

// A project's lead and default assignee are active members of it who are
// not its guests (M3 design 3.19), on the wired app. Of alice's Web, carol,
// whom she adds as a guest, bob, whom she adds as a member and removes,
// and dave, acme's member and none of Web's, are each refused as lead and
// as default assignee: 422 naming the field not_allowed, and the project
// as it was. erin, its member, is taken as both.
func TestTheLeadAndTheDefaultAssigneeAreActiveMembersWhoAreNoGuests(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	aliceID := accountID(t, contract, base, alice)
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	path := base + "/api/v0/projects/" + web.String()
	ids := map[string]uuid.UUID{}
	for _, name := range []string{"bob", "carol", "dave", "erin"} {
		ids[name] = accountID(t, contract, base, registerAccount(t, contract, base, name+"@example.com").AccessToken)
		inWorkspaceOf(t, pool, web, ids[name], aliceID, shared.RoleMember)
	}
	if status, body := call(t, contract, http.MethodPost, path+"/members", alice, fmt.Sprintf(
		`{"members":[{"member_id":"%s","role":15},{"member_id":"%s","role":5},{"member_id":"%s","role":15}]}`, ids["bob"], ids["carol"],
		ids["erin"])); status != http.StatusCreated {
		t.Fatalf("adding bob, carol and erin = %d %s", status, body)
	}
	if status, body := call(t, contract, http.MethodDelete, base+"/api/v0/project-members/"+projectMemberships(t, pool, ids["bob"], web)[0].String(),
		alice, ""); status != http.StatusNoContent {
		t.Fatalf("alice's removing bob = %d %s", status, body)
	}
	stored := func() string {
		t.Helper()
		var s string
		if err := pool.QueryRow(context.Background(), `SELECT coalesce(project_lead_id::text, 'none') || ' ' ||
			coalesce(default_assignee_id::text, 'none') || ' ' || updated_at FROM projects WHERE id = $1`, web).Scan(&s); err != nil {
			t.Fatal(err)
		}
		return s
	}
	before := stored()
	for _, name := range []string{"carol", "bob", "dave"} {
		for _, field := range []string{"project_lead_id", "default_assignee_id"} {
			status, body := call(t, contract, http.MethodPatch, path, alice, fmt.Sprintf(`{"%s":"%s"}`, field, ids[name]))
			if want := `"errors":[{"field":"` + field + `","code":"not_allowed"`; status != http.StatusUnprocessableEntity ||
				!strings.Contains(body, want) {
				t.Errorf("%s as %s = %d %s, want 422 with %s", name, field, status, body, want)
			}
		}
	}
	if got := stored(); got != before {
		t.Errorf("Web after the refusals: %s, want %s", got, before)
	}
	erin := ids["erin"].String()
	if status, body := call(t, contract, http.MethodPatch, path, alice,
		`{"project_lead_id":"`+erin+`","default_assignee_id":"`+erin+`"}`); status != http.StatusOK || !strings.HasPrefix(stored(), erin+" "+erin+" ") {
		t.Errorf("erin as both = %d %s, stored %s; want 200 and erin both", status, body, stored())
	}
}

// An add that names an account twice is refused whole, on the wired app
// (M3 design 5.2): alice adds bob, acme's member, to her Web as a member
// and as an admin, either of which his workspace role allows. The answer
// is 422 naming his second place duplicate, alone, and bob has neither a
// membership of Web nor display settings in it.
func TestAnAddThatNamesAnAccountTwiceWritesNothing(t *testing.T) {
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	base := startApp(t, testConfig(t, dbURL, false), migrations.FS())
	pool := openPool(t, dbURL)
	alice := registerAccount(t, contract, base, "alice@example.com").AccessToken
	if status, body := call(t, contract, http.MethodPost, base+"/api/v0/workspaces", alice, `{"name":"Acme","slug":"acme"}`); status != http.StatusCreated {
		t.Fatalf("creating acme = %d %s", status, body)
	}
	web := createdProject(t, contract, base, alice, "acme", "Web", "WEB")
	bobID := accountID(t, contract, base, registerAccount(t, contract, base, "bob@example.com").AccessToken)
	inWorkspaceOf(t, pool, web, bobID, accountID(t, contract, base, alice), shared.RoleMember)
	status, body := call(t, contract, http.MethodPost, base+"/api/v0/projects/"+web.String()+"/members", alice, fmt.Sprintf(
		`{"members":[{"member_id":"%s","role":15},{"member_id":"%s","role":20}]}`, bobID, bobID))
	if want := `{"status":422,"code":"validation_failed","title":"Unprocessable Entity","detail":"The request has invalid values.",` +
		`"errors":[{"field":"members[1].member_id","code":"duplicate","message":"is listed before"}]}`; status != http.StatusUnprocessableEntity ||
		strings.TrimSpace(body) != want {
		t.Errorf("adding bob twice = %d %s, want 422 %s", status, body, want)
	}
	var rows int
	if err := pool.QueryRow(context.Background(), `SELECT (SELECT count(*) FROM project_members WHERE project_id = $1 AND member_id = $2) +
		(SELECT count(*) FROM project_user_properties WHERE project_id = $1 AND user_id = $2)`, web, bobID).Scan(&rows); err != nil || rows != 0 {
		t.Errorf("bob's rows in Web after the refusal: %d, %v; want none", rows, err)
	}
}
