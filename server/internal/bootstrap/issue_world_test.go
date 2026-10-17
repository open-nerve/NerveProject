package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// issueWorld is the wired app on a database of its own, as the tests of
// the work-item side's writes find it: acme, whose admin alice created
// its public project Web, with Web's state Todo and its label Bug; bob,
// a member of acme and of Web; and three accounts who may not write on
// Web's work items: walt, a member of acme who is none of Web, and wanda,
// an admin of acme who is none of Web either, see Web (403); olga, no
// member of acme, does not (404).
type issueWorld struct {
	contract             *apitest.Contract
	base                 string
	pool                 *pgxpool.Pool
	tokens               map[string]string    // each account's, by its name
	ids                  map[string]uuid.UUID // each account's, by its name
	acme, web, todo, bug uuid.UUID
}

// newIssueWorld makes the world: each account registered through the API,
// acme, Web, bob's membership of Web and Bug through the API too, and the
// memberships of acme through the workspace store (inWorkspaceOf).
func newIssueWorld(t *testing.T) issueWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	w := issueWorld{contract: contract, base: startApp(t, testConfig(t, dbURL, false), migrations.FS()), pool: openPool(t, dbURL),
		tokens: map[string]string{}, ids: map[string]uuid.UUID{}}
	for _, name := range []string{"alice", "bob", "walt", "wanda", "olga"} {
		w.tokens[name] = registerAccount(t, contract, w.base, name+"@example.com").AccessToken
		w.ids[name] = accountID(t, contract, w.base, w.tokens[name])
	}
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Acme","slug":"acme"}`, http.StatusCreated), &created)
	w.acme = created.ID
	w.web = createdProject(t, contract, w.base, w.tokens["alice"], "acme", "Web", "WEB")
	for name, role := range map[string]shared.Role{"bob": shared.RoleMember, "walt": shared.RoleMember, "wanda": shared.RoleAdmin} {
		inWorkspaceOf(t, w.pool, w.web, w.ids[name], w.ids["alice"], role)
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+w.ids["bob"].String()+`","role":15}]}`, http.StatusCreated)
	w.todo = stateID(t, w.pool, w.web, "Todo")
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"Bug"}`, http.StatusCreated),
		&created)
	w.bug = created.ID
	return w
}

// send sends method path with body as the account name, checks the answer
// against the contract and that it is want, and returns its body.
func (w issueWorld) send(t *testing.T, name, method, path, body string, want int) string {
	t.Helper()
	status, answer := call(t, w.contract, method, w.base+path, w.tokens[name], body)
	if status != want {
		t.Fatalf("%s %s as %s = %d %s, want %d", method, path, name, status, answer, want)
	}
	return answer
}

// fullIssue is the body of a work item in Todo, assigned to bob and
// labelled Bug.
func (w issueWorld) fullIssue() string {
	return `{"name":"Fix","state_id":"` + w.todo.String() + `","assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` +
		w.bug.String() + `"]}`
}
