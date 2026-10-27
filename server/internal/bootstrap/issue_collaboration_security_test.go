package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// collaborationState includes tombstones and audit fields, across all new tables
// and the existing work-item tables which these operations can affect.
func collaborationState(t *testing.T, w issueWorld) map[string]string {
	t.Helper()
	state := map[string]string{}
	for _, table := range []string{"issues", "issue_links", "issue_assignees", "issue_labels", "issue_relations", "issue_subscribers", "issue_mentions", "issue_comments", "issue_reactions", "comment_reactions", "issue_activities", "issue_description_versions"} {
		var rows string
		if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+table+" r").Scan(&rows); err != nil {
			t.Fatal(err)
		}
		state[table] = rows
	}
	return state
}

func TestCollaborationSecurity(t *testing.T) {
	t.Run("hidden equals absent for every operation", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		comment := w.newComment(t, "alice", id)
		version := savedVersion(t, w, id)
		w.adminGuest(t)
		missing := uuid.NewV7()
		for _, op := range []struct{ name, method, hidden, absent, body string }{
			{"listIssueComments", http.MethodGet, commentsOf(id), commentsOf(missing), ""},
			{"createIssueComment", http.MethodPost, commentsOf(id), commentsOf(missing), `{"comment_html":"<p>Denied</p>"}`},
			{"updateIssueComment", http.MethodPatch, commentPath(comment.ID), commentPath(missing), `{"comment_html":"<p>Denied</p>"}`},
			{"deleteIssueComment", http.MethodDelete, commentPath(comment.ID), commentPath(missing), ""},
			{"listIssueReactions", http.MethodGet, reactionsOf(id), reactionsOf(missing), ""},
			{"addIssueReaction", http.MethodPost, reactionsOf(id), reactionsOf(missing), `{"reaction":"128077"}`},
			{"removeIssueReaction", http.MethodDelete, reactionsOf(id) + "/128077", reactionsOf(missing) + "/128077", ""},
			{"addCommentReaction", http.MethodPost, commentReactionsOf(comment.ID), commentReactionsOf(missing), `{"reaction":"128077"}`},
			{"removeCommentReaction", http.MethodDelete, commentReactionsOf(comment.ID) + "/128077", commentReactionsOf(missing) + "/128077", ""},
			{"listIssueActivities", http.MethodGet, activitiesOf(id), activitiesOf(missing), ""},
			{"listDescriptionVersions", http.MethodGet, versionsOf(id), versionsOf(missing), ""},
			{"getDescriptionVersion", http.MethodGet, versionPath(version.ID), versionPath(missing), ""},
		} {
			t.Run(op.name, func(t *testing.T) {
				before := collaborationState(t, w)
				answer := func(path string) wholeAnswer {
					req := newRequest(t, op.method, w.base+path, w.tokens["walt"], []byte(op.body))
					res, body := send(t, req)
					w.contract.CheckResponse(t, req, res)
					if res.StatusCode != 404 {
						t.Fatal(res.StatusCode, string(body))
					}
					return wholeAnswerOf(res, body)
				}
				hiddenReply, absentReply := answer(op.hidden), answer(op.absent)
				if !reflect.DeepEqual(hiddenReply, absentReply) {
					t.Fatal("not-found differs", hiddenReply, absentReply)
				}
				if !reflect.DeepEqual(before, collaborationState(t, w)) {
					t.Fatal("refused operation changed collaboration state")
				}
			})
		}
	})
	t.Run("author precheck uses no write resources", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		c := w.newComment(t, "alice", id)
		one := poolOfOne(t, w.pool.Config().ConnString())
		store := issuepg.New(one)
		projects := issueProjects{projects: project.Provide(one).IssueProjects}
		resources := &collaborationWriteResources{}
		locks := issueapp.NewLocks(projects, store, authorizerOn(one), resources, resources)
		edit := issueapp.NewUpdateIssueComment(issueapp.UpdateIssueCommentDeps{Events: discardIssueEvents{}, Locks: locks, Comments: store, Members: projects, Sanitizer: content.NewSanitizer(), Clock: clock.System{}})
		remove := issueapp.NewDeleteIssueComment(issueapp.DeleteIssueCommentDeps{Events: discardIssueEvents{}, Locks: locks, Comments: store, Clock: clock.System{}})
		ctx := shared.WithActor(pgtest.Soon(t), shared.Actor{UserID: w.ids["bob"]})
		before := collaborationState(t, w)
		// Exclusive table locks permit precheck SELECTs while excluding any row
		// locks or writes. The real authorizer/store share a single-connection pool.
		held := lockedTables(t, w.pool, issueTables, "EXCLUSIVE")
		_, err := edit.Execute(ctx, c.ID, "<p>Denied</p>")
		if err != domain.ErrNotAuthor {
			t.Fatal("edit did not refuse at author precheck", err)
		}
		if err := remove.Execute(ctx, c.ID); err != domain.ErrNotAuthor {
			t.Fatal("delete did not refuse at author precheck", err)
		}
		release(t, held)
		if resources.gates != 0 || resources.transactions != 0 || one.Stat().AcquiredConns() != 0 {
			t.Fatal("403 acquired write resources", resources, one.Stat().AcquiredConns())
		}
		if !reflect.DeepEqual(before, collaborationState(t, w)) {
			t.Fatal("author refusal changed rows")
		}
		for _, method := range []string{http.MethodPatch, http.MethodDelete} {
			body := ""
			if method == http.MethodPatch {
				body = `{"comment_html":"<p>Denied</p>"}`
			}
			w.send(t, "bob", method, commentPath(c.ID), body, 403)
		}
	})
	t.Run("administrator guest and ordinary guest", func(t *testing.T) {
		w := newIssueWorld(t)
		w.adminGuest(t)
		own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Owned"}`)
		c := w.newComment(t, "alice", own)
		w.send(t, "walt", http.MethodGet, versionsOf(own), "", 200)
		w.send(t, "walt", http.MethodDelete, commentPath(c.ID), "", 204)
		c = w.newComment(t, "alice", own)
		// Removing workspace-admin authority leaves project-guest visibility intact.
		member := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])[0]
		w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+member.String(), `{"role":15}`, 200)
		w.send(t, "walt", http.MethodGet, commentsOf(own), "", 200)
		w.send(t, "walt", http.MethodPost, commentsOf(own), `{"comment_html":"<p>Guest comment</p>"}`, 201)
		w.send(t, "walt", http.MethodPost, reactionsOf(own), `{"reaction":"128077"}`, 201)
		w.send(t, "walt", http.MethodPost, commentReactionsOf(c.ID), `{"reaction":"128077"}`, 201)
		w.send(t, "walt", http.MethodGet, versionsOf(own), "", 403)
		w.send(t, "walt", http.MethodGet, versionPath(savedVersion(t, w, own).ID), "", 403)
		w.send(t, "walt", http.MethodDelete, commentPath(c.ID), "", 403)
	})
	t.Run("archive precedes unchanged comment", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.doneWorkItem(t)
		c := w.newComment(t, "bob", id)
		w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+id.String()+"/archive", "", 200)
		check := func(want string) {
			t.Helper()
			before := collaborationState(t, w)
			answer := w.send(t, "bob", http.MethodPatch, commentPath(c.ID), fmt.Sprintf(`{"comment_html":%q}`, c.HTML), 409)
			if refusalOf(t, []byte(answer), false) != want {
				t.Fatal("archive precedence", answer)
			}
			if !reflect.DeepEqual(before, collaborationState(t, w)) {
				t.Fatal("archived no-op changed rows")
			}
		}
		check("issue.archived")
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
		check("project.archived")
	})
	t.Run("reaction absence belongs only to caller", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		c := w.newComment(t, "alice", id)
		w.send(t, "alice", http.MethodPost, reactionsOf(id), `{"reaction":"128077"}`, 201)
		addCommentReaction(t, w, "alice", c.ID, "128077")
		for _, path := range []string{reactionsOf(id), commentReactionsOf(c.ID)} {
			before := collaborationState(t, w)
			present := w.send(t, "bob", http.MethodDelete, path+"/128077", "", 404)
			absent := w.send(t, "bob", http.MethodDelete, path+"/1", "", 404)
			if present != absent {
				t.Fatal("other actor reaction leaked", present, absent)
			}
			if !reflect.DeepEqual(before, collaborationState(t, w)) {
				t.Fatal("deleted another actor's reaction")
			}
		}
	})
	t.Run("real HTTP validation and custom sources", func(t *testing.T) {
		w := newIssueWorld(t)
		id := w.newWorkItem(t)
		c := w.newComment(t, "alice", id)
		invalid := func(method, path, body string, want int) {
			t.Helper()
			req := newRequest(t, method, w.base+path, w.tokens["alice"], []byte(body))
			res, data := send(t, req)
			w.contract.CheckResponse(t, req, res)
			if res.StatusCode != want {
				t.Fatal(method, path, res.StatusCode, string(data))
			}
		}
		for _, path := range []string{commentsOf(id), commentPath(c.ID), reactionsOf(id), commentReactionsOf(c.ID)} {
			method := http.MethodPost
			if path == commentPath(c.ID) {
				method = http.MethodPatch
			}
			for _, body := range []string{`[]`, `null`, `{"unexpected":true}`} {
				invalid(method, path, body, 400)
			}
		}
		invalid(http.MethodPost, commentsOf(id), `{"comment_html":"<script>removed()</script>"}`, 422)
		for _, path := range []string{reactionsOf(id), commentReactionsOf(c.ID)} {
			invalid(http.MethodPost, path, `{"reaction":"not-hex"}`, 422)
		}
		for _, path := range []string{commentsOf(id), activitiesOf(id), versionsOf(id)} {
			invalid(http.MethodGet, path+"?limit=0", "", 422)
			invalid(http.MethodGet, path+"?cursor=broken&limit=0", "", 400)
		}
		html := `<p>Safe text</p><image-component src="javascript:alert(1)"></image-component><div data-type="emoji" data-emoji-url="javascript:alert(2)">x</div>`
		var safe answeredComment
		decodeAnswer(t, w.send(t, "alice", http.MethodPost, commentsOf(id), fmt.Sprintf(`{"comment_html":%q}`, html), 201), &safe)
		if strings.Contains(safe.HTML, "javascript:") || !strings.Contains(safe.HTML, "Safe text") {
			t.Fatal("custom source was not sanitized over HTTP", safe.HTML)
		}
		if got := commentRow(t, w, safe.ID); strings.Contains(got, "javascript:") {
			t.Fatal("unsafe source persisted", got)
		}
	})
}

// These guards observe write-only resources; precheck queries may borrow and
// return the real pool's connection, which is intentionally not forbidden.
type collaborationWriteResources struct{ gates, transactions int }

func (r *collaborationWriteResources) Enter(context.Context, ...uuid.UUID) (func(), error) {
	r.gates++
	return func() {}, nil
}
func (r *collaborationWriteResources) WithinTx(context.Context, func(context.Context) error) error {
	r.transactions++
	return fmt.Errorf("precheck acquired write transaction")
}

var _ shared.TxManager = (*collaborationWriteResources)(nil)
