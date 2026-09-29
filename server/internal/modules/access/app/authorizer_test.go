package app_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type ctxKey struct{}

// membership is one (workspace, user) pair of fakeRoles.
type membership struct{ workspace, user uuid.UUID }

// fakeRoles answers the role of each (workspace, user) it holds, and records
// every call with the value of ctxKey in its context, "(none)" when the
// context has none, so a context dropped on the way fails the call's
// assertion rather than the fake.
type fakeRoles struct {
	roles map[membership]shared.Role
	err   error
	calls []string
}

func (f *fakeRoles) ActiveRole(ctx context.Context, workspaceID, userID uuid.UUID) (shared.Role, bool, error) {
	value, ok := ctx.Value(ctxKey{}).(string)
	if !ok {
		value = "(none)"
	}
	f.calls = append(f.calls, workspaceID.String()+" "+userID.String()+" "+value)
	if f.err != nil {
		return 0, false, f.err
	}
	role, ok := f.roles[membership{workspaceID, userID}]
	return role, ok, nil
}

var (
	w1, w2 = uuid.NewV7(), uuid.NewV7()
	a, b   = uuid.NewV7(), uuid.NewV7()
)

// Authorize reads the caller's role in the target's workspace, in the
// caller's context: two users, two workspaces, and each answer is the one of
// its own pair.
func TestAuthorizeReadsTheCallersRoleInTheTargetsWorkspace(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{
		{w1, a}: shared.RoleAdmin, {w2, a}: shared.RoleGuest, {w1, b}: shared.RoleMember,
	}}
	auth := app.NewAuthorizer(roles)
	tests := []struct {
		user, workspace uuid.UUID
		want            shared.Role // 0: not visible
	}{
		{a, w1, shared.RoleAdmin},
		{a, w2, shared.RoleGuest},
		{b, w1, shared.RoleMember},
		{b, w2, 0},
	}
	for _, tt := range tests {
		roles.calls = nil
		ctx := context.WithValue(context.Background(), ctxKey{}, "tx")
		grant, err := auth.Authorize(ctx, shared.Actor{UserID: tt.user}, "workspace.read", shared.Target{WorkspaceID: tt.workspace})
		want := tt.workspace.String() + " " + tt.user.String() + " tx"
		if len(roles.calls) != 1 || roles.calls[0] != want {
			t.Errorf("ActiveRole calls = %q, want [%q]", roles.calls, want)
		}
		if tt.want == 0 {
			if !errors.Is(err, shared.ErrNotVisible) {
				t.Errorf("user %s in %s: Authorize() = %+v, %v; want ErrNotVisible", tt.user, tt.workspace, grant, err)
			}
			continue
		}
		if err != nil || grant != (shared.Grant{WorkspaceRole: tt.want}) {
			t.Errorf("user %s in %s: Authorize() = %+v, %v; want role %d", tt.user, tt.workspace, grant, err, tt.want)
		}
	}
}

// Nothing is cached: a membership that ends between two calls is not
// visible at the second.
func TestAuthorizeReadsOnEveryCall(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	target := shared.Target{WorkspaceID: w1}
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", target); err != nil {
		t.Fatalf("first Authorize() = %v", err)
	}
	delete(roles.roles, membership{w1, a})
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", target); !errors.Is(err, shared.ErrNotVisible) {
		t.Errorf("Authorize() after the membership ended = %v, want ErrNotVisible", err)
	}
}

// An action without a row fails closed, before anything is read, and is not
// a refusal the caller could act on: an internal error.
func TestAuthorizeRefusesAnActionWithoutARule(t *testing.T) {
	roles := &fakeRoles{roles: map[membership]shared.Role{{w1, a}: shared.RoleAdmin}}
	auth := app.NewAuthorizer(roles)
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	grant, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "no.such.action", shared.Target{WorkspaceID: w1})
	var se *shared.Error
	if err == nil || errors.As(err, &se) || grant != (shared.Grant{}) {
		t.Errorf("Authorize() = %+v, %v; want an internal error", grant, err)
	}
	if len(roles.calls) != 0 {
		t.Errorf("ActiveRole calls = %q, want none", roles.calls)
	}
}

// The port's failure is Authorize's.
func TestAuthorizeReturnsThePortsError(t *testing.T) {
	failure := errors.New("connection reset")
	auth := app.NewAuthorizer(&fakeRoles{err: failure})
	ctx := context.WithValue(context.Background(), ctxKey{}, "request")
	if _, err := auth.Authorize(ctx, shared.Actor{UserID: a}, "workspace.read", shared.Target{WorkspaceID: w1}); !errors.Is(err, failure) {
		t.Errorf("Authorize() = %v, want %v", err, failure)
	}
}
