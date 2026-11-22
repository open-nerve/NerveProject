package bootstrap

import (
	"net/http"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// visitsFollowWorld is TestRecentVisitsFollowDeletedWorkItems' world:
// issueWorld's acme and Web, Web's intake turned on. bob makes the
// three-level subtree Parent, Child and Grandchild of Web; alice makes
// Kept and Old, in Done, of Web, and Shelf, a project of acme whose member
// bob is, with its work item Shelved; bob's own workspace Elsewhere has
// his project Side.
type visitsFollowWorld struct {
	issueWorld
	parent, child, grandchild, kept, old, shelf, shelved, side uuid.UUID
}

// What follows a work item's deletion, and its project's, among the recent
// visits on the real app (H17's deletions, M7 P3d spec 2.6, 4, R5, R10),
// each cell a subtest in order on one world (visitsFollowWorld). Seeded so:
// alice and bob both visit each target in acme through the API (bothVisit),
// one row each a target, and bob visits Side; the subtree is bob's, and
// alice, not its creator, deletes it. The visits of Shelf, Shelved and
// Side, which no deletion takes, stay as they were, byte for byte, through
// every cell. The start-time runs are waited for before the first deletion
// (waitForStartupRuns). Every deletion through the app has its actor, so
// the cascade of nerve's own, with none, is the unit tests'
// (TestForgettingADeletedWorkItemsVisits, TestAWorkItemsDeletionForgetsEveryonesVisitsOfIt).
func TestRecentVisitsFollowDeletedWorkItems(t *testing.T) {
	w := visitsFollowWorld{issueWorld: newIssueWorld(t)}
	w.intakeOn(t)
	w.parent = w.newWorkItemBy(t, "bob", w.web, `{"name":"Parent"}`)
	w.child = w.newWorkItemBy(t, "bob", w.web, `{"name":"Child","parent_id":"`+w.parent.String()+`"}`)
	w.grandchild = w.newWorkItemBy(t, "bob", w.web, `{"name":"Grandchild","parent_id":"`+w.child.String()+`"}`)
	w.kept = w.newWorkItemIn(t, w.web, `{"name":"Kept"}`)
	w.old = w.doneWorkItem(t)
	w.shelf = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shelf", "SHF")
	w.makeMember(t, "bob", w.shelf, shared.RoleMember)
	w.shelved = w.newWorkItemIn(t, w.shelf, `{"name":"Shelved"}`)
	w.send(t, "bob", http.MethodPost, "/api/v0/workspaces", `{"name":"Elsewhere","slug":"elsewhere"}`, http.StatusCreated)
	w.side = createdProject(t, w.contract, w.base, w.tokens["bob"], "elsewhere", "Side", "SID")
	w.bothVisit(t, "project", w.web)
	for _, id := range []uuid.UUID{w.parent, w.child, w.grandchild, w.kept, w.old} {
		w.bothVisit(t, "issue", id)
	}
	w.bothVisit(t, "project", w.shelf)
	w.bothVisit(t, "issue", w.shelved)
	w.visit(t, "bob", "project", w.side)
	untouched := append(liveVisits(t, w.pool, w.both(), w.shelf, w.shelved), liveVisit(t, w.pool, w.ids["bob"], w.side))
	rows := visitsText(t, w.pool, untouched...)
	waitForStartupRuns(t, w.pool)
	var forgotten []visitRow
	t.Run("subtree", func(t *testing.T) { forgotten = forgetsADeletedSubtree(t, w) })
	t.Run("intake", func(t *testing.T) { forgotten = append(forgotten, forgetsADeletedIntakeItem(t, w)...) })
	t.Run("archives", func(t *testing.T) { keepsVisitsThroughArchives(t, w) })
	t.Run("the project after its work items", func(t *testing.T) { deletesTheRestWithTheProject(t, w, forgotten) })
	if got := visitsText(t, w.pool, untouched...); got != rows {
		t.Errorf("the visits no deletion takes changed:\n%s\nwere\n%s", got, rows)
	}
}

// both is alice and bob.
func (w visitsFollowWorld) both() []uuid.UUID { return []uuid.UUID{w.ids["alice"], w.ids["bob"]} }

// bothVisit is alice's and bob's recordings of their visits of the target
// of kind with id.
func (w visitsFollowWorld) bothVisit(t *testing.T, kind string, id uuid.UUID) {
	t.Helper()
	w.visit(t, "alice", kind, id)
	w.visit(t, "bob", kind, id)
}

// forgottenAt checks that the visits rows, alice's and bob's of the
// targets ids, as they were before, are deleted at the moment of the work
// item first's deletion, by its deleter, their visits kept; that neither
// lists any of the targets; and that each one's recording of each answers
// as an id that never was. It answers the rows as they are now.
func (w visitsFollowWorld) forgottenAt(t *testing.T, first uuid.UUID, rows []visitRow, ids ...uuid.UUID) []visitRow {
	t.Helper()
	at, by := stampOf(t, w.pool, "issues", "deleted_at", first)
	if by != w.ids["alice"] {
		t.Fatalf("the deletion of %s by %s; want alice's", first, by)
	}
	visitsDeletedAt(t, w.pool, at, by, rows...)
	for _, actor := range []string{"alice", "bob"} {
		if listed := w.listedTargets(t, actor, "acme"); containsAny(listed, ids...) {
			t.Errorf("%s lists %v; want none of %v", actor, listed, ids)
		}
		for _, id := range ids {
			w.visitAnsweredAsAbsent(t, actor, "issue", id)
		}
	}
	now := make([]visitRow, len(rows))
	for n, r := range rows {
		now[n] = visitRowsWhere(t, w.pool, "id = $1", r.id)[0]
	}
	return now
}

// forgetsADeletedSubtree is the subtree's cell (spec 2.6, R10, M4 M7-04):
// alice deletes Parent, which bob made, and with it Child and Grandchild.
// Every account's visits of each, alice's and bob's, are deleted at the
// deletion's moment, by alice, their visits kept; neither lists them, and
// each one's recording of them answers as an id that never was. The visits
// of Web, Kept and Old are as they were, byte for byte. It answers the
// deleted visits as they are now.
func forgetsADeletedSubtree(t *testing.T, w visitsFollowWorld) []visitRow {
	subtree := []uuid.UUID{w.parent, w.child, w.grandchild}
	visits := liveVisits(t, w.pool, w.both(), subtree...)
	others := liveVisits(t, w.pool, w.both(), w.web, w.kept, w.old)
	kept := visitsText(t, w.pool, others...)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+w.parent.String(), "", http.StatusNoContent)
	deleted := w.forgottenAt(t, w.parent, visits, subtree...)
	if got := visitsText(t, w.pool, others...); got != kept {
		t.Errorf("the visits of Web, Kept and Old after Parent's deletion:\n%s\nwere\n%s", got, kept)
	}
	return deleted
}

// forgetsADeletedIntakeItem is the intake's cell (spec 2.6, R6): bob
// submits Unaccepted and Accepted to Web's intake, alice and bob visit
// both, and alice accepts Accepted. alice's deletion of Unaccepted, an
// intake's work item not accepted, deletes it as a work item and every
// account's visits of it with it, at its moment, by her. Her deletion of
// Accepted, accepted, deletes its intake row alone: the work item stays,
// and the visits of it are as they were, byte for byte, both listing it. It
// answers the deleted visits as they are now.
func forgetsADeletedIntakeItem(t *testing.T, w visitsFollowWorld) []visitRow {
	unaccepted, accepted := w.submit(t, "bob", `{"name":"Unaccepted"}`).ID, w.submit(t, "bob", `{"name":"Accepted"}`).ID
	w.bothVisit(t, "issue", unaccepted)
	w.bothVisit(t, "issue", accepted)
	w.triage(t, "alice", accepted, `{"status":"accepted"}`, http.StatusOK)
	visits, stays := liveVisits(t, w.pool, w.both(), unaccepted), liveVisits(t, w.pool, w.both(), accepted)
	kept := visitsText(t, w.pool, stays...)
	w.send(t, "alice", http.MethodDelete, intakeItemPath(unaccepted), "", http.StatusNoContent)
	deleted := w.forgottenAt(t, unaccepted, visits, unaccepted)
	w.send(t, "alice", http.MethodDelete, intakeItemPath(accepted), "", http.StatusNoContent)
	if at, _ := w.intakeRowDeletion(t, accepted); at == nil {
		t.Fatal("Accepted's intake row is undeleted; want it deleted")
	}
	if got := visitsText(t, w.pool, stays...); got != kept {
		t.Errorf("the visits of Accepted after its intake row's deletion:\n%s\nwere\n%s", got, kept)
	}
	for _, actor := range []string{"alice", "bob"} {
		if listed := w.listedTargets(t, actor, "acme"); len(listed) == 0 || listed[0] != accepted {
			t.Errorf("%s lists %v; want Accepted at the top", actor, listed)
		}
	}
	return deleted
}

// keepsVisitsThroughArchives is the archives' cell (spec 2.6, R5, PE-17):
// alice archives Old, and archives Shelf and restores it; alice's and
// bob's visits of Old, Shelf and Shelved are as they were, byte for byte,
// after each, and both list them.
func keepsVisitsThroughArchives(t *testing.T, w visitsFollowWorld) {
	visits := liveVisits(t, w.pool, w.both(), w.old, w.shelf, w.shelved)
	kept := visitsText(t, w.pool, visits...)
	for _, path := range []string{"/api/v0/issues/" + w.old.String() + "/archive", "/api/v0/projects/" + w.shelf.String() + "/archive",
		"/api/v0/projects/" + w.shelf.String() + "/unarchive"} {
		w.send(t, "alice", http.MethodPost, path, "", http.StatusOK)
		if got := visitsText(t, w.pool, visits...); got != kept {
			t.Errorf("the visits of Old, Shelf and Shelved after POST %s:\n%s\nwere\n%s", path, got, kept)
		}
		for _, actor := range []string{"alice", "bob"} {
			if listed := w.listedTargets(t, actor, "acme"); !containsAll(listed, w.old, w.shelf, w.shelved) {
				t.Errorf("%s lists %v after POST %s; want Old, Shelf and Shelved among them", actor, listed, path)
			}
		}
	}
}

// deletesTheRestWithTheProject is the cell of a project deleted after some
// of its work items (spec 2.6, R10, re-92 I3 ⑤): alice deletes Web. Once
// the deletion job has run, every undeleted visit under Web, of Web itself
// and of Kept, Old and Accepted, alice's and bob's, is deleted at Web's
// deletion's moment, by alice, their visits kept; the visits the work
// items' deletions deleted keep those deletions, byte for byte. Neither
// lists any of Web's, and each one's recording of Web and Kept answers as
// an id that never was.
func deletesTheRestWithTheProject(t *testing.T, w visitsFollowWorld, forgotten []visitRow) {
	rest := visitRowsWhere(t, w.pool, "project_id = $1 AND deleted_at IS NULL", w.web)
	if len(rest) != 8 {
		t.Fatalf("the undeleted visits under Web: %v; want alice's and bob's of Web, Kept, Old and Accepted", rest)
	}
	before := visitsText(t, w.pool, forgotten...)
	w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+w.web.String(), "", http.StatusNoContent)
	waitNavigationDeletion(t, w.pool, "project", w.web)
	at, by := stampOf(t, w.pool, "projects", "deleted_at", w.web)
	if by != w.ids["alice"] {
		t.Fatalf("Web's deletion by %s; want alice's", by)
	}
	visitsDeletedAt(t, w.pool, at, by, rest...)
	if got := visitsText(t, w.pool, forgotten...); got != before {
		t.Errorf("the visits the work items' deletions deleted, after Web's:\n%s\nwere\n%s", got, before)
	}
	for _, actor := range []string{"alice", "bob"} {
		if listed := w.listedTargets(t, actor, "acme"); containsAny(listed, w.web, w.kept, w.old) {
			t.Errorf("%s lists %v; want none of Web's", actor, listed)
		}
		w.visitAnsweredAsAbsent(t, actor, "project", w.web)
		w.visitAnsweredAsAbsent(t, actor, "issue", w.kept)
	}
}
