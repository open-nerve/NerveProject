package bootstrap

import (
	"net/http"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// answersOne404 is the cell of the one 404 (spec 2.2, 2.5, 2.7, R7): each
// recording of a target one may not record answers its own 404, the same
// answer byte for byte as of an id that never was, and writes nothing (the
// table compared whole): olga's of Web and Fix, of a workspace she is no
// member of; bob's of Vault and Secret, a private project he is none of and
// its work item; walt's of Web, a public project he sees (getProject
// answers him 200) and is none of, and of Fix; and wanda's of Web and Fix,
// acme's admin who is none of Web's members (Readable). A guest's of
// another's work item is recordsAsTheGuestRuleSays'.
func answersOne404(t *testing.T, w recentVisitsWorld) {
	w.send(t, "walt", http.MethodGet, "/api/v0/projects/"+w.web.String(), "", http.StatusOK)
	for _, c := range []struct {
		actor, kind string
		id          uuid.UUID
	}{
		{"olga", "project", w.web}, {"olga", "issue", w.fix}, {"bob", "project", w.vault}, {"bob", "issue", w.secret},
		{"walt", "project", w.web}, {"walt", "issue", w.fix}, {"wanda", "project", w.web}, {"wanda", "issue", w.fix},
	} {
		w.visitAnsweredAsAbsent(t, c.actor, c.kind, c.id)
	}
}

// recordsAsTheGuestRuleSays is H19's project-level cells and the workspace
// guest's (spec 2.2, 2.5, design 3.10), which the matrix of workspace
// columns cannot hold. While Web does not show its guests everything, as a
// project starts: adam, acme's admin who is Web's guest, records Web and
// his own work item Adam's; gus, acme's guest who is Web's guest, records
// Web; each lists his own; and each is refused Fix, alice's, as an id that
// never was, gus Adam's and Vault too, a workspace guest answered 404, not
// 403. Shown everything, each records Fix, gus Adam's, and lists them. No
// longer shown it, neither lists another's work item, the rows kept
// undeleted, and each is refused Fix again.
func recordsAsTheGuestRuleSays(t *testing.T, w recentVisitsWorld) {
	adam, gus := w.ids["adam"], w.ids["gus"]
	adams := w.newWorkItemBy(t, "adam", w.web, `{"name":"Adam's"}`)
	w.visit(t, "adam", "project", w.web)
	w.visit(t, "adam", "issue", adams)
	w.visit(t, "gus", "project", w.web)
	for _, c := range []struct {
		actor, kind string
		id          uuid.UUID
	}{{"adam", "issue", w.fix}, {"gus", "issue", w.fix}, {"gus", "issue", adams}, {"gus", "project", w.vault}} {
		w.visitAnsweredAsAbsent(t, c.actor, c.kind, c.id)
	}
	adamsOwn := []visitRow{liveVisit(t, w.pool, adam, adams), liveVisit(t, w.pool, adam, w.web)}
	gusOwn := liveVisit(t, w.pool, gus, w.web)
	w.listsVisits(t, "adam", "acme", "", adamsOwn...)
	w.listsVisits(t, "gus", "acme", "", gusOwn)

	w.changeProject(t, w.web, `{"guest_view_all_features":true}`)
	w.visit(t, "adam", "issue", w.fix)
	w.visit(t, "gus", "issue", w.fix)
	w.visit(t, "gus", "issue", adams)
	others := []visitRow{liveVisit(t, w.pool, adam, w.fix), liveVisit(t, w.pool, gus, w.fix), liveVisit(t, w.pool, gus, adams)}
	kept := visitsText(t, w.pool, others...)
	w.listsVisits(t, "adam", "acme", "", others[0], adamsOwn[0], adamsOwn[1])
	w.listsVisits(t, "gus", "acme", "", others[2], others[1], gusOwn)

	w.changeProject(t, w.web, `{"guest_view_all_features":false}`)
	w.listsVisits(t, "adam", "acme", "", adamsOwn...)
	w.listsVisits(t, "gus", "acme", "", gusOwn)
	if got := visitsText(t, w.pool, others...); got != kept {
		t.Errorf("the guests' visits of others' work items, no longer shown them:\n%s\nwere\n%s", got, kept)
	}
	w.visitAnsweredAsAbsent(t, "adam", "issue", w.fix)
	w.visitAnsweredAsAbsent(t, "gus", "issue", w.fix)
}

// followsMembership is the cell of leaving and joining again (spec 2.2,
// 2.5): bob leaves Web; he lists none of his visits of Web and Fix, their
// rows kept as they were, byte for byte, and his recordings of them answer
// as ids that never were; he joins Web again, and lists them as before.
func followsMembership(t *testing.T, w recentVisitsWorld) {
	rows := liveVisits(t, w.pool, []uuid.UUID{w.ids["bob"]}, w.web, w.fix)
	kept := visitsText(t, w.pool, rows...)
	before := w.listedVisits(t, "bob", "acme", "")
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/leave", "", http.StatusNoContent)
	if listed := w.listedTargets(t, "bob", "acme"); containsAny(listed, w.web, w.fix) {
		t.Errorf("bob lists %v out of Web; want neither Web nor Fix", listed)
	}
	if got := visitsText(t, w.pool, rows...); got != kept {
		t.Errorf("bob's visits of Web and Fix, out of Web:\n%s\nwere\n%s", got, kept)
	}
	w.visitAnsweredAsAbsent(t, "bob", "project", w.web)
	w.visitAnsweredAsAbsent(t, "bob", "issue", w.fix)
	w.send(t, "bob", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/join", "", http.StatusOK)
	if got := w.listedVisits(t, "bob", "acme", ""); !reflect.DeepEqual(got, before) {
		t.Errorf("bob lists %+v in Web again; want %+v", got, before)
	}
}

// listsTheArchived is the archived targets' cell (spec 2.5, 2.6, R5): bob,
// a member of Shelf, records Shelf and its work item Shelved; alice
// archives Old, a work item of Web in Done, and Shelf, which leaves bob's
// visits as they were, byte for byte. bob records Old, archived, and Shelf
// and Shelved again, an archived project and a work item of it, each 204,
// and lists the three, each with its target now, above Web and Fix.
func listsTheArchived(t *testing.T, w recentVisitsWorld) {
	bob := w.ids["bob"]
	old := w.doneWorkItem(t)
	shelf := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Shelf", "SHF")
	w.makeMember(t, "bob", shelf, shared.RoleMember)
	shelved := w.newWorkItemIn(t, shelf, `{"name":"Shelved"}`)
	w.visit(t, "bob", "project", shelf)
	w.visit(t, "bob", "issue", shelved)
	visits := liveVisits(t, w.pool, []uuid.UUID{bob}, shelf, shelved)
	kept := visitsText(t, w.pool, visits...)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+old.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+shelf.String()+"/archive", "", http.StatusOK)
	if got := visitsText(t, w.pool, visits...); got != kept {
		t.Errorf("bob's visits of Shelf and Shelved, archived:\n%s\nwere\n%s", got, kept)
	}
	w.visit(t, "bob", "issue", old)
	w.visit(t, "bob", "project", shelf)
	w.visit(t, "bob", "issue", shelved)
	w.listsVisits(t, "bob", "acme", "", liveVisits(t, w.pool, []uuid.UUID{bob}, shelved, shelf, old, w.web, w.fix)...)
}

// keepsTriageOutUntilAccepted is the intake's cell (spec 2.5, R6): with
// Web's intake turned on, alice submits Pending, Accepted and Declined; bob
// records each, 204, and lists none. alice accepts Accepted, which he then
// lists at the top with its work item now, and declines Declined, which he
// does not; their rows are kept undeleted. It returns Pending.
func keepsTriageOutUntilAccepted(t *testing.T, w recentVisitsWorld) uuid.UUID {
	bob := w.ids["bob"]
	w.intakeOn(t)
	pending, accepted, declined := w.submit(t, "alice", `{"name":"Pending"}`).ID, w.submit(t, "alice", `{"name":"Accepted"}`).ID,
		w.submit(t, "alice", `{"name":"Declined"}`).ID
	before := w.listedVisits(t, "bob", "acme", "")
	for _, id := range []uuid.UUID{pending, accepted, declined} {
		w.visit(t, "bob", "issue", id)
	}
	if got := w.listedVisits(t, "bob", "acme", ""); !reflect.DeepEqual(got, before) {
		t.Errorf("bob lists %+v with his visits of three items in Triage; want %+v", got, before)
	}
	w.triage(t, "alice", accepted, `{"status":"accepted"}`, http.StatusOK)
	w.triage(t, "alice", declined, `{"status":"declined"}`, http.StatusOK)
	listed := w.listedVisits(t, "bob", "acme", "")
	if len(listed) != len(before)+1 || !reflect.DeepEqual(listed[0], w.shownVisit(t, "bob", liveVisit(t, w.pool, bob, accepted))) ||
		!reflect.DeepEqual(listed[1:], before) {
		t.Errorf("bob lists %+v once Accepted is accepted and Declined declined; want Accepted's above %+v", listed, before)
	}
	liveVisits(t, w.pool, []uuid.UUID{bob}, pending, declined)
	return pending
}
