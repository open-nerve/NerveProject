package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	acme = domain.Workspace{ID: uuid.NewV7(), Name: "Acme", Slug: "acme", Timezone: "UTC", TotalMembers: 3, CreatedAt: now, UpdatedAt: now}
	beta = domain.Workspace{ID: uuid.NewV7(), Name: "Beta", Slug: "beta", Timezone: "UTC", TotalMembers: 1, CreatedAt: now, UpdatedAt: now}
)

// ListWorkspaces is the caller's list, asked of the store for the caller:
// two callers, two lists.
func TestListWorkspacesIsTheCallersList(t *testing.T) {
	log := &callLog{}
	aliceList := []domain.Workspace{acme, beta}
	store := &fakeWorkspaces{log: log, lists: map[uuid.UUID][]domain.Workspace{alice.ID: aliceList, bob.ID: {beta}}}
	uc := app.NewListWorkspaces(store)
	for user, want := range map[app.AccountState][]domain.Workspace{alice: aliceList, bob: {beta}, carol: nil} {
		log.calls = nil
		got, err := uc.Execute(as(user))
		if err != nil || !slices.Equal(got, want) {
			t.Errorf("%s: Execute() = %v, %v; want %v", user.Email, got, err, want)
		}
		if want := []string{"ListWorkspaces " + user.ID.String() + " outside tx"}; !slices.Equal(log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", user.Email, log.calls, want)
		}
	}
	if _, err := uc.Execute(context.Background()); err == nil {
		t.Error("Execute() without an actor = nil, want an error")
	}
}

// GetWorkspace reads the workspace, then asks the Authorizer for
// workspace.read on it, for the caller, without a transaction: the answer
// carries the caller's role from the grant. Two callers, two workspaces.
func TestGetWorkspaceDecidesOnTheWorkspaceFound(t *testing.T) {
	log := &callLog{}
	store := &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{
		{alice.ID, acme.ID}: {WorkspaceRole: shared.RoleGuest},
		{alice.ID, beta.ID}: {WorkspaceRole: shared.RoleAdmin},
		{bob.ID, beta.ID}:   {WorkspaceRole: shared.RoleMember},
	}}
	uc := app.NewGetWorkspace(store, auth)
	tests := []struct {
		user app.AccountState
		w    domain.Workspace
		role shared.Role
	}{
		{alice, acme, shared.RoleGuest},
		{alice, beta, shared.RoleAdmin},
		{bob, beta, shared.RoleMember},
	}
	for _, tt := range tests {
		log.calls = nil
		got, err := uc.Execute(as(tt.user), tt.w.Slug)
		want := tt.w
		want.Role = tt.role
		if err != nil || got != want {
			t.Errorf("%s, %s: Execute() = %+v, %v; want %+v", tt.user.Email, tt.w.Slug, got, err, want)
		}
		wantCalls := []string{
			"WorkspaceBySlug " + tt.w.Slug + " outside tx",
			"Authorize " + tt.user.ID.String() + " workspace.read on " + tt.w.ID.String() + "/" + uuid.Nil().String() + " outside tx",
		}
		if !slices.Equal(log.calls, wantCalls) {
			t.Errorf("%s, %s: calls = %q, want %q", tt.user.Email, tt.w.Slug, log.calls, wantCalls)
		}
	}
}

// A workspace the caller cannot see and one that is not there are the same
// workspace.not_found; a refusal that is not "not visible" is not turned
// into one.
func TestGetWorkspaceNotFound(t *testing.T) {
	log := &callLog{}
	failure := errors.New("connection reset")
	store := &fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme, beta}}
	auth := &fakeAuthorizer{log: log, errs: map[grantKey]error{
		{bob.ID, beta.ID}:   shared.Forbidden(),
		{carol.ID, beta.ID}: failure,
	}}
	uc := app.NewGetWorkspace(store, auth)
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		want  error
		calls int
	}{
		{"not visible", bob, "acme", domain.ErrNotFound, 2},
		{"no such workspace", alice, "nothing", domain.ErrNotFound, 1},
		{"forbidden", bob, "beta", shared.Forbidden(), 2},
		{"the Authorizer failed", carol, "beta", failure, 2},
	}
	for _, tt := range tests {
		log.calls = nil
		if _, err := uc.Execute(as(tt.user), tt.slug); !errors.Is(err, tt.want) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		if len(log.calls) != tt.calls {
			t.Errorf("%s: calls = %q, want %d", tt.name, log.calls, tt.calls)
		}
	}
}

// CheckSlug looks up only a slug that could be used.
func TestCheckSlug(t *testing.T) {
	log := &callLog{}
	uc := app.NewCheckSlug(&fakeWorkspaces{log: log, workspaces: []domain.Workspace{acme}})
	tests := []struct {
		slug   string
		want   domain.SlugReason
		lookup bool
	}{
		{"free", "", true},
		{"acme", domain.SlugTaken, true},
		{"Acme", domain.SlugInvalid, false},
		{"", domain.SlugInvalid, false},
		{"settings", domain.SlugReserved, false},
		{"assets", domain.SlugReserved, false},
	}
	for _, tt := range tests {
		log.calls = nil
		got, err := uc.Execute(context.Background(), tt.slug)
		if err != nil || got != tt.want {
			t.Errorf("Execute(%q) = %q, %v; want %q", tt.slug, got, err, tt.want)
		}
		if lookup := len(log.calls) > 0; lookup != tt.lookup || (lookup && log.calls[0] != "SlugTaken "+tt.slug+" outside tx") {
			t.Errorf("Execute(%q): calls = %q, want a lookup: %v", tt.slug, log.calls, tt.lookup)
		}
	}
}
