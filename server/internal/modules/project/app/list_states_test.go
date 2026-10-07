package app_test

import (
	"cmp"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// statesListed are the calls of user's list of project's states: the
// project's workspace, the decision, the list; none in a transaction.
func statesListed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionStateList, acme.ID, project),
		fmt.Sprintf("ListStates %s outside tx", project)}
}

// sortOrder is a sort of a list, named for the failure.
type sortOrder[T any] struct {
	name  string
	order func(a, b T) int
}

// sortedByNone fails the test when list, a store's order that a use case
// answers as it is, is in the order of one of sorts: a use case that
// sorted it so would pass.
func sortedByNone[T any](t *testing.T, list []T, sorts []sortOrder[T]) {
	t.Helper()
	for _, by := range sorts {
		if slices.IsSortedFunc(list, by.order) {
			t.Fatalf("the store's order %v is %s: a use case that sorted so would pass", list, by.name)
		}
	}
}

// ListStates decides state.list on the project, then answers the store's
// list as it is, in the store's order, which no sort by sequence, either
// way, nor by id gives: a use case that sorted the states would answer
// another. A read opens no transaction.
func TestListStates(t *testing.T) {
	f, s := newStates()
	want := s.of(webID)
	sortedByNone(t, want, []sortOrder[domain.State]{
		{"the sequence's", func(a, b domain.State) int { return cmp.Compare(a.Sequence, b.Sequence) }},
		{"the sequence's, reversed", func(a, b domain.State) int { return cmp.Compare(b.Sequence, a.Sequence) }},
		{"the id's", func(a, b domain.State) int { return slices.Compare(a.ID[:], b.ID[:]) }},
	})
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
	for _, tt := range readRefusals(statesListed, "the list", "ListStates") {
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
