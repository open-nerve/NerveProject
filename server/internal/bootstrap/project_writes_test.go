package bootstrap

import (
	"context"
	"fmt"
	"net/http"
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

// Each write on a project as bootstrap wires it stamps the rows it writes
// with the time of its request, from the clock project.New takes, and with
// its caller (M3 design 3.6): alice, acme's admin, writes on her project
// Web, one write after another; the statement of each reads the rows it
// wrote, by $1 Web's id and $2 alice's. Bob, whom she adds, is acme's
// member.
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
	for _, w := range []struct {
		name, method, path, body string
		status                   int
		stamps                   string // the rows written: the time each took, and whether alice wrote it as the write does
	}{
		{"updateProject", http.MethodPatch, "/api/v0/projects/" + web.String(), `{"name":"Site"}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM projects WHERE id = $1"},
		{"archiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK,
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1"},
		// Archived again, it takes the new time.
		{"archiveProject again", http.MethodPost, "/api/v0/projects/" + web.String() + "/archive", "", http.StatusOK,
			"SELECT archived_at, updated_by_id = $2 AND updated_at = archived_at FROM projects WHERE id = $1"},
		{"unarchiveProject", http.MethodPost, "/api/v0/projects/" + web.String() + "/unarchive", "", http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 AND archived_at IS NULL FROM projects WHERE id = $1"},
		{"updateProjectPreferences", http.MethodPatch, "/api/v0/me/projects/" + web.String() + "/preferences", `{"sort_order":5}`, http.StatusOK,
			"SELECT updated_at, updated_by_id = $2 FROM project_user_properties WHERE project_id = $1 AND user_id = $2 AND deleted_at IS NULL"},
		// Bob's membership and his display settings, made.
		{"addProjectMembers", http.MethodPost, "/api/v0/projects/" + web.String() + "/members",
			`{"members":[{"member_id":"` + bobID.String() + `","role":15}]}`, http.StatusCreated,
			"SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND updated_at = created_at FROM project_members " +
				"WHERE project_id = $1 AND member_id <> $2 UNION ALL SELECT created_at, created_by_id = $2 AND updated_by_id = $2 AND " +
				"updated_at = created_at FROM project_user_properties WHERE project_id = $1 AND user_id <> $2"},
	} {
		before := time.Now().Truncate(time.Microsecond)
		status, body := call(t, contract, w.method, base+w.path, alice, w.body)
		after := time.Now()
		if status != w.status {
			t.Fatalf("%s = %d %s, want %d", w.name, status, body, w.status)
		}
		rows, err := pool.Query(context.Background(), w.stamps, web, aliceID)
		if err != nil {
			t.Fatal(err)
		}
		n := 0
		for rows.Next() {
			var at time.Time
			var hers bool
			if err := rows.Scan(&at, &hers); err != nil {
				t.Fatal(err)
			}
			if n++; at.Before(before) || at.After(after) || !hers {
				t.Errorf("%s wrote a row at %v, by alice %v; want within the request, %v to %v, by alice", w.name, at, hers, before, after)
			}
		}
		if err := rows.Err(); err != nil || n == 0 {
			t.Errorf("%s: %d rows written, %v; want one at least", w.name, n, err)
		}
	}
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
// whom she adds as a guest, bob, whose membership ended (P5's removal ends
// one; SQL stands in), and dave, acme's member and none of Web's, are each
// refused as lead and as default assignee: 422 naming the field
// not_allowed, and the project as it was. erin, its member, is taken as
// both.
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
	if tag, err := pool.Exec(context.Background(), "UPDATE project_members SET is_active = false WHERE project_id = $1 AND member_id = $2",
		web, ids["bob"]); err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("ending bob's membership: %v, %v", tag, err)
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
