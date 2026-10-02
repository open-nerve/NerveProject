package app_test

import (
	"context"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListMembers is the project's active memberships, in storeOrder: an order
// of the store's that no sort by account gives.
func (f *fakeStore) ListMembers(ctx context.Context, projectID uuid.UUID) ([]domain.Member, error) {
	f.log.add(ctx, "ListMembers %s", projectID)
	if err := f.fail("ListMembers"); err != nil {
		return nil, err
	}
	var out []domain.Member
	if f.missing {
		return out, nil
	}
	members := f.projects[projectID].members
	var active []uuid.UUID
	for user, m := range members {
		if m.Active {
			active = append(active, user)
		}
	}
	for _, user := range storeOrder(active) {
		m := members[user]
		out = append(out, domain.Member{ID: m.ID, ProjectID: projectID, MemberID: user, Role: m.Role, CreatedAt: now})
	}
	return out, nil
}

// storeOrder is accounts by their ids, the first moved last: of three or
// more, an order that neither an ascending nor a descending sort by
// account gives.
func storeOrder(accounts []uuid.UUID) []uuid.UUID {
	sorted := slices.SortedFunc(slices.Values(accounts), byAccount)
	if len(sorted) == 0 {
		return nil
	}
	return slices.Concat(sorted[1:], sorted[:1])
}

// byAccount orders accounts by their ids.
func byAccount(a, b uuid.UUID) int { return strings.Compare(a.String(), b.String()) }

// listed are the calls of user's list of project's members: the project's
// workspace, the decision, the list; none in a transaction.
func listed(user, project uuid.UUID) []string {
	return []string{"ProjectWorkspace " + project.String() + " outside tx",
		fmt.Sprintf("Authorize %s %s on %s/%s outside tx", user, domain.ActionMemberList, acme.ID, project),
		fmt.Sprintf("ListMembers %s outside tx", project)}
}

// ListProjectMembers decides project_member.list on the project, then
// answers the store's list as it is, in the store's order, which is no
// sort by account, ascending or descending: web's three active members,
// not dave, whose membership ended. A read opens no transaction.
func TestListProjectMembers(t *testing.T) {
	order := storeOrder([]uuid.UUID{alice, bob, carol})
	if slices.IsSortedFunc(order, byAccount) || slices.IsSortedFunc(order, func(a, b uuid.UUID) int { return byAccount(b, a) }) {
		t.Fatalf("the store's order %v is a sort by account: a use case that sorted would pass", order)
	}
	f := newWrites()
	got, err := app.NewListProjectMembers(f.store, f.auth).Execute(as(bob), webID)
	members := f.store.projects[webID].members
	var want []domain.Member
	for _, user := range order {
		want = append(want, domain.Member{ID: members[user].ID, ProjectID: webID, MemberID: user, Role: members[user].Role, CreatedAt: now})
	}
	if err != nil || !reflect.DeepEqual(got, want) || !slices.Equal(f.log.calls, listed(bob, webID)) {
		t.Errorf("Execute() = %+v, %v, calls %q; want %+v, calls %q", got, err, f.log.calls, want, listed(bob, webID))
	}
}

// Refusals, each in its place: no caller; a project not there, or not
// visible: 404; a member's 403 from the Authorizer. Every port's failure
// comes back as itself, after the calls before it and none after.
func TestListProjectMembersRefuses(t *testing.T) {
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
		{"not seen", as(erin), webID, nil, domain.ErrNotFound, listed(erin, webID)[:2]},
		{"forbidden", as(alice), webID, nil, shared.Forbidden(), listed(alice, webID)[:2]},
		{"the project failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ProjectWorkspace": errDisk} }, errDisk,
			listed(bob, webID)[:1]},
		{"the decision failing", as(bob), webID, func(f *writeFixture) { f.auth.errs[grantKey{bob, acme.ID}] = errDisk }, errDisk,
			listed(bob, webID)[:2]},
		{"the list failing", as(bob), webID, func(f *writeFixture) { f.store.errs = map[string]error{"ListMembers": errDisk} }, errDisk,
			listed(bob, webID)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newWrites()
			if tt.fail != nil {
				tt.fail(f)
			}
			got, err := app.NewListProjectMembers(f.store, f.auth).Execute(tt.ctx, tt.id)
			if !refusedAs(err, tt.want) || got != nil || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %+v, %v, calls %q; want %v, calls %q", got, err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}
