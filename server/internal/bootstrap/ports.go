package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// The values that cross from one module's port to another's: modules do not
// import each other, so where an implementation answers its own type, a few
// lines here convert it (M3 design 6.5).

// workspaceAccounts is identity's Accounts as workspace's port: the same
// locks, identity's AccountState converted into workspace's.
type workspaceAccounts struct {
	accounts identity.Accounts
}

func (a workspaceAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (workspace.AccountState, bool, error) {
	state, found, err := a.accounts.ShareAccount(ctx, id)
	return workspace.AccountState(state), found, err
}

func (a workspaceAccounts) ShareAccountByEmail(ctx context.Context, email string) (workspace.AccountState, bool, error) {
	state, found, err := a.accounts.ShareAccountByEmail(ctx, email)
	return workspace.AccountState(state), found, err
}

// workspaceProfiles is identity's PublicProfiles as workspace's
// MemberProfiles: the same read, each profile converted.
type workspaceProfiles struct {
	profiles identity.PublicProfiles
}

func (p workspaceProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]workspace.PublicProfile, error) {
	profiles, err := p.profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]workspace.PublicProfile, len(profiles))
	for i, profile := range profiles {
		out[i] = workspace.PublicProfile(profile)
	}
	return out, nil
}

// projectWorkspaces is workspace's WorkspaceDirectory as project's port:
// the same reads and locks, each workspace converted.
type projectWorkspaces struct {
	directory workspace.WorkspaceDirectory
}

func (d projectWorkspaces) WorkspaceBySlug(ctx context.Context, slug string) (project.Workspace, bool, error) {
	w, found, err := d.directory.WorkspaceBySlug(ctx, slug)
	return project.Workspace(w), found, err
}

func (d projectWorkspaces) ShareWorkspaceBySlug(ctx context.Context, slug string) (project.Workspace, bool, error) {
	w, found, err := d.directory.ShareWorkspaceBySlug(ctx, slug)
	return project.Workspace(w), found, err
}

func (d projectWorkspaces) ShareWorkspaceByID(ctx context.Context, id uuid.UUID) (project.Workspace, bool, error) {
	w, found, err := d.directory.ShareWorkspaceByID(ctx, id)
	return project.Workspace(w), found, err
}

// accessProjects is project's ProjectAccess as access's port: the same read,
// the facts converted.
type accessProjects struct {
	projects project.ProjectAccess
}

func (a accessProjects) ProjectFacts(ctx context.Context, projectID, userID uuid.UUID) (access.ProjectFacts, bool, error) {
	f, found, err := a.projects.ProjectFacts(ctx, projectID, userID)
	return access.ProjectFacts(f), found, err
}
