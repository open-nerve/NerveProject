package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// Two changes of a work item serialize on its project's row (M4 design
// 3.5, 3.7; interleaving 1, the change's branch): alice's renames her Fix
// Mend and labels it Bug, and holds Web FOR NO KEY UPDATE and its rows
// past its writes; bob's, sent then, which renames it Repair and takes
// every label off, passes his precheck on the work item as it was and
// waits for Web's row. Once hers has committed he reads the work item
// again under his lock and changes it from what she left: his activity
// renames Mend, not Fix, and takes off the Bug she put on, which is gone,
// at a moment no earlier than the gate's opening, which he read under his
// lock.
func TestTwoChangesOfAWorkItemSerialize(t *testing.T) {
	w := newIssueWorld(t)
	aWorkItem(t, w)
	var hers, his exchange
	w.serialize(t, changesIssue("alice", func(_ *testing.T, w issueWorld) string {
		return `{"name":"Mend","label_ids":["` + w.bug.String() + `"]}`
	}), changesIssue("bob", func(*testing.T, issueWorld) string { return `{"name":"Repair","label_ids":[]}` }), &hers, &his)
	w.answered(t, hers, http.StatusOK)
	var changed struct {
		Name      string      `json:"name"`
		LabelIDs  []uuid.UUID `json:"label_ids"`
		UpdatedAt time.Time   `json:"updated_at"`
	}
	decodeAnswer(t, w.answered(t, his, http.StatusOK), &changed)
	if changed.Name != "Repair" || len(changed.LabelIDs) != 0 || changed.UpdatedAt.Before(hers.released) {
		t.Errorf("bob's change answers %+v; want Repair, no label, changed no earlier than %v", changed, hers.released)
	}
	said := w.activityBy(t, w.workItem(t), w.ids["bob"])
	if want := []string{"name: Mend → Repair", "labels: Bug → "}; !slices.Equal(said, want) {
		t.Errorf("bob's activity says %q, want %q", said, want)
	}
}

// A work item deleted, or moved into another project, while a change of it
// waits for its project's row is not found under the lock (M4 design 3.5:
// the work item read again under the lock, and found still in the project
// locked). No operation of P1b deletes or moves a work item, so a
// transaction does it in the database: it holds Web's row FOR NO KEY
// UPDATE and deletes alice's Fix, or moves it into Ops; bob's change,
// which passed his precheck on Fix as it was, waits for Web's row, and
// once the transaction commits finds no Fix in Web: 404 issue.not_found,
// and Fix keeps its name. The test probes for his wait on a row of
// projects: the transaction holds Fix's row too, and a wait for it, by a
// precheck that locked the work item, would be answered by the precheck,
// not under the lock.
func TestAWorkItemGoneWhileItsChangeWaitsIsNotFound(t *testing.T) {
	for _, moved := range []bool{false, true} {
		t.Run(map[bool]string{false: "deleted", true: "moved"}[moved], func(t *testing.T) {
			w := newIssueWorld(t)
			aWorkItem(t, w)
			item := w.workItem(t)
			gone, args := "UPDATE issues SET deleted_at = now() WHERE id = $1", []any{item}
			if moved {
				ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
				gone, args = "UPDATE issues SET project_id = $2 WHERE id = $1", []any{item, ops}
			}
			held := holding(t, w.pool, "SELECT 1 FROM projects WHERE id = $1 FOR NO KEY UPDATE", w.web)
			if _, err := held.Exec(pgtest.Soon(t), gone, args...); err != nil {
				t.Fatal(err)
			}
			change := newRequest(t, http.MethodPatch, w.base+"/api/v0/issues/"+item.String(), w.tokens["bob"], []byte(`{"name":"Mend"}`))
			w.contract.CheckRequest(t, change)
			changed := sendInBackground(change)
			pgtest.WaitForLockWaitOn(t, w.pool, "projects", 5*time.Second)
			if err := held.Commit(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			a := receiveWithin(t, changed, 5*time.Second, "answer to bob's change")
			if a.err != nil {
				t.Fatal(a.err)
			}
			w.contract.CheckResponse(t, change, a.res)
			if a.res.StatusCode != http.StatusNotFound || problemCode(t, a.body) != "issue.not_found" {
				t.Errorf("bob's change = %d %s; want 404 issue.not_found", a.res.StatusCode, a.body)
			}
			var name string
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT name FROM issues WHERE id = $1", item).Scan(&name); err != nil || name != "Fix" {
				t.Errorf("the work item is named %q (%v) after the refused change; want Fix", name, err)
			}
		})
	}
}
