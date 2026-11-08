package bootstrap

import (
	"net/http"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// targetNamed is the id of Web's row of table, cycles or modules, named
// name, deleted or not.
func (w issueWorld) targetNamed(t *testing.T, table, name string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM "+table+" WHERE project_id = $1 AND name = $2", w.web, name).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// deletesTarget is alice's deletion of Web's cycle or module named name,
// table cycles or modules, through the planning module's route, its
// deletion published to the issue module's subscribers in its transaction.
func deletesTarget(table, name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newPlanningRoute(t, w.pool, tx), "alice", http.MethodDelete, "/api/v0/"+table+"/"+w.targetNamed(t, table, name).String(), "")
	}
}

// joinsTarget is bob's change of Web's one work item (workItem) into Web's
// cycle Sprint, table cycles, or its module Auth, modules.
func joinsTarget(table string) racingWrite {
	return changesIssue("bob", func(t *testing.T, w issueWorld) string {
		if table == "cycles" {
			return `{"cycle_id":"` + w.targetNamed(t, table, "Sprint").String() + `"}`
		}
		return modulesBody(w.targetNamed(t, table, "Auth"))
	})
}

// A work item's joining a cycle or a module and the target's deletion
// serialize on Web's row, which each takes FOR NO KEY UPDATE first (M6
// design 6, 10; P3a spec A5), in both orders. Deleted first, the change,
// waiting, reads the target's facts under the lock it then takes and finds
// it gone: 422, cycle_id or module_ids[0] not_allowed, and nothing written.
// Joined first, the deletion, waiting, then takes the work item out in its
// own transaction (the issue module's subscriber): bob's row, made at his
// change, deleted at the deletion's moment by alice, a row of activity for
// the join, his, then for the deletion, hers, and the work item last changed
// by her then; its answer reads it in neither.
func TestJoiningAndDeletingATargetInterleave(t *testing.T) {
	for _, tc := range []struct {
		table, name, rows, field, kind string
	}{
		{"cycles", "Sprint", "cycle_issues", "cycle_id", "cycle"},
		{"modules", "Auth", "module_issues", "module_ids[0]", "module"},
	} {
		t.Run(tc.table, func(t *testing.T) {
			prepare := func(t *testing.T, w issueWorld) {
				if tc.table == "cycles" {
					w.turnCyclesOn(t, w.web)
					w.newCycle(t, "alice", `{"name":"Sprint"}`)
				} else {
					w.turnModulesOn(t, w.web)
					w.newModule(t, `{"name":"Auth"}`)
				}
				aWorkItem(t, w)
			}
			inBothOrdersAt(t, "projects", prepare, "join", joinsTarget(tc.table), "deletion", deletesTarget(tc.table, tc.name),
				func(t *testing.T, w issueWorld, joinFirst bool, join, deletion exchange) {
					w.answered(t, deletion, http.StatusNoContent)
					item, target := w.workItem(t), w.targetNamed(t, tc.table, tc.name)
					n := cycleNames{w: w, cycles: map[uuid.UUID]string{target: tc.name}, moments: map[string]string{}}
					var deletedAt time.Time
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+tc.table+" WHERE id = $1", target).Scan(&deletedAt); err != nil {
						t.Fatal(err)
					}
					n.moment("deleted", deletedAt)
					created := n.cycleActivity(t, item)[:1]
					if !joinFirst {
						w.refused(t, join, http.StatusUnprocessableEntity, "validation_failed "+tc.field+" not_allowed")
						wantRows(t, "rows after the refused join", n.snapshot(t, tc.rows, item).rows, nil)
						wantRows(t, "activity after the refused join", n.cycleActivity(t, item), created)
						return
					}
					var joined joinedItem
					decodeAnswer(t, w.answered(t, join, http.StatusOK), &joined)
					n.moment("joined", joined.UpdatedAt)
					var gone joinedItem
					decodeAnswer(t, w.send(t, "bob", http.MethodGet, "/api/v0/issues/"+item.String(), "", http.StatusOK), &gone)
					if gone.CycleID != nil || len(gone.ModuleIDs) != 0 {
						t.Errorf("the work item read in %v and %v after the deletion; want neither", gone.CycleID, gone.ModuleIDs)
					}
					n.wantSnapshot(t, "the work item after the deletion", tc.rows, item, membershipSnapshot{
						rows: []string{tc.name + " made by bob at joined, left by alice at deleted, written at deleted"},
						activity: append(created, tc.table+` created ""→"`+tc.name+`" -→`+tc.name+` "added `+tc.kind+` " by bob at joined`,
							tc.table+` deleted "`+tc.name+`"→"" `+tc.name+`→- "removed this work item from `+tc.kind+` " by alice at deleted`),
						audit: "by alice at deleted",
					})
				})
		})
	}
}
