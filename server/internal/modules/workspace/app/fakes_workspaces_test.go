package app_test

import (
	"context"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// fakeWorkspaces is the repositories: it logs every call with its
// arguments, answers from the workspaces it holds, and fails a call with
// the error set for it; a read fails for the argument its error is set
// for, wrapped as the store wraps it, so the use case must match with
// errors.Is. The workspaces' methods are here; the members' and the display
// settings' are in fakes_members_test.go and fakes_preferences_test.go.
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
	memberships map[uuid.UUID][]domain.Membership // by workspace, for ListMembers and MemberByID
	membersErr  error                             // for ListMembers and MemberByID
	roleErr     error                             // for UpdateMemberRole
	onLock      func()                            // run by LockWorkspace once it has locked: what changed while it waited
	prefs       map[prefsKey]domain.Preferences
	prefIDs     map[prefsKey]uuid.UUID // the id each row UpsertPreferences inserted took
	prefsErr    error                  // for Preferences and UpsertPreferences
	upserts     []app.PreferencesRow
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
	return domain.Workspace{ID: w.ID, Name: w.Name, Slug: w.Slug, OrganizationSize: w.OrganizationSize, Timezone: w.Timezone,
		CreatedAt: stored(w.Now), UpdatedAt: stored(w.Now)}, nil
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
	w.UpdatedAt = stored(now)
	return w, nil
}

func (f *fakeWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error) {
	f.log.add(ctx, "ShareWorkspaceBySlug %s", slug)
	return f.lock(slug)
}

func (f *fakeWorkspaces) DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspace", id, by, now)
}

func (f *fakeWorkspaces) DeleteWorkspaceInvitations(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.deleteStep(ctx, "DeleteWorkspaceInvitations", workspaceID, by, now)
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
