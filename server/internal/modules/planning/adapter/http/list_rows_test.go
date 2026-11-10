package httpadapter_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"testing"
	"time"
	"uuid"

	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/http"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// withFields is the JSON object obj with the fields of the JSON object more too, by key, as the server writes them.
func withFields(t *testing.T, obj, more string) string {
	t.Helper()
	var fields, extra map[string]json.RawMessage
	if err := json.Unmarshal([]byte(obj), &fields); err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(more), &extra); err != nil {
		t.Fatal(err)
	}
	maps.Copy(fields, extra)
	out, err := json.Marshal(fields)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// noCounts are the six counts of a list row without work items.
const noCounts = `"total_issues":0,"backlog_issues":0,"unstarted_issues":0,"started_issues":0,"completed_issues":0,"cancelled_issues":0`

// uncountedCycle is the cycle as POST answers it, cycle, in a list row without work items: no assignee, not frozen.
func uncountedCycle(t *testing.T, cycle string) string {
	return withFields(t, cycle, `{`+noCounts+`,"assignee_ids":[],"progress_snapshot":null}`)
}

// uncountedModule is the module as POST answers it, module, in a list row without work items.
func uncountedModule(t *testing.T, module string) string {
	return withFields(t, module, `{`+noCounts+`}`)
}

// A cycle's list row answers the use case's numbers by their names (M6 P4 spec 2.3): its six counts, its assignees in
// their order, and its frozen counts with their moment, an object; without assignees an array, without frozen counts
// null.
func TestListCyclesAnswersTheNumbers(t *testing.T) {
	sprint, backlog := cycles(t)
	frozenAt := time.Date(2026, 11, 14, 16, 0, 0, 123456000, time.UTC)
	list := &fakeList{list: []domain.CycleListItem{
		{ReadCycle: sprint, Live: domain.Counts{Total: 21, Backlog: 1, Unstarted: 2, Started: 3, Completed: 7, Cancelled: 8},
			AssigneeIDs: []uuid.UUID{bobID, aliceID},
			Snapshot:    &domain.FrozenCounts{Counts: domain.Counts{Total: 15, Backlog: 1, Unstarted: 2, Started: 3, Completed: 4, Cancelled: 5}, FrozenAt: frozenAt}},
		{ReadCycle: backlog, Live: domain.Counts{Total: 1, Started: 1}},
	}}
	res, body := do(t, newServer(t, httpadapter.UseCases{ListCycles: list}), request(http.MethodGet, "/api/v0/projects/"+webID.String()+"/cycles", "alice", ""))
	want := `{"data":[` + withFields(t, sprintJSON, `{"total_issues":21,"backlog_issues":1,"unstarted_issues":2,"started_issues":3,`+
		`"completed_issues":7,"cancelled_issues":8,"assignee_ids":["`+bobID.String()+`","`+aliceID.String()+`"],"progress_snapshot":`+
		`{"backlog_issues":1,"cancelled_issues":5,"completed_issues":4,"frozen_at":"2026-11-14T16:00:00.123456Z","started_issues":3,`+
		`"total_issues":15,"unstarted_issues":2}}`) + `,` +
		withFields(t, backlogJSON, `{"total_issues":1,"backlog_issues":0,"unstarted_issues":0,"started_issues":1,"completed_issues":0,`+
			`"cancelled_issues":0,"assignee_ids":[],"progress_snapshot":null}`) + `]}`
	if res.StatusCode != http.StatusOK || body != want+"\n" {
		t.Errorf("GET = %d %s\nwant 200 %s", res.StatusCode, body, want)
	}
}

// A module's list row answers the use case's six counts by their names (M6 P4 spec 2.3).
func TestListModulesAnswersTheNumbers(t *testing.T) {
	billing, auth := modules(t)
	list := &fakeListModules{list: []domain.ModuleListItem{
		{Module: billing, Live: domain.Counts{Total: 21, Backlog: 1, Unstarted: 2, Started: 3, Completed: 7, Cancelled: 8}},
		{Module: auth},
	}}
	res, body := do(t, newServer(t, httpadapter.UseCases{ListModules: list}), request(http.MethodGet, "/api/v0/projects/"+webID.String()+"/modules", "alice", ""))
	want := `{"data":[` + withFields(t, billingJSON, `{"total_issues":21,"backlog_issues":1,"unstarted_issues":2,"started_issues":3,`+
		`"completed_issues":7,"cancelled_issues":8}`) + `,` + uncountedModule(t, authJSON) + `]}`
	if res.StatusCode != http.StatusOK || body != want+"\n" {
		t.Errorf("GET = %d %s\nwant 200 %s", res.StatusCode, body, want)
	}
}
