package bootstrap

import (
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// updateIssue as bootstrap wires it stamps every row it writes at one
// moment within its request, from the clock issue.New takes, and as its
// caller (M4 design 3.7, M2 handoff 3): alice creates a work item in Web
// of the world, in Todo, assigned to herself and labelled Bug; bob renames
// it, makes it urgent, moves it to Done, assigns it to walt, a member of
// Web, alone and takes Bug off. The work item is changed last by bob then,
// and completed then, its creation alice's as it was; alice's assignment
// and Bug's row are deleted by bob then; walt's assignment and his
// subscription are bob's, then, and bob, who changed it, does not
// subscribe to it (3.12); its activity gains six rows of his, then: the
// name, the priority, the state by its names, Bug removed, walt added and
// alice removed, each account by his display name as identity has it.
// The rows bob deleted keep their making by alice, and the creation's
// other rows, its two activities and alice's subscription, are as they
// were. The answer shows the work item as stored: made by alice at its
// creation, changed last by bob then.
func TestUpdatingAWorkItemStampsItsRows(t *testing.T) {
	w := newIssueWorld(t)
	alice, bob, walt := w.ids["alice"], w.ids["bob"], w.ids["walt"]
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/members",
		`{"members":[{"member_id":"`+walt.String()+`","role":15}]}`, http.StatusCreated)
	item := w.newWorkItemIn(t, w.web, `{"name":"Fix","state_id":"`+w.todo.String()+`","assignee_ids":["`+alice.String()+`"],"label_ids":["`+
		w.bug.String()+`"]}`)
	before := time.Now().Truncate(time.Microsecond)
	answer := w.send(t, "bob", http.MethodPatch, "/api/v0/issues/"+item.String(), `{"name":"Mend","priority":"urgent","state_id":"`+
		w.done.String()+`","assignee_ids":["`+walt.String()+`"],"label_ids":[]}`, http.StatusOK)
	after := time.Now()

	var made, changed time.Time
	var completed *time.Time
	if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT created_at, updated_at, completed_at FROM issues WHERE id = $1", item).Scan(&made,
		&changed, &completed); err != nil {
		t.Fatal(err)
	}
	if changed.Equal(made) || changed.Before(before) || changed.After(after) || completed == nil || !completed.Equal(changed) {
		t.Fatalf("the work item made at %v, changed at %v, completed at %v; want it changed and completed at one moment within %v to %v",
			made, changed, completed, before, after)
	}
	creation := "issue_activities made by alice made, changed by alice made, of alice"
	update := "issue_activities made by bob changed, changed by bob changed, of bob"
	want := []string{creation, creation, update, update, update, update, update, update,
		"issue_assignees made by alice made, changed by bob changed, deleted changed, of alice",
		"issue_assignees made by bob changed, changed by bob changed, of walt",
		"issue_labels made by alice made, changed by bob changed, deleted changed",
		"issue_subscribers made by alice made, changed by alice made, of alice",
		"issue_subscribers made by bob changed, changed by bob changed, of walt",
		"issues made by alice made, changed by bob changed"}
	if got := w.workItemStamps(t, item); !slices.Equal(got, want) {
		t.Errorf("the work item's rows:\n%q\nwant\n%q", got, want)
	}

	said := w.activityBy(t, item, bob)
	names := displayNames(t, w, alice, walt)
	if want := []string{"name: Fix → Mend", "priority: none → urgent", "state: Todo → Done", "labels: Bug → ", "assignees:  → " + names[1],
		"assignees: " + names[0] + " → "}; !slices.Equal(said, want) {
		t.Errorf("bob's activity says %q, want %q", said, want)
	}

	var shown struct {
		CreatedByID uuid.UUID `json:"created_by_id"`
		UpdatedByID uuid.UUID `json:"updated_by_id"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}
	if decodeAnswer(t, answer, &shown); shown.CreatedByID != alice || shown.UpdatedByID != bob || !shown.CreatedAt.Equal(made) ||
		!shown.UpdatedAt.Equal(changed) {
		t.Errorf("updateIssue answers %s; want made by alice at %v, changed last by bob at %v", answer, made, changed)
	}
}

// displayNames are the accounts' display names as identity has them, in
// their order.
func displayNames(t *testing.T, w issueWorld, ids ...uuid.UUID) []string {
	t.Helper()
	names := make([]string, len(ids))
	for i, id := range ids {
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT display_name FROM users WHERE id = $1", id).Scan(&names[i]); err != nil {
			t.Fatal(err)
		}
	}
	return names
}

// getIssue as bootstrap wires it answers the work item as stored, every
// field (M4 design 5.2): Web's second work item, under the first and
// archived (in the database: no operation of P1b archives a work item),
// moved into Done by bob after alice made it and changed by him again
// after that, so that each field that may be null has a value of its own,
// each of its stamps is its own, and it was made, completed, changed last
// and archived at four moments, in that order.
func TestAWorkItemReadsAsStored(t *testing.T) {
	w := newIssueWorld(t)
	first, second := w.newWorkItem(t), w.newWorkItem(t)
	path := "/api/v0/issues/" + second.String()
	w.send(t, "bob", http.MethodPatch, path, `{"state_id":"`+w.done.String()+`"}`, http.StatusOK)
	w.send(t, "bob", http.MethodPatch, path, `{"priority":"low","label_ids":[]}`, http.StatusOK)
	if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET parent_id = $2, archived_at = updated_at + interval '1 minute' WHERE id = $1",
		second, first); err != nil {
		t.Fatal(err)
	}
	var shown storedIssue
	decodeAnswer(t, w.send(t, "bob", http.MethodGet, path, "", http.StatusOK), &shown)
	var stored storedIssue
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT id, workspace_id, project_id, sequence_id, name, state_id, priority, parent_id, sort_order,
		completed_at, archived_at, created_by_id, updated_by_id, created_at, updated_at,
		ARRAY(SELECT assignee_id FROM issue_assignees WHERE issue_id = i.id AND deleted_at IS NULL ORDER BY assignee_id),
		ARRAY(SELECT label_id FROM issue_labels WHERE issue_id = i.id AND deleted_at IS NULL ORDER BY label_id)
		FROM issues i WHERE id = $1`, second).Scan(&stored.ID, &stored.WorkspaceID, &stored.ProjectID, &stored.SequenceID, &stored.Name,
		&stored.StateID, &stored.Priority, &stored.ParentID, &stored.SortOrder, &stored.CompletedAt, &stored.ArchivedAt, &stored.CreatedByID,
		&stored.UpdatedByID, &stored.CreatedAt, &stored.UpdatedAt, &stored.AssigneeIDs, &stored.LabelIDs); err != nil {
		t.Fatal(err)
	}
	shown.inUTC()
	stored.inUTC()
	if !reflect.DeepEqual(shown, stored) || stored.ParentID == nil || stored.CompletedAt == nil || stored.ArchivedAt == nil ||
		*stored.UpdatedByID == *stored.CreatedByID || len(stored.LabelIDs) != 0 || !stored.CreatedAt.Before(*stored.CompletedAt) ||
		!stored.CompletedAt.Before(stored.UpdatedAt) || !stored.UpdatedAt.Before(*stored.ArchivedAt) {
		t.Errorf("getIssue answers %+v; want the work item as stored, %+v, under the first, made, completed, changed last by bob and "+
			"archived at four moments in that order, without labels", shown, stored)
	}
}

// storedIssue is every field of a work item, as getIssue answers it and as
// the database has it.
type storedIssue struct {
	ID          uuid.UUID   `json:"id"`
	WorkspaceID uuid.UUID   `json:"workspace_id"`
	ProjectID   uuid.UUID   `json:"project_id"`
	SequenceID  int         `json:"sequence_id"`
	Name        string      `json:"name"`
	StateID     uuid.UUID   `json:"state_id"`
	Priority    string      `json:"priority"`
	ParentID    *uuid.UUID  `json:"parent_id"`
	SortOrder   float64     `json:"sort_order"`
	CompletedAt *time.Time  `json:"completed_at"`
	ArchivedAt  *time.Time  `json:"archived_at"`
	CreatedByID *uuid.UUID  `json:"created_by_id"`
	UpdatedByID *uuid.UUID  `json:"updated_by_id"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
	AssigneeIDs []uuid.UUID `json:"assignee_ids"`
	LabelIDs    []uuid.UUID `json:"label_ids"`
}

// inUTC puts each moment of s in UTC, where an answer and the database
// read it in zones of their own.
func (s *storedIssue) inUTC() {
	s.CreatedAt, s.UpdatedAt = s.CreatedAt.UTC(), s.UpdatedAt.UTC()
	for _, at := range []**time.Time{&s.CompletedAt, &s.ArchivedAt} {
		if *at != nil {
			utc := (*at).UTC()
			*at = &utc
		}
	}
}
