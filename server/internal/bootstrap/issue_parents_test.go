package bootstrap

import (
	"net/http"
	"reflect"
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
// Up, and Part under Up: Part is answered and stored under Up, and its
// creation writes no row of its parent. A work item under one not there,
// with a label not there, is refused with both problems at once (422),
// and not made.
func TestAWorkItemIsPutUnderAnother(t *testing.T) {
	w := newIssueWorld(t)
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
}

// The guest rule holds for a parent (M4 design 3.4, 3.9): walt, acme's
// admin and Web's guest (adminGuest), sees only his own work items in Web.
// His new work item under alice's Up is refused, 422 at parent_id, as
// under one not there: no work item made. Under his own work item he
// makes one.
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
	below := w.newWorkItemBy(t, "walt", w.web, underParent("Below", his))
	if got := w.parentOf(t, below); !reflect.DeepEqual(got, &his) {
		t.Errorf("Below is under %v, want His, %s", got, his)
	}
}
