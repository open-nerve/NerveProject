package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeIdentifiers holds the identifiers taken, by workspace; it logs each
// question and fails with err.
type fakeIdentifiers struct {
	log   *callLog
	taken map[uuid.UUID][]string
	err   error
}

func (f *fakeIdentifiers) IdentifierTaken(ctx context.Context, workspaceID uuid.UUID, identifier string) (bool, error) {
	f.log.add(ctx, "IdentifierTaken %s %s", workspaceID, identifier)
	return slices.Contains(f.taken[workspaceID], identifier), f.err
}

// newCheck is CheckProjectIdentifier over fakes sharing one log: acme has
// WEB, beta OPS; alice is acme's member, carol its guest.
func newCheck() (*app.CheckProjectIdentifier, *fakeDirectory, *fakeIdentifiers, *fakeAuthorizer) {
	log := &callLog{}
	workspaces := &fakeDirectory{log: log, workspaces: map[string]app.Workspace{"acme": acme, "beta": beta}}
	projects := &fakeIdentifiers{log: log, taken: map[uuid.UUID][]string{acme.ID: {"WEB"}, beta.ID: {"OPS"}}}
	auth := &fakeAuthorizer{log: log, grants: map[grantKey]shared.Grant{{alice, acme.ID}: {WorkspaceRole: shared.RoleMember}},
		errs: map[grantKey]error{{carol, acme.ID}: shared.Forbidden()}}
	return app.NewCheckProjectIdentifier(workspaces, projects, auth), workspaces, projects, auth
}

// The use case finds the workspace without a lock and outside any
// transaction, decides project_identifier.check in it, then asks for the
// identifier upper-cased: available unless the workspace has it. An
// identifier createProject would refuse is not available, and the store is
// not asked.
func TestCheckProjectIdentifier(t *testing.T) {
	decided := []string{"WorkspaceBySlug acme outside tx",
		fmt.Sprintf("Authorize %s project_identifier.check on %s/%s outside tx", alice, acme.ID, uuid.UUID{})}
	asked := func(id string) []string {
		return append(slices.Clone(decided), fmt.Sprintf("IdentifierTaken %s %s outside tx", acme.ID, id))
	}
	tests := []struct {
		identifier string
		want       bool
		calls      []string
	}{
		{"WEB", false, asked("WEB")},
		{"web", false, asked("WEB")},
		{"OPS", true, asked("OPS")},
		{"çay1", true, asked("ÇAY1")},
		{"WEB-2", false, decided},
		{"ABCDEFGHIJK", false, decided},
	}
	for _, tt := range tests {
		uc, workspaces, _, _ := newCheck()
		got, err := uc.Execute(as(alice), "acme", tt.identifier)
		if err != nil || got != tt.want || !slices.Equal(workspaces.log.calls, tt.calls) {
			t.Errorf("Execute(%q) = %v, %v, calls %q; want %v, calls %q", tt.identifier, got, err, workspaces.log.calls, tt.want, tt.calls)
		}
	}
}

// A workspace not there, or not visible, is workspace.not_found; a guest
// is refused; every failure comes back as itself; without a caller nothing
// is read. None of them asks about the identifier.
func TestCheckProjectIdentifierRefuses(t *testing.T) {
	failure := errors.New("connection reset")
	tests := []struct {
		name       string
		ctx        context.Context
		slug       string
		dirErr     error
		storeErr   error
		want       error
		identifier bool // the store was asked
	}{
		{"no workspace", as(alice), "gone", nil, nil, domain.ErrWorkspaceNotFound, false},
		{"a workspace he is not in", as(dave), "acme", nil, nil, domain.ErrWorkspaceNotFound, false},
		{"a guest", as(carol), "acme", nil, nil, shared.Forbidden(), false},
		{"the directory failing", as(alice), "acme", failure, nil, failure, false},
		{"the store failing", as(alice), "acme", nil, failure, failure, true},
		{"no caller", context.Background(), "acme", nil, nil, shared.Unauthenticated(), false},
	}
	for _, tt := range tests {
		uc, workspaces, projects, _ := newCheck()
		workspaces.err, projects.err = tt.dirErr, tt.storeErr
		got, err := uc.Execute(tt.ctx, tt.slug, "NEW")
		asked := slices.ContainsFunc(workspaces.log.calls, func(c string) bool { return c == fmt.Sprintf("IdentifierTaken %s NEW outside tx", acme.ID) })
		if !errors.Is(err, tt.want) || got || asked != tt.identifier {
			t.Errorf("%s: Execute() = %v, %v, calls %q; want false, %v", tt.name, got, err, workspaces.log.calls, tt.want)
		}
	}
}
