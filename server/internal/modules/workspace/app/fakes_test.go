package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var now = clocktest.At(time.Date(2026, 9, 29, 10, 0, 0, 123456000, time.UTC)).Now()

// fakeTx runs fn in a context marked as inside the transaction; the fakes
// record whether each call happened there.
type fakeTx struct{ calls int }

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

// callLog records the calls of the fakes that share it, in order, each with
// its arguments, and " outside tx" when it ran outside a transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, format string, args ...any) {
	call := fmt.Sprintf(format, args...)
	if ctx.Value(inTxKey{}) != true {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

// fakeAccounts answers the state of the accounts it holds, by id and by
// address, and logs each lock.
type fakeAccounts struct {
	log      *callLog
	accounts []app.AccountState
	err      error
}

func (f *fakeAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccount %s", id)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.ID == id })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

func (f *fakeAccounts) ShareAccountByEmail(ctx context.Context, email string) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccountByEmail %s", email)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.Email == email })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

// fakeWorkspaces is the repositories: it logs every call with its
// arguments, answers from the workspaces it holds, and fails a call with
// the error set for it.
type fakeWorkspaces struct {
	log        *callLog
	workspaces []domain.Workspace // by slug for WorkspaceBySlug and SlugTaken
	lists      map[uuid.UUID][]domain.Workspace
	createErr  error
	listErr    error
	members    []app.MemberRow
}

func (f *fakeWorkspaces) CreateWorkspace(ctx context.Context, w app.WorkspaceRow) (domain.Workspace, error) {
	size := "<nil>"
	if w.OrganizationSize != nil {
		size = *w.OrganizationSize
	}
	f.log.add(ctx, "CreateWorkspace %s %q %s %s %s by %s at %s", w.ID, w.Name, w.Slug, size, w.Timezone, w.CreatedBy, w.Now.Format(time.RFC3339Nano))
	if f.createErr != nil {
		return domain.Workspace{}, f.createErr
	}
	// As stored: the database's clock has no nanoseconds either.
	return domain.Workspace{ID: w.ID, Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: w.Timezone,
		CreatedAt: w.Now, UpdatedAt: w.Now}, nil
}

func (f *fakeWorkspaces) CreateMember(ctx context.Context, m app.MemberRow) error {
	f.log.add(ctx, "CreateMember %s in %s as %d by %s at %s", m.MemberID, m.WorkspaceID, m.Role, m.CreatedBy, m.Now.Format(time.RFC3339Nano))
	f.members = append(f.members, m)
	return nil
}

func (f *fakeWorkspaces) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	f.log.add(ctx, "ListWorkspaces %s", userID)
	return f.lists[userID], f.listErr
}

func (f *fakeWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error) {
	f.log.add(ctx, "WorkspaceBySlug %s", slug)
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return domain.Workspace{}, app.ErrNotFound
	}
	return f.workspaces[i], nil
}

func (f *fakeWorkspaces) SlugTaken(ctx context.Context, slug string) (bool, error) {
	f.log.add(ctx, "SlugTaken %s", slug)
	return slices.ContainsFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug }), nil
}

// grantKey is one (user, workspace) pair of fakeAuthorizer.
type grantKey struct{ user, workspace uuid.UUID }

// fakeAuthorizer answers each (user, workspace) pair its grant, or its
// error, and ErrNotVisible for any other; it logs every call.
type fakeAuthorizer struct {
	log    *callLog
	grants map[grantKey]shared.Grant
	errs   map[grantKey]error
}

func (f *fakeAuthorizer) Authorize(ctx context.Context, actor shared.Actor, action shared.Action, t shared.Target) (shared.Grant, error) {
	f.log.add(ctx, "Authorize %s %s on %s/%s", actor.UserID, action, t.WorkspaceID, t.ProjectID)
	key := grantKey{actor.UserID, t.WorkspaceID}
	if err := f.errs[key]; err != nil {
		return shared.Grant{}, err
	}
	if g, ok := f.grants[key]; ok {
		return g, nil
	}
	return shared.Grant{}, shared.ErrNotVisible
}
