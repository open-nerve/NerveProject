package bootstrap

import (
	"fmt"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// accessChange is a change of an account's access through the API: alice's call, method on path with body.
type accessChange struct {
	name, method, path, body string
	want                     int
}

// recheckCase is a comment of alice's on a work item mentioning accounts by name, the changes of their access made
// while its fan-out waits for the work item's row, and the accounts who are then sent their mention.
type recheckCase struct {
	name               string
	item               uuid.UUID
	mentioned, receive []string
	changes            []accessChange
}

// The fan-out decides who reads a work item's notifications when it runs, after it has locked the work item's row (M7
// P1b spec 2.4, 2.7, 2.9): an account mentioned on it whose access changes after the write, while its fan-out waits
// for the row, is decided on as he is then, as Authorize would decide issue.read, and the others are sent theirs.
// Each comment's fan-out is held until a transaction holds the work item's row, then released, and waits at its
// ShareIssue behind it (the statement probe), when the changes are made through the API; the transaction then lets
// go. On alice's work item of Web, a public project: bob, removed from Web, sees Web as acme's member and may not read
// it (Forbidden); walt, whose membership of acme ended, sees nothing of acme; olga, made Web's guest, sees only the
// work items she created; wanda, acme's admin made Web's guest, too: the access rule of issue.read allows a project
// guest, and the guest rule is the project role's alone, the workspace's admin included (OwnOnly, as decideProject
// sets it), so she may read Web's work items, hers only, and is sent nothing of alice's. keith, Web's member still, is
// sent his mention. On wanda's work item of Ops, a private project: bob, removed from Ops, sees nothing of it; wanda,
// made Ops' guest, is sent hers, the work item being one she created (the issue module's HiddenFrom over the creator
// the work item's columns name); keith is sent his.
func TestTheFanoutDecidesItsReceiversWhenItRuns(t *testing.T) {
	w := newHeldFanoutWorld(t, func(*config.Config) {})
	w.tokens["keith"] = registerAccount(t, w.contract, w.base, "keith@example.com").AccessToken
	w.ids["keith"] = accountID(t, w.contract, w.base, w.tokens["keith"])
	for _, name := range []string{"olga", "keith"} {
		inWorkspaceOf(t, w.pool, w.web, w.ids[name], w.ids["alice"], shared.RoleMember)
	}
	ops := createdProjectFrom(t, w.contract, w.base, w.tokens["alice"], "acme", `{"name":"Ops","identifier":"OPS","network":0}`)
	for _, m := range []struct {
		project uuid.UUID
		names   []string
	}{{w.web, []string{"walt", "olga", "wanda", "keith"}}, {ops, []string{"bob", "wanda", "keith"}}} {
		for _, name := range m.names {
			role := shared.RoleMember
			if name == "wanda" {
				role = shared.RoleAdmin // acme's admin joins a project as its admin (M3 design 3.5)
			}
			w.makeMember(t, name, m.project, role)
		}
	}
	public, private := w.newWorkItemBy(t, "alice", w.web, `{"name":"Public"}`), w.newWorkItemBy(t, "wanda", ops, `{"name":"Private"}`)
	releaseFanouts(t, w.pool)
	waitForFanouts(t, w.pool)

	member := func(project uuid.UUID, name string) string {
		return "/api/v0/project-members/" + queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id = $1 AND member_id = $2 AND is_active AND deleted_at IS NULL",
			project, w.ids[name])[0].String()
	}
	walt := "/api/v0/workspace-members/" + queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id = $1 AND member_id = $2", w.acme,
		w.ids["walt"])[0].String()
	guest := `{"role":5}`
	for _, c := range []recheckCase{
		{name: "a public project's", item: public, mentioned: []string{"bob", "walt", "olga", "wanda", "keith"}, receive: []string{"keith"},
			changes: []accessChange{{"bob removed", http.MethodDelete, member(w.web, "bob"), "", http.StatusNoContent},
				{"walt out of acme", http.MethodDelete, walt, "", http.StatusNoContent},
				{"olga made a guest", http.MethodPatch, member(w.web, "olga"), guest, http.StatusOK},
				{"wanda made a guest", http.MethodPatch, member(w.web, "wanda"), guest, http.StatusOK}}},
		{name: "a private project's", item: private, mentioned: []string{"bob", "wanda", "keith"}, receive: []string{"wanda", "keith"},
			changes: []accessChange{{"bob removed", http.MethodDelete, member(ops, "bob"), "", http.StatusNoContent},
				{"wanda made a guest", http.MethodPatch, member(ops, "wanda"), guest, http.StatusOK}}},
	} {
		t.Run(c.name, func(t *testing.T) {
			mentioned := make([]uuid.UUID, len(c.mentioned))
			for i, name := range c.mentioned {
				mentioned[i] = w.ids[name]
			}
			var comment answeredComment
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, commentsOf(c.item), fmt.Sprintf(`{"comment_html":%q}`, mentionHTML(mentioned...)),
				http.StatusCreated), &comment)
			held := holding(t, w.pool, "SELECT 1 FROM issues WHERE id = $1 FOR NO KEY UPDATE", c.item)
			if n := releaseFanouts(t, w.pool); n != 1 {
				t.Fatalf("released %d fan-outs; want the comment's", n)
			}
			pgtest.WaitForStatementLockWaitBehind(t, w.pool, held.Conn().PgConn(), 5*time.Second, "ShareIssue")
			for _, change := range c.changes {
				if status, body := call(t, w.contract, change.method, w.base+change.path, w.tokens["alice"], change.body); status != change.want {
					t.Fatalf("%s: %s %s = %d %s; want %d", change.name, change.method, change.path, status, body, change.want)
				}
			}
			if err := held.Rollback(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			waitForFanouts(t, w.pool)
			if runs := fanoutRunsOf(t, w.pool, "issue.comment_created", comment.ID); !reflect.DeepEqual(runs, []fanoutRun{{"completed", 1, 0}}) {
				t.Fatalf("the comment's fan-outs %+v; want one completed in its first attempt", runs)
			}
			job := fanoutOf(t, w.pool, "issue.comment_created", comment.ID)
			var want []notified
			for _, name := range c.receive {
				want = append(want, notified{w.ids[name], "mentioned", int(mentionedAt(t, job, w.ids[name])), "comment"})
			}
			slices.SortFunc(want, func(a, b notified) int { return a.Seq - b.Seq })
			if got := notifiedOf(t, w.pool, c.item); !reflect.DeepEqual(got, want) {
				t.Errorf("the notifications of %s %+v; want %v's mentions alone, %+v", c.item, got, c.receive, want)
			}
		})
	}
}
