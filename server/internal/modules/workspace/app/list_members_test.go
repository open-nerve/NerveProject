package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// membersFixture is ListWorkspaceMembers over fakes sharing one log. acme's
// memberships: alice (admin), bob (member), carol's ended one (guest, her
// account deactivated); beta's: bob alone. The Authorizer gives alice admin
// in acme, bob member in acme and guest in beta.
type membersFixture struct {
	log        *callLog
	workspaces *fakeWorkspaces
	profiles   *fakeProfiles
	auth       *fakeAuthorizer
	uc         *app.ListWorkspaceMembers
}

var (
	aliceInAcme = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: alice.ID, Role: shared.RoleAdmin, IsActive: true, CreatedAt: now}
	bobInAcme   = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: bob.ID, Role: shared.RoleMember, IsActive: true,
		CreatedAt: now.Add(time.Minute)}
	carolInAcme = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: carol.ID, Role: shared.RoleGuest, CreatedAt: now.Add(time.Hour)}
	bobInBeta   = domain.Membership{ID: uuid.NewV7(), WorkspaceID: beta.ID, MemberID: bob.ID, Role: shared.RoleGuest, IsActive: true, CreatedAt: now}
	profiles    = []app.PublicProfile{
		{ID: carol.ID, Email: carol.Email, DisplayName: "carol"},
		{ID: alice.ID, Email: alice.Email, FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: bob.Email, DisplayName: "bob"},
	}
)

func newMembers() *membersFixture {
	log := &callLog{}
	f := &membersFixture{log: log,
		workspaces: &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}, memberships: map[uuid.UUID][]domain.Membership{
			acme.ID: {aliceInAcme, bobInAcme, carolInAcme}, beta.ID: {bobInBeta},
		}},
		profiles: &fakeProfiles{log: log, profiles: profiles},
		auth: &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
			{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleAdmin},
			{bob.ID, acme.ID}:   {WorkspaceRole: shared.RoleMember},
			{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleGuest},
		}}}
	f.uc = app.NewListWorkspaceMembers(f.workspaces, f.profiles, f.auth)
	return f
}

// withUser is m with its member's profile from profiles, the address only
// when seen.
func withUser(m domain.Membership, seen bool) domain.Member {
	i := slices.IndexFunc(profiles, func(p app.PublicProfile) bool { return p.ID == m.MemberID })
	p := profiles[i]
	user := domain.MemberUser{ID: p.ID, DisplayName: p.DisplayName, FirstName: p.FirstName, LastName: p.LastName}
	if seen {
		user.Email = &p.Email
	}
	return domain.Member{Membership: m, User: user}
}

// sameMembers compares two lists, the addresses by value.
func sameMembers(a, b []domain.Member) bool {
	return slices.EqualFunc(a, b, func(x, y domain.Member) bool {
		ex, ey := x.User.Email, y.User.Email
		x.User.Email, y.User.Email = nil, nil
		return x == y && (ex == nil) == (ey == nil) && (ex == nil || *ex == *ey)
	})
}

// The list is the workspace's memberships in the store's order, the ended
// one too, each with its member's profile, a deactivated account's too; the
// profiles are read once, for the listed members, without a transaction.
// An admin and a member see every address, a guest none, his own neither.
func TestListWorkspaceMembersShowsAddressesByRole(t *testing.T) {
	acmeIDs := fmt.Sprint([]uuid.UUID{alice.ID, bob.ID, carol.ID})
	tests := []struct {
		name string
		user app.AccountState
		w    domain.Workspace
		ids  string
		want []domain.Member
	}{
		{"acme's admin", alice, acme, acmeIDs, []domain.Member{withUser(aliceInAcme, true), withUser(bobInAcme, true), withUser(carolInAcme, true)}},
		{"acme's member", bob, acme, acmeIDs, []domain.Member{withUser(aliceInAcme, true), withUser(bobInAcme, true), withUser(carolInAcme, true)}},
		{"beta's guest", bob, beta, fmt.Sprint([]uuid.UUID{bob.ID}), []domain.Member{withUser(bobInBeta, false)}},
	}
	for _, tt := range tests {
		f := newMembers()
		got, err := f.uc.Execute(as(tt.user), tt.w.Slug)
		if err != nil || !sameMembers(got, tt.want) {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, got, err, tt.want)
		}
		want := []string{
			"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace_member.list on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
			"ListMembers " + tt.w.ID.String() + " outside tx",
			"PublicProfiles " + tt.ids + " outside tx",
		}
		if !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}

// A workspace that is not there and one the caller cannot see are the same
// workspace.not_found, and nothing is listed; a failure is the answer,
// never a partial list, and so is a member without an account.
func TestListWorkspaceMembersRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *membersFixture)
		want  error
		calls int
	}{
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, 1},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, 2},
		{"alice cannot see beta", alice, "beta", nil, domain.ErrNotFound, 2},
		{"forbidden", alice, "acme", func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), 2},
		{"the Authorizer failed", alice, "acme", func(f *membersFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} }, failure, 2},
		{"the list failed", alice, "acme", func(f *membersFixture) { f.workspaces.membersErr = failure }, failure, 3},
		{"the profiles failed", alice, "acme", func(f *membersFixture) { f.profiles.err = failure }, failure, 4},
	}
	for _, tt := range tests {
		f := newMembers()
		if tt.set != nil {
			tt.set(f)
		}
		if got, err := f.uc.Execute(as(tt.user), tt.slug); !errors.Is(err, tt.want) || got != nil {
			t.Errorf("%s: Execute() = %+v, %v; want no list and %v", tt.name, got, err, tt.want)
		}
		if len(f.log.calls) != tt.calls {
			t.Errorf("%s: calls = %q, want %d", tt.name, f.log.calls, tt.calls)
		}
	}
	// carol has no account: the error names her membership, and is no
	// problem of the contract (so a 500), never a 404 or a 403.
	f := newMembers()
	f.profiles.profiles = profiles[1:]
	got, err := f.uc.Execute(as(alice), "acme")
	var se *shared.Error
	if err == nil || errors.As(err, &se) || !strings.Contains(err.Error(), carolInAcme.ID.String()) || got != nil {
		t.Errorf("a member without an account: Execute() = %+v, %v; want an error naming %s that is no *shared.Error", got, err, carolInAcme.ID)
	}
	f = newMembers()
	if _, err := f.uc.Execute(context.Background(), "acme"); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and no call", err, f.log.calls)
	}
}
