package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
)

// The app refuses a transfer in the order of M6 P4 spec 2.3, on the business
// clock at noon in Shanghai, Web's time zone. While Sprint runs, a transfer
// of its work items is refused as of a cycle not completed, as is one of
// Later's, ahead, and of a draft's; a target that is none of the project's
// is refused before that. Sprint ended, with a in Todo: of Shelved, ended
// and archived, refused as archived before its target is looked at; to a
// cycle that never was, to Gone, deleted, to Elsewhere, of Api, and to
// Sprint itself, at target_cycle_id; to Shelved, archived, which has ended
// too, as archived; to Old, which has ended, as completed. A body without
// its target, with a value that is no id, or with another field, is a bad
// request. walt, acme's admin who sees only his own work items in Web, and
// wanda, acme's admin and none of Web's members, are refused; olga, who does
// not see Web, and a cycle that never was, are not found. With Web's cycles
// turned off, and then Web archived, bob is refused so; walt and olga as
// before. Nothing is frozen or moved.
func TestTransferCycleIssuesRefusalsThroughTheApp(t *testing.T) {
	w, now := newNoonWorld(t)
	waitForStartupRuns(t, w.pool)
	w.turnCyclesOn(t, w.web)
	w.adminGuest(t)
	day := func(days int) string { return shanghaiDay(t, now, days) }
	sprint := w.newCycle(t, "alice", datesBody("Sprint", day(-10), day(1)))
	next := w.newCycle(t, "alice", datesBody("Next", day(5), day(10)))
	later := w.newCycle(t, "alice", datesBody("Later", day(12), day(15)))
	draft := w.newCycle(t, "alice", `{"name":"Draft"}`)
	a := w.newWorkItemBy(t, "alice", w.web, workItemBody("a", w.todo, nil, nil, `,"cycle_id":"`+sprint.String()+`"`))
	const notCompleted, notAllowed = "planning.cycle_not_completed", "validation_failed target_cycle_id not_allowed"
	for _, from := range []uuid.UUID{sprint, later, draft} {
		w.refusedTransfer(t, "bob", from, transferBody(next), http.StatusConflict, notCompleted)
	}
	w.refusedTransfer(t, "bob", sprint, transferBody(uuid.NewV7()), http.StatusUnprocessableEntity, notAllowed)

	w.send(t, "alice", http.MethodPatch, "/api/v0/cycles/"+sprint.String(), `{"start_date":"`+day(-10)+`","end_date":"`+day(-1)+`"}`, http.StatusOK)
	old := w.newCycle(t, "alice", datesBody("Old", day(-30), day(-20)))
	shelved := w.newCycle(t, "alice", datesBody("Shelved", day(-19), day(-15)))
	w.send(t, "alice", http.MethodPost, "/api/v0/cycles/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newCycle(t, "alice", `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+gone.String(), "", http.StatusNoContent)
	api := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Api", "API")
	w.turnCyclesOn(t, api)
	elsewhere := w.cycleIn(t, api, datesBody("Elsewhere", day(5), day(10)))
	for _, tt := range []struct {
		name     string
		from, to uuid.UUID
		status   int
		refusal  string
	}{
		{"bob", shelved, next, http.StatusConflict, "planning.archived"},
		{"bob", shelved, uuid.NewV7(), http.StatusConflict, "planning.archived"},
		{"bob", sprint, uuid.NewV7(), http.StatusUnprocessableEntity, notAllowed},
		{"bob", sprint, gone, http.StatusUnprocessableEntity, notAllowed},
		{"bob", sprint, elsewhere, http.StatusUnprocessableEntity, notAllowed},
		{"bob", sprint, sprint, http.StatusUnprocessableEntity, notAllowed},
		{"bob", sprint, shelved, http.StatusConflict, "planning.archived"},
		{"bob", sprint, old, http.StatusConflict, "planning.cycle_completed"},
		{"walt", sprint, next, http.StatusForbidden, "forbidden"},
		{"wanda", sprint, next, http.StatusForbidden, "forbidden"},
		{"olga", sprint, next, http.StatusNotFound, "planning.cycle_not_found"},
		{"bob", uuid.NewV7(), next, http.StatusNotFound, "planning.cycle_not_found"},
	} {
		w.refusedTransfer(t, tt.name, tt.from, transferBody(tt.to), tt.status, tt.refusal)
	}
	for body, refusal := range map[string]string{`{}`: "bad_request target_cycle_id required",
		`{"target_cycle_id":"next"}`: "bad_request target_cycle_id invalid_format",
		`{"target_cycle_id":"` + next.String() + `","new_cycle_id":"` + next.String() + `"}`: "bad_request new_cycle_id not_allowed"} {
		w.refusedShape(t, "bob", http.MethodPost, transferPath(sprint), body, refusal)
	}

	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"cycle_view":false}`, http.StatusOK)
	w.refusedTransfer(t, "bob", sprint, transferBody(next), http.StatusConflict, "planning.cycles_disabled")
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", http.StatusOK)
	w.refusedTransfer(t, "bob", sprint, transferBody(next), http.StatusConflict, "project.archived")
	w.refusedTransfer(t, "walt", sprint, transferBody(next), http.StatusForbidden, "forbidden")
	w.refusedTransfer(t, "olga", sprint, transferBody(next), http.StatusNotFound, "planning.cycle_not_found")

	if got := frozenOf(t, w.pool, sprint); got != nil {
		t.Errorf("Sprint frozen as %+v; want no refusal to freeze it", got)
	}
	if in := queryIDs(t, w.pool, "SELECT issue_id FROM cycle_issues WHERE cycle_id = $1 AND deleted_at IS NULL", sprint); !slices.Equal(in, []uuid.UUID{a}) {
		t.Errorf("Sprint has %v; want a, moved by no refusal", in)
	}
}
