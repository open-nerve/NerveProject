package bootstrap

import (
	"fmt"
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The automatic archival keeps the work items still in a cycle that has
// not ended or in a module whose target day has not passed, on the day the
// job's clock falls on in the project's time zone (M6 P3a spec R9, A4,
// design 7). Web, in Shanghai, has the cycles Old (1–10 March 2044), Last
// night (20–31 March), Today (1 April alone), Later (20–30 April), and Draft
// and Gone without dates, and the modules No target, Due today (1 April),
// Due later (10 April), Overdue (31 March), Shelved (10 April, completed)
// and Gone, without a target. The work items join them through the API;
// then Shelved is archived, the cycle and the module Gone are deleted, and
// one work item leaves Draft. Every work item, done and last changed 40
// days before, is due under Web's one month. The job's clock stands at
// 16:30 UTC on 31 March, already 1 April in Shanghai, and far from the
// app's own days. Its run archives the work items in no open target: in
// none, in Old and Last night, which have ended, in Overdue, whose day has
// passed, in the deleted ones and the one that left Draft; it keeps those
// in Today, which ends that day, Later, Draft, No target, Due today, Due
// later and Shelved, archived but not yet due, and those in both a closed
// and an open one. A second run on 1 May archives the work items kept by
// targets that have closed since, and keeps those of Draft and No target.
func TestAutoArchiveKeepsTheWorkOfOpenCyclesAndModules(t *testing.T) {
	w := newIssueWorld(t)
	w.turnCyclesOn(t, w.web)
	w.turnModulesOn(t, w.web)
	at, later := time.Date(2044, 3, 31, 16, 30, 0, 0, time.UTC), time.Date(2044, 5, 1, 0, 30, 0, 0, time.UTC)
	if day := shanghaiDay(t, at, 0); day != "2044-04-01" || at.Format(time.DateOnly) != "2044-03-31" {
		t.Fatalf("the job's clock %s must be on 1 April in Shanghai (%s) and on 31 March in UTC", at, day)
	}
	cycles := map[string]uuid.UUID{
		"Old":        w.newCycle(t, "alice", datesBody("Old", "2044-03-01", "2044-03-10")),
		"Last night": w.newCycle(t, "alice", datesBody("Last night", "2044-03-20", "2044-03-31")),
		"Today":      w.newCycle(t, "alice", datesBody("Today", "2044-04-01", "2044-04-01")),
		"Later":      w.newCycle(t, "alice", datesBody("Later", "2044-04-20", "2044-04-30")),
		"Draft":      w.newCycle(t, "alice", `{"name":"Draft"}`),
		"Gone":       w.newCycle(t, "alice", `{"name":"Gone"}`),
	}
	modules := map[string]uuid.UUID{
		"No target": w.newModule(t, `{"name":"No target"}`),
		"Due today": w.newModule(t, `{"name":"Due today","target_date":"2044-04-01"}`),
		"Due later": w.newModule(t, `{"name":"Due later","target_date":"2044-04-10"}`),
		"Overdue":   w.newModule(t, `{"name":"Overdue","target_date":"2044-03-31"}`),
		"Shelved":   w.newModule(t, `{"name":"Shelved","status":"completed","target_date":"2044-04-10"}`),
		"Gone":      w.newModule(t, `{"name":"Gone"}`),
	}
	// first and second: archived by the run at at, by the one at later.
	items := []struct {
		name, cycle, module string
		first, second       bool
	}{
		{name: "Free", first: true},
		{name: "In Old", cycle: "Old", first: true},
		{name: "In Last night", cycle: "Last night", first: true},
		{name: "In Today", cycle: "Today", second: true},
		{name: "In Later", cycle: "Later", second: true},
		{name: "In Draft", cycle: "Draft"},
		{name: "Left Draft", cycle: "Draft", first: true},
		{name: "In the deleted cycle", cycle: "Gone", first: true},
		{name: "In No target", module: "No target"},
		{name: "In Due today", module: "Due today", second: true},
		{name: "In Due later", module: "Due later", second: true},
		{name: "In Overdue", module: "Overdue", first: true},
		{name: "In Shelved", module: "Shelved", second: true},
		{name: "In the deleted module", module: "Gone", first: true},
		{name: "In Old and No target", cycle: "Old", module: "No target"},
		{name: "In Later and Overdue", cycle: "Later", module: "Overdue", second: true},
	}
	ids := make([]uuid.UUID, len(items))
	for i, it := range items {
		body := fmt.Sprintf(`{"name":%q`, it.name)
		if it.cycle != "" {
			body += fmt.Sprintf(`,"cycle_id":%q`, cycles[it.cycle])
		}
		if it.module != "" {
			body += fmt.Sprintf(`,"module_ids":[%q]`, modules[it.module])
		}
		ids[i] = w.newWorkItemIn(t, w.web, body+"}")
		if it.name == "Left Draft" {
			w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+ids[i].String(), `{"cycle_id":null}`, http.StatusOK)
		}
	}
	w.send(t, "alice", http.MethodPost, "/api/v0/modules/"+modules["Shelved"].String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodDelete, "/api/v0/cycles/"+cycles["Gone"].String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodDelete, "/api/v0/modules/"+modules["Gone"].String(), "", http.StatusNoContent)
	w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"archive_in":1}`, http.StatusOK)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2, updated_at=$3 WHERE project_id=$1", w.web, w.done, at.Add(-40*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for r, run := range []time.Time{at, later} {
		want := 0
		for _, it := range items {
			if it.first && r == 0 || it.second && r == 1 {
				want++
			}
		}
		if n, err := autoArchiveOn(w, clocktest.At(run), discardIssueEvents{}).Execute(pgtest.Soon(t)); err != nil || n != want {
			t.Errorf("the run at %s archived %d, %v; want %d", run, n, err, want)
		}
		for i, it := range items {
			wantAt := "never"
			switch {
			case it.first:
				wantAt = at.String()
			case it.second && r == 1:
				wantAt = later.String()
			}
			var archived *time.Time
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT archived_at FROM issues WHERE id = $1", ids[i]).Scan(&archived); err != nil {
				t.Fatal(err)
			}
			if got := archivedAt(archived); got != wantAt {
				t.Errorf("after the run at %s, %s is archived %s; want %s", run, it.name, got, wantAt)
			}
		}
	}
}

// archivedAt is when archived says a work item was archived, in UTC, or
// "never".
func archivedAt(archived *time.Time) string {
	if archived == nil {
		return "never"
	}
	return archived.UTC().String()
}
