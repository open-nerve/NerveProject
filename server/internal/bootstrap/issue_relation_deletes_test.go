package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// relationPath is the path of the relation id.
func relationPath(id uuid.UUID) string {
	return "/api/v0/issue-relations/" + id.String()
}

// relationDeletion is how the relation id stands (rowDeletion).
func (w issueWorld) relationDeletion(t *testing.T, id uuid.UUID, at time.Time) string {
	t.Helper()
	return w.rowDeletion(t, "issue_relations", id, at)
}

// rowDeletion is how the row id of table stands: "undeleted", or "deleted
// by <account> at the moment" when it was deleted and written last at at,
// else "... at another moment", an account by its name in the world.
func (w issueWorld) rowDeletion(t *testing.T, table string, id uuid.UUID, at time.Time) string {
	t.Helper()
	var deleted *time.Time
	var by uuid.UUID
	var written time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at, updated_by_id, updated_at FROM "+table+" WHERE id = $1", id).Scan(
		&deleted, &by, &written); err != nil {
		t.Fatal(err)
	}
	if deleted == nil {
		return "undeleted"
	}
	when := "at the moment"
	if !deleted.Equal(at) || !written.Equal(at) {
		when = "at another moment"
	}
	return "deleted by " + w.names()[by] + " " + when
}

// A relation is deleted from either of its work items, as bootstrap wires
// it (M4 design 3.5, 3.7, 3.10, 3.13). alice makes Ops, of acme, of which
// bob is no member, Near in Web and Far in Ops, and relates Near to Far,
// blocking it: stored as Far's blocked_by. bob's deletion is decided on
// Far, which he sees and may not change, then on Near, which he may: 204.
// The relation is deleted by bob at the moment Near and Far are changed
// last, by him; each has a row of its activity on it, in its project, the
// type as it read it, the other by its identifier, after alice's of its
// making; neither lists it. Deleted, it is not found again.
func TestARelationIsDeletedFromEitherWorkItem(t *testing.T) {
	w := newIssueWorld(t)
	ops := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS")
	near := w.newWorkItemIn(t, w.web, `{"name":"Near"}`)
	far := w.newWorkItemIn(t, ops, `{"name":"Far"}`)
	made := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(near), relating("blocking", far), http.StatusCreated))

	w.send(t, "bob", http.MethodDelete, relationPath(made[0].ID), "", http.StatusNoContent)
	_, at, _ := archivalOf(t, w, near)
	if got := w.relationDeletion(t, made[0].ID, at); got != "deleted by bob at the moment" {
		t.Errorf("the relation is %s, want deleted by bob at the moment", got)
	}
	for _, item := range []uuid.UUID{near, far} {
		if _, changed, by := archivalOf(t, w, item); !changed.Equal(at) || by != w.ids["bob"] {
			t.Errorf("the work item %s is changed last by %s at %v, want by bob at %v", item, by, changed, at)
		}
	}
	for _, tt := range []struct {
		item uuid.UUID
		want []string
	}{
		{near, []string{"alice: updated blocking  → OPS-1, none → Far, in WEB, added blocking relation",
			"bob: deleted blocking OPS-1 → , Far → none, in WEB, deleted blocking relation at the moment"}},
		{far, []string{"alice: updated blocked_by  → WEB-1, none → Near, in OPS, added blocked_by relation",
			"bob: deleted blocked_by WEB-1 → , Near → none, in OPS, deleted blocked_by relation at the moment"}},
	} {
		if got := w.relationRows(t, tt.item, at); !slices.Equal(got, tt.want) {
			t.Errorf("the rows of %s's relations:\n%q\nwant\n%q", tt.item, got, tt.want)
		}
		if listed := w.listedRelations(t, "alice", tt.item); len(listed) != 0 {
			t.Errorf("%s lists %q, want none", tt.item, listed)
		}
	}
	answer := w.send(t, "bob", http.MethodDelete, relationPath(made[0].ID), "", http.StatusNotFound)
	if got := refusalOf(t, []byte(answer), false); got != "issue.relation_not_found" {
		t.Errorf("deleting it again answers %s, want issue.relation_not_found", got)
	}
}

// Who may not delete a relation, and the relations that are not deleted
// (M4 design 3.4, 3.8, 3.10), each refused with nothing written: walt,
// acme's member and none of Web, sees Near and Up and may change neither
// (403); olga, no member of acme, sees neither (404
// issue.relation_not_found), nor walt the relation of two work items of
// Lab, a private project he is none of, each the very answer of a relation
// not there. Then a relation of an archived work item, Shelved, from
// either end, and one of a work item of an archived project, Old's, are
// not deleted (409), the archived one's end stored as issue_id or not;
// nor are those of Old's work item to Shelved, stored from Old's, and of
// Shelved to Old's second, stored from Shelved: the archived project,
// Old, is answered before the archived work item, Shelved, whichever end
// each is (409 project.archived).
func TestWhoMayNotDeleteARelation(t *testing.T) {
	w := newIssueWorld(t)
	lab := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Lab","identifier":"LAB","network":0}`)
	old := createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Old", "OLD")
	near, up := w.newWorkItemIn(t, w.web, `{"name":"Near"}`), w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	shelved := w.newWorkItemIn(t, w.web, `{"name":"Shelved","state_id":"`+w.done.String()+`"}`)
	labs, lab2 := w.newWorkItemIn(t, lab, `{"name":"Lab's"}`), w.newWorkItemIn(t, lab, `{"name":"Lab's too"}`)
	olds, old2 := w.newWorkItemIn(t, old, `{"name":"Old's"}`), w.newWorkItemIn(t, old, `{"name":"Old's too"}`)
	related := func(from, to uuid.UUID, kind string) uuid.UUID {
		return listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(from), relating(kind, to), http.StatusCreated))[0].ID
	}
	ofUp, ofLab := related(near, up, "relates_to"), related(labs, lab2, "relates_to")
	toShelved, fromShelved := related(near, shelved, "relates_to"), related(shelved, up, "relates_to")
	toOld, fromOld := related(up, olds, "relates_to"), related(olds, near, "relates_to")
	fromOldToShelved, fromShelvedToOld := related(olds, shelved, "relates_to"), related(shelved, old2, "relates_to")
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+old.String()+"/archive", "", http.StatusOK)
	before := w.relationRows(t, near, time.Time{})

	for _, tt := range []struct {
		name     string
		relation uuid.UUID
		status   int
		code     string
	}{
		{"walt", ofUp, http.StatusForbidden, "forbidden"},
		{"olga", ofUp, http.StatusNotFound, "issue.relation_not_found"},
		{"walt", ofLab, http.StatusNotFound, "issue.relation_not_found"},
		{"bob", toShelved, http.StatusConflict, "issue.archived"},
		{"bob", fromShelved, http.StatusConflict, "issue.archived"},
		{"alice", toOld, http.StatusConflict, "project.archived"},
		{"alice", fromOld, http.StatusConflict, "project.archived"},
		{"alice", fromOldToShelved, http.StatusConflict, "project.archived"},
		{"alice", fromShelvedToOld, http.StatusConflict, "project.archived"},
	} {
		answer := w.send(t, tt.name, http.MethodDelete, relationPath(tt.relation), "", tt.status)
		if got := refusalOf(t, []byte(answer), false); got != tt.code || w.relationDeletion(t, tt.relation, time.Time{}) != "undeleted" {
			t.Errorf("%s's deletion of %s answers %s, the relation %s; want %s, undeleted", tt.name, tt.relation, got,
				w.relationDeletion(t, tt.relation, time.Time{}), tt.code)
		}
		if tt.status != http.StatusNotFound {
			continue
		}
		if absent := w.send(t, tt.name, http.MethodDelete, relationPath(uuid.NewV7()), "", http.StatusNotFound); answer != absent {
			t.Errorf("%s's deletion of %s answers %s, want as for a relation not there: %s", tt.name, tt.relation, answer, absent)
		}
	}
	if after := w.relationRows(t, near, time.Time{}); !slices.Equal(after, before) {
		t.Errorf("Near's rows on its relations:\n%q\nwant them as they were:\n%q", after, before)
	}
}

// The guest rule holds for deleting a relation (M4 design 3.4, 3.10):
// walt, acme's admin and Web's guest (adminGuest), sees only his own work
// items in Web. alice relates Hers to his His, stored from Hers, which he
// does not see: his deletion is decided on Hers, hidden from him, then on
// His, which he may change as acme's admin: 204. The relation of Up and
// Hers, alice's both, he finds not: the very 404 of a relation not there.
func TestTheGuestRuleHoldsForDeletingARelation(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	hers := w.newWorkItemIn(t, w.web, `{"name":"Hers"}`)
	ofHis := listIn[answeredRelation](t, w.send(t, "alice", http.MethodPost, relationsOf(hers), relating("relates_to", his, up), http.StatusCreated))
	if len(ofHis) != 2 {
		t.Fatalf("alice made %d relations, want 2", len(ofHis))
	}
	byID := map[uuid.UUID]uuid.UUID{ofHis[0].Issue.ID: ofHis[0].ID, ofHis[1].Issue.ID: ofHis[1].ID}

	w.send(t, "walt", http.MethodDelete, relationPath(byID[his]), "", http.StatusNoContent)
	_, at, _ := archivalOf(t, w, his)
	if got := w.relationDeletion(t, byID[his], at); got != "deleted by walt at the moment" {
		t.Errorf("the relation of His is %s, want deleted by walt at the moment", got)
	}
	answer := w.send(t, "walt", http.MethodDelete, relationPath(byID[up]), "", http.StatusNotFound)
	absent := w.send(t, "walt", http.MethodDelete, relationPath(uuid.NewV7()), "", http.StatusNotFound)
	if got := refusalOf(t, []byte(answer), false); got != "issue.relation_not_found" || answer != absent ||
		w.relationDeletion(t, byID[up], at) != "undeleted" {
		t.Errorf("walt's deletion of the relation of Up answers %s; want issue.relation_not_found, as for a relation not there (%s), and it "+
			"undeleted", answer, absent)
	}
}
