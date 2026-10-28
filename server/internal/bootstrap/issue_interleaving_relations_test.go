package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The interleavings of relations across projects (M4 design 3.5, 3.6,
// 3.10, convention 7; 9.3, interleavings 10, 11, 23 and 26), on
// issueWorld's real database, as issue_interleaving_test.go runs them.

// nearAndFar makes Ops, of acme, by alice, with bob its member, then bob's
// Near in Web and alice's Far in Ops: the two ends of the interleavings'
// relations.
func nearAndFar(t *testing.T, w issueWorld) {
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	w.makeMember(t, "bob", ops, shared.RoleMember)
	w.newWorkItemBy(t, "bob", w.web, `{"name":"Near"}`)
	w.newWorkItemIn(t, ops, `{"name":"Far"}`)
}

// projectWith is the id of the world's one project of the identifier.
func (w issueWorld) projectWith(t *testing.T, identifier string) uuid.UUID {
	t.Helper()
	var id uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM projects WHERE identifier = $1", identifier).Scan(&id); err != nil {
		t.Fatal(err)
	}
	return id
}

// deletedAt is the moment the row id of table was deleted.
func (w issueWorld) deletedAt(t *testing.T, table string, id uuid.UUID) time.Time {
	t.Helper()
	var at time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at FROM "+table+" WHERE id = $1", id).Scan(&at); err != nil {
		t.Fatal(err)
	}
	return at
}

// relationRowOf is the id of the one row of the work item item's activity
// on its relations.
func (w issueWorld) relationRowOf(t *testing.T, item uuid.UUID) uuid.UUID {
	t.Helper()
	ids := queryIDs(t, w.pool, `SELECT id FROM issue_activities WHERE issue_id = $1
		AND field IN ('relates_to', 'duplicate', 'blocked_by', 'blocking')`, item)
	if len(ids) != 1 {
		t.Fatalf("the work item %s has %d rows of activity on its relations, want 1", item, len(ids))
	}
	return ids[0]
}

// unrelates is name's deletion of the world's one relation.
func unrelates(name string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRoute(t, w.pool, tx), name, http.MethodDelete,
			relationPath(queryIDs(t, w.pool, "SELECT id FROM issue_relations")[0]), "")
	}
}

// deletesOps is alice's deletion of Ops through the project module's
// route.
var deletesOps = aliceOnWeb(http.MethodDelete, func(t *testing.T, w issueWorld) string {
	return "/api/v0/projects/" + w.projectWith(t, "OPS").String()
}, "")

// Two relations of the same two work items, each the other way round,
// serialize on their projects' rows (M4 design 3.5, 3.10, convention 7;
// interleaving 10), in both orders: alice relates Near, of Web, to Far, of
// Ops, as blocking it, while bob relates Far to Near, as blocking it. Each
// locks Web and Ops at once, in ascending id order, so neither holds a row
// the other waits for while waiting for one of the other's: the second
// waits for the first's projects, then reads under its locks the relation
// the first made, either way, and is refused at its place, 422 duplicate,
// nothing written. The first's relation is the one stored, with its row
// of each work item's activity.
func TestOppositeRelationsOfTwoWorkItemsSerialize(t *testing.T) {
	inBothOrders(t, nearAndFar, "alice's", relates("alice", "Near", "Far", "blocking"), "bob's", relates("bob", "Far", "Near", "blocking"),
		func(t *testing.T, w issueWorld, aliceFirst bool, hers, his exchange) {
			first, second, stored := hers, his, "Far blocked_by Near, in OPS, by alice, alice at the moment"
			if !aliceFirst {
				first, second, stored = his, hers, "Near blocked_by Far, in WEB, by bob, bob at the moment"
			}
			made := listIn[answeredRelation](t, w.answered(t, first, http.StatusCreated))
			if answer := w.answered(t, second, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), false) !=
				"validation_failed issue_ids[0] duplicate" {
				t.Errorf("the second relation answers %s, want issue_ids[0] duplicate", answer)
			}
			if got := w.storedRelations(t, made[0].CreatedAt); !slices.Equal(got, []string{stored}) {
				t.Errorf("the relations stored:\n%q\nwant\n%q", got, []string{stored})
			}
			for _, item := range []string{"Near", "Far"} {
				w.relationRowOf(t, w.workItemNamed(t, item))
			}
		})
}

// The project request owns only its original P lock. Relation contention is
// between its committed deletion job and the ordinary synchronous issue write.
// Both orders retain the first deletion's precise audit stamp.
func TestDeletingARelatedWorkItemAndTheOtherEndsProjectSerialize(t *testing.T) {
	for _, nearFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "Near deleted first", false: "Ops deleted first"}[nearFirst], func(t *testing.T) {
			w := newIssueWorld(t)
			nearAndFar(t, w)
			near, ops := w.workItemNamed(t, "Near"), w.projectWith(t, "OPS")
			farToo := w.newWorkItemIn(t, ops, `{"name":"Far too"}`)
			w.send(t, "alice", http.MethodPost, relationsOf(near), relating("relates_to", w.workItemNamed(t, "Far")), 201)
			w.send(t, "alice", http.MethodPost, relationsOf(farToo), relating("duplicate", near), 201)
			relations := queryIDs(t, w.pool, "SELECT id FROM issue_relations ORDER BY id")
			if len(relations) != 2 {
				t.Fatal("relation fixture", relations)
			}
			nearProbe := requestTx(w.pool)
			g := newGate()
			var nearDeleted, opsDeleted exchange
			var nearDone <-chan error
			defer func() { releaseAndJoinGate(t, g, nearDone) }()
			if nearFirst {
				nearDone = run(deletesNamed("bob", "Near")(t, w, heldTx{nearProbe, g}, &nearDeleted))
				held(t, pgtest.Soon(t), g, nearDone, "Near deletion")
			}
			// The request still waits for P, then commits independently of issue rows.
			projectHolder := holding(t, w.pool, "SELECT 1 FROM projects WHERE id=$1 FOR NO KEY UPDATE", ops)
			projectProbe := requestTx(w.pool)
			collectProject := requestBackground(t, deletesOps(t, w, projectProbe, &opsDeleted), func() { requestRelease(t, projectHolder) })
			requestBlocked(t, w.pool, projectProbe, projectHolder.Conn().PgConn().PID(), "projects")
			// Worker-first: hold its second sorted relation so it owns the first while
			// the ordinary write arrives. The worker itself supplies the competing lock.
			var releaseRelation func()
			var secondPID uint32
			if !nearFirst {
				second := holding(t, w.pool, "SELECT 1 FROM issue_relations WHERE id=$1 FOR UPDATE", relations[1])
				secondPID = second.Conn().PgConn().PID()
				releaseRelation = func() { requestRelease(t, second) }
				defer releaseRelation()
			}
			requestRelease(t, projectHolder)
			collectProject()
			w.answered(t, opsDeleted, 204)
			w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+w.workItemNamed(t, "Far").String(), "", 404)
			job := projectDeletionJob(t, w.pool, ops)
			if nearFirst {
				deletionWorkerBlockedBy(t, w.pool, requestPID(t, nearProbe), job, "issue_relations")
				requestOpen(g)
				if err := result(t, pgtest.Soon(t), nearDone, "Near deletion"); err != nil {
					t.Fatal(err)
				}
				nearDone = nil
			} else {
				worker := deletionWorkerBlockedBy(t, w.pool, secondPID, job, "issue_relations")
				workerProbe := &requestProbe{}
				workerProbe.pid.Store(worker)
				if got := requestRowLock(t, w.pool, w.pool, workerProbe, "issue_relations WHERE id=$1", relations[0]); got != "FOR NO KEY UPDATE" {
					t.Fatal("worker must own first relation", got)
				}
				collectNear := requestBackground(t, deletesNamed("bob", "Near")(t, w, nearProbe, &nearDeleted), releaseRelation)
				requestBlocked(t, w.pool, nearProbe, worker, "issue_relations")
				releaseRelation()
				collectNear()
			}
			w.answered(t, nearDeleted, 204)
			waitDeletionJob(t, w.pool, job, "completed")
			at, by := w.deletedAt(t, "projects", ops), "alice"
			if nearFirst {
				at, by = w.deletedAt(t, "issues", near), "bob"
			}
			for _, id := range relations {
				if got := w.relationDeletion(t, id, at); got != "deleted by "+by+" at the moment" {
					t.Errorf("relation %s: %s want %s", id, got, by)
				}
			}
		})
	}
}

// Relating a work item to another and deleting the other's project
// serialize on the other's project's row (M4 design 3.5, 3.6, 3.10,
// convention 7; interleaving 23), in both orders: bob relates Near, of
// Web, to Far, of Ops, as blocking it, while alice deletes Ops. Deleted
// first: the relation, which passed its precheck with Far reached, waits
// for Ops' row, and under its locks finds Ops deleted: Far is not
// reached, 422 at its place, and nothing is written. Related first: the
// deletion waits for Ops' row, then its committed job deletes the relation,
// stored from Far, and Far's row of its activity at the request's moment,
// by alice; Near's row stays, and no undeleted relation names a
// deleted work item.
func TestRelatingAWorkItemAndDeletingTheOtherEndsProjectSerialize(t *testing.T) {
	var holder pgx.Tx
	inBothOrders(t, func(t *testing.T, w issueWorld) {
		nearAndFar(t, w)
		ops := w.projectWith(t, "OPS")
		createDraftHTTP(t, w, "alice", `{"project_id":"`+ops.String()+`","payload":{}}`)
		holder = holdProjectDeletion(t, w.pool, ops)
	}, "Ops deleted", deletesOps, "related", relates("bob", "Near", "Far", "blocking"),
		func(t *testing.T, w issueWorld, deletedFirst bool, deleted, related exchange) {
			w.answered(t, deleted, http.StatusNoContent)
			if deletedFirst {
				if answer := w.answered(t, related, http.StatusUnprocessableEntity); refusalOf(t, []byte(answer), false) !=
					"validation_failed issue_ids[0] not_allowed" {
					t.Errorf("the relation answers %s, want issue_ids[0] not_allowed", answer)
				}
				if n := len(queryIDs(t, w.pool, "SELECT id FROM issue_relations")); n != 0 || len(w.relationRows(t, w.workItemNamed(t, "Near"),
					time.Time{})) != 0 {
					t.Errorf("%d relations and Near's rows on them %q after the refusal; want none", n, w.relationRows(t, w.workItemNamed(t, "Near"),
						time.Time{}))
				}
				requestRelease(t, holder)
				waitProjectDeletion(t, w.pool, w.projectWith(t, "OPS"))
				return
			}
			w.send(t, "alice", http.MethodGet, "/api/v0/issues/"+w.workItemNamed(t, "Far").String(), "", 404)
			got := listIn[answeredRelation](t, w.send(t, "bob", http.MethodGet, relationsOf(w.workItemNamed(t, "Near")), "", 200))
			if len(got) != 0 {
				t.Fatal("live endpoint leaked hidden relation", got)
			}
			requestRelease(t, holder)
			waitProjectDeletion(t, w.pool, w.projectWith(t, "OPS"))
			made := listIn[answeredRelation](t, w.answered(t, related, http.StatusCreated))
			at := w.deletedAt(t, "projects", w.projectWith(t, "OPS"))
			for _, tt := range []struct {
				what, table string
				id          uuid.UUID
				want        string
			}{
				{"the relation", "issue_relations", made[0].ID, "deleted by alice at the moment"},
				{"Far's row on it", "issue_activities", w.relationRowOf(t, w.workItemNamed(t, "Far")), "deleted by alice at the moment"},
				{"Near's row on it", "issue_activities", w.relationRowOf(t, w.workItemNamed(t, "Near")), "undeleted"},
			} {
				if got := w.rowDeletion(t, tt.table, tt.id, at); got != tt.want {
					t.Errorf("%s is %s, want %s", tt.what, got, tt.want)
				}
			}
			if n := len(queryIDs(t, w.pool, `SELECT r.id FROM issue_relations r JOIN issues i ON i.id IN (r.issue_id, r.related_issue_id)
				WHERE r.deleted_at IS NULL AND i.deleted_at IS NOT NULL`)); n != 0 {
				t.Errorf("%d undeleted relations name a deleted work item, want none", n)
			}
		})
}

// Relating work items across projects and deleting their workspace
// serialize on the projects' rows (M4 design 3.5, 3.6, convention 7), in
// both orders: bob relates Near, of Web, to Far, of Ops, as blocking it,
// while alice deletes acme. The relation locks Web and Ops at once, and
// the deletion, under acme's row, its projects, each in ascending id
// order, so neither holds a row the other waits for while waiting for one
// of the other's. Deleted first: the relation, which passed its precheck,
// waits for the projects' rows and under its locks finds its own project
// deleted: 404 issue.not_found, and nothing is written. Related first: the
// deletion waits for them, then deletes the relation with every other
// row, at acme's moment, by alice.
func TestRelatingAcrossProjectsAndDeletingTheWorkspaceSerialize(t *testing.T) {
	inBothOrders(t, nearAndFar, "acme deleted", deletesAcme, "related", relates("bob", "Near", "Far", "blocking"),
		func(t *testing.T, w issueWorld, deletedFirst bool, deleted, related exchange) {
			w.answered(t, deleted, http.StatusNoContent)
			if deletedFirst {
				w.refused(t, related, http.StatusNotFound, "issue.not_found")
				if n := len(queryIDs(t, w.pool, "SELECT id FROM issue_relations")); n != 0 {
					t.Errorf("%d relations after the refusal, want none", n)
				}
				return
			}
			waitWorkspaceDeletion(t, w.pool, w.acme)
			made := listIn[answeredRelation](t, w.answered(t, related, http.StatusCreated))
			if got := w.relationDeletion(t, made[0].ID, w.deletedAt(t, "workspaces", w.acme)); got != "deleted by alice at the moment" {
				t.Errorf("the relation is %s, want deleted by alice at the moment", got)
			}
		})
}

// Two deletions of one relation serialize on its projects' rows (M4
// design 3.5, 3.10), in both orders: alice and bob each delete the
// relation of Near, of Web, to Far, of Ops, which alice made. The second
// waits for the first's projects, then reads the relation again under its
// locks, finds it deleted, and is refused, 404 issue.relation_not_found,
// writing nothing: the relation is deleted once, by the first's deleter,
// at the moment of the row of each work item's activity on its deletion.
func TestTwoDeletionsOfARelationSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		nearAndFar(t, w)
		w.send(t, "alice", http.MethodPost, relationsOf(w.workItemNamed(t, "Near")), relating("relates_to", w.workItemNamed(t, "Far")),
			http.StatusCreated)
	}
	inBothOrders(t, prepare, "alice's", unrelates("alice"), "bob's", unrelates("bob"),
		func(t *testing.T, w issueWorld, aliceFirst bool, hers, his exchange) {
			first, second, by := hers, his, "alice"
			if !aliceFirst {
				first, second, by = his, hers, "bob"
			}
			w.answered(t, first, http.StatusNoContent)
			w.refused(t, second, http.StatusNotFound, "issue.relation_not_found")
			relation := queryIDs(t, w.pool, "SELECT id FROM issue_relations")[0]
			at := w.deletedAt(t, "issue_relations", relation)
			if got := w.relationDeletion(t, relation, at); got != "deleted by "+by+" at the moment" {
				t.Errorf("the relation is %s, want deleted by %s at the moment", got, by)
			}
			for _, tt := range []struct {
				item string
				want []string
			}{
				{"Near", []string{"alice: updated relates_to  → OPS-1, none → Far, in WEB, added relates_to relation",
					by + ": deleted relates_to OPS-1 → , Far → none, in WEB, deleted relates_to relation at the moment"}},
				{"Far", []string{"alice: updated relates_to  → WEB-1, none → Near, in OPS, added relates_to relation",
					by + ": deleted relates_to WEB-1 → , Near → none, in OPS, deleted relates_to relation at the moment"}},
			} {
				if got := w.relationRows(t, w.workItemNamed(t, tt.item), at); !slices.Equal(got, tt.want) {
					t.Errorf("the rows of %s's relations:\n%q\nwant\n%q", tt.item, got, tt.want)
				}
			}
		})
}

// Deleting a relation and deleting the work item at its other end
// serialize on that end's project's row (M4 design 3.5, 3.10; interleaving
// 26), in both orders: bob deletes the relation of his Near, of Web, to
// alice's Far, of Ops, which he made, deciding on Near, while alice
// deletes Far. The relation's deletion locks both projects whichever work
// item it is decided on, since it writes on both. Relation deleted first:
// Far's deletion waits for Ops' row, then finds the relation deleted and
// leaves it as bob deleted it. Far deleted first: the relation's deletion,
// which passed its precheck, waits for Ops' row, then reads Far again
// under its locks, finds it deleted, and is refused, 404
// issue.relation_not_found, writing nothing: the relation was deleted with
// Far, by alice at Far's moment.
func TestDeletingARelationAndItsOtherEndSerialize(t *testing.T) {
	prepare := func(t *testing.T, w issueWorld) {
		nearAndFar(t, w)
		w.send(t, "bob", http.MethodPost, relationsOf(w.workItemNamed(t, "Near")), relating("relates_to", w.workItemNamed(t, "Far")),
			http.StatusCreated)
	}
	inBothOrders(t, prepare, "the relation deleted", unrelates("bob"), "Far deleted", deletesNamed("alice", "Far"),
		func(t *testing.T, w issueWorld, relationFirst bool, unrelated, farDeleted exchange) {
			w.answered(t, farDeleted, http.StatusNoContent)
			relation := queryIDs(t, w.pool, "SELECT id FROM issue_relations")[0]
			by, at := "alice", w.deletedAt(t, "issues", w.workItemNamed(t, "Far"))
			if relationFirst {
				w.answered(t, unrelated, http.StatusNoContent)
				by, at = "bob", w.deletedAt(t, "issue_relations", relation)
			} else {
				w.refused(t, unrelated, http.StatusNotFound, "issue.relation_not_found")
			}
			if got := w.relationDeletion(t, relation, at); got != "deleted by "+by+" at the moment" {
				t.Errorf("the relation is %s, want deleted by %s at the moment", got, by)
			}
		})
}
