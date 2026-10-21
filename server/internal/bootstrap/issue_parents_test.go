package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// parentNotAllowed is the refusal of a parent a work item may not have, as
// refusalOf writes it with its messages (M4 design 3.9).
const parentNotAllowed = "validation_failed parent_id not_allowed: must be a work item of the project that is not archived, and neither " +
	"this one nor under it"

// parentOf is the parent of the work item id as the database has it, nil
// for none.
func (w issueWorld) parentOf(t *testing.T, id uuid.UUID) *uuid.UUID {
	t.Helper()
	var parent *uuid.UUID
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT parent_id FROM issues WHERE id = $1", id).Scan(&parent); err != nil {
		t.Fatal(err)
	}
	return parent
}

// parentRows are the rows of the work item item's activity on its parent
// (row 5 of M4 design 3.13), in their ids' order, each as "<actor>:
// <old value> → <new value>, <old identifier> → <new identifier>", an
// account by its name in the world, an identifier none for NULL.
func (w issueWorld) parentRows(t *testing.T, item uuid.UUID) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT actor_id, old_value, new_value, coalesce(old_identifier::text, 'none'),
		coalesce(new_identifier::text, 'none') FROM issue_activities WHERE issue_id = $1 AND field = 'parent' ORDER BY id`, item)
	if err != nil {
		t.Fatal(err)
	}
	names := make(map[uuid.UUID]string, len(w.ids))
	for name, account := range w.ids {
		names[account] = name
	}
	said, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (string, error) {
		var actor uuid.UUID
		var oldValue, newValue, oldID, newID string
		if err := row.Scan(&actor, &oldValue, &newValue, &oldID, &newID); err != nil {
			return "", err
		}
		return names[actor] + ": " + oldValue + " → " + newValue + ", " + oldID + " → " + newID, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return said
}

// underParent is the body of a work item named name under parent.
func underParent(name string, parent uuid.UUID) string {
	return `{"name":"` + name + `","parent_id":"` + parent.String() + `"}`
}

// Parents as bootstrap wires them (M4 design 3.5, 3.9, 3.13). alice makes
// Top, Up, and Part under Up, WEB-1 to WEB-3: Part is answered and stored
// under Up, and its creation writes no row of its parent. A work item
// under one not there, with a label not there, is refused with both
// problems at once (422), and not made. bob moves Part under Top, then
// from under it, then from under none again: each answered with the
// parent as stored; the first two each a row of his activity, from WEB-2
// to WEB-1, then from WEB-1 to "", their ids the old and the new parent's,
// the third, which changes nothing, none. Then each parent Part may not
// have is refused, 422 at parent_id, nothing written: Part itself; Low,
// under Mid under Part, however deep, Mid archived; Ops's work item; an
// archived work item; a deleted one; one not there. Low may move under
// Top, which is under none.
func TestAWorkItemIsPutUnderAnother(t *testing.T) {
	w := newIssueWorld(t)
	top := w.newWorkItemIn(t, w.web, `{"name":"Top"}`)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	var part storedIssue
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", underParent("Part", up),
		http.StatusCreated), &part)
	if !reflect.DeepEqual(part.ParentID, &up) || !reflect.DeepEqual(w.parentOf(t, part.ID), &up) || len(w.parentRows(t, part.ID)) != 0 {
		t.Errorf("Part answered under %v, stored under %v, its parent's rows %q; want under Up, %s, and none", part.ParentID,
			w.parentOf(t, part.ID), w.parentRows(t, part.ID), up)
	}
	count := w.workItems(t)
	lost := w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/issues", `{"name":"Lost","parent_id":"`+
		uuid.NewV7().String()+`","label_ids":["`+uuid.NewV7().String()+`"]}`, http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(lost), false); got != "validation_failed label_ids[0] not_allowed; parent_id not_allowed" ||
		w.workItems(t) != count {
		t.Errorf("a work item under one not there, with a label not there, answers %s, and Web has %d work items; want label_ids[0] and "+
			"parent_id not_allowed at once, and %d", got, w.workItems(t), count)
	}
	path := "/api/v0/issues/" + part.ID.String()
	for _, step := range []struct {
		body string
		to   *uuid.UUID
	}{{`{"parent_id":"` + top.String() + `"}`, &top}, {`{"parent_id":null}`, nil}, {`{"parent_id":null}`, nil}} {
		var moved storedIssue
		decodeAnswer(t, w.send(t, "bob", http.MethodPatch, path, step.body, http.StatusOK), &moved)
		if !reflect.DeepEqual(moved.ParentID, step.to) || !reflect.DeepEqual(w.parentOf(t, part.ID), step.to) {
			t.Errorf("PATCH %s answers it under %v, stored under %v; want %v", step.body, moved.ParentID, w.parentOf(t, part.ID), step.to)
		}
	}
	if said, want := w.parentRows(t, part.ID), []string{"bob: WEB-2 → WEB-1, " + up.String() + " → " + top.String(),
		"bob: WEB-1 → , " + top.String() + " → none"}; !slices.Equal(said, want) {
		t.Errorf("Part's rows of its parent say %q, want %q", said, want)
	}

	mid := w.newWorkItemIn(t, w.web, `{"name":"Mid","state_id":"`+w.done.String()+`","parent_id":"`+part.ID.String()+`"}`)
	low := w.newWorkItemIn(t, w.web, underParent("Low", mid))
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+mid.String()+"/archive", "", http.StatusOK)
	far := w.newWorkItemIn(t, createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", "Ops", "OPS"), `{"name":"Far"}`)
	shelved := w.doneWorkItem(t)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+shelved.String()+"/archive", "", http.StatusOK)
	gone := w.newWorkItemIn(t, w.web, `{"name":"Gone"}`)
	w.send(t, "alice", http.MethodDelete, "/api/v0/issues/"+gone.String(), "", http.StatusNoContent)
	for name, parent := range map[string]uuid.UUID{"itself": part.ID, "under it": low, "Ops's": far, "archived": shelved, "deleted": gone,
		"not there": uuid.NewV7()} {
		_, changed, by := archivalOf(t, w, part.ID)
		answer := w.send(t, "bob", http.MethodPatch, path, `{"parent_id":"`+parent.String()+`"}`, http.StatusUnprocessableEntity)
		if got := refusalOf(t, []byte(answer), true); got != parentNotAllowed {
			t.Errorf("Part under %s answers %s, want %s", name, got, parentNotAllowed)
		}
		if _, at, who := archivalOf(t, w, part.ID); w.parentOf(t, part.ID) != nil || !at.Equal(changed) || who != by {
			t.Errorf("after Part under %s it is under %v, changed last by %s at %v; want under none, as before, by %s at %v", name,
				w.parentOf(t, part.ID), who, at, by, changed)
		}
	}
	w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+low.String(), `{"parent_id":"`+top.String()+`"}`, http.StatusOK)
	if !reflect.DeepEqual(w.parentOf(t, low), &top) {
		t.Errorf("Low is under %v, want Top, %s", w.parentOf(t, low), top)
	}
}

// The guest rule holds for a parent (M4 design 3.4, 3.9): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// His new work item under alice's Up is refused, and so is his work item
// put under it, 422 at parent_id, as under one not there: no work item
// made, nothing written. bob, Web's member, makes one under Up, answered
// and stored under it. Under his own work item walt makes one and moves
// one, and bob puts his own work item under Up.
func TestTheGuestRuleHoldsForAParent(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	up := w.newWorkItemIn(t, w.web, `{"name":"Up"}`)
	his := w.newWorkItemBy(t, "walt", w.web, `{"name":"His"}`)
	create := "/api/v0/projects/" + w.web.String() + "/issues"
	answer := w.send(t, "walt", http.MethodPost, create, underParent("Part", up), http.StatusUnprocessableEntity)
	if got := refusalOf(t, []byte(answer), true); got != parentNotAllowed || w.workItems(t) != 2 {
		t.Errorf("walt's work item under Up answers %s, and Web has %d work items; want %s, and 2", got, w.workItems(t), parentNotAllowed)
	}
	_, changed, by := archivalOf(t, w, his)
	answer = w.send(t, "walt", http.MethodPatch, "/api/v0/issues/"+his.String(), `{"parent_id":"`+up.String()+`"}`,
		http.StatusUnprocessableEntity)
	if _, at, who := archivalOf(t, w, his); refusalOf(t, []byte(answer), true) != parentNotAllowed || w.parentOf(t, his) != nil ||
		!at.Equal(changed) || who != by {
		t.Errorf("walt's work item put under Up answers %s, and is under %v, changed last by %s at %v; want %s, under none, as before",
			answer, w.parentOf(t, his), who, at, parentNotAllowed)
	}
	var sub storedIssue
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, create, underParent("Sub", up), http.StatusCreated), &sub)
	if !reflect.DeepEqual(sub.ParentID, &up) || !reflect.DeepEqual(w.parentOf(t, sub.ID), &up) {
		t.Errorf("bob's work item under Up is answered under %v, stored under %v; want under Up, %s", sub.ParentID,
			w.parentOf(t, sub.ID), up)
	}
	below := w.newWorkItemBy(t, "walt", w.web, underParent("Below", his))
	moved := w.newWorkItemBy(t, "walt", w.web, `{"name":"Moved"}`)
	w.send(t, "walt", http.MethodPatch, "/api/v0/issues/"+moved.String(), `{"parent_id":"`+his.String()+`"}`, http.StatusOK)
	bobs := w.newWorkItemBy(t, "bob", w.web, `{"name":"Bob's"}`)
	w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+bobs.String(), `{"parent_id":"`+up.String()+`"}`, http.StatusOK)
	for _, tt := range []struct {
		item uuid.UUID
		want *uuid.UUID
	}{{below, &his}, {moved, &his}, {bobs, &up}} {
		if got := w.parentOf(t, tt.item); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("the work item %s is under %v, want %v", tt.item, got, tt.want)
		}
	}
}
