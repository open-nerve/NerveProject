package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// The group of the trim and a subtree's deletion of
// TestRecentVisitWritesInterleave (navigation_recent_visits_interleaving_test.go).

// trimsBesideASubtreesDeletion is the group of the trim and a subtree's
// deletion (spec 2.5, 2.8, 4; re-92 I4 ②, m1). A work item's deletion
// soft-deletes its subtree's visits in its transaction work item by work
// item, in the subtree's id order (an issue.deleted each), not in one id
// order over the visits; the recording's trim locks the visits past the
// cap in id order. One recorder's cell each (trimBesideASubtree):
//   - One past the cap: erin's visit of the subtree's first work item is
//     her one visit past the cap once she records. The deletion holds it;
//     a trim that waited for it would wait for the whole deletion while
//     holding its parents' and its advisory lock.
//   - Two past the cap (a state only a rolled back deletion leaves, so
//     seeded): finn's visits of both work items are his two past the cap,
//     the second's of an id before the first's. A trim that waited would
//     lock the second's and wait for the first's, which the deletion holds;
//     let go, the deletion would wait for the second's: a cycle, 40P01, one
//     of the two answered 500 (spec 2.8).
func trimsBesideASubtreesDeletion(t *testing.T, w visitRaceWorld) {
	t.Run("one past the cap", func(t *testing.T) { w.trimBesideASubtree(t, "erin", false) })
	t.Run("two past the cap", func(t *testing.T) { w.trimBesideASubtree(t, "finn", true) })
}

// trimBesideASubtree is one cell of trimsBesideASubtreesDeletion's, as
// TestMarkAllReadPassesOverTheNotificationsAWorkItemsDeletionHolds builds
// it: alice makes Parent of Web, and Child under it, each assigned to bob,
// so Parent's id is before Child's; actor, who has no other visit in acme,
// has 19 visits there of no work item and, before them all, his visit of
// Parent, and when both is set of Child too, its row's id before
// Parent's (seedPastVisits). Once no fan-out is pending, a transaction
// holds bob's first notification of Child, and alice's deletion of Parent,
// sent through the API, is proven by its own request probe waiting for it
// in DeleteIssueNotifications: Parent's issue.deleted has run, its cascade
// holding actor's visit of Parent soft-deleted, and Child's has not. Then
// actor records his visit of Ops, outside Web, whose FOR SHARE would
// otherwise wait for the deletion: his trim finds his visit of Parent past
// the cap, and of Child when both is set. The recording answers 204 having
// waited for no row (answeredUnlessWaiting); then the notification is let
// go, and both answers are in before any is judged, so that a 500 of
// either shows. The deletion answers 204. actor has 20 undeleted visits in
// acme, Ops' among them: his visit of Parent deleted at the deletion's
// moment, by alice, his visit of Child, when both is set, deleted for good
// by the trim, before the deletion's cascade reached it.
func (w visitRaceWorld) trimBesideASubtree(t *testing.T, actor string, both bool) {
	owner := w.ids[actor]
	toBob := fmt.Sprintf(`"assignee_ids":%s}`, idArray(w.ids["bob"]))
	parent := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"%s's parent",`, actor)+toBob)
	child := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"%s's child","parent_id":%q,`, actor, parent)+toBob)
	items, ids := append(freshIDs(navigationdomain.RecentVisitCap-1), parent), freshIDs(navigationdomain.RecentVisitCap)
	if both {
		rows := freshIDs(2)
		items, ids = append(items, child), append(ids[:navigationdomain.RecentVisitCap-1], rows[1], rows[0])
	}
	seedPastVisits(t, w.pool, w.web, owner, items, ids...)
	waitForFanouts(t, w.pool)
	ofChild := queryIDs(t, w.pool, "SELECT id FROM notifications WHERE receiver_id = $1 AND issue_id = $2 ORDER BY id", w.ids["bob"], child)
	ofParent := queryIDs(t, w.pool, "SELECT id FROM notifications WHERE receiver_id = $1 AND issue_id = $2 ORDER BY id", w.ids["bob"], parent)
	if parent.Compare(child) >= 0 || len(ofChild) == 0 || len(ofParent) == 0 {
		t.Fatalf("Parent %s, Child %s, bob's notifications of Parent %v, of Child %v; want Parent's id before Child's, and some of each",
			parent, child, ofParent, ofChild)
	}
	visitOf := func(item uuid.UUID) uuid.UUID { return ids[slices.Index(items, item)] }

	note := holding(t, w.pool, "SELECT 1 FROM notifications WHERE id = $1 FOR UPDATE", ofChild[0])
	deletion := newRequest(t, http.MethodDelete, w.base+"/api/v0/issues/"+parent.String(), w.tokens["alice"], nil)
	w.contract.CheckRequest(t, deletion)
	deleting := w.requests.observe(deletion)
	deleted := sendInBackground(deletion)
	pgtest.WaitForStatementLockWaitBehind(t, w.pool, note.Conn().PgConn(), 5*time.Second, "DeleteIssueNotifications")
	if sample, waiting := readRequestLock(t, pgtest.Soon(t), w.pool, deleting); !waiting ||
		!strings.Contains(sample.query, "-- name: DeleteIssueNotifications ") || !slices.Contains(sample.blockers, int32(note.Conn().PgConn().PID())) {
		t.Fatalf("alice's deletion of Parent waits (%t) in %q behind %v; want its DeleteIssueNotifications behind the held notification",
			waiting, strings.Join(strings.Fields(sample.query), " "), sample.blockers)
	}
	locks := map[uuid.UUID]string{parent: "FOR NO KEY UPDATE"}
	if both {
		locks[child] = "no lock"
	}
	for item, want := range locks {
		if got := requestRowLock(t, w.pool, w.pool, deleting, "user_recent_visits WHERE id = $1", visitOf(item)); got != want {
			t.Fatalf("alice's deletion holds %s's visit of %s %s while it waits; want %s", actor, item, got, want)
		}
	}

	recording := newRequest(t, http.MethodPost, w.base+recordVisitPath, w.tokens[actor], []byte(visitBody("project", w.ops)))
	w.contract.CheckRequest(t, recording)
	recorded := sendInBackground(recording)
	a, unwaited := answeredUnlessWaiting(t, w.pool, recorded, "TrimRecentVisits")
	release(t, note)
	if !unwaited {
		a = receiveWithin(t, recorded, 10*time.Second, "the answer to "+actor+"'s recording")
	}
	d := receiveWithin(t, deleted, 10*time.Second, "the answer to alice's deletion of Parent")
	if d.err != nil {
		t.Fatal(d.err)
	}
	w.contract.CheckResponse(t, deletion, d.res)
	w.recorded(t, recording, a)
	if d.res.StatusCode != http.StatusNoContent {
		t.Errorf("alice's deletion of Parent = %d %s; want 204", d.res.StatusCode, d.body)
	}
	if !unwaited {
		t.Errorf("%s's recording waited in its trim for a visit alice's deletion of Parent held; want it passed over", actor)
	}

	if n := w.undeletedVisits(t, owner); n != navigationdomain.RecentVisitCap {
		t.Errorf("%s has %d undeleted visits in acme; want %d", actor, n, navigationdomain.RecentVisitCap)
	}
	liveVisit(t, w.pool, owner, w.ops)
	at, by := stampOf(t, w.pool, "issues", "deleted_at", parent)
	if v := visitRowsWhere(t, w.pool, "id = $1", visitOf(parent)); len(v) != 1 || v[0].deletedAt == nil || !v[0].deletedAt.Equal(at) ||
		!v[0].updatedAt.Equal(at) || v[0].updatedBy == nil || *v[0].updatedBy != by || by != w.ids["alice"] {
		t.Errorf("%s's visit of Parent: %v; want it deleted and last changed at the deletion, %s, by alice", actor, v, at.UTC().Format(time.RFC3339Nano))
	}
	if both {
		if v := visitRowsWhere(t, w.pool, "id = $1", visitOf(child)); len(v) != 0 {
			t.Errorf("%s's visit of Child: %v; want it deleted for good by his trim", actor, v)
		}
	}
}
