package bootstrap

import (
	"net/http"
	"slices"
	"testing"
	"uuid"
)

// addsReview adds Review to Web's started group, through the API.
func addsReview(t *testing.T, w issueWorld) {
	w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/states", `{"name":"Review","color":"#F59E0B","group":"started"}`,
		http.StatusCreated)
}

// inReview is the body of a work item in Web's Review.
func inReview(t *testing.T, w issueWorld) string {
	return `{"name":"Fix","state_id":"` + stateID(t, w.pool, w.web, "Review").String() + `"}`
}

// labelledBug is the body of a work item labelled Web's Bug.
func labelledBug(_ *testing.T, w issueWorld) string {
	return `{"name":"Fix","label_ids":["` + w.bug.String() + `"]}`
}

// alice's deletions of Web's Review and of its Bug.
var (
	deletesReview = aliceOnWeb(http.MethodDelete, func(t *testing.T, w issueWorld) string {
		return "/api/v0/states/" + stateID(t, w.pool, w.web, "Review").String()
	}, "")
	deletesBug = aliceOnWeb(http.MethodDelete, func(_ *testing.T, w issueWorld) string { return "/api/v0/labels/" + w.bug.String() }, "")
)

// Deleting a state and creating a work item in it serialize on Web's row
// (M4 design 3.6, 3.7; interleaving 6, the creation's branch), in both
// orders: Review deleted first, the creation finds no such state under its
// lock: 422 state_id, and no work item; created first, the deletion finds
// the work item in Review under its lock: 409 project.state_in_use, and
// Review is kept.
func TestDeletingAStateAndCreatingAWorkItemInItSerialize(t *testing.T) {
	inBothOrders(t, addsReview, "deleted", deletesReview, "created", createsIssue("bob", inReview), func(t *testing.T, w issueWorld,
		deletedFirst bool, deletion, create exchange) {
		if deletedFirst {
			w.answered(t, deletion, http.StatusNoContent)
			w.refused(t, create, http.StatusUnprocessableEntity, "validation_failed state_id not_allowed")
			if n := w.workItems(t); n != 0 {
				t.Errorf("Web has %d work items, want none", n)
			}
			return
		}
		var created struct {
			StateID uuid.UUID `json:"state_id"`
		}
		decodeAnswer(t, w.answered(t, create, http.StatusCreated), &created)
		w.refused(t, deletion, http.StatusConflict, "project.state_in_use")
		w.stateKept(t, created.StateID, "Review, the work item's state")
	})
}

// Deleting a label and creating a work item labelled with it serialize on
// Web's row (M4 design 3.6, 3.7; interleaving 7, the creation's branch),
// in both orders: Bug deleted first, the creation finds no such label
// under its lock: 422 label_ids[0], and no work item; created first, the
// deletion takes Bug off the work item at its moment, by alice.
func TestDeletingALabelAndCreatingAWorkItemWithItSerialize(t *testing.T) {
	inBothOrders(t, nil, "deleted", deletesBug, "created", createsIssue("bob", labelledBug), func(t *testing.T, w issueWorld,
		deletedFirst bool, deletion, create exchange) {
		w.answered(t, deletion, http.StatusNoContent)
		if deletedFirst {
			w.refused(t, create, http.StatusUnprocessableEntity, "validation_failed label_ids[0] not_allowed")
			if n := w.workItems(t); n != 0 {
				t.Errorf("Web has %d work items, want none", n)
			}
			return
		}
		w.answered(t, create, http.StatusCreated)
		if got, want := w.workItemLabels(t, w.bug, "alice"), []string{"Bug taken off by alice with Bug"}; !slices.Equal(got, want) {
			t.Errorf("the work item's labels after Bug's deletion: %q; want %q", got, want)
		}
	})
}
