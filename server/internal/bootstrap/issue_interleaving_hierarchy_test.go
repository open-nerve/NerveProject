package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// workItemNamed is the id of the world's one work item named name, of any
// project: the test fails unless there is exactly one.
func (w issueWorld) workItemNamed(t *testing.T, name string) uuid.UUID {
	t.Helper()
	ids := queryIDs(t, w.pool, "SELECT id FROM issues WHERE name = $1", name)
	if len(ids) != 1 {
		t.Fatalf("the work items named %s: %v; want one", name, ids)
	}
	return ids[0]
}

// movesUnder is name's change of Web's work item named child, which puts
// it under Web's work item named parent (workItemNamed).
func movesUnder(name, child, parent string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPatch, "/api/v0/issues/"+w.workItemNamed(t, child).String(),
			`{"parent_id":"`+w.workItemNamed(t, parent).String()+`"}`)
	}
}

// Two moves, each allowed alone, that together would make a loop
// serialize on Web's row (M4 design 3.5, 3.9; interleaving 9), in both
// orders: alice puts A under B while bob puts B under A. The first holds
// Web FOR NO KEY UPDATE past its writes; the second, which passed its
// precheck, waits for Web's row, and once the first commits reads the
// ancestry of its parent under its lock, finds its own work item in it,
// and is refused: 422 at parent_id, nothing written. The first's work item
// is under the other, which is under none, and only the first writes a row
// of a parent.
func TestTwoMovesThatWouldMakeALoopSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.newWorkItemIn(t, w.web, `{"name":"A"}`)
		w.newWorkItemIn(t, w.web, `{"name":"B"}`)
	}
	inBothOrders(t, prepare, "A under B", movesUnder("alice", "A", "B"), "B under A", movesUnder("bob", "B", "A"), func(t *testing.T,
		w issueWorld, aFirst bool, ea, eb exchange) {
		first, second, under, over := ea, eb, w.workItemNamed(t, "A"), w.workItemNamed(t, "B")
		if !aFirst {
			first, second, under, over = eb, ea, over, under
		}
		w.answered(t, first, http.StatusOK)
		if answer := w.answered(t, second, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), true) != parentNotAllowed {
			t.Errorf("the second move answers %s, want %s", answer, parentNotAllowed)
		}
		if !reflect.DeepEqual(w.parentOf(t, under), &over) || w.parentOf(t, over) != nil || len(w.parentRows(t, under)) != 1 ||
			len(w.parentRows(t, over)) != 0 {
			t.Errorf("the first's work item is under %v, the other under %v, their rows of a parent %q and %q; want under %s, under none, "+
				"one row and none", w.parentOf(t, under), w.parentOf(t, over), w.parentRows(t, under), w.parentRows(t, over), over)
		}
	})
}
