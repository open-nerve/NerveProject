package bootstrap

import (
	"context"
	"maps"
	"net/http"
	"net/url"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The workspace's list across its projects (H13's list, M7 P3a spec 2.2,
// 2.3, 4), on the real app, in newWorkspaceListWorld's world. Walt, the
// caller, is Web's guest who sees only his own and Alpha's member: his
// list is a7, a3, w3, a2, w2, a1 by -created_at. It leaves out w1, w4 and
// w5, which are not his, in Web; Open, which he is no member of though it is
// public; the work item in Alpha's intake (a6), the archived (a4) and the
// deleted (a5) ones; Shelf, archived; Former, which he left. w1, a4, a5,
// a6 and Shelf's and Former's work items are each his or assigned to him,
// so that a view of his would show it if it were let in. A cursor only
// says where a page starts: the filters and the projects seen are those of
// the request, in one read-only snapshot.
func TestWorkspaceIssueList(t *testing.T) {
	w := newWorkspaceListWorld(t)
	everything := []uuid.UUID{w.a7, w.a3, w.w3, w.a2, w.w2, w.a1}
	t.Run("default views", func(t *testing.T) {
		p := w.listed(t, "walt", "", 6, false, everything...)
		// w2's parent, w1, is not his: the row does not name it; w3's, w2, is.
		if p.Data[4].ParentID != nil || p.Data[2].ParentID == nil || *p.Data[2].ParentID != w.w2 {
			t.Fatal("parents", p.Data[4], p.Data[2])
		}
		w.listed(t, "walt", "assignee_id=me", 2, false, w.a2, w.w2)
		w.listed(t, "walt", "created_by_id=me", 3, false, w.a3, w.w3, w.w2)
		// Subscribed by making a3, w3, w2, by being assigned a2, by hand a1.
		w.listed(t, "walt", "subscriber_id=me", 5, false, w.a3, w.w3, w.a2, w.w2, w.a1)
	})
	t.Run("sub-issue counts", func(t *testing.T) {
		// A row counts the children its reader sees, project by project, in
		// either way the count is read: w2's, in Web, his own w3 and not
		// bob's w5; a3's, in Alpha, bob's a7. Alice, Web's admin, counts both
		// of w2's.
		want := map[uuid.UUID]int{w.a7: 0, w.a3: 1, w.w3: 0, w.a2: 0, w.w2: 1, w.a1: 0}
		for _, query := range []string{"", "order_by=-sub_issues_count"} {
			if got := w.subIssueCounts(t, "walt", query); !maps.Equal(got, want) {
				t.Errorf("walt ?%s: %v; want %v", query, got, want)
			}
			if got := w.subIssueCounts(t, "alice", query); got[w.w2] != 2 || got[w.a3] != 1 {
				t.Errorf("alice ?%s: w2 %d, a3 %d; want 2, 1", query, got[w.w2], got[w.a3])
			}
		}
	})
	t.Run("projects not his", func(t *testing.T) {
		// A project filter only narrows the projects he is a member of.
		for _, p := range []uuid.UUID{w.open, w.shelf, w.former} {
			w.listed(t, "walt", "project_id="+p.String(), 0, false)
		}
		// Acme's admin, wanda, sees Alpha's alone, the project she is an
		// admin of: neither Web nor Open, public both.
		w.all(t, "wanda", w.a7, w.a3, w.a2, w.a1)
		w.listed(t, "wanda", "project_id="+w.open.String(), 0, false)
		w.listed(t, "wanda", "project_id="+w.web.String(), 0, false)
	})
	t.Run("filters", func(t *testing.T) {
		// w1, in Web and not his, is urgent, dated and labelled as a2 is,
		// and mentions bob as a3 does; w4, bob's, is assigned to bob as a7.
		for _, tc := range []struct {
			query string
			want  []uuid.UUID
		}{
			{"project_id=" + w.alpha.String(), []uuid.UUID{w.a7, w.a3, w.a2, w.a1}},
			{"project_id=" + w.web.String(), []uuid.UUID{w.w3, w.w2}},
			{"state_id=" + w.alphaDone.String(), []uuid.UUID{w.a3}},
			{"state_group=completed", []uuid.UUID{w.a3}},
			{"priority=urgent", []uuid.UUID{w.a2}},
			{"priority=high", []uuid.UUID{w.w2}},
			{"label_id=" + w.ops.String() + "," + w.bug.String(), []uuid.UUID{w.a2, w.w2}},
			{"assignee_id=" + w.ids["bob"].String(), []uuid.UUID{w.a7}},
			{"created_by_id=" + w.ids["bob"].String(), []uuid.UUID{w.a7, w.a2}},
			{"mention_id=" + w.ids["bob"].String(), []uuid.UUID{w.a3}},
			{"subscriber_id=" + w.ids["walt"].String(), []uuid.UUID{w.a3, w.w3, w.a2, w.w2, w.a1}},
			{"start_date=2026-10-10..2026-10-13", []uuid.UUID{w.a2}},
			{"target_date=2026-10-14..2026-10-16", []uuid.UUID{w.a2}},
			{"sub_issue=false", []uuid.UUID{w.a3, w.a2, w.a1}},
			{"project_id=" + w.alpha.String() + "&assignee_id=me", []uuid.UUID{w.a2}},
		} {
			t.Run(tc.query, func(t *testing.T) {
				w.listed(t, "walt", tc.query, int64(len(tc.want)), false, tc.want...)
			})
		}
	})
	t.Run("paging", func(t *testing.T) {
		var seen []uuid.UUID
		query := "limit=2"
		for page := 0; page < 3; page++ {
			p := w.listed(t, "walt", query, 6, page < 2, everything[2*page:2*page+2]...)
			for _, row := range p.Data {
				seen = append(seen, row.ID)
			}
			if p.NextCursor != nil {
				query = "limit=2&cursor=" + url.QueryEscape(*p.NextCursor)
			}
		}
		if !slices.Equal(seen, everything) {
			t.Fatal(seen)
		}
		// Past his last row, the page is empty and still counts his.
		w.listed(t, "walt", "cursor="+url.QueryEscape(w.cursorAt(t, w.a1)), 6, false)
	})
	t.Run("tampered cursor", func(t *testing.T) {
		// A cursor made at a row he does not see starts after it, on his rows.
		w.listed(t, "walt", "cursor="+url.QueryEscape(w.cursorAt(t, w.w4)), 6, false, w.a2, w.w2, w.a1)
		w.listed(t, "walt", "cursor="+url.QueryEscape(w.cursorAt(t, w.w1)), 6, false, w.a1)
	})
	t.Run("refusal order", func(t *testing.T) {
		// A bad query is refused before the workspace is looked at: to
		// olga, no member of acme, and in a workspace that does not exist.
		for _, tc := range []struct{ actor, path, want string }{
			{"olga", workspaceIssuesPath + "?group_by=priority", "group_by not_allowed"},
			{"olga", workspaceIssuesPath + "?cycle_id=" + uuid.NewV7().String(), "cycle_id not_allowed"},
			{"alice", "/api/v0/workspaces/absent/issues?relation=assigned", "relation not_allowed"},
		} {
			if got := oneError(t, []byte(w.send(t, tc.actor, http.MethodGet, tc.path, "", http.StatusUnprocessableEntity))); got != tc.want {
				t.Errorf("%s %s: %s; want %s", tc.actor, tc.path, got, tc.want)
			}
		}
		absent := w.send(t, "alice", http.MethodGet, "/api/v0/workspaces/absent/issues", "", http.StatusNotFound)
		unseen := w.send(t, "olga", http.MethodGet, workspaceIssuesPath, "", http.StatusNotFound)
		if problemCode(t, []byte(absent)) != "workspace.not_found" || !reflect.DeepEqual(problemObject(t, unseen), problemObject(t, absent)) {
			t.Fatal(unseen, absent)
		}
		// A single-valued parameter given twice is refused as it is bound,
		// before the token is: alike with one and with none. It breaks the
		// contract's request shape, so only the answer is checked.
		for _, token := range []string{w.tokens["walt"], ""} {
			req := newRequest(t, http.MethodGet, w.base+workspaceIssuesPath+"?priority=urgent&priority=high", token, nil)
			res, body := send(t, req)
			w.contract.CheckResponse(t, req, res)
			if res.StatusCode != http.StatusUnprocessableEntity || oneError(t, body) != "priority not_allowed" {
				t.Errorf("priority given twice = %d %s", res.StatusCode, body)
			}
		}
	})
	t.Run("read-only snapshot", func(t *testing.T) {
		workspaceListSnapshot(t, w, everything)
	})
	t.Run("workspace admin as project guest", func(t *testing.T) {
		// Acme's admin, walt is still Web's guest who sees only his own.
		wm := queryIDs(t, w.pool, "SELECT id FROM workspace_members WHERE workspace_id=$1 AND member_id=$2", w.acme, w.ids["walt"])[0]
		w.send(t, "alice", http.MethodPatch, "/api/v0/workspace-members/"+wm.String(), `{"role":20}`, http.StatusOK)
		w.all(t, "walt", everything...)
		w.listed(t, "walt", "project_id="+w.open.String(), 0, false)
	})
}

// workspaceListSnapshot runs the use case on a pool of one connection, its
// read transaction checked (checkedListSnapshot), as the project's and the
// member's lists' snapshot tests do: the workspace, the decision, Visible,
// the directories, the rows, the pruned parent, and the total of the empty
// page past the end each come from the one read-only REPEATABLE READ
// transaction of its call, or the call waits for the connection and fails.
func workspaceListSnapshot(t *testing.T, w workspaceListWorld, want []uuid.UUID) {
	t.Helper()
	one := poolOfOne(t, w.pool.Config().ConnString())
	projects, workspaces := project.Provide(one, nil), workspace.Provide(one, nil)
	auth := access.New(access.Deps{WorkspaceRoles: workspaces.WorkspaceRoles, ProjectAccess: accessProjects{projects: projects.ProjectAccess}})
	dirs := issueProjects{projects: projects.IssueProjects}
	store := issuepg.New(one)
	listing := issueapp.NewListing(issueapp.ListingDeps{Attachments: attachmentCounterOn(one), Rows: store, States: dirs, Labels: dirs, Assignees: store, Profiles: issueProfiles{profiles: identity.Provide(one).PublicProfiles}, Issues: store})
	tx := &checkedListSnapshot{t: t, pool: one, tx: postgres.NewTxManager(one, 3*time.Second, postgres.ReadOnlySnapshot())}
	u := issueapp.NewListWorkspaceIssues(issueapp.ListWorkspaceIssuesDeps{ReadTx: tx, Authorizer: auth, Workspaces: issueWorkspaces{directory: workspaces.WorkspaceDirectory}, Projects: dirs, Listing: listing})
	ctx, cancel := context.WithTimeout(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), 5*time.Second)
	defer cancel()
	page, err := u.Execute(ctx, "acme", nil)
	if err != nil || page.Total != int64(len(want)) || len(page.Data) != len(want) {
		t.Fatal(page, err)
	}
	for n, item := range page.Data {
		if item.ID != want[n] || item.ID == w.w2 && item.ParentID != nil {
			t.Fatal(n, item)
		}
	}
	page, err = u.Execute(ctx, "acme", map[string][]string{"cursor": {w.cursorAt(t, w.a1)}})
	if err != nil || page.Total != int64(len(want)) || page.Data == nil || len(page.Data) != 0 || page.NextCursor != "" || tx.calls != 2 {
		t.Fatal(page, err, tx.calls)
	}
}
