package app_test

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// labelsListed are the calls of user's list of project's labels: the
// project's workspace, the decision, the list; none in a transaction.
func labelsListed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionLabelList, acme.ID, project),
		fmt.Sprintf("ListLabels %s outside tx", project)}
}

// ListLabels decides label.list on the project, then answers the store's
// list as it is, in the store's order, which no sort by sort order, either
// way, nor by id gives: a use case that sorted the labels would answer
// another. ops, archived, has its labels listed as any other's (3.19). A
// read opens no transaction.
func TestListLabels(t *testing.T) {
	for _, project := range []uuid.UUID{webID, opsID} {
		f, l := newLabels()
		want := l.of(project)
		if project == webID {
			for _, by := range []struct {
				name  string
				order func(a, b domain.Label) int
			}{
				{"the sort order's", func(a, b domain.Label) int { return cmp.Compare(a.SortOrder, b.SortOrder) }},
				{"the sort order's, reversed", func(a, b domain.Label) int { return cmp.Compare(b.SortOrder, a.SortOrder) }},
				{"the id's", func(a, b domain.Label) int { return slices.Compare(a.ID[:], b.ID[:]) }},
			} {
				if slices.IsSortedFunc(want, by.order) {
					t.Fatalf("the store's order %v is %s: a use case that sorted so would pass", want, by.name)
				}
			}
		}
		got, err := app.NewListLabels(l, f.auth).Execute(as(bob), project)
		if err != nil || len(got) == 0 || !reflect.DeepEqual(got, want) || !slices.Equal(f.log.calls, labelsListed(bob, project)) {
			t.Errorf("Execute(%s) = %+v, %v, calls %q; want %+v, calls %q", project, got, err, f.log.calls, want, labelsListed(bob, project))
		}
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; the Authorizer's 403 for one who sees it and is not its
// member. Every port's failure comes back as itself, after the calls
// before it and none after.
func TestListLabelsRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		fail  func(f *writeFixture)
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, nil, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), nil, domain.ErrNotFound, []string{"ProjectWorkspace " + uuid.Nil().String() + " outside tx"}},
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, labelsListed(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), labelsListed(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			labelsListed(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			labelsListed(bob, webID)[:2]},
		{"the list failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ListLabels": errDisk} }, errDisk,
			labelsListed(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, l := newLabels()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewListLabels(l, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}
