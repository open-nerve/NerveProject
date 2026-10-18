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

// Deleting a state and putting a work item in it serialize on Web's row (M4
// design 3.6, 3.7; interleaving 6, the creation's branch and the change's),
// in both orders: Review deleted first, the creation, or the change of
// alice's Fix, finds no such state under its lock: 422 state_id, and it
// wrote nothing; written first, the deletion finds the work item in Review
// under its lock: 409 project.state_in_use, and Review is kept.
func TestDeletingAStateAndPuttingAWorkItemInItSerialize(t *testing.T) {
	for _, iw := range itemWrites {
		t.Run(iw.name, func(t *testing.T) {
			inBothOrders(t, iw.after(addsReview), "deleted", deletesReview, iw.name, iw.write("bob", inReview), func(t *testing.T,
				w issueWorld, deletedFirst bool, deletion, written exchange) {
				if deletedFirst {
					w.answered(t, deletion, http.StatusNoContent)
					w.refused(t, written, http.StatusUnprocessableEntity, "validation_failed state_id not_allowed")
					w.wroteNothing(t, iw)
					return
				}
				var put struct {
					StateID uuid.UUID `json:"state_id"`
				}
				decodeAnswer(t, w.answered(t, written, iw.done), &put)
				w.refused(t, deletion, http.StatusConflict, "project.state_in_use")
				w.stateKept(t, put.StateID, "Review, the work item's state")
			})
		})
	}
}

// Deleting a label and labelling a work item with it serialize on Web's row
// (M4 design 3.6, 3.7; interleaving 7, the creation's branch and the
// change's), in both orders: Bug deleted first, the creation, or the change
// of alice's Fix, finds no such label under its lock: 422 label_ids[0], and
// it wrote nothing; written first, the deletion takes Bug off the work item
// at its moment, by alice.
func TestDeletingALabelAndLabellingAWorkItemWithItSerialize(t *testing.T) {
	for _, iw := range itemWrites {
		t.Run(iw.name, func(t *testing.T) {
			inBothOrders(t, iw.prepare, "deleted", deletesBug, iw.name, iw.write("bob", labelledBug), func(t *testing.T, w issueWorld,
				deletedFirst bool, deletion, written exchange) {
				w.answered(t, deletion, http.StatusNoContent)
				if deletedFirst {
					w.refused(t, written, http.StatusUnprocessableEntity, "validation_failed label_ids[0] not_allowed")
					w.wroteNothing(t, iw)
					return
				}
				w.answered(t, written, iw.done)
				if got, want := w.workItemLabels(t, w.bug, "alice"), []string{"Bug taken off by alice with Bug"}; !slices.Equal(got, want) {
					t.Errorf("the work item's labels after Bug's deletion: %q; want %q", got, want)
				}
			})
		})
	}
}
