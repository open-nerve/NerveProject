package bootstrap

import (
	"fmt"
	"maps"
	"net/http"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// collaborationAt shares the real module wiring, with independent transactions
// supplied by serializeAt, including when the clock is controlled by the test.
func collaborationAt(actor, method, path, body string, c interface{ Now() time.Time }) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return w.sending(e, newIssueRouteWithClock(t, w.pool, tx, c), actor, method, path, body)
	}
}

func collaborationRequest(actor, method string, target func(*testing.T, issueWorld) string, body string) racingWrite {
	return func(t *testing.T, w issueWorld, tx shared.TxManager, e *exchange) func() error {
		return collaborationAt(actor, method, target(t, w), body, clock.System{})(t, w, tx, e)
	}
}

func TestCommentAndIssueDeletionSerialize(t *testing.T) {
	var id uuid.UUID
	var beforeActivities string
	prepare := func(t *testing.T, w issueWorld) {
		id = w.newWorkItemIn(t, w.web, `{"name":"Delete with comment"}`)
		beforeActivities = descriptionRows(t, w, "issue_activities", id)
	}
	comment := collaborationRequest("bob", http.MethodPost, func(*testing.T, issueWorld) string { return commentsOf(id) }, `{"comment_html":"<p>Concurrent comment</p>"}`)
	deletion := collaborationRequest("alice", http.MethodDelete, func(*testing.T, issueWorld) string { return "/api/v0/issues/" + id.String() }, "")
	inBothOrdersAt(t, "projects", prepare, "comment", comment, "deletion", deletion, func(t *testing.T, w issueWorld, commentFirst bool, c, d exchange) {
		w.answered(t, d, http.StatusNoContent)
		var deletionStamped bool
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT updated_at=deleted_at AND updated_by_id=$2 FROM issues WHERE id=$1", id, w.ids["alice"]).Scan(&deletionStamped); err != nil || !deletionStamped {
			t.Fatal("issue deletion audit overwritten", deletionStamped, err)
		}
		wantComments, wantSubscribers := 0, 1
		if commentFirst {
			w.answered(t, c, http.StatusCreated)
			wantComments, wantSubscribers = 1, 2
		} else {
			w.refused(t, c, http.StatusNotFound, "issue.not_found")
		}
		for table, want := range map[string]int{"issue_comments": wantComments, "issue_subscribers": wantSubscribers} {
			var all, deleted int
			err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*),count(*) FILTER (WHERE r.deleted_at=i.deleted_at AND r.updated_at=i.deleted_at AND r.updated_by_id=$2) FROM `+table+` r JOIN issues i ON i.id=r.issue_id WHERE i.id=$1`, id, w.ids["alice"]).Scan(&all, &deleted)
			if err != nil || all != want || deleted != want {
				t.Fatalf("%s deletion chain: all=%d stamped=%d want=%d: %v", table, all, deleted, want, err)
			}
		}
		// No comment activity is fabricated. Deletion changes only its audit columns.
		var activities int
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_activities WHERE issue_id=$1", id).Scan(&activities); err != nil || activities != 1 {
			t.Fatal("comment audit", activities, err, beforeActivities)
		}
		if got := w.activityBy(t, id, w.ids["bob"]); len(got) != 0 {
			t.Fatal("comment fabricated activity", got)
		}
	})
}

func TestCommentAndReactionsSerialize(t *testing.T) {
	for _, removeComment := range []bool{false, true} {
		for _, removeReaction := range []bool{false, true} {
			name := fmt.Sprintf("delete_comment=%t/remove_reaction=%t", removeComment, removeReaction)
			t.Run(name, func(t *testing.T) {
				var id uuid.UUID
				var c, neighbor answeredComment
				var old answeredCommentReaction
				var oldRow, neighborRow, activities string
				prepare := func(t *testing.T, w issueWorld) {
					id = w.newWorkItemIn(t, w.web, `{"name":"Reaction race"}`)
					c = w.newComment(t, "bob", id)
					neighbor = w.newComment(t, "alice", id)
					old = addCommentReaction(t, w, "alice", c.ID, "1")
					w.send(t, "alice", http.MethodDelete, commentReactionsOf(c.ID)+"/1", "", 204)
					oldRow = collaborationRows(t, w, id)["comment_reactions"]
					neighborRow = commentRow(t, w, neighbor.ID)
					if removeReaction {
						addCommentReaction(t, w, "alice", c.ID, "128077")
					}
					activities = descriptionRows(t, w, "issue_activities", id)
				}
				method, body := http.MethodPatch, `{"comment_html":"<p>Edited while reacting</p>"}`
				if removeComment {
					method, body = http.MethodDelete, ""
				}
				change := collaborationRequest("bob", method, func(*testing.T, issueWorld) string { return commentPath(c.ID) }, body)
				reactionMethod, reactionBody := http.MethodPost, `{"reaction":"128077"}`
				if removeReaction {
					reactionMethod, reactionBody = http.MethodDelete, ""
				}
				react := collaborationRequest("alice", reactionMethod, func(*testing.T, issueWorld) string {
					path := commentReactionsOf(c.ID)
					if removeReaction {
						return path + "/128077"
					}
					return path
				}, reactionBody)
				inBothOrdersAt(t, "projects", prepare, "comment", change, "reaction", react, func(t *testing.T, w issueWorld, commentFirst bool, ce, re exchange) {
					want := http.StatusOK
					if removeComment {
						want = http.StatusNoContent
					}
					w.answered(t, ce, want)
					if removeComment && commentFirst {
						w.refused(t, re, http.StatusNotFound, "issue.comment_not_found")
					} else {
						want = http.StatusCreated
						if removeReaction {
							want = http.StatusNoContent
						}
						w.answered(t, re, want)
					}
					live := queryIDs(t, w.pool, "SELECT id FROM comment_reactions WHERE comment_id=$1 AND deleted_at IS NULL", c.ID)
					all := queryIDs(t, w.pool, "SELECT id FROM comment_reactions WHERE comment_id=$1", c.ID)
					wantAll := 2
					if removeComment && commentFirst && !removeReaction {
						wantAll = 1
					}
					if len(all) != wantAll {
						t.Fatal("refused reaction left a new row", all, wantAll)
					}
					wantLive := 1
					if removeComment || removeReaction {
						wantLive = 0
					}
					if len(live) != wantLive {
						t.Fatal("live reaction lost or orphaned", live, wantLive)
					}
					// A reaction removed earlier is outside the later deletion chain.
					var kept string
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]')::text FROM comment_reactions r WHERE id=$1", old.ID).Scan(&kept); err != nil || kept != oldRow {
						t.Fatal("old deleted reaction changed", kept, oldRow, err)
					}
					if neighborRow != commentRow(t, w, neighbor.ID) || activities != descriptionRows(t, w, "issue_activities", id) {
						t.Fatal("comment/reaction touched neighbor or activities")
					}
					if !removeComment {
						var edited answeredComment
						decodeAnswer(t, w.answered(t, ce, 200), &edited)
						if edited.HTML != "<p>Edited while reacting</p>" {
							t.Fatal("edit lost", edited)
						}
						if !commentFirst && len(edited.Reactions) != wantLive {
							t.Fatal("edit reply lost committed reactions", edited)
						}
					} else {
						var bad int
						if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM comment_reactions r JOIN issue_comments c ON c.id=r.comment_id WHERE c.id=$1 AND r.id<>$2 AND (r.deleted_at IS NULL OR (NOT $3 AND (r.deleted_at<>c.deleted_at OR r.updated_at<>c.deleted_at OR r.updated_by_id<>$4)))`, c.ID, old.ID, removeReaction, w.ids["bob"]).Scan(&bad); err != nil || bad != 0 {
							t.Fatal("reaction deletion did not follow comment", bad, err)
						}
					}
				})
			})
		}
	}
}

func TestCommentWritesAndArchivalSerialize(t *testing.T) {
	for _, project := range []bool{false, true} {
		for _, operation := range []string{"create", "edit", "delete"} {
			t.Run(fmt.Sprintf("project=%t/%s", project, operation), func(t *testing.T) {
				var id uuid.UUID
				var c answeredComment
				var before map[string]string
				var beforeIssue, beforeActivities string
				var stamps []string
				prepare := func(t *testing.T, w issueWorld) {
					id = w.doneWorkItem(t)
					if operation != "create" {
						c = w.newComment(t, "bob", id)
					}
					// Keep full snapshots; only a work-item archival may change its own issue and activity.
					before = collaborationRows(t, w, id)
					beforeActivities = before["issue_activities"]
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT (to_jsonb(i)-'archived_at'-'updated_at'-'updated_by_id')::text FROM issues i WHERE id=$1", id).Scan(&beforeIssue); err != nil {
						t.Fatal(err)
					}
					stamps = w.workItemStamps(t, id)
				}
				method, body := http.MethodPost, `{"comment_html":"<p>Before archival</p>"}`
				if operation == "edit" {
					method = http.MethodPatch
				}
				if operation == "delete" {
					method, body = http.MethodDelete, ""
				}
				write := collaborationRequest("bob", method, func(*testing.T, issueWorld) string {
					if operation == "create" {
						return commentsOf(id)
					}
					return commentPath(c.ID)
				}, body)
				archive := archivesIssue("alice")
				if project {
					archive = archivesWeb
				}
				inBothOrdersAt(t, "projects", prepare, "archive", archive, "comment", write, func(t *testing.T, w issueWorld, archiveFirst bool, a, b exchange) {
					w.answered(t, a, 200)
					activityCount := 1
					if !project {
						activityCount = 2
					}
					if got := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1", id); len(got) != activityCount {
						t.Fatal("unexpected archival/comment activity", got, activityCount)
					}
					if archiveFirst {
						code := "issue.archived"
						if project {
							code = "project.archived"
						}
						w.refused(t, b, 409, code)
						after := collaborationRows(t, w, id)
						if !project {
							var afterIssue, keptActivities string
							var stamped bool
							if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT (to_jsonb(i)-'archived_at'-'updated_at'-'updated_by_id')::text,updated_at=archived_at AND updated_by_id=$2 FROM issues i WHERE id=$1", id, w.ids["alice"]).Scan(&afterIssue, &stamped); err != nil || !stamped || afterIssue != beforeIssue {
								t.Fatal("refused comment changed archived issue or audit", afterIssue, beforeIssue, stamped, err)
							}
							if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(a) ORDER BY id),'[]')::text FROM issue_activities a WHERE issue_id=$1 AND field IS DISTINCT FROM 'archived_at'", id).Scan(&keptActivities); err != nil || keptActivities != beforeActivities {
								t.Fatal("refused comment changed old activity", keptActivities, beforeActivities, err)
							}
							delete(before, "issues")
							delete(after, "issues")
							delete(before, "issue_activities")
							delete(after, "issue_activities")
						}
						if !maps.Equal(before, after) {
							t.Fatal("refused archived comment wrote rows", before, after, stamps, w.workItemStamps(t, id))
						}
					} else {
						want := 201
						if operation == "edit" {
							want = 200
						}
						if operation == "delete" {
							want = 204
						}
						w.answered(t, b, want)
						var n int
						query := "SELECT count(*) FROM issue_comments WHERE issue_id=$1 AND deleted_at IS NULL AND comment_html='<p>Before archival</p>'"
						expected := 1
						if operation == "delete" {
							query = "SELECT count(*) FROM issue_comments WHERE issue_id=$1 AND deleted_at IS NOT NULL"
						}
						if err := w.pool.QueryRow(pgtest.Soon(t), query, id).Scan(&n); err != nil || n != expected {
							t.Fatal("successful comment lost during archival", n, err)
						}
					}
					if got := w.activityBy(t, id, w.ids["bob"]); len(got) != 0 {
						t.Fatal("comment fabricated activity", got)
					}
				})
			})
		}
	}
}

// collaborationRows complements workItemStamps with values and row identities.
func collaborationRows(t *testing.T, w issueWorld, id uuid.UUID) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, table := range []string{"issues", "issue_subscribers", "issue_mentions", "issue_comments", "issue_reactions", "issue_activities", "issue_description_versions"} {
		out[table] = descriptionRows(t, w, table, id)
	}
	var reactions string
	if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]')::text FROM comment_reactions r WHERE comment_id IN (SELECT id FROM issue_comments WHERE issue_id=$1)`, id).Scan(&reactions); err != nil {
		t.Fatal(err)
	}
	out["comment_reactions"] = reactions
	return out
}

// sameCollaborationRows also checks the older work-item tables' audit stamps.
func sameCollaborationRows(t *testing.T, w issueWorld, id uuid.UUID, rows map[string]string, stamps []string) {
	t.Helper()
	if !maps.Equal(rows, collaborationRows(t, w, id)) || !slices.Equal(stamps, w.workItemStamps(t, id)) {
		t.Fatal("refused collaboration write changed rows or audit")
	}
}
