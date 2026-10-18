package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/access"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
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

// issueProjects is project's IssueProjects as issue's ProjectPorts: the
// same reads and lock, each value converted.
type issueProjects struct {
	projects project.IssueProjects
}

func (p issueProjects) Find(ctx context.Context, id uuid.UUID) (uuid.UUID, bool, error) {
	return p.projects.ProjectWorkspace(ctx, id)
}

func (p issueProjects) Lock(ctx context.Context, id uuid.UUID) (issue.LockedProject, bool, error) {
	locked, found, err := p.projects.LockIssueProject(ctx, id)
	return issue.LockedProject(locked), found, err
}

func (p issueProjects) Next(ctx context.Context, id uuid.UUID) (int, error) {
	return p.projects.NextIssueSequence(ctx, id)
}

func (p issueProjects) States(ctx context.Context, id uuid.UUID) ([]issue.StateRef, error) {
	states, err := p.projects.IssueStates(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]issue.StateRef, len(states))
	for i, s := range states {
		out[i] = issue.StateRef{ID: s.ID, Name: s.Name, Group: issue.StateGroup(s.Group), Default: s.Default}
	}
	return out, nil
}

func (p issueProjects) Labels(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]issue.LabelRef, error) {
	labels, err := p.projects.IssueLabels(ctx, id, ids)
	if err != nil {
		return nil, err
	}
	out := make([]issue.LabelRef, len(labels))
	for i, l := range labels {
		out[i] = issue.LabelRef(l)
	}
	return out, nil
}

func (p issueProjects) Roles(ctx context.Context, id uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]shared.Role, error) {
	return p.projects.ActiveRoles(ctx, id, ids)
}

// issueProfiles is identity's PublicProfiles as issue's: each account's
// display name, by account.
type issueProfiles struct {
	profiles identity.PublicProfiles
}

func (p issueProfiles) DisplayNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error) {
	profiles, err := p.profiles.PublicProfiles(ctx, ids)
	if err != nil {
		return nil, err
	}
	names := make(map[uuid.UUID]string, len(profiles))
	for _, profile := range profiles {
		names[profile.ID] = profile.DisplayName
	}
	return names, nil
}
