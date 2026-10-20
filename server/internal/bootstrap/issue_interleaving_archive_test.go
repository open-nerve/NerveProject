package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// archivesIssue is name's archiving of Web's one work item (workItem).
func archivesIssue(name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, "/api/v0/issues/"+w.workItem(t).String()+"/archive", "")
	}
}

// inProgress is the body of a change into Web's In Progress, a state of
// the started group.
func inProgress(t *testing.T, w issueWorld) string {
	return `{"state_id":"` + stateID(t, w.pool, w.web, "In Progress").String() + `"}`
}

// Archiving a work item and moving it out of the completed group serialize
// on Web's row (M4 design 3.8; interleaving 12), in both orders: alice
// archives her Fix in Done (doneWorkItem), bob moves it into In Progress.
// Moved first, holding Web FOR NO KEY UPDATE: the archiving, which passed
// its precheck, finds it in the started group under its lock: 409
// issue.not_archivable, and it is not archived. Archived first: the
// change, which passed its precheck, finds it archived under its lock: 409
// issue.archived, and it stays in Done. Only the write that went through
// is in the activity: bob's move, or the archiving.
func TestArchivingAWorkItemAndMovingItSerialize(t *testing.T) {
	inBothOrders(t, func(t *testing.T, w issueWorld) { w.doneWorkItem(t) }, "archived", archivesIssue("alice"), "moved",
		changesIssue("bob", inProgress), func(t *testing.T, w issueWorld, archivedFirst bool, archived, moved exchange) {
			state, said, archivals := "In Progress", []string{"state: Done → In Progress"}, 0
			if archivedFirst {
				w.answered(t, archived, http.StatusOK)
				w.refused(t, moved, http.StatusConflict, "issue.archived")
				state, said, archivals = "Done", nil, 1
			} else {
				w.answered(t, moved, http.StatusOK)
				w.refused(t, archived, http.StatusConflict, "issue.not_archivable")
			}
			item := w.workItem(t)
			var in uuid.UUID
			var isArchived bool
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT state_id, archived_at IS NOT NULL FROM issues WHERE id = $1", item).Scan(&in,
				&isArchived); err != nil {
				t.Fatal(err)
			}
			if got := w.activityBy(t, item, w.ids["bob"]); in != stateID(t, w.pool, w.web, state) || isArchived != archivedFirst ||
				!slices.Equal(got, said) || len(archivalRows(t, w, item)) != archivals {
				t.Errorf("the work item in %s, archived %v, bob's activity %q, %d archival rows; want in %s, archived %v, %q, %d", in,
					isArchived, got, len(archivalRows(t, w, item)), state, archivedFirst, said, archivals)
			}
		})
}
