package app

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateProject creates a project in a workspace: POST
// /api/v0/workspaces/{slug}/projects (M3 design 3.6, 3.17–3.19).
type CreateProject struct {
	d CreateProjectDeps
}

// CreateProjectDeps are the use case's ports.
type CreateProjectDeps struct {
	Workspaces WorkspaceDirectory
	Members    WorkspaceMembers
	Projects   ProjectCreator
	Auth       shared.Authorizer
	Tx         shared.TxManager
	Clock      Clock
}

// NewCreateProject returns the use case.
func NewCreateProject(d CreateProjectDeps) *CreateProject {
	return &CreateProject{d: d}
}

// Execute checks in (domain.CheckNewProject), then, in one transaction, in
// the order of M3 design 3.6: the workspace FOR SHARE; the decision on
// project.create; the creator's and the lead's memberships of the workspace
// FOR SHARE in id order (convention 3); the lead's check, after the
// decision, so that a caller who may not create learns nothing of the lead
// (422 project_lead_id not_allowed unless the lead is an active admin or
// member); then the project, the creator's and the lead's memberships as
// its admins, each one's display settings first in his sidebar
// (SortOrderFirst), and the six default states. The project's time zone is
// the one given, or the workspace's. The rows are new, so the clock is read
// once, before the transaction: no row it reads under the locks has a time
// the new rows must follow (P3's ruling (c), as createWorkspaceInvitations).
// The answer is the project as stored, as the caller sees it.
func (u *CreateProject) Execute(ctx context.Context, slug string, in domain.NewProject) (domain.Project, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return domain.Project{}, err
	}
	in, err = domain.CheckNewProject(in)
	if err != nil {
		return domain.Project{}, err
	}
	now := u.d.Clock.Now()
	id := uuid.NewV7()
	var created domain.Project
	err = u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		ws, found, err := u.d.Workspaces.ShareWorkspaceBySlug(ctx, slug)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrWorkspaceNotFound
		}
		if _, err = decide(ctx, u.d.Auth, actor, domain.ActionCreate, ws.ID, uuid.UUID{}, domain.ErrWorkspaceNotFound); err != nil {
			return err
		}
		admins := []uuid.UUID{actor.UserID}
		if in.LeadID != nil && *in.LeadID != actor.UserID {
			admins = append(admins, *in.LeadID)
		}
		roles, err := u.d.Members.ShareMembers(ctx, ws.ID, admins)
		if err != nil {
			return err
		}
		if in.LeadID != nil {
			if role, active := roles[*in.LeadID]; !active || !domain.CanLead(role) {
				return domain.LeadNotAllowed()
			}
		}
		if err := u.insert(ctx, u.row(id, ws, in, actor.UserID, now), admins); err != nil {
			return err
		}
		created, found, err = u.d.Projects.GetProject(ctx, id, actor.UserID)
		switch {
		case err != nil:
			return err
		case !found:
			return fmt.Errorf("create project: project %s is not there after its insert", id)
		}
		return nil
	})
	if err != nil {
		return domain.Project{}, err
	}
	return created, nil
}

// row is the project to insert: in's values, in the workspace ws, its time
// zone the workspace's unless in gives one.
func (u *CreateProject) row(id uuid.UUID, ws Workspace, in domain.NewProject, by uuid.UUID, now time.Time) ProjectRow {
	timezone := ws.Timezone
	if in.Timezone != nil {
		timezone = *in.Timezone
	}
	return ProjectRow{ID: id, WorkspaceID: ws.ID, Name: in.Name, Description: in.Description, Identifier: in.Identifier, Network: *in.Network,
		LeadID: in.LeadID, LogoProps: in.LogoProps, Timezone: timezone, CreatedBy: by, Now: now}
}

// insert writes p, each of admins its admin with his display settings, and
// the default states.
func (u *CreateProject) insert(ctx context.Context, p ProjectRow, admins []uuid.UUID) error {
	if err := u.d.Projects.CreateProject(ctx, p); err != nil {
		return err
	}
	for _, user := range admins {
		if err := u.d.Projects.CreateMember(ctx, MemberRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: p.ID, MemberID: user,
			Role: shared.RoleAdmin, CreatedBy: p.CreatedBy, Now: p.Now}); err != nil {
			return err
		}
		lowest, err := u.d.Projects.LowestSortOrder(ctx, p.WorkspaceID, user)
		if err != nil {
			return err
		}
		if err := u.d.Projects.CreatePreferences(ctx, PreferencesRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: p.ID, UserID: user,
			SortOrder: domain.SortOrderFirst(lowest), CreatedBy: p.CreatedBy, Now: p.Now}); err != nil {
			return err
		}
	}
	var states []StateRow
	for _, s := range domain.DefaultStates() {
		states = append(states, StateRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: p.ID, State: s, CreatedBy: p.CreatedBy, Now: p.Now})
	}
	return u.d.Projects.CreateStates(ctx, states)
}
