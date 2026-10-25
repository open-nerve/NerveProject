package bootstrap

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func TestCollaborationRechecksMembership(t *testing.T) {
	for _, operation := range []string{"description hidden", "description own", "comment create", "comment edit", "comment delete", "issue reaction add", "issue reaction remove", "comment reaction add", "comment reaction remove"} {
		for _, removed := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/removed=%t", operation, removed), func(t *testing.T) {
				w := newIssueWorld(t)
				// A private project makes removal invisible even to a workspace member.
				w.send(t, "alice", http.MethodPatch, "/api/v0/projects/"+w.web.String(), `{"network":0}`, 200)
				creator := "alice"
				if operation == "description own" {
					creator = "bob"
				}
				id := w.newWorkItemBy(t, creator, w.web, `{"name":"Membership race"}`)
				c := w.newComment(t, "bob", id)
				method, path, body, code := http.MethodPatch, "/api/v0/issues/"+id.String(), `{"description_html":"<p>Denied after wait</p>"}`, "issue.not_found"
				targetID := id
				switch operation {
				case "comment create":
					method, path, body = http.MethodPost, commentsOf(id), `{"comment_html":"<p>Denied after wait</p>"}`
				case "comment edit", "comment delete":
					path, body, code, targetID = commentPath(c.ID), `{"comment_html":"<p>Denied after wait</p>"}`, "issue.comment_not_found", c.ID
					if operation == "comment delete" {
						method, body = http.MethodDelete, ""
					}
				case "issue reaction add", "issue reaction remove":
					method, path, body = http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`
					if operation == "issue reaction remove" {
						w.send(t, "bob", http.MethodPost, path, body, 201)
						method, path, body = http.MethodDelete, path+"/128077", ""
					}
				case "comment reaction add", "comment reaction remove":
					method, path, body, code, targetID = http.MethodPost, commentReactionsOf(c.ID), `{"reaction":"128077"}`, "issue.comment_not_found", c.ID
					if operation == "comment reaction remove" {
						addCommentReaction(t, w, "bob", c.ID, "128077")
						method, path, body = http.MethodDelete, path+"/128077", ""
					}
				}
				before, stamps := collaborationRows(t, w, id), w.workItemStamps(t, id)
				var changed, written exchange
				change, changedStatus := demotesBob, http.StatusOK
				if removed {
					change, changedStatus = removesBob, http.StatusNoContent
				}
				w.serializeAt(t, "projects", change, collaborationAt("bob", method, path, body, clock.System{}), &changed, &written)
				w.answered(t, changed, changedStatus)
				status := http.StatusNotFound
				if !removed && operation == "description own" {
					status, code = http.StatusForbidden, "forbidden"
				}
				w.refused(t, written, status, code)
				sameCollaborationRows(t, w, id, before, stamps)
				if status == http.StatusNotFound {
					route := newIssueRoute(t, w.pool, nil)
					missing := strings.Replace(path, targetID.String(), uuid.NewV7().String(), 1)
					req, rec := route.send(method, missing, w.ids["bob"], body)
					w.contract.CheckResponse(t, req, rec.Result())
					actual, want := wholeAnswerOf(written.rec.Result(), written.rec.Body.Bytes()), wholeAnswerOf(rec.Result(), rec.Body.Bytes())
					if actual != want {
						t.Fatal("hidden target differs from missing target", actual, want)
					}
				}
			})
		}
	}
	// A guest retains comment permission on his own issue after demotion;
	// visibility is distinct from the description writer's role restriction.
	t.Run("guest can comment on own issue", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItemBy(t, "bob", w.web, `{"name":"Own comment"}`)
		var demotion, comment exchange
		w.serializeAt(t, "projects", demotesBob, collaborationAt("bob", http.MethodPost, commentsOf(id), `{"comment_html":"<p>Still allowed</p>"}`, clock.System{}), &demotion, &comment)
		w.answered(t, demotion, 200)
		w.answered(t, comment, 201)
		if got := queryIDs(t, w.pool, "SELECT id FROM issue_comments WHERE issue_id=$1 AND actor_id=$2 AND deleted_at IS NULL", id, w.ids["bob"]); len(got) != 1 {
			t.Fatal("guest's own comment missing", got)
		}
		if got := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1", id); len(got) != 1 {
			t.Fatal("only own creation activity expected", got)
		}
		var n int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_subscribers WHERE issue_id=$1 AND subscriber_id=$2 AND deleted_at IS NULL", id, w.ids["bob"]).Scan(&n); err != nil || n != 1 {
			t.Fatal("own subscription duplicated", n, err)
		}
	})
}
