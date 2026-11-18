package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// intakeBreaks are the breaks of the intake's invariant (M7 P2a spec 2.3,
// DP-7) among every work item of project, deleted or not, each a line,
// sorted, and how many work items are in its intake: a work item has an
// undeleted intake row whose status is not accepted if and only if it is
// in the project's triage state; and none such, by either mark, has a
// parent, an undeleted sub-item, an undeleted relation, or a row of
// cycle_issues or module_issues.
func intakeBreaks(t *testing.T, pool *pgxpool.Pool, project uuid.UUID) (int, []string) {
	t.Helper()
	const items = `WITH items AS (
		SELECT i.id, i.name, i.parent_id, s."group" = 'triage' AS triage_state, coalesce(ii.status <> 'accepted', false) AS in_intake
		FROM issues i JOIN states s ON s.id = i.state_id LEFT JOIN intake_issues ii ON ii.issue_id = i.id AND ii.deleted_at IS NULL
		WHERE i.project_id = $1)`
	var in int
	if err := pool.QueryRow(pgtest.Soon(t), items+` SELECT count(*) FROM items WHERE in_intake`, project).Scan(&in); err != nil {
		t.Fatal(err)
	}
	rows, err := pool.Query(pgtest.Soon(t), items+`
		SELECT format('%s: in the intake %s, in the triage state %s', name, in_intake, triage_state) FROM items WHERE in_intake <> triage_state
		UNION ALL
		SELECT format('%s, in the intake, %s', i.name, f.what) FROM items i CROSS JOIN LATERAL (VALUES
			('has a parent', i.parent_id IS NOT NULL),
			('has a sub-item', EXISTS (SELECT 1 FROM issues c WHERE c.parent_id = i.id AND c.deleted_at IS NULL)),
			('has a relation', EXISTS (SELECT 1 FROM issue_relations r WHERE i.id IN (r.issue_id, r.related_issue_id) AND r.deleted_at IS NULL)),
			('is in a cycle', EXISTS (SELECT 1 FROM cycle_issues c WHERE c.issue_id = i.id)),
			('is in a module', EXISTS (SELECT 1 FROM module_issues m WHERE m.issue_id = i.id))) AS f(what, broken)
		WHERE (i.in_intake OR i.triage_state) AND f.broken
		ORDER BY 1`, project)
	if err != nil {
		t.Fatal(err)
	}
	breaks, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return in, breaks
}

// The intake's invariant holds through the app (M7 P2a spec 2.3, DP-7):
// after each step, Web's work items in its intake are those of the step,
// and no work item breaks it (intakeBreaks). bob submits seven work items;
// alice snoozes one, snoozes another and ends its snooze, declines one,
// marks one a duplicate of Ordinary, accepts two, and deletes one of those.
// On each still in the intake, each write a work item in the intake
// refuses is refused 409 issue.in_triage, a change of its parent, its
// cycle or its modules too; each write that would make it Ordinary's
// parent, sub-item or relation, 422. The accepted one goes into Sprint
// and the module Login, becomes Ordinary's parent and is related to it.
func TestTheIntakeRowAndTheTriageStateGoTogether(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	now := time.Now()
	sprint := w.newCycle(t, "alice", datesBody("Sprint", shanghaiDay(t, now, -1), shanghaiDay(t, now, 7)))
	login := w.newModule(t, `{"name":"Login"}`)
	ordinary := w.newWorkItemIn(t, w.web, `{"name":"Ordinary"}`)
	w.intakeOn(t)
	holds := func(step string, want int) {
		t.Helper()
		if in, breaks := intakeBreaks(t, w.pool, w.web); in != want || len(breaks) > 0 {
			t.Errorf("after %s: %d work items in the intake, breaks %q; want %d, none", step, in, breaks, want)
		}
	}
	holds("the start", 0)
	items := map[string]uuid.UUID{}
	for _, name := range []string{"Pending", "Snoozed", "Woken", "Declined", "Duplicate", "Accepted", "Gone"} {
		items[name] = w.submit(t, "bob", `{"name":"`+name+`"}`).ID
	}
	holds("the submissions", 7)
	till := now.Add(time.Hour)
	for _, step := range []struct {
		name, body string
		in         int
	}{
		{"Snoozed", snoozedTill(till), 7}, {"Woken", snoozedTill(till), 7}, {"Woken", `{"status":"pending"}`, 7},
		{"Declined", `{"status":"declined"}`, 7}, {"Duplicate", duplicateOf(ordinary), 7},
		{"Accepted", `{"status":"accepted"}`, 6}, {"Gone", `{"status":"accepted"}`, 5},
	} {
		w.triage(t, "alice", items[step.name], step.body, http.StatusOK)
		holds("the triage of "+step.name+" "+step.body, step.in)
	}
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+items["Gone"].String(), "", http.StatusNoContent)
	holds("the deletion of an accepted work item", 5)

	under, relating := fmt.Sprintf(`{"sub_issue_ids":[%q]}`, ordinary), fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, ordinary)
	for _, name := range []string{"Pending", "Snoozed", "Woken", "Declined", "Duplicate"} {
		path := "/api/v0/issues/" + items[name].String()
		for _, write := range []struct{ method, path, body string }{
			{http.MethodPatch, path, `{"name":"Mended"}`}, {http.MethodPatch, path, fmt.Sprintf(`{"parent_id":%q}`, ordinary)},
			{http.MethodPatch, path, fmt.Sprintf(`{"cycle_id":%q}`, sprint)}, {http.MethodPatch, path, modulesBody(login)},
			{http.MethodPost, path + "/archive", ""}, {http.MethodPost, path + "/unarchive", ""}, {http.MethodDelete, path, ""},
			{http.MethodPost, path + "/sub-issues", under}, {http.MethodPost, path + "/relations", relating},
		} {
			w.refusesAs(t, "alice", write.method, write.path, write.body, http.StatusConflict, "issue.in_triage")
		}
	}
	holds("the writes a work item in the intake refuses", 5)
	other := "/api/v0/issues/" + ordinary.String()
	for _, name := range []string{"Pending", "Declined"} {
		item := items[name]
		for _, write := range []struct{ method, path, body string }{
			{http.MethodPost, "/api/v0/projects/" + w.web.String() + "/issues", fmt.Sprintf(`{"name":"Under","parent_id":%q}`, item)},
			{http.MethodPatch, other, fmt.Sprintf(`{"parent_id":%q}`, item)},
			{http.MethodPost, other + "/sub-issues", fmt.Sprintf(`{"sub_issue_ids":[%q]}`, item)},
			{http.MethodPost, other + "/relations", fmt.Sprintf(`{"relation_type":"blocking","issue_ids":[%q]}`, item)},
		} {
			w.send(t, "alice", write.method, write.path, write.body, http.StatusUnprocessableEntity)
		}
	}
	holds("the writes that would make one another's parent, sub-item or relation", 5)
	accepted := "/api/v0/issues/" + items["Accepted"].String()
	w.send(t, "alice", http.MethodPatch, accepted, fmt.Sprintf(`{"cycle_id":%q}`, sprint), http.StatusOK)
	w.send(t, "alice", http.MethodPatch, accepted, modulesBody(login), http.StatusOK)
	w.send(t, "alice", http.MethodPatch, other, fmt.Sprintf(`{"parent_id":%q}`, items["Accepted"]), http.StatusOK)
	w.send(t, "alice", http.MethodPost, accepted+"/relations", relating, http.StatusCreated)
	holds("the accepted work item's writes", 5)
}
