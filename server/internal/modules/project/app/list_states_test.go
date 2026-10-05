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

// statesListed are the calls of user's list of project's states: the
// project's workspace, the decision, the list; none in a transaction.
func statesListed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionStateList, acme.ID, project),
		fmt.Sprintf("ListStates %s outside tx", project)}
}

// ListStates decides state.list on the project, then answers the store's
// list as it is, in the store's order, which no sort by sequence, either
// way, nor by id gives: a use case that sorted the states would answer
// another. A read opens no transaction.
func TestListStates(t *testing.T) {
	f, s := newStates()
	want := s.of(webID)
	for _, by := range []struct {
		name  string
		order func(a, b domain.State) int
	}{
		{"the sequence's", func(a, b domain.State) int { return cmp.Compare(a.Sequence, b.Sequence) }},
		{"the sequence's, reversed", func(a, b domain.State) int { return cmp.Compare(b.Sequence, a.Sequence) }},
		{"the id's", func(a, b domain.State) int { return slices.Compare(a.ID[:], b.ID[:]) }},
	} {
		if slices.IsSortedFunc(want, by.order) {
			t.Fatalf("the store's order %v is %s: a use case that sorted so would pass", want, by.name)
		}
	}
	got, err := app.NewListStates(s, f.auth).Execute(as(bob), webID)
	if err != nil || !reflect.DeepEqual(got, want) || !slices.Equal(f.log.calls, statesListed(bob, webID)) {
		t.Errorf("Execute() = %+v, %v, calls %q; want %+v, calls %q", got, err, f.log.calls, want, statesListed(bob, webID))
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; the Authorizer's 403 for one who sees it and is not its
// member. Every port's failure comes back as itself, after the calls
// before it and none after.
func TestListStatesRefuses(t *testing.T) {
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
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, statesListed(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), statesListed(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			statesListed(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			statesListed(bob, webID)[:2]},
		{"the list failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ListStates": errDisk} }, errDisk,
			statesListed(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, s := newStates()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewListStates(s, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}
