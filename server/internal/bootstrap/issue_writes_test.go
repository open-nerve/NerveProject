package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// issueStamp is a row createIssue writes, as TestCreatingAWorkItemStampsItsRows
// reads it: its table, when it was made and last changed, and whether
// alice made it and changed it last, and acted, for an activity.
type issueStamp struct {
	table            string
	created, updated time.Time
	hers             bool
}

// createIssue as bootstrap wires it makes every row it writes at the time
// of its request, one moment, from the clock issue.New takes, and as its
// caller (M4 design 3.7): alice creates a work item in Web of the world,
// in Todo, assigned to bob and labelled Bug. The work item, its assignee,
// its label, its two subscribers, alice and bob, and its two activities
// are seven rows, each made and last changed by alice at one moment within
// the request. The second activity names bob by his display name as
// identity has it (bootstrap/ports.go's issueProfiles).
func TestCreatingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	before := time.Now().Truncate(time.Microsecond)
	item := w.newWorkItem(t)
	after := time.Now()

	rows, err := w.pool.Query(pgtest.Soon(t), `
		SELECT 'issues', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issues WHERE id = $1
		UNION ALL SELECT 'issue_assignees', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_assignees
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_labels', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_labels
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_subscribers', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 FROM issue_subscribers
			WHERE issue_id = $1
		UNION ALL SELECT 'issue_activities', created_at, updated_at, created_by_id = $2 AND updated_by_id = $2 AND actor_id = $2
			FROM issue_activities WHERE issue_id = $1`, item, w.ids["alice"])
	if err != nil {
		t.Fatal(err)
	}
	stamps, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (issueStamp, error) {
		var s issueStamp
		return s, row.Scan(&s.table, &s.created, &s.updated, &s.hers)
	})
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for _, s := range stamps {
		tables = append(tables, s.table)
		if !s.created.Equal(stamps[0].created) || !s.updated.Equal(s.created) || s.created.Before(before) || s.created.After(after) || !s.hers {
			t.Errorf("a row of %s made at %v, changed at %v, by alice %v; want made and changed by alice at one moment within %v to %v",
				s.table, s.created, s.updated, s.hers, before, after)
		}
	}
	if want := []string{"issue_activities", "issue_activities", "issue_assignees", "issue_labels", "issue_subscribers", "issue_subscribers",
		"issues"}; !slices.Equal(slices.Sorted(slices.Values(tables)), want) {
		t.Errorf("createIssue wrote rows of %q, want %q", tables, want)
	}

	var named, displayName string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT a.new_value, u.display_name FROM issue_activities a, users u
		WHERE a.issue_id = $1 AND a.field = 'assignees' AND a.new_identifier = $2 AND u.id = $2`, item, w.ids["bob"]).Scan(&named,
		&displayName); err != nil {
		t.Fatal(err)
	}
	if named != displayName {
		t.Errorf("the activity assigning bob names him %q; want his display name %q", named, displayName)
	}
}

// newIssueRoute is the issue module (issue.New) on pool, with tx its
// transactions, the Authorizer on pool and project's and identity's ports
// converted, all as bootstrap wires them.
func newIssueRoute(t *testing.T, pool *pgxpool.Pool, tx shared.TxManager) moduleRoute {
	t.Helper()
	return newRoute(t, issue.New(issue.Deps{Pool: pool, Tx: tx, Clock: clock.System{}, Authorizer: authorizerOn(pool),
		Projects: issueProjects{projects: project.Provide(pool).IssueProjects}, Profiles: issueProfiles{profiles: identity.Provide(pool).PublicProfiles}}).Register)
}

// Each write on the work-item side runs every statement of its transaction
// on that transaction's connection (M4 design 3.5, 6.7): its lock, the
// work item read again under it, the decision under it, its checks of the
// states, the members and the labels, its number, its writes, the display
// names, and its answer. The issue module and the Authorizer are wired as
// bootstrap wires them, on a pool of one connection (poolOfOne): a
// statement sent through the pool rather than the transaction would wait
// for a second connection that never comes, and its request fail at the
// request's deadline. On Web of the world, alice creates a work item in
// Todo, assigned to bob and labelled Bug; with bob made Web's default
// assignee, one without assignees, which he takes; one labelled with a
// label no project has, which the check under the lock refuses (422); walt,
// who sees Web, is refused before the lock (403). She changes the first:
// into Done, assigned to herself instead of bob, without Bug; then its name
// as it is, which writes nothing; then with a label no project has (422),
// and walt is refused (403); and she reads it, without a transaction. With
// Web archived, her creation and her change are refused under the lock
// (409).
func TestTheWritesOnAWorkItemRunOnTheirTransactionsConnection(t *testing.T) {
	w := newIssueWorld(t)
	one := poolOfOne(t, w.pool.Config().ConnString())
	route := newIssueRoute(t, one, postgres.NewTxManager(one, 2*time.Second))
	path := "/api/v0/projects/" + w.web.String() + "/issues"
	alice := w.ids["alice"]
	send := func(caller uuid.UUID, body string, want int) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodPost, path, caller, body, want)
	}

	var first struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, send(alice, w.fullIssue(), http.StatusCreated), &first)
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET default_assignee_id = $2 WHERE id = $1", w.web, w.ids["bob"]); err != nil {
		t.Fatal(err)
	}
	var defaulted struct {
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	}
	if decodeAnswer(t, send(alice, `{"name":"Defaulted"}`, http.StatusCreated), &defaulted); !slices.Equal(defaulted.AssigneeIDs,
		[]uuid.UUID{w.ids["bob"]}) {
		t.Errorf("a work item without assignees is assigned to %v; want bob, Web's default assignee", defaulted.AssigneeIDs)
	}
	send(alice, `{"name":"Fix","label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	send(w.ids["walt"], `{"name":"Fix"}`, http.StatusForbidden)
	change := func(caller uuid.UUID, body string, want int) string {
		t.Helper()
		return route.answerWithin(t, w.contract, http.MethodPatch, "/api/v0/issues/"+first.ID.String(), caller, body, want)
	}
	var changed struct {
		AssigneeIDs []uuid.UUID `json:"assignee_ids"`
		LabelIDs    []uuid.UUID `json:"label_ids"`
	}
	if decodeAnswer(t, change(alice, `{"state_id":"`+w.done.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":[]}`,
		http.StatusOK), &changed); !slices.Equal(changed.AssigneeIDs, []uuid.UUID{alice}) || len(changed.LabelIDs) != 0 {
		t.Errorf("the work item changed is assigned to %v, labelled %v; want alice alone, no label", changed.AssigneeIDs, changed.LabelIDs)
	}
	change(alice, `{"name":"Fix"}`, http.StatusOK)
	change(alice, `{"label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	change(w.ids["walt"], `{"name":"Mend"}`, http.StatusForbidden)
	route.answerWithin(t, w.contract, http.MethodGet, "/api/v0/issues/"+first.ID.String(), alice, "", http.StatusOK)
	if _, err := w.pool.Exec(context.Background(), "UPDATE projects SET archived_at = now() WHERE id = $1", w.web); err != nil {
		t.Fatal(err)
	}
	send(alice, `{"name":"Fix"}`, http.StatusConflict)
	change(alice, `{"name":"Mend"}`, http.StatusConflict)
}
