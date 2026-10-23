package bootstrap

import (
	"context"
	"net/http"
	"slices"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// deleteState refuses a state a work item is in, an archived one too, as
// bootstrap wires it (M4 design 3.6): in Web of the world, alice adds
// Review to the started group, and bob creates a work item in it, one in
// Todo and one in the default state, Backlog. Deleting Review answers 409
// project.state_in_use and keeps it, and so it does once its work item is
// archived; once the work item is deleted, Review is deleted. Todo, its
// group's only state, and Backlog, the default and its group's only state,
// each answer project.state_in_use: the work items are checked before the
// default and the group (M4 design 3.5).
func TestDeletingAStateAWorkItemIsInIsRefused(t *testing.T) {
	w := newIssueWorld(t)
	web := "/api/v0/projects/" + w.web.String()
	var review, fix struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, web+"/states", `{"name":"Review","color":"#F59E0B","group":"started"}`,
		http.StatusCreated), &review)
	decodeAnswer(t, w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Fix","state_id":"`+review.ID.String()+`"}`,
		http.StatusCreated), &fix)
	w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Plan","state_id":"`+w.todo.String()+`"}`, http.StatusCreated)
	w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Idea"}`, http.StatusCreated)
	refused := func(state uuid.UUID, what string) {
		t.Helper()
		answer := w.send(t, "alice", http.MethodDelete, "/api/v0/states/"+state.String(), "", http.StatusConflict)
		if got := refusalOf(t, []byte(answer), true); got != "project.state_in_use" {
			t.Errorf("deleting %s = %s; want project.state_in_use", what, answer)
		}
		w.stateKept(t, state, what)
	}
	fixed := func(set string) {
		t.Helper()
		if _, err := w.pool.Exec(context.Background(), "UPDATE issues SET "+set+" WHERE id = $1", fix.ID); err != nil {
			t.Fatal(err)
		}
	}
	refused(review.ID, "Review, with a work item in it")
	refused(w.todo, "Todo, its group's only state, with a work item in it")
	refused(stateID(t, w.pool, w.web, "Backlog"), "Backlog, the default and its group's only state, with a work item in it")
	fixed("archived_at = now()")
	refused(review.ID, "Review, with its work item archived")
	fixed("deleted_at = now()")
	w.send(t, "alice", http.MethodDelete, "/api/v0/states/"+review.ID.String(), "", http.StatusNoContent)
}

// deleteLabel takes its label, and the labels under it, off every work
// item, at the moment of the deletion and by its caller, as bootstrap
// wires it (M4 design 3.6): in Web of the world, alice adds UI under Bug,
// and Feature; bob creates a work item labelled Bug and Feature, and one
// labelled UI; the first is archived. Once alice deletes Bug, the rows
// that put Bug and UI on them are deleted at Bug's moment, by her, the
// archived work item's too: deleting a label is a write on the project,
// not on a work item. Feature's row is kept.
func TestDeletingALabelTakesItOffTheWorkItems(t *testing.T) {
	w := newIssueWorld(t)
	web := "/api/v0/projects/" + w.web.String()
	var ui, feature struct {
		ID uuid.UUID `json:"id"`
	}
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, web+"/labels", `{"name":"UI","parent_id":"`+w.bug.String()+`"}`, http.StatusCreated), &ui)
	decodeAnswer(t, w.send(t, "alice", http.MethodPost, web+"/labels", `{"name":"Feature"}`, http.StatusCreated), &feature)
	for _, labels := range [][]uuid.UUID{{w.bug, feature.ID}, {ui.ID}} {
		w.send(t, "bob", http.MethodPost, web+"/issues", `{"name":"Fix","label_ids":`+idArray(labels...)+`}`, http.StatusCreated)
	}
	tag, err := w.pool.Exec(pgtest.Soon(t), `UPDATE issues SET archived_at = now()
		WHERE id = (SELECT issue_id FROM issue_labels WHERE label_id = $1)`, w.bug)
	if err != nil || tag.RowsAffected() != 1 {
		t.Fatalf("archiving the work item labelled Bug: %v, %d rows", err, tag.RowsAffected())
	}
	w.send(t, "alice", http.MethodDelete, "/api/v0/labels/"+w.bug.String(), "", http.StatusNoContent)
	got := w.workItemLabels(t, w.bug, "alice")
	if want := []string{"Bug taken off by alice with Bug", "Feature kept", "UI taken off by alice with Bug"}; !slices.Equal(got, want) {
		t.Errorf("the work items' labels after Bug's deletion: %q; want %q", got, want)
	}
}

// stateKept checks that state, which what names, is not deleted after its
// refused deletion.
func (w issueWorld) stateKept(t *testing.T, state uuid.UUID, what string) {
	t.Helper()
	var kept bool
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT deleted_at IS NULL FROM states WHERE id = $1", state).Scan(&kept); err != nil || !kept {
		t.Errorf("%s after its refused deletion: kept %v, %v; want kept", what, kept, err)
	}
}

// workItemLabels is each row that puts a label on a work item, in the
// order of the labels' names, as "<label> kept"; "<label> taken off by
// <deleter> with <deleted>" when the deletion of the label deleted took it
// off, at that deletion's moment and by the account deleter; or "<label>
// taken off otherwise".
func (w issueWorld) workItemLabels(t *testing.T, deleted uuid.UUID, deleter string) []string {
	t.Helper()
	rows, err := w.pool.Query(pgtest.Soon(t), `SELECT format('%s %s', l.name, CASE WHEN il.deleted_at IS NULL THEN 'kept'
		WHEN il.deleted_at = d.deleted_at AND il.updated_at = d.deleted_at AND il.updated_by_id = $2
		THEN format('taken off by %s with %s', $3::text, d.name) ELSE 'taken off otherwise' END)
		FROM issue_labels il JOIN labels l ON l.id = il.label_id, labels d WHERE d.id = $1 ORDER BY l.name`, deleted, w.ids[deleter], deleter)
	if err != nil {
		t.Fatal(err)
	}
	got, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		t.Fatal(err)
	}
	return got
}
