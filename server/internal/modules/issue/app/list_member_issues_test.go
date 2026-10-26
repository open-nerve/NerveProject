package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type memberListDirectory struct {
	app.VisibleProjects
	log      *callLog
	exists   bool
	projects []app.VisibleProject
}

func (f *memberListDirectory) MemberExists(ctx context.Context, wid, uid uuid.UUID) (bool, error) {
	return f.exists, f.log.add(ctx, "MemberExists", wid, uid)
}
func (f *memberListDirectory) Visible(ctx context.Context, wid, uid uuid.UUID) ([]app.VisibleProject, error) {
	return f.projects, f.log.add(ctx, "Visible", wid, uid)
}
func memberList(t *testing.T) (*app.ListMemberIssues, *world, *listingPorts, *memberListDirectory) {
	t.Helper()
	_, w, _ := newGet()
	l, p, _ := listingFixture(t)
	p.pages = [][]app.ListRecord{{listRecord(carol, nil)}}
	d := &memberListDirectory{log: w.log, exists: true, projects: []app.VisibleProject{{ID: web, Role: shared.RoleGuest, OwnOnly: true}}}
	u := app.NewListMemberIssues(app.ListMemberIssuesDeps{ReadTx: listSnapshot{fakeTx{log: w.log}, p}, Authorizer: w.auth, Workspaces: &fakeWorkspaces{log: w.log, slug: "acme", id: acme}, Members: d, Projects: d, Listing: l})
	return u, w, p, d
}
func TestListMemberIssues(t *testing.T) {
	for _, relation := range []string{"assigned", "created", "subscribed"} {
		t.Run(relation, func(t *testing.T) {
			u, w, p, d := memberList(t)
			page, err := u.Execute(as(alice), "acme", bob, map[string][]string{"relation": {relation}, "order_by": {"sort_order"}})
			if err != nil || page.Total != 3 || len(page.Data) != 1 || page.Data[0].ID != carol {
				t.Fatal(page, err)
			}
			want := []string{"Begin", line("WorkspaceBySlug", "acme"), line("Authorize", alice, domain.ActionListMemberIssues, acme, uuid.Nil()), line("MemberExists", acme, bob), line("Visible", acme, alice)}
			if !slices.Equal(w.log.calls, want) {
				t.Fatal(w.log.calls, want)
			}
			read := p.reads[0]
			if read.ActorID != alice || read.TargetMemberID == nil || *read.TargetMemberID != bob || read.Query.Relation != relation || !reflect.DeepEqual(read.Directories.Projects, d.projects) {
				t.Fatal(read)
			}
		})
	}
}
func TestListMemberIssuesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name, fail string
		ctx        context.Context
		slug       string
		raw        map[string][]string
		auth       error
		missing    bool
		want       error
		calls      int
	}{
		{name: "no actor", ctx: context.Background(), want: shared.Unauthenticated()},
		{name: "missing relation", ctx: as(alice), raw: map[string][]string{}, want: memberRelationRequired()},
		{name: "missing workspace", ctx: as(alice), slug: "missing", want: domain.ErrWorkspaceNotFound, calls: 2},
		{name: "hidden workspace", ctx: as(alice), auth: shared.ErrNotVisible, want: domain.ErrWorkspaceNotFound, calls: 3},
		{name: "guest", ctx: as(alice), auth: shared.Forbidden(), want: shared.Forbidden(), calls: 3},
		{name: "missing target", ctx: as(alice), missing: true, want: domain.ErrMemberNotFound, calls: 4},
		{name: "workspace error", ctx: as(alice), fail: "WorkspaceBySlug", want: errDisk, calls: 2},
		{name: "auth error", ctx: as(alice), fail: "Authorize", want: errDisk, calls: 3},
		{name: "target error", ctx: as(alice), fail: "MemberExists", want: errDisk, calls: 4},
		{name: "projects error", ctx: as(alice), fail: "Visible", want: errDisk, calls: 5},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, w, _, d := memberList(t)
			w.log.fail = tt.fail
			d.exists = !tt.missing
			w.auth.answers = []error{tt.auth}
			slug := tt.slug
			if slug == "" {
				slug = "acme"
			}
			raw := tt.raw
			if raw == nil {
				raw = map[string][]string{"relation": {"assigned"}}
			}
			page, err := u.Execute(tt.ctx, slug, bob, raw)
			if !reflect.DeepEqual(err, tt.want) || !reflect.DeepEqual(page, app.IssueListPage{}) || len(w.log.calls) != tt.calls {
				t.Fatal(page, err, tt.want, w.log.calls)
			}
		})
	}
	for _, field := range []string{"target_member_id", "user_id", "archived", "sub_group_by"} {
		u, w, _, _ := memberList(t)
		_, err := u.Execute(as(alice), "acme", bob, map[string][]string{"relation": {"assigned"}, field: {"bad"}})
		if err == nil || len(w.log.calls) != 0 {
			t.Fatal(field, err, w.log.calls)
		}
	}
	for _, fail := range []string{"states", "labels", "assignees", "profiles", "flat", "parents", "total"} {
		t.Run(fail, func(t *testing.T) {
			u, _, p, _ := memberList(t)
			p.fail = fail
			raw := map[string][]string{"relation": {"assigned"}, "order_by": {"sort_order"}}
			if fail == "parents" {
				p.pages[0][0].Issue.ParentID = &bob
			}
			if fail == "total" {
				p.pages = [][]app.ListRecord{nil, nil}
				cursor, _ := domain.EncodeListCursor(domain.ListCursor{Order: "sort_order", Key: []byte("1"), ID: bob})
				raw["cursor"] = []string{cursor}
			}
			if _, err := u.Execute(as(alice), "acme", bob, raw); !errors.Is(err, errDisk) {
				t.Fatal(err)
			}
		})
	}
	_, w, p, d := memberList(t)
	l, _, _ := listingFixture(t)
	u := app.NewListMemberIssues(app.ListMemberIssuesDeps{ReadTx: activityReadFailureTx{}, Authorizer: w.auth, Members: d, Projects: d, Listing: l})
	if _, err := u.Execute(as(alice), "acme", bob, map[string][]string{"relation": {"created"}}); !errors.Is(err, errDisk) || len(w.log.calls) != 0 || len(p.reads) != 0 {
		t.Fatal(err, w.log.calls)
	}
}

func memberRelationRequired() error {
	_, err := domain.ParseListQuery(nil, domain.MemberList, alice)
	return err
}
