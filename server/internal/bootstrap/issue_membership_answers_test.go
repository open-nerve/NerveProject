package bootstrap

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// issueMemberships is a work item's cycle_id and module_ids as each answer
// must have them.
type issueMemberships struct {
	cycle   *uuid.UUID
	modules []uuid.UUID
}

// seedPlanningTarget writes a cycle or a module (table) of Web named name,
// directly, and returns its id: planning owns the table, the test writes
// it as a fixture.
func (w issueWorld) seedPlanningTarget(t *testing.T, table, name string) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	listSeedSQL(t, w, `INSERT INTO `+table+` (id, workspace_id, project_id, name) VALUES ($1, $2, $3, $4)`, id, w.acme, w.web, name)
	return id
}

// seedMembership writes the row of table that puts the work item issue in
// the cycle or module target (column its id's), directly, deleted when gone
// is true: the membership writes come with the work item's fields (M6 P3a
// spec 2.5), so the answers' test writes them as a fixture.
func (w issueWorld) seedMembership(t *testing.T, table, column string, issue, target uuid.UUID, gone bool) {
	t.Helper()
	listSeedSQL(t, w, `INSERT INTO `+table+` (id, workspace_id, project_id, `+column+`, issue_id, deleted_at)
		VALUES ($1, $2, $3, $4, $5, CASE WHEN $6 THEN now() END)`, uuid.NewV7(), w.acme, w.web, target, issue, gone)
}

// Every answer that has work items has each one's cycle_id and module_ids
// (M6 P3a spec 2.3), the same in each: the detail of a write or a read, by
// id or by identifier, the relations and sub-items written and read, the
// flat, grouped and subgrouped lists, the member's list and the search. a
// left Sprint for Old, an archived cycle, and left M3; it is in M2 and M1,
// archived, stored in that order, and answers them in id order. b is in
// Sprint and M2; c left M1 and is in none, as a work item just created or
// published; walt made them all. e, alice's, is in Sprint and M3. Once walt
// is a guest who sees only his own work items, he reads the same values of
// his own, and e is not found.
func TestIssueMembershipsAcrossRepresentations(t *testing.T) {
	w := newIssueWorld(t)
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	project := "/api/v0/projects/" + w.web.String() + "/issues"
	created := w.send(t, "walt", http.MethodPost, project, `{"name":"Membership A"}`, http.StatusCreated)
	var a struct {
		ID         uuid.UUID `json:"id"`
		SequenceID int       `json:"sequence_id"`
	}
	decodeAnswer(t, created, &a)
	checkIssueMemberships(t, created, map[uuid.UUID]issueMemberships{a.ID: {}})
	b := w.newWorkItemBy(t, "walt", w.web, `{"name":"Membership B"}`)
	c := w.newWorkItemBy(t, "walt", w.web, `{"name":"Membership C"}`)
	e := w.newWorkItemIn(t, w.web, `{"name":"Membership E"}`)
	d := createDraftHTTP(t, w, "walt", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Membership D"}}`, w.web))
	published := w.send(t, "walt", http.MethodPost, publishPath(d.ID), "", http.StatusCreated)
	var p struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, published, &p)
	checkIssueMemberships(t, published, map[uuid.UUID]issueMemberships{p.ID: {}})

	sprint, old := w.seedPlanningTarget(t, "cycles", "Sprint"), w.seedPlanningTarget(t, "cycles", "Old")
	modules := []uuid.UUID{w.seedPlanningTarget(t, "modules", "M1"), w.seedPlanningTarget(t, "modules", "M2"), w.seedPlanningTarget(t, "modules", "M3")}
	slices.SortFunc(modules, uuid.UUID.Compare)
	m1, m2, m3 := modules[0], modules[1], modules[2]
	listSeedSQL(t, w, "UPDATE cycles SET archived_at = now() WHERE id = $1", old)
	listSeedSQL(t, w, "UPDATE modules SET archived_at = now() WHERE id = $1", m1)
	for _, row := range []struct {
		table, column string
		issue, target uuid.UUID
		gone          bool
	}{
		{"cycle_issues", "cycle_id", a.ID, sprint, true}, {"cycle_issues", "cycle_id", a.ID, old, false},
		{"cycle_issues", "cycle_id", b, sprint, false},
		{"module_issues", "module_id", a.ID, m3, true}, {"module_issues", "module_id", a.ID, m2, false},
		{"module_issues", "module_id", a.ID, m1, false}, {"module_issues", "module_id", b, m2, false},
		{"module_issues", "module_id", c, m1, true},
		{"cycle_issues", "cycle_id", e, sprint, false}, {"module_issues", "module_id", e, m3, false},
	} {
		w.seedMembership(t, row.table, row.column, row.issue, row.target, row.gone)
	}
	want := map[uuid.UUID]issueMemberships{a.ID: {cycle: &old, modules: []uuid.UUID{m1, m2}}, b: {cycle: &sprint, modules: []uuid.UUID{m2}},
		c: {}, p.ID: {}, e: {cycle: &sprint, modules: []uuid.UUID{m3}}}
	only := func(ids ...uuid.UUID) map[uuid.UUID]issueMemberships {
		out := map[uuid.UUID]issueMemberships{}
		for _, id := range ids {
			out[id] = want[id]
		}
		return out
	}
	path := "/api/v0/issues/" + a.ID.String()
	byIdentifier := "/api/v0/workspaces/acme/issues/WEB-" + strconv.Itoa(a.SequenceID)
	for _, q := range []struct{ method, path, body string }{
		{http.MethodGet, path, ""}, {http.MethodGet, byIdentifier, ""}, {http.MethodPatch, path, `{}`},
		{http.MethodPatch, path, `{"name":"Membership A renamed"}`}, {http.MethodPatch, path, fmt.Sprintf(`{"state_id":%q}`, w.done)},
		{http.MethodPost, path + "/archive", ""}, {http.MethodPost, path + "/unarchive", ""},
	} {
		checkIssueMemberships(t, w.send(t, "walt", q.method, q.path, q.body, http.StatusOK), only(a.ID))
	}
	// Relation writes and nested relation reads read through GetIssues; the
	// sub-items' write too, their read through SubIssues.
	checkIssueMemberships(t, w.send(t, "alice", http.MethodPost, relationsOf(b), relating("relates_to", a.ID, c), http.StatusCreated), only(a.ID, c))
	checkIssueMemberships(t, w.send(t, "alice", http.MethodPost, subIssues(b), subIssueIDs(a.ID, c), http.StatusOK), only(a.ID, c))
	walts := only(a.ID, b, c, p.ID)
	// reads are walt's reads: seen in the project's lists and the search,
	// his own in his list.
	reads := func(t *testing.T, seen map[uuid.UUID]issueMemberships) {
		t.Helper()
		checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, path, "", http.StatusOK), only(a.ID))
		checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, byIdentifier, "", http.StatusOK), only(a.ID))
		checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, "/api/v0/issues/"+b.String(), "", http.StatusOK), only(b))
		checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, relationsOf(b), "", http.StatusOK), only(a.ID, c))
		checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, subIssues(b), "", http.StatusOK), only(a.ID, c))
		for _, query := range []string{"?sub_issue=true", "?sub_issue=true&group_by=state_group",
			"?sub_issue=true&group_by=state_group&sub_group_by=priority"} {
			checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, project+query, "", http.StatusOK), seen)
		}
		checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, memberIssuesPath(w, "walt")+"?relation=created&sub_issue=true", "", http.StatusOK), walts)
		for _, query := range []string{"", "?q=Membership"} {
			checkIssueMemberships(t, w.send(t, "walt", http.MethodGet, searchPath+query, "", http.StatusOK), seen)
		}
	}
	t.Run("member", func(t *testing.T) { reads(t, want) })
	// A guest sees only his own work items (OwnOnly), each with the same
	// memberships.
	listSeedSQL(t, w, "UPDATE project_members SET role = $3 WHERE project_id = $1 AND member_id = $2", w.web, w.ids["walt"], shared.RoleGuest)
	w.send(t, "walt", http.MethodGet, "/api/v0/issues/"+e.String(), "", http.StatusNotFound)
	t.Run("guest", func(t *testing.T) { reads(t, walts) })
}

// checkIssueMemberships checks each work item of an answer, at any depth
// (a relation's, a group's or a cell's too), has the required cycle_id and
// module_ids want has for it, and that the answer has each work item of
// want.
func checkIssueMemberships(t *testing.T, answer string, want map[uuid.UUID]issueMemberships) {
	t.Helper()
	var body any
	if err := json.NewDecoder(strings.NewReader(answer)).Decode(&body); err != nil {
		t.Fatal(err)
	}
	seen := map[uuid.UUID]bool{}
	var visit func(any)
	visit = func(value any) {
		switch v := value.(type) {
		case []any:
			for _, child := range v {
				visit(child)
			}
		case map[string]any:
			if _, isIssue := v["sequence_id"]; isIssue {
				id, err := uuid.Parse(fmt.Sprint(v["id"]))
				if err != nil {
					t.Fatal(err)
				}
				expected, ok := want[id]
				if !ok {
					t.Fatalf("unexpected work item %s in %s", id, answer)
				}
				if got, want := answeredMemberships(t, v), fmt.Sprint(cycleText(expected.cycle), " ", idsText(expected.modules)); got != want {
					t.Errorf("work item %s is in %s, want %s: %s", id, got, want, answer)
				}
				seen[id] = true
			}
			for _, child := range v {
				visit(child)
			}
		}
	}
	visit(body)
	if len(seen) != len(want) {
		t.Fatalf("work items %v, want %v in %s", seen, want, answer)
	}
}

// answeredMemberships is a work item's cycle_id and module_ids as an answer has
// them, failing the test when either is missing or of another type.
func answeredMemberships(t *testing.T, issue map[string]any) string {
	t.Helper()
	cycle, ok := issue["cycle_id"]
	if !ok {
		t.Fatalf("work item %v lacks cycle_id", issue["id"])
	}
	modules, ok := issue["module_ids"].([]any)
	if !ok {
		t.Fatalf("work item %v has module_ids %v, not an array", issue["id"], issue["module_ids"])
	}
	ids := make([]string, len(modules))
	for n, m := range modules {
		ids[n] = fmt.Sprint(m)
	}
	if cycle == nil {
		return "none [" + strings.Join(ids, " ") + "]"
	}
	return fmt.Sprint(cycle) + " [" + strings.Join(ids, " ") + "]"
}

// cycleText is the cycle as answeredMemberships shows it.
func cycleText(id *uuid.UUID) string {
	if id == nil {
		return "none"
	}
	return id.String()
}

// idsText is the modules as answeredMemberships shows them.
func idsText(ids []uuid.UUID) string {
	out := make([]string, len(ids))
	for n, id := range ids {
		out[n] = id.String()
	}
	return "[" + strings.Join(out, " ") + "]"
}
