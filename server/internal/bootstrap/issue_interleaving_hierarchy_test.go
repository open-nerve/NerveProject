package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
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

// deletesNamed is name's deletion of the world's work item named item
// (workItemNamed).
func deletesNamed(name, item string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete, "/api/v0/issues/"+w.workItemNamed(t, item).String(), "")
	}
}

// addsUnder is name's addition of Web's work item named child under Web's
// work item named parent (workItemNamed).
func addsUnder(name, child, parent string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodPost, subIssues(w.workItemNamed(t, parent)),
			subIssueIDs(w.workItemNamed(t, child)))
	}
}

// Deleting a work item and putting another under it serialize on Web's
// row (M4 design 3.5, 3.6, 3.9; interleaving 13), in both orders: alice
// deletes Up while bob puts Part under it. Deleted first, holding Web FOR
// NO KEY UPDATE: the addition, which passed its precheck on Up as it was,
// finds no Up under its lock: 404 issue.not_found, and Part is as it was,
// undeleted, under none, without a row of a parent. Added first: the
// deletion waits for Web's row, then reads Up's subtree under its lock,
// Part in it, and deletes both with their rows, bob's row of Part's parent
// among them, seven, by alice, at its one moment, no earlier than the
// addition's commit.
func TestDeletingAWorkItemAndPuttingAnotherUnderItSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
		w.newWorkItemIn(t, w.web, `{"name":"Part"}`)
	}
	inBothOrders(t, prepare, "deleted", deletesNamed("alice", "Up"), "added", addsUnder("bob", "Part", "Up"), func(t *testing.T,
		w issueWorld, deletedFirst bool, deleted, added exchange) {
		w.answered(t, deleted, http.StatusNoContent)
		if !deletedFirst {
			w.answered(t, added, http.StatusOK)
			w.deletedWith(t, "issues", w.workItemNamed(t, "Up"), added.released, 7)
			return
		}
		w.refused(t, added, http.StatusNotFound, "issue.not_found")
		part := w.workItemNamed(t, "Part")
		var gone bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NOT NULL FROM issues WHERE id = $1", part).Scan(&gone); err != nil {
			t.Fatal(err)
		}
		if gone || w.parentOf(t, part) != nil || len(w.parentRows(t, part)) != 0 {
			t.Errorf("Part is deleted %v, under %v, with the rows of a parent %q; want undeleted, under none, without", gone,
				w.parentOf(t, part), w.parentRows(t, part))
		}
	})
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

// Putting a work item under another and moving that other under the first
// serialize on Web's row (M4 design 3.5, 3.9; interleaving 9's addSubIssues
// half), in both orders: alice puts A under B while bob moves B under A.
// The first holds Web FOR NO KEY UPDATE past its writes; the second, which
// passed its precheck, waits for Web's row, and once the first commits
// reads the ancestry under its lock, finds the loop and is refused at its
// place, nothing written: the move 422 at parent_id, the addition 422 at
// sub_issue_ids[0]. The first's work item is under the other, which is
// under none, and only the first writes a row of a parent.
func TestAddingSubItemsAndAMoveThatWouldMakeALoopSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		w.newWorkItemIn(t, w.web, `{"name":"A"}`)
		w.newWorkItemIn(t, w.web, `{"name":"B"}`)
	}
	inBothOrders(t, prepare, "A added under B", addsUnder("alice", "A", "B"), "B moved under A", movesUnder("bob", "B", "A"),
		func(t *testing.T, w issueWorld, addedFirst bool, added, moved exchange) {
			under, over := w.workItemNamed(t, "A"), w.workItemNamed(t, "B")
			if addedFirst {
				w.answered(t, added, http.StatusOK)
				if answer := w.answered(t, moved, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), true) != parentNotAllowed {
					t.Errorf("the move answers %s, want %s", answer, parentNotAllowed)
				}
			} else {
				under, over = over, under
				w.answered(t, moved, http.StatusOK)
				if answer := w.answered(t, added, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), false) !=
					"validation_failed sub_issue_ids[0] not_allowed" {
					t.Errorf("the addition answers %s, want sub_issue_ids[0] not_allowed", answer)
				}
			}
			if !reflect.DeepEqual(w.parentOf(t, under), &over) || w.parentOf(t, over) != nil || len(w.parentRows(t, under)) != 1 ||
				len(w.parentRows(t, over)) != 0 {
				t.Errorf("the first's work item is under %v, the other under %v, their rows of a parent %q and %q; want under %s, under "+
					"none, one row and none", w.parentOf(t, under), w.parentOf(t, over), w.parentRows(t, under), w.parentRows(t, over), over)
			}
		})
}
