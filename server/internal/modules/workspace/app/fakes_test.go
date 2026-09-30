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
// record whether each call happened there. commitErr, when set, is the
// commit failing after fn succeeded.
type fakeTx struct {
	calls     int
	commitErr error
}

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	if err := fn(context.WithValue(ctx, inTxKey{}, true)); err != nil {
		return err
	}
	return f.commitErr
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
// the error set for it; a read fails for the argument its error is set
// for, wrapped as the store wraps it, so the use case must match with
// errors.Is.
type fakeWorkspaces struct {
	log         *callLog
	workspaces  []domain.Workspace // by slug for WorkspaceBySlug, SlugTaken and the locks; by id for UpdateWorkspace
	lists       map[uuid.UUID][]domain.Workspace
	createErr   error
	memberErr   error               // for CreateMember, which then stores nothing
	updateErr   error               // for UpdateWorkspace
	deleteErrs  map[string]error    // by step, e.g. "DeleteWorkspaceMembers"
	listErrs    map[uuid.UUID]error // by user, for ListWorkspaces
	slugErrs    map[string]error    // by slug, for WorkspaceBySlug and SlugTaken
	lockErrs    map[string]error    // by slug, for the locks
	members     []app.MemberRow
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers
	membersErr  error                             // for ListMembers
	prefs       map[prefsKey]domain.Preferences
	prefsErr    error // for Preferences and UpsertPreferences
	upserts     []app.PreferencesRow
}

// prefsKey is one (workspace, user) pair of fakeWorkspaces' preferences.
type prefsKey struct{ workspace, user uuid.UUID }

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
	if f.memberErr != nil {
		return fmt.Errorf("create workspace member: %w", f.memberErr)
	}
	f.members = append(f.members, m)
	return nil
}

func (f *fakeWorkspaces) ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error) {
	f.log.add(ctx, "ListWorkspaces %s", userID)
	if err := f.listErrs[userID]; err != nil {
		return nil, fmt.Errorf("list workspaces: %w", err)
	}
	return f.lists[userID], nil
}

func (f *fakeWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error) {
	f.log.add(ctx, "WorkspaceBySlug %s", slug)
	if err := f.slugErrs[slug]; err != nil {
		return domain.Workspace{}, fmt.Errorf("workspace by slug: %w", err)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return domain.Workspace{}, app.ErrNotFound
	}
	return f.workspaces[i], nil
}

func (f *fakeWorkspaces) SlugTaken(ctx context.Context, slug string) (bool, error) {
	f.log.add(ctx, "SlugTaken %s", slug)
	if err := f.slugErrs[slug]; err != nil {
		return false, fmt.Errorf("check slug: %w", err)
	}
	return slices.ContainsFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug }), nil
}

func (f *fakeWorkspaces) LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

// lock answers the id of the workspace with slug, app.ErrNotFound when it
// holds none, and the error set for slug wrapped as the store wraps it.
func (f *fakeWorkspaces) lock(slug string) (uuid.UUID, error) {
	if err := f.lockErrs[slug]; err != nil {
		return uuid.UUID{}, fmt.Errorf("lock the workspace row: %w", err)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.Slug == slug })
	if i < 0 {
		return uuid.UUID{}, app.ErrNotFound
	}
	return f.workspaces[i].ID, nil
}

func (f *fakeWorkspaces) UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error) {
	f.log.add(ctx, "UpdateWorkspace %s name=%s size=%s timezone=%s by %s at %s", id, show(p.Name), show(p.OrganizationSize), show(p.Timezone),
		by, now.Format(time.RFC3339Nano))
	if f.updateErr != nil {
		return domain.Workspace{}, fmt.Errorf("update workspace: %w", f.updateErr)
	}
	i := slices.IndexFunc(f.workspaces, func(w domain.Workspace) bool { return w.ID == id })
	if i < 0 {
		return domain.Workspace{}, fmt.Errorf("update workspace %s: no such row", id)
	}
	w := f.workspaces[i]
	if p.Name != nil {
		w.Name = *p.Name
	}
	if p.OrganizationSize != nil {
		w.OrganizationSize = p.OrganizationSize
	}
	if p.Timezone != nil {
		w.Timezone = *p.Timezone
	}
	w.UpdatedAt = now
	return w, nil
}

func (f *fakeWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "ShareWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

func (f *fakeWorkspaces) Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (domain.Preferences, bool, error) {
	f.log.add(ctx, "Preferences %s %s", workspaceID, userID)
	if f.prefsErr != nil {
		return domain.Preferences{}, false, fmt.Errorf("read workspace preferences: %w", f.prefsErr)
	}
	p, found := f.prefs[prefsKey{workspaceID, userID}]
	return p, found, nil
}

// UpsertPreferences logs the row without its id, which the use case makes
// anew each time; upserts keeps it whole.
func (f *fakeWorkspaces) UpsertPreferences(ctx context.Context, r app.PreferencesRow) (domain.Preferences, error) {
	var limit *string
	if r.Patch.NavigationProjectLimit != nil {
		limit = ptr(fmt.Sprint(*r.Patch.NavigationProjectLimit))
	}
	f.log.add(ctx, "UpsertPreferences %s %s mode=%s limit=%s at %s", r.WorkspaceID, r.UserID, show(r.Patch.NavigationControl), show(limit),
		r.Now.Format(time.RFC3339Nano))
	f.upserts = append(f.upserts, r)
	if f.prefsErr != nil {
		return domain.Preferences{}, fmt.Errorf("write workspace preferences: %w", f.prefsErr)
	}
	key := prefsKey{r.WorkspaceID, r.UserID}
	p, found := f.prefs[key]
	if !found {
		p = domain.DefaultPreferences()
	}
	if f.prefs == nil {
		f.prefs = map[prefsKey]domain.Preferences{}
	}
	f.prefs[key] = p.Apply(r.Patch)
	return f.prefs[key], nil
}

func (f *fakeWorkspaces) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspace", id, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceMembers", workspaceID, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspacePreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspacePreferences", workspaceID, by, now)
}

// deleteStep logs a step of the deletion and fails it with the error set
// for it, wrapped as the store wraps it.
func (f *fakeWorkspaces) deleteStep(ctx context.Context, step string, id, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "%s %s by %s at %s", step, id, by, now.Format(time.RFC3339Nano))
	if err := f.deleteErrs[step]; err != nil {
		return fmt.Errorf("%s: %w", step, err)
	}
	return nil
}

func (f *fakeWorkspaces) ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error) {
	f.log.add(ctx, "ListMembers %s", workspaceID)
	if f.membersErr != nil {
		return nil, fmt.Errorf("list workspace members: %w", f.membersErr)
	}
	return f.memberships[workspaceID], nil
}

// fakeProfiles answers the profiles it holds of the ids asked for, in the
// order it holds them, and logs each call with its ids.
type fakeProfiles struct {
	log      *callLog
	profiles []app.PublicProfile
	err      error
}

func (f *fakeProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error) {
	f.log.add(ctx, "PublicProfiles %v", ids)
	if f.err != nil {
		return nil, fmt.Errorf("read public profiles: %w", f.err)
	}
	var out []app.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, nil
}

// show is *s quoted, or <nil>.
func show(s *string) string {
	if s == nil {
		return "<nil>"
	}
	return fmt.Sprintf("%q", *s)
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
