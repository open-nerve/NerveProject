package bootstrap

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	issuedomain "github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type checkedListSnapshot struct {
	t     *testing.T
	pool  *pgxpool.Pool
	tx    shared.TxManager
	calls int
}

func (s *checkedListSnapshot) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	s.calls++
	return s.tx.WithinTx(ctx, func(ctx context.Context) error {
		var isolation, readOnly string
		if err := postgres.DB(ctx, s.pool).QueryRow(ctx, "SELECT current_setting('transaction_isolation'), current_setting('transaction_read_only')").Scan(&isolation, &readOnly); err != nil {
			return err
		}
		if isolation != "repeatable read" || readOnly != "on" {
			s.t.Fatalf("snapshot %s/%s", isolation, readOnly)
		}
		return fn(ctx)
	})
}
func TestIssueListReadSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	w.adminGuest(t)
	parent := w.newWorkItem(t)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own","label_ids":`+idArray(w.bug)+`}`)
	w.send(t, "alice", http.MethodPatch, "/api/v0/issues/"+own.String(), `{"parent_id":"`+parent.String()+`"}`, 200)
	one := poolOfOne(t, w.pool.Config().ConnString())
	projects := project.Provide(one)
	workspaces := workspace.Provide(one)
	auth := access.New(access.Deps{WorkspaceRoles: workspaces.WorkspaceRoles, ProjectAccess: accessProjects{projects: projects.ProjectAccess}})
	dirs := issueProjects{projects: projects.IssueProjects}
	store := issuepg.New(one)
	listing := issueapp.NewListing(issueapp.ListingDeps{Rows: store, States: dirs, Labels: dirs, Assignees: store, Profiles: issueProfiles{profiles: identity.Provide(one).PublicProfiles}, Issues: store})
	tx := &checkedListSnapshot{t: t, pool: one, tx: postgres.NewTxManager(one, 3*time.Second, postgres.ReadOnlySnapshot())}
	u := issueapp.NewListIssues(issueapp.ListIssuesDeps{ReadTx: tx, Locks: issueapp.NewLocks(dirs, store, auth, nil, nil), Listing: listing})
	ctx, cancel := context.WithTimeout(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), 5*time.Second)
	defer cancel()
	for _, raw := range []map[string][]string{nil, {"group_by": {"label_id"}}, {"group_by": {"label_id"}, "sub_group_by": {"assignee_id"}}} {
		page, err := u.Execute(ctx, w.web, raw)
		if err != nil || page.Total != 1 {
			t.Fatal(page, err)
		}
		items := page.Data
		if raw != nil {
			items = page.Groups[0].Data
			if raw["sub_group_by"] != nil {
				items = page.Groups[0].Subgroups[0].Data
			}
		}
		if len(items) != 1 || items[0].ID != own || items[0].ParentID != nil {
			t.Fatal(items)
		}
	}
	cursor, err := issuedomain.EncodeListCursor(issuedomain.ListCursor{Order: "sort_order", Key: []byte("9999999999"), ID: own})
	if err != nil {
		t.Fatal(err)
	}
	page, err := u.Execute(ctx, w.web, map[string][]string{"order_by": {"sort_order"}, "cursor": {cursor}, "label_id": {w.bug.String()}, "group_by": {"label_id"}, "sub_group_by": {"assignee_id"}, "group_key": {w.bug.String()}, "sub_group_key": {"none"}})
	if err != nil || page.Total != 1 || page.Data == nil || len(page.Data) != 0 || page.NextCursor != "" || tx.calls != 4 {
		t.Fatal(page, err, tx.calls)
	}
}

// The personal chain includes real workspace authorization, target membership,
// ordinary project directory, SQL, parent pruning and empty-cursor totals.
func TestMemberIssueReadSnapshot(t *testing.T) {
	w := newIssueWorld(t)
	ids := memberRelations(t, w)
	own := w.newWorkItemBy(t, "walt", w.web, `{"name":"Own","assignee_ids":`+idArray(w.ids["bob"])+`,"parent_id":"`+ids[0].String()+`"}`)
	pm := queryIDs(t, w.pool, "SELECT id FROM project_members WHERE project_id=$1 AND member_id=$2", w.web, w.ids["walt"])[0]
	w.send(t, "alice", http.MethodPatch, "/api/v0/project-members/"+pm.String(), `{"role":5}`, 200)
	one := poolOfOne(t, w.pool.Config().ConnString())
	projects, workspaces := project.Provide(one), workspace.Provide(one)
	auth := access.New(access.Deps{WorkspaceRoles: workspaces.WorkspaceRoles, ProjectAccess: accessProjects{projects: projects.ProjectAccess}})
	dirs := issueProjects{projects: projects.IssueProjects}
	store := issuepg.New(one)
	listing := issueapp.NewListing(issueapp.ListingDeps{Rows: store, States: dirs, Labels: dirs, Assignees: store, Profiles: issueProfiles{profiles: identity.Provide(one).PublicProfiles}, Issues: store})
	tx := &checkedListSnapshot{t: t, pool: one, tx: postgres.NewTxManager(one, 3*time.Second, postgres.ReadOnlySnapshot())}
	u := issueapp.NewListMemberIssues(issueapp.ListMemberIssuesDeps{ReadTx: tx, Authorizer: auth, Workspaces: issueWorkspaces{directory: workspaces.WorkspaceDirectory}, Members: issueMembers{members: workspaces.WorkspaceMemberDirectory}, Projects: dirs, Listing: listing})
	ctx, cancel := context.WithTimeout(shared.WithActor(context.Background(), shared.Actor{UserID: w.ids["walt"]}), 5*time.Second)
	defer cancel()
	for _, grouped := range []bool{false, true} {
		raw := map[string][]string{"relation": {"assigned"}}
		if grouped {
			raw["group_by"] = []string{"project_id"}
		}
		page, err := u.Execute(ctx, "acme", w.ids["bob"], raw)
		if err != nil || page.Total != 1 {
			t.Fatal(page, err)
		}
		rows := page.Data
		if grouped {
			rows = page.Groups[0].Data
		}
		if len(rows) != 1 || rows[0].ID != own || rows[0].ParentID != nil {
			t.Fatal(rows)
		}
	}
	cursor, err := issuedomain.EncodeListCursor(issuedomain.ListCursor{Order: "sort_order", Key: []byte("9999999999"), ID: own})
	if err != nil {
		t.Fatal(err)
	}
	page, err := u.Execute(ctx, "acme", w.ids["bob"], map[string][]string{"relation": {"assigned"}, "order_by": {"sort_order"}, "cursor": {cursor}})
	if err != nil || page.Total != 1 || len(page.Data) != 0 || page.NextCursor != "" || tx.calls != 3 {
		t.Fatal(page, err, tx.calls)
	}
}
