package bootstrap

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	navigationdomain "github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// visitRaceWorld is TestRecentVisitWritesInterleave's world: issueWorld's
// acme and Web on an observed app (newIssueWorldOnRequests), whose request
// probes attribute each lock wait to its request's own backend; Ops,
// alice's project of acme; seen, Seen 0 to Seen 20, alice's work items of
// Web. Besides bob, Web's member, walt, acme's member, is Web's member too,
// and wanda, acme's admin, Web's admin; erin and finn, acme's members
// registered here (joinAcme), are Ops' members. Each is made through the API but for the memberships of acme.
type visitRaceWorld struct {
	issueWorld
	ops  uuid.UUID
	seen []uuid.UUID
}

// A recording of one's recent visit, recordRecentVisit, interleaved with
// the writes that decide its target, with another of his recordings, and
// with the writes that hold one of his visits (M7 P3d spec 2.5, 2.6, 2.8,
// 4; re-92 I2, I4, m8), on the app as it runs, each write through its API
// but for the test's own transactions:
//   - A work item's deletion, its parent's, a project's deletion, a
//     workspace's deletion, a member's removal and a project's archive,
//     each with a recording of its target, in both orders
//     (visitsRaceATarget): both writes wait for the row the other
//     transaction holds, the target's project's or workspace's, the first
//     proven waiting at its statement and the second behind it, by each
//     one's request probe (issueWorld.race, P3c's construction).
//   - Two recordings of his at once, of two new targets past the cap, and
//     of one target twice (recordsTogether), each proven waiting for the
//     advisory lock of his visits a transaction of the test holds.
//   - A recording whose trim meets a visit of his past the cap that a
//     transaction of the test holds, soft-deleting it as a work item's
//     deletion does, which then commits or rolls back
//     (trimsPastAHeldVisit).
//   - A recording whose trim meets the visits a subtree's deletion holds,
//     stopped between the work items' cascades by a notification held
//     (trimsBesideASubtreesDeletion).
//
// One world (visitRaceWorld): each cell makes its own target, or records
// as its own account, so that none sees another's; once the start-time
// runs have run, so that none queues on a row a cell holds (03 §5); each
// write that holds or waits for a row is sent once no fan-out is pending
// (waitForFanouts), and each deletion's jobs are waited for, so that none
// takes the app's four connections from the next cell's writes. Each group
// is a subtest, and each cell one of it.
func TestRecentVisitWritesInterleave(t *testing.T) {
	dbURL := pgtest.NewDatabase(t)
	w := visitRaceWorld{issueWorld: newIssueWorldOnRequests(t, dbURL, openPool(t, dbURL), func(*config.Config) {}, &requestQueries{})}
	w.makeMember(t, "walt", w.web, shared.RoleMember)
	w.makeMember(t, "wanda", w.web, shared.RoleAdmin)
	w.ops = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	for _, name := range []string{"erin", "finn"} {
		w.joinAcme(t, name, shared.RoleMember)
		w.makeMember(t, name, w.ops, shared.RoleMember)
	}
	for n := range navigationdomain.RecentVisitCap + 1 {
		w.seen = append(w.seen, w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Seen %d"}`, n)))
	}
	waitForStartupRuns(t, w.pool)
	for _, c := range visitCompetitors(w) {
		t.Run(c.name, func(t *testing.T) { visitsRaceATarget(t, w, c) })
	}
	t.Run("two recordings at once", func(t *testing.T) { recordsTogether(t, w) })
	t.Run("the trim and a held visit", func(t *testing.T) { trimsPastAHeldVisit(t, w) })
	t.Run("the trim and a subtree's deletion", func(t *testing.T) { trimsBesideASubtreesDeletion(t, w) })
}

// visitFate is what becomes of the visit a recording wrote first, once the
// write that decides its target has committed.
type visitFate int

const (
	// forgottenWithIt: the deletion's own transaction deletes every
	// account's visits of the work item, at its moment, by its deleter (the
	// cascade of issue.deleted, spec 2.6).
	forgottenWithIt visitFate = iota
	// forgottenByItsJob: the deletion job's third step deletes the visits
	// under the project, at the deletion's moment, by its deleter (spec
	// 2.6, R10).
	forgottenByItsJob
	// keptUnlisted: the visit stays, and the list no longer shows it (a
	// member's removal, spec 2.5).
	keptUnlisted
	// keptListed: the visit stays, and the list shows it (an archive, R5,
	// PE-17).
	keptListed
)

// visitRace is one cell's: the target recorded and its project; the row
// the other transaction holds, the project's or the workspace's, by id; the
// rows the recording holds no lock on while it waits (free); and the write
// raced with the recording.
type visitRace struct {
	target, project, held uuid.UUID
	free                  []lockedRow
	competitor            favoriteRacer
}

// visitCompetitor is a write that decides a recording's target, raced with
// recorder's recording of his visit of a target of kind: make makes the
// cell n's own target and its race. Both writes wait for the row of table
// the other transaction holds FOR NO KEY UPDATE, the recording at waits,
// and the competitor answers answers. For the fates that delete the visit,
// stamp waits for the competitor's jobs and reads the deletion's moment and
// deleter, and left reads the undeleted visits the deletion was to leave
// none of: of the work item, under the project or under the workspace.
type visitCompetitor struct {
	name, recorder, kind, table, waits string
	answers                            int
	fate                               visitFate
	make                               func(t *testing.T, n int) visitRace
	stamp                              func(t *testing.T, r visitRace) (time.Time, uuid.UUID)
	left                               func(t *testing.T, r visitRace) []visitRow
}

// visitCompetitors are, each alice's through the API:
//   - The deletion of a work item of Web, and of the parent of one, which
//     takes its child with it: bob's recording of the work item, waiting
//     for Web's row, which the deletion locks FOR NO KEY UPDATE
//     (LockIssueProject) for its whole transaction; its cascade deletes
//     the visit, the child's in its own issue.deleted.
//   - The deletion of a project of acme, bob's member: his recording of
//     the project, waiting for its row (LockProject); the deletion's job
//     deletes the visit.
//   - The deletion of a workspace of hers, Doomed n: her recording of a
//     work item of its project Gone, waiting for the workspace's row
//     (LockWorkspaceBySlug), which the recording shares first
//     (ShareDirectoryWorkspaceByID); the jobs of the deletion of Gone
//     delete the visit.
//   - The removal of bob from a project of acme, and its archive: his
//     recording of the project, waiting for its row (LockProject); the
//     visit stays, unlisted once he has left, listed when archived.
func visitCompetitors(w visitRaceWorld) []visitCompetitor {
	bobs := func(t *testing.T, name, identifier string) uuid.UUID {
		p := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, identifier)
		w.makeMember(t, "bob", p, shared.RoleMember)
		return p
	}
	alices := func(method, path, statement string) favoriteRacer {
		return favoriteRacer{actor: "alice", method: method, path: path, statement: statement}
	}
	row := func(table string, id uuid.UUID) lockedRow { return lockedRow{table + " WHERE id = $1", id} }
	left := func(column string, id func(visitRace) uuid.UUID) func(t *testing.T, r visitRace) []visitRow {
		return func(t *testing.T, r visitRace) []visitRow {
			return visitRowsWhere(t, w.pool, column+" = $1 AND deleted_at IS NULL", id(r))
		}
	}
	itemDeleted := func(t *testing.T, r visitRace) (time.Time, uuid.UUID) {
		return stampOf(t, w.pool, "issues", "deleted_at", r.target)
	}
	projectDeleted := func(t *testing.T, r visitRace) (time.Time, uuid.UUID) {
		waitProjectDeletion(t, w.pool, r.project)
		return stampOf(t, w.pool, "projects", "deleted_at", r.project)
	}
	target, project, held := func(r visitRace) uuid.UUID { return r.target }, func(r visitRace) uuid.UUID { return r.project },
		func(r visitRace) uuid.UUID { return r.held }
	return []visitCompetitor{
		{name: "a work item's deletion", recorder: "bob", kind: "issue", table: "projects", waits: "ShareNavigationProject",
			answers: http.StatusNoContent, fate: forgottenWithIt, stamp: itemDeleted, left: left("entity_identifier", target),
			make: func(t *testing.T, n int) visitRace {
				item := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Deleted %d"}`, n))
				return visitRace{target: item, project: w.web, held: w.web, free: []lockedRow{row("issues", item)},
					competitor: alices(http.MethodDelete, "/api/v0/issues/"+item.String(), "LockIssueProject")}
			}},
		{name: "its parent's deletion", recorder: "bob", kind: "issue", table: "projects", waits: "ShareNavigationProject",
			answers: http.StatusNoContent, fate: forgottenWithIt, stamp: itemDeleted, left: left("entity_identifier", target),
			make: func(t *testing.T, n int) visitRace {
				parent := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Parent %d"}`, n))
				child := w.newWorkItemIn(t, w.web, fmt.Sprintf(`{"name":"Child %d","parent_id":%q}`, n, parent))
				return visitRace{target: child, project: w.web, held: w.web, free: []lockedRow{row("issues", parent), row("issues", child)},
					competitor: alices(http.MethodDelete, "/api/v0/issues/"+parent.String(), "LockIssueProject")}
			}},
		{name: "a project's deletion", recorder: "bob", kind: "project", table: "projects", waits: "ShareNavigationProject",
			answers: http.StatusNoContent, fate: forgottenByItsJob, stamp: projectDeleted, left: left("project_id", project),
			make: func(t *testing.T, n int) visitRace {
				p := bobs(t, fmt.Sprintf("Deleted %d", n), fmt.Sprintf("DEL%d", n))
				return visitRace{target: p, project: p, held: p, competitor: alices(http.MethodDelete, "/api/v0/projects/"+p.String(), "LockProject")}
			}},
		{name: "a workspace's deletion", recorder: "alice", kind: "issue", table: "workspaces", waits: "ShareDirectoryWorkspaceByID",
			answers: http.StatusNoContent, fate: forgottenByItsJob, left: left("workspace_id", held),
			stamp: func(t *testing.T, r visitRace) (time.Time, uuid.UUID) {
				waitWorkspaceDeletion(t, w.pool, r.held)
				return stampOf(t, w.pool, "projects", "deleted_at", r.project)
			},
			make: func(t *testing.T, n int) visitRace {
				slug := fmt.Sprintf("doomed-%d", n)
				var doomed struct {
					ID uuid.UUID `json:"id"`
				}
				decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", fmt.Sprintf(`{"name":"Doomed %d","slug":%q}`, n, slug),
					http.StatusCreated), &doomed)
				gone := createdProject(t, w.contract, w.base, w.tokens["alice"], slug, "Gone", "GNE")
				item := w.newWorkItemIn(t, gone, `{"name":"Gone's"}`)
				return visitRace{target: item, project: gone, held: doomed.ID, free: []lockedRow{row("projects", gone), row("issues", item)},
					competitor: alices(http.MethodDelete, "/api/v0/workspaces/"+slug, "LockWorkspaceBySlug")}
			}},
		{name: "a member's removal", recorder: "bob", kind: "project", table: "projects", waits: "ShareNavigationProject",
			answers: http.StatusNoContent, fate: keptUnlisted,
			make: func(t *testing.T, n int) visitRace {
				p := bobs(t, fmt.Sprintf("Left %d", n), fmt.Sprintf("LFT%d", n))
				membership := projectMemberships(t, w.pool, w.ids["bob"], p)[0]
				return visitRace{target: p, project: p, held: p,
					competitor: alices(http.MethodDelete, "/api/v0/project-members/"+membership.String(), "LockProject")}
			}},
		{name: "a project's archive", recorder: "bob", kind: "project", table: "projects", waits: "ShareNavigationProject",
			answers: http.StatusOK, fate: keptListed,
			make: func(t *testing.T, n int) visitRace {
				p := bobs(t, fmt.Sprintf("Archived %d", n), fmt.Sprintf("ARC%d", n))
				return visitRace{target: p, project: p, held: p, competitor: alices(http.MethodPost, "/api/v0/projects/"+p.String()+"/archive", "LockProject")}
			}},
	}
}

// visitsRaceATarget races c with its recorder's recording of a target of
// c's, in both orders, each cell on a target of its own.
func visitsRaceATarget(t *testing.T, w visitRaceWorld, c visitCompetitor) {
	for n, ours := range []bool{false, true} {
		first := "the recording second"
		if ours {
			first = "the recording first"
		}
		t.Run(first, func(t *testing.T) { raceAVisit(t, w, c, n+1, ours) })
	}
}

// raceAVisit is one cell of c's: its recorder's recording of the cell's
// target, first when ours is set, else second, each write waiting for the
// row the other transaction holds at its statement, the recording holding
// no lock on its free rows meanwhile (spec 2.8: the project's FOR SHARE
// stands for the work item's row). Neither answers 500.
//
// c's write answers as it would alone. Second, the recording decides under
// its locks on what c's left (spec 2.7): a target deleted, or one its
// recorder no longer sees, is its 404, byte for byte as an id that never
// was, and it writes no visit; an archived one it records, 204 (R5).
// First, it records, 204, one visit of the target, the recorder's, made by
// him at its visit. Then: when c deletes the target or its project, the
// visit is deleted at c's moment, by its deleter, after its visit, by c's
// transaction or its jobs; when c keeps it, it is live as recorded, last
// changed by its recorder at its visit, and listed unless c removed him.
// Whichever came first, no visit of a deleted work item, under a deleted
// project or under a deleted workspace is left undeleted once c's jobs
// have run.
func raceAVisit(t *testing.T, w visitRaceWorld, c visitCompetitor, n int, ours bool) {
	r := c.make(t, n)
	recording := favoriteRacer{actor: c.recorder, method: http.MethodPost, path: recordVisitPath, body: visitBody(c.kind, r.target),
		statement: c.waits, free: r.free}
	var mine, theirs racedAnswer
	if ours {
		mine, theirs = w.race(t, c.table, "FOR NO KEY UPDATE", r.held, recording, r.competitor)
	} else {
		theirs, mine = w.race(t, c.table, "FOR NO KEY UPDATE", r.held, r.competitor, recording)
	}
	answeredRace(t, theirs, c.answers)
	var at time.Time
	var by uuid.UUID
	if c.stamp != nil {
		at, by = c.stamp(t, r)
	}
	owner := w.ids[c.recorder]
	rows := visitRowsWhere(t, w.pool, "user_id = $1 AND entity_identifier = $2", owner, r.target)
	switch {
	case !ours && c.fate != keptListed:
		w.refusedVisitAsAbsent(t, mine, c.recorder, c.kind)
		if len(rows) != 0 {
			t.Errorf("%s's recording refused under its locks left %v; want no visit written", c.recorder, rows)
		}
	case !recordedOnce(t, mine, rows, owner, r.project, c.kind):
	case c.fate == forgottenWithIt || c.fate == forgottenByItsJob:
		if v := rows[0]; v.deletedAt == nil || !v.deletedAt.Equal(at) || !v.updatedAt.Equal(at) || v.updatedBy == nil || *v.updatedBy != by ||
			v.visitedAt.After(at) {
			t.Errorf("%s's visit recorded first: %s; want it deleted and last changed by %s at %s, after its visit", c.recorder, v, by,
				at.UTC().Format(time.RFC3339Nano))
		}
	default:
		if v := rows[0]; v.deletedAt != nil || !v.updatedAt.Equal(v.visitedAt) || v.updatedBy == nil || *v.updatedBy != owner {
			t.Errorf("%s's visit: %s; want it undeleted, as recorded", c.recorder, v)
		}
		if listed := slices.Contains(w.listedTargets(t, c.recorder, "acme"), r.target); listed != (c.fate == keptListed) {
			t.Errorf("%s lists his visit of %s: %t; want %t", c.recorder, r.target, listed, c.fate == keptListed)
		}
	}
	if c.left != nil {
		if left := c.left(t, r); len(left) != 0 {
			t.Errorf("undeleted visits once the deletion and its jobs are done: %v; want none", left)
		}
	}
}

// recordedOnce checks that a, owner's recording of his visit of a target
// of kind in project, answered 204, and that rows, his visits of it,
// deleted or not, are one visit, his, of that target, made by him at its
// visit: whether they are.
func recordedOnce(t *testing.T, a racedAnswer, rows []visitRow, owner, project uuid.UUID, kind string) bool {
	t.Helper()
	answeredRace(t, a, http.StatusNoContent)
	if len(rows) != 1 || rows[0].userID != owner || rows[0].projectID != project || rows[0].kind != kind || rows[0].createdBy == nil ||
		*rows[0].createdBy != owner || !rows[0].createdAt.Equal(rows[0].visitedAt) {
		t.Errorf("%s's visits once recorded: %v; want one, his, of a %s in %s, made by him at its visit", owner, rows, kind, project)
		return false
	}
	return true
}

// refusedVisitAsAbsent checks that a, actor's recording of his visit of a
// target of kind, answered its 404, issue.not_found or project.not_found,
// byte for byte as a recording of an id no target of kind ever had (spec
// 2.7).
func (w visitRaceWorld) refusedVisitAsAbsent(t *testing.T, a racedAnswer, actor, kind string) {
	t.Helper()
	code := map[string]string{"issue": "issue.not_found", "project": "project.not_found"}[kind]
	absent := w.whole(t, actor, http.MethodPost, recordVisitPath, visitBody(kind, uuid.NewV7()))
	if absent.status != http.StatusNotFound || problemCode(t, []byte(absent.body)) != code {
		t.Fatalf("%s's recording of an id no %s ever had: %+v; want 404 %s", actor, kind, absent, code)
	}
	if a.whole != absent {
		t.Errorf("%s %s as %s:\n%+v\nwant, as an absent target's,\n%+v", a.req.Method, a.req.URL.Path, actor, a.whole, absent)
	}
}
