package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// issueWorld is the wired app on a database of its own, as the tests of
// the work-item side's writes find it: acme, whose admin alice created
// its public project Web, with Web's states Todo and Done and its label
// Bug, and no work item yet; bob, a member of acme and of Web; and three
// accounts who may not write on Web's work items: walt, a member of acme
// who is none of Web, and wanda, an admin of acme who is none of Web
// either, see Web (403); olga, no member of acme, does not (404).
type issueWorld struct {
	contract                   *apitest.Contract
	base                       string
	pool                       *pgxpool.Pool
	tokens                     map[string]string    // each account's, by its name
	ids                        map[string]uuid.UUID // each account's, by its name
	acme, web, todo, done, bug uuid.UUID
}

// newIssueWorld makes the world: each account registered through the API,
// acme, Web, bob's membership of Web and Bug through the API too, and the
// memberships of acme through the workspace store (inWorkspaceOf).
func newIssueWorld(t *testing.T) issueWorld {
	t.Helper()
	return newIssueWorldWith(t, func(*config.Config) {})
}

// newIssueWorldWith is newIssueWorld in an app whose configuration,
// testConfig's, configure changes first.
func newIssueWorldWith(t *testing.T, configure func(cfg *config.Config)) issueWorld {
	t.Helper()
	contract := apitest.Load(t)
	dbURL := pgtest.NewDatabase(t)
	cfg := testConfig(t, dbURL, false)
	configure(&cfg)
	w := issueWorld{contract: contract, base: startApp(t, cfg, migrations.FS()), pool: openPool(t, dbURL), tokens: map[string]string{},
		ids: map[string]uuid.UUID{}}
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
	w.todo, w.done = stateID(t, w.pool, w.web, "Todo"), stateID(t, w.pool, w.web, "Done")
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

// newWorkItem is the id of a work item alice creates in Web through the
// API: fullIssue's.
func (w issueWorld) newWorkItem(t *testing.T) uuid.UUID {
	t.Helper()
	return w.newWorkItemIn(t, w.web, w.fullIssue())
}

// newWorkItemIn is the id of a work item alice creates in project through
// the API, with body.
func (w issueWorld) newWorkItemIn(t *testing.T, project uuid.UUID, body string) uuid.UUID {
	t.Helper()
	var created struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/issues", body, http.StatusCreated), &created)
	return created.ID
}

// fullIssue is the body of a work item in Todo, assigned to bob and
// labelled Bug.
func (w issueWorld) fullIssue() string {
	return `{"name":"Fix","state_id":"` + w.todo.String() + `","assignee_ids":["` + w.ids["bob"].String() + `"],"label_ids":["` +
		w.bug.String() + `"]}`
}

// workItemStamps are the rows of the work item id in the work-item side's
// five tables, each a line, sorted: its table, who made it and changed it
// last, and when, when it was deleted if it was, and whose it is, for an
// assignee, a subscriber or an activity (the account assigned, subscribed
// or acting). An account goes by its name in the world, a moment by the
// work item's: made, its creation, or changed, its last change.
func (w issueWorld) workItemStamps(t *testing.T, id uuid.UUID) []string {
	t.Helper()
	var made, changed time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at, updated_at FROM issues WHERE id = $1", id).Scan(&made, &changed); err != nil {
		t.Fatal(err)
	}
	when := func(at time.Time) string {
		switch {
		case at.Equal(made):
			return "made"
		case at.Equal(changed):
			return "changed"
		}
		return at.String()
	}
	names := make(map[uuid.UUID]string, len(w.ids))
	for name, account := range w.ids {
		names[account] = name
	}
	who := func(account *uuid.UUID) string {
		if account == nil {
			return "no one"
		}
		if name, ok := names[*account]; ok {
			return name
		}
		return account.String()
	}
	rows, err := w.pool.Query(pgtest.Soon(t), `
		SELECT 'issues', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL::uuid FROM issues WHERE id = $1
		UNION ALL SELECT 'issue_assignees', created_by_id, created_at, updated_by_id, updated_at, deleted_at, assignee_id FROM issue_assignees
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_labels', created_by_id, created_at, updated_by_id, updated_at, deleted_at, NULL FROM issue_labels
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_subscribers', created_by_id, created_at, updated_by_id, updated_at, deleted_at, subscriber_id
			FROM issue_subscribers WHERE issue_id = $1
		UNION ALL SELECT 'issue_activities', created_by_id, created_at, updated_by_id, updated_at, deleted_at, actor_id FROM issue_activities
			WHERE issue_id = $1`, id)
	if err != nil {
		t.Fatal(err)
	}
	lines, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var table string
		var madeBy, changedBy, of *uuid.UUID
		var madeAt, changedAt time.Time
		var deletedAt *time.Time
		if err := row.Scan(&table, &madeBy, &madeAt, &changedBy, &changedAt, &deletedAt, &of); err != nil {
			return "", err
		}
		line := table + " made by " + who(madeBy) + " " + when(madeAt) + ", changed by " + who(changedBy) + " " + when(changedAt)
		if deletedAt != nil {
			line += ", deleted " + when(*deletedAt)
		}
		if of != nil {
			line += ", of " + who(of)
		}
		return line, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return slices.Sorted(slices.Values(lines))
}
