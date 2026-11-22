package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"strconv"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// recentVisitsWorld is TestRecentVisitsThroughTheApp's world: issueWorld's
// acme and Web, in an app whose deletion jobs outlast the test's row locks
// (barriersOutlasted), with Fix, alice's work item of Web in Todo, assigned
// to bob (fullIssue). Vault, acme's private project, is alice's alone, with
// her work item Secret. Elsewhere is bob's own workspace, with his project
// Side. Besides issueWorld's accounts: adam, acme's admin who is Web's
// guest (adminGuest's case); gus, acme's guest who is Web's guest; and
// carl, acme's and Web's member, with his own workspace Carl's and its
// project Own. Each is made through the API, but for the memberships of
// acme (joinAcme).
type recentVisitsWorld struct {
	issueWorld
	fix, vault, secret, side, own uuid.UUID
}

func newRecentVisitsWorld(t *testing.T) recentVisitsWorld {
	t.Helper()
	w := recentVisitsWorld{issueWorld: newIssueWorldWith(t, barriersOutlasted)}
	w.fix = w.newWorkItem(t)
	w.vault = createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Vault","identifier":"VLT","network":0}`)
	w.secret = w.newWorkItemIn(t, w.vault, `{"name":"Secret"}`)
	w.send(t, "bob", http.MethodPost, "/api/v0/workspaces", `{"name":"Elsewhere","slug":"elsewhere"}`, http.StatusCreated)
	w.side = createdProject(t, w.contract, w.base, w.tokens["bob"], "elsewhere", "Side", "SID")
	w.joinAcme(t, "adam", shared.RoleMember)
	w.makeMember(t, "adam", w.web, shared.RoleGuest)
	w.promoteInAcme(t, "adam")
	w.joinAcme(t, "gus", shared.RoleGuest)
	w.makeMember(t, "gus", w.web, shared.RoleGuest)
	w.joinAcme(t, "carl", shared.RoleMember)
	w.makeMember(t, "carl", w.web, shared.RoleMember)
	w.send(t, "carl", http.MethodPost, "/api/v0/workspaces", `{"name":"Carl's","slug":"carls"}`, http.StatusCreated)
	w.own = createdProject(t, w.contract, w.base, w.tokens["carl"], "carls", "Own", "OWN")
	return w
}

// One's recent visits on the real app (H17, H19's project-level cells, M7
// P3d spec 2.2–2.5, 2.7, 4, R5–R8), each cell a subtest in order on one
// world (newRecentVisitsWorld): bob's visits of Web and Fix, recorded in
// the first, are the ones the later cells revisit, list, change and hide,
// so the test ends unless the first two pass. Seeded so: alice's and bob's
// visits of the same targets, and bob's of Side in his second workspace
// (listsHisOwnHere); two of alice's visits stamped at one moment by SQL
// (listsVisitTiesByID); adam, acme's admin who is Web's guest, and gus,
// acme's guest who is Web's guest, while Web shows its guests everything
// and while it does not (recordsAsTheGuestRuleSays); an archived work item,
// an archived project and its work item (listsTheArchived); three of Web's
// intake items, pending, accepted and declined
// (keepsTriageOutUntilAccepted); carl's visit made first but not visited
// first, the pending one hidden among his 20, and his visit in his own
// workspace (capsAtTwenty); the projects of the deletion window with their
// views turned on and one view each, which the window holds, after the
// start-time runs (visitsDeletionWindow).
func TestRecentVisitsThroughTheApp(t *testing.T) {
	w := newRecentVisitsWorld(t)
	var bobs []visitRow
	var pending uuid.UUID
	if !t.Run("each kind", func(t *testing.T) { bobs = recordsEachKind(t, w) }) || !t.Run("revisit", func(t *testing.T) { bobs = revisits(t, w, bobs) }) {
		t.FailNow()
	}
	// The filter's cell (spec 2.4, R8): bob's list of the projects' visits
	// answers his visit of Web alone, of the work items' Fix's alone, of
	// either both.
	t.Run("filter", func(t *testing.T) {
		w.listsVisits(t, "bob", "acme", "?entity_type=project", bobs[0])
		w.listsVisits(t, "bob", "acme", "?entity_type=issue", bobs[1])
		w.listsVisits(t, "bob", "acme", "", bobs...)
	})
	t.Run("his own, here", func(t *testing.T) { listsHisOwnHere(t, w, bobs) })
	t.Run("list order", func(t *testing.T) { listsVisitTiesByID(t, w) })
	t.Run("summaries now", func(t *testing.T) { showsTargetsAsTheyAreNow(t, w, bobs) })
	t.Run("reads write nothing", func(t *testing.T) { readsWriteNoVisit(t, w) })
	t.Run("refusal order", func(t *testing.T) { refusesInOrder(t, w) })
	t.Run("one 404", func(t *testing.T) { answersOne404(t, w) })
	t.Run("guest rule", func(t *testing.T) { recordsAsTheGuestRuleSays(t, w) })
	t.Run("leaving", func(t *testing.T) { followsMembership(t, w) })
	t.Run("archived", func(t *testing.T) { listsTheArchived(t, w) })
	t.Run("triage", func(t *testing.T) { pending = keepsTriageOutUntilAccepted(t, w) })
	t.Run("cap", func(t *testing.T) { capsAtTwenty(t, w, pending) })
	t.Run("deletion window", func(t *testing.T) { visitsDeletionWindow(t, w) })
}

// recordsEachKind is the two kinds' cell (spec 2.3–2.5, R7): bob records
// his visit of Web and of Fix. Each answers 204 and writes one row, his, in
// acme and Web, which the server finds from the target, made and last
// changed by him, visited, made and changed at one moment within the
// request. His list answers them, the last first, each with its target now.
// It returns them, Web's first.
func recordsEachKind(t *testing.T, w recentVisitsWorld) []visitRow {
	bob := w.ids["bob"]
	var rows []visitRow
	for _, target := range []struct {
		kind string
		id   uuid.UUID
	}{{"project", w.web}, {"issue", w.fix}} {
		from := time.Now().UTC().Truncate(time.Microsecond)
		w.visit(t, "bob", target.kind, target.id)
		to := time.Now()
		all := visitRowsWhere(t, w.pool, "user_id = $1 AND entity_identifier = $2", bob, target.id)
		if len(all) != 1 {
			t.Fatalf("bob's rows of the %s %s: %v; want one", target.kind, target.id, all)
		}
		r := all[0]
		if r.workspaceID != w.acme || r.projectID != w.web || r.userID != bob || r.kind != target.kind || r.createdBy == nil || *r.createdBy != bob ||
			r.updatedBy == nil || *r.updatedBy != bob || !r.createdAt.Equal(r.visitedAt) || !r.updatedAt.Equal(r.visitedAt) || r.visitedAt.Before(from) ||
			r.visitedAt.After(to) || r.deletedAt != nil {
			t.Errorf("bob's visit of the %s: %s; want his in acme/Web, made and changed by him at once within [%s, %s]", target.kind, r, from, to)
		}
		rows = append(rows, r)
	}
	w.listsVisits(t, "bob", "acme", "", rows[1], rows[0])
	return rows
}

// revisits is the revisit's cell (spec 2.5, PE-13): bob records Web again,
// 204. His one row of it is kept, its id, its owner, its target and its
// creation as they were, its visit and its last change stamped at the new
// moment, by him; it moves to the top of his list. It returns his visits,
// Web's first, as they are now.
func revisits(t *testing.T, w recentVisitsWorld, bobs []visitRow) []visitRow {
	bob := w.ids["bob"]
	web, fix := bobs[0], bobs[1]
	w.visit(t, "bob", "project", w.web)
	all := visitRowsWhere(t, w.pool, "user_id = $1 AND entity_identifier = $2", bob, w.web)
	if len(all) != 1 {
		t.Fatalf("bob's rows of Web after a revisit: %v; want his one", all)
	}
	again := all[0]
	want := web
	want.visitedAt, want.updatedAt, want.updatedBy = again.visitedAt, again.visitedAt, &bob
	if !sameVisitRow(again, want) || !again.visitedAt.After(fix.visitedAt) {
		t.Errorf("bob's visit of Web revisited: %s; was %s, want it visited and changed by him at one moment past Fix's", again, web)
	}
	w.listsVisits(t, "bob", "acme", "", again, fix)
	return []visitRow{again, fix}
}

// listsHisOwnHere is the cell of whose and where (spec 2.5, re-92 I3 ①):
// alice records her visits of Fix, of Web and of Fix again, the targets of
// bob's; bob records his visit of Side in Elsewhere. Each lists his own
// alone, in the workspace listed alone: alice hers in acme, bob his in acme
// as before and his of Side alone in Elsewhere.
func listsHisOwnHere(t *testing.T, w recentVisitsWorld, bobs []visitRow) {
	alice, bob := w.ids["alice"], w.ids["bob"]
	w.visit(t, "alice", "issue", w.fix)
	w.visit(t, "alice", "project", w.web)
	w.visit(t, "alice", "issue", w.fix)
	w.visit(t, "bob", "project", w.side)
	w.listsVisits(t, "alice", "acme", "", liveVisit(t, w.pool, alice, w.fix), liveVisit(t, w.pool, alice, w.web))
	w.listsVisits(t, "bob", "acme", "", bobs...)
	w.listsVisits(t, "bob", "elsewhere", "", liveVisit(t, w.pool, bob, w.side))
}

// listsVisitTiesByID is the list order's cell (spec 2.4, R8): alice's visit of
// Web, her visit after Fix's, stamped by SQL at the moment of Fix's
// revisit, as two recordings at one moment would leave them (the app reads
// its clock once a request, so it makes none): of the two at one moment
// her list answers Web's first, its id the greater, then Fix's.
func listsVisitTiesByID(t *testing.T, w recentVisitsWorld) {
	alice := w.ids["alice"]
	fix, web := liveVisit(t, w.pool, alice, w.fix), liveVisit(t, w.pool, alice, w.web)
	if web.id.Compare(fix.id) <= 0 || !fix.visitedAt.After(web.visitedAt) {
		t.Fatalf("alice's visits of Web %s and of Fix %s: want Web's the later id and Fix's the later visit", web, fix)
	}
	if tag, err := w.pool.Exec(pgtest.Soon(t), "UPDATE user_recent_visits SET visited_at = $1 WHERE id = $2", fix.visitedAt, web.id); err != nil ||
		tag.RowsAffected() != 1 {
		t.Fatalf("stamping alice's visit of Web at Fix's moment: %d rows, %v", tag.RowsAffected(), err)
	}
	web.visitedAt = fix.visitedAt
	w.listsVisits(t, "alice", "acme", "", web, fix)
}

// showsTargetsAsTheyAreNow is the cell of the summaries (spec 2.4, 2.5):
// alice renames Fix, gives it another priority and state, and assigns it to
// herself in bob's place, whose assignee row is deleted; and renames Web,
// with another identifier and an icon. bob lists both as they are now,
// value for value, his assignment no longer among Fix's assignees.
func showsTargetsAsTheyAreNow(t *testing.T, w recentVisitsWorld, bobs []visitRow) {
	alice := w.ids["alice"]
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+w.fix.String(), `{"name":"Fixed","priority":"urgent","state_id":"`+w.done.String()+
		`","assignee_ids":["`+alice.String()+`"]}`, http.StatusOK)
	w.changeProject(t, w.web, `{"name":"Web App","identifier":"WEBAPP","logo_props":`+webLogo+`}`)
	if gone := queryIDs(t, w.pool, "SELECT id FROM issue_assignees WHERE issue_id = $1 AND assignee_id = $2 AND deleted_at IS NOT NULL", w.fix,
		w.ids["bob"]); len(gone) != 1 {
		t.Fatalf("bob's assignment to Fix deleted: %v; want one", gone)
	}
	w.listsVisits(t, "bob", "acme", "", bobs...)
	var logo map[string]any
	decodeAnswer(t, webLogo, &logo)
	project := recentVisitAnswerOwner{ID: w.web, Name: "Web App", Identifier: "WEBAPP", LogoProps: logo}
	listed := w.listedVisits(t, "bob", "acme", "")
	if len(listed) != 2 || !reflect.DeepEqual(listed[0].Project, project) || !reflect.DeepEqual(listed[1].Project, project) || listed[1].Issue == nil ||
		listed[1].Issue.Name != "Fixed" || listed[1].Issue.Priority != "urgent" || listed[1].Issue.StateID != w.done ||
		!slices.Equal(listed[1].Issue.AssigneeIDs, []uuid.UUID{alice}) {
		t.Errorf("bob lists %+v; want Web and Fix as alice changed them", listed)
	}
}

// readsWriteNoVisit is the cell of the reads (spec 2.5, D38): bob reads Fix
// by id and by its identifier, Web, acme's projects, his workspaces and his
// recent visits, of each kind and in Elsewhere; no row of recent visits is
// written or changed, column for column.
func readsWriteNoVisit(t *testing.T, w recentVisitsWorld) {
	before := visitTable(t, w.pool)
	identifier := w.projectNow(t, "bob", w.web).Identifier + "-" + strconv.Itoa(w.issueNow(t, "bob", w.fix).SequenceID)
	for _, path := range []string{"/api/v0/issues/" + w.fix.String(), "/api/v0/workspaces/acme/issues/" + identifier,
		"/api/v0/projects/" + w.web.String(), "/api/v0/workspaces/acme/projects", "/api/v0/workspaces", recentVisitsPath("acme"),
		recentVisitsPath("acme") + "?entity_type=issue", recentVisitsPath("acme") + "?entity_type=project", recentVisitsPath("elsewhere")} {
		w.send(t, "bob", http.MethodGet, path, "", http.StatusOK)
	}
	if after := visitTable(t, w.pool); after != before {
		t.Errorf("bob's reads wrote recent visits:\n%s\nwere\n%s", after, before)
	}
}

// refusesInOrder is the refusal order's crossed cells (spec 2.7, DP-19): a
// kind neither of the two is the one 422, invalid_format on entity_type,
// byte for byte, before anything is looked at: bob's and olga's, acme's
// stranger, recordings of no target and of Fix, and their lists of acme
// and of a workspace that is none. Without it, olga's list of acme is the
// 404 of a workspace that is none. Nothing is written.
func refusesInOrder(t *testing.T, w recentVisitsWorld) {
	before := visitTable(t, w.pool)
	bad := func(id uuid.UUID) string { return `{"entity_type":"page","entity_id":"` + id.String() + `"}` }
	want := w.wholeOffContract(t, "bob", http.MethodPost, recordVisitPath, bad(uuid.NewV7()))
	if want.status != http.StatusUnprocessableEntity || refusalOf(t, []byte(want.body), false) != "validation_failed entity_type invalid_format" {
		t.Fatalf("bob's recording of a page that is none: %+v; want 422 invalid_format on entity_type", want)
	}
	for _, c := range []struct{ actor, method, path, body string }{
		{"bob", http.MethodPost, recordVisitPath, bad(w.fix)}, {"olga", http.MethodPost, recordVisitPath, bad(uuid.NewV7())},
		{"olga", http.MethodPost, recordVisitPath, bad(w.fix)}, {"bob", http.MethodGet, recentVisitsPath("acme") + "?entity_type=page", ""},
		{"olga", http.MethodGet, recentVisitsPath("acme") + "?entity_type=page", ""}, {"bob", http.MethodGet, recentVisitsPath("absent") + "?entity_type=page", ""},
	} {
		if got := w.wholeOffContract(t, c.actor, c.method, c.path, c.body); got != want {
			t.Errorf("%s %s %s as %s:\n%+v\nwant\n%+v", c.method, c.path, c.body, c.actor, got, want)
		}
	}
	w.answeredAsAbsent(t, "olga", http.MethodGet, recentVisitsPath("acme"), recentVisitsPath("absent"), "", "workspace.not_found")
	if after := visitTable(t, w.pool); after != before {
		t.Errorf("the refusals wrote recent visits:\n%s\nwere\n%s", after, before)
	}
}
