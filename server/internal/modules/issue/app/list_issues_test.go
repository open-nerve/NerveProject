package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type listSnapshot struct {
	tx    fakeTx
	ports *listingPorts
}

func (s listSnapshot) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return s.tx.WithinTx(ctx, func(ctx context.Context) error { s.ports.ctx = ctx; return fn(ctx) })
}
func projectList(t *testing.T) (*app.ListIssues, *world, *listingPorts) {
	t.Helper()
	_, w, _ := newGet()
	l, p, _ := listingFixture(t)
	p.pages = [][]app.ListRecord{{listRecord(bob, nil)}}
	u := app.NewListIssues(app.ListIssuesDeps{Locks: w.locks, Listing: l, ReadTx: listSnapshot{fakeTx{log: w.log}, p}})
	return u, w, p
}
func TestListIssues(t *testing.T) {
	for _, archived := range []bool{false, true} {
		t.Run(map[bool]string{false: "active", true: "archived"}[archived], func(t *testing.T) {
			u, w, p := projectList(t)
			w.auth.grants = []shared.Grant{{ProjectRole: shared.RoleGuest, OwnOnly: true}}
			raw := map[string][]string{"order_by": {"sort_order"}}
			action := domain.ActionRead
			if archived {
				raw["archived"] = []string{"true"}
				action = domain.ActionArchive
			}
			page, err := u.Execute(as(alice), web, raw)
			if err != nil || page.Total != 3 || len(page.Data) != 1 || page.Data[0].ID != bob {
				t.Fatal(page, err)
			}
			want := []string{"Begin", line("Find", web), line("Authorize", alice, action, acme, web)}
			if !slices.Equal(w.log.calls, want) {
				t.Fatal(w.log.calls, want)
			}
			read := p.reads[0]
			if read.ActorID != alice || read.TargetMemberID != nil || read.Query.Relation != "" || read.Query.Filters.Archived != archived || !reflect.DeepEqual(read.Directories.Projects, []app.VisibleProject{{ID: web, Role: shared.RoleGuest, OwnOnly: true}}) {
				t.Fatal(read)
			}
		})
	}
}
func TestListIssuesRefusals(t *testing.T) {
	for _, tt := range []struct {
		name  string
		setup func(*world, *listingPorts)
		raw   map[string][]string
		ctx   context.Context
		want  error
		calls int
	}{
		{name: "no actor", ctx: context.Background(), want: shared.Unauthenticated()},
		{name: "unknown query", ctx: as(alice), raw: map[string][]string{"target_member_id": {bob.String()}}, want: shared.Invalid(shared.FieldError{Field: "target_member_id", Code: "not_allowed", Message: "is not allowed for this list"})},
		{name: "missing project", ctx: as(alice), setup: func(w *world, _ *listingPorts) { w.project.found = false }, want: domain.ErrProjectNotFound, calls: 2},
		{name: "invisible", ctx: as(alice), setup: func(w *world, _ *listingPorts) { w.auth.answers = []error{shared.ErrNotVisible} }, want: domain.ErrProjectNotFound, calls: 3},
		{name: "forbidden", ctx: as(alice), setup: func(w *world, _ *listingPorts) { w.auth.answers = []error{shared.Forbidden()} }, want: shared.Forbidden(), calls: 3},
		{name: "find", ctx: as(alice), setup: func(w *world, _ *listingPorts) { w.log.fail = "Find" }, want: errDisk, calls: 2},
		{name: "authorize", ctx: as(alice), setup: func(w *world, _ *listingPorts) { w.log.fail = "Authorize" }, want: errDisk, calls: 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, w, p := projectList(t)
			if tt.setup != nil {
				tt.setup(w, p)
			}
			got, err := u.Execute(tt.ctx, web, tt.raw)
			if !reflect.DeepEqual(err, tt.want) || !reflect.DeepEqual(got, app.IssueListPage{}) || len(w.log.calls) != tt.calls {
				t.Fatal(got, err, w.log.calls, tt.want)
			}
		})
	}
	for _, fail := range []string{"states", "labels", "assignees", "profiles", "flat", "parents", "total"} {
		t.Run(fail, func(t *testing.T) {
			u, _, p := projectList(t)
			p.fail = fail
			raw := map[string][]string{"order_by": {"sort_order"}}
			if fail == "parents" {
				u, w, pp := projectList(t)
				p = pp
				p.fail = fail
				w.auth.grants = []shared.Grant{ownOnly}
				p.pages[0][0].Issue.ParentID = &carol
				_, err := u.Execute(as(alice), web, raw)
				if !errors.Is(err, errDisk) {
					t.Fatal(err)
				}
				return
			}
			if fail == "total" {
				p.pages = [][]app.ListRecord{nil, nil}
				cursor, _ := domain.EncodeListCursor(domain.ListCursor{Order: "sort_order", Key: []byte("1"), ID: bob})
				raw["cursor"] = []string{cursor}
			}
			_, err := u.Execute(as(alice), web, raw)
			if !errors.Is(err, errDisk) {
				t.Fatal(err)
			}
		})
	}
	_, w, p := projectList(t)
	l, _, _ := listingFixture(t)
	u := app.NewListIssues(app.ListIssuesDeps{Locks: w.locks, Listing: l, ReadTx: activityReadFailureTx{}})
	if _, err := u.Execute(as(alice), web, nil); !errors.Is(err, errDisk) || len(w.log.calls) != 0 || len(p.reads) != 0 {
		t.Fatal(err, w.log.calls)
	}
}
