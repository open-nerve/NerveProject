package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateProject inserts p. An identifier or a name an undeleted project of
// the workspace has is domain.ErrIdentifierTaken or domain.ErrNameTaken.
// The domain checked every value, so a CHECK violation is a bug: an
// internal error (500), not a domain error.
func (s *Store) CreateProject(ctx context.Context, p app.ProjectRow) error {
	logo, err := json.Marshal(p.LogoProps)
	if err != nil {
		return fmt.Errorf("create project: %w", err)
	}
	err = s.queries(ctx).CreateProject(ctx, gen.CreateProjectParams{
		ID: p.ID, WorkspaceID: p.WorkspaceID, Name: p.Name, Description: p.Description, Identifier: p.Identifier,
		Network: int16(p.Network), ProjectLeadID: p.LeadID, LogoProps: logo, Timezone: p.Timezone, CreatedBy: &p.CreatedBy, Now: p.Now,
	})
	switch {
	case uniqueViolation(err, "projects_workspace_id_identifier_key"):
		return domain.ErrIdentifierTaken
	case uniqueViolation(err, "projects_workspace_id_name_key"):
		return domain.ErrNameTaken
	case err != nil:
		return fmt.Errorf("create project: %w", err)
	}
	return nil
}

// IdentifierTaken reports whether an undeleted project of workspaceID has
// identifier, compared as stored, in upper case: the caller upper-cases it.
func (s *Store) IdentifierTaken(ctx context.Context, workspaceID uuid.UUID, identifier string) (bool, error) {
	taken, err := s.queries(ctx).IdentifierTaken(ctx, gen.IdentifierTakenParams{WorkspaceID: workspaceID, Identifier: identifier})
	if err != nil {
		return false, fmt.Errorf("check identifier %q: %w", identifier, err)
	}
	return taken, nil
}

// GetProject returns the undeleted project id, archived or not, as userID
// sees it: his role and his place in his sidebar while his membership is
// active, and the active members' accounts. found is false when there is no
// such project. It reads in the transaction ctx carries.
func (s *Store) GetProject(ctx context.Context, id, userID uuid.UUID) (p domain.Project, found bool, err error) {
	r, err := s.queries(ctx).GetProject(ctx, gen.GetProjectParams{ID: id, UserID: userID})
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Project{}, false, nil
	case err != nil:
		return domain.Project{}, false, fmt.Errorf("read project %s: %w", id, err)
	}
	p, err = projectOf(r)
	return p, err == nil, err
}

// ListProjects lists workspaceID's undeleted projects that userID sees
// with v, the archived ones alone when archived is true and the others
// otherwise, each as he sees it, in the order of M3 design 3.12.
func (s *Store) ListProjects(ctx context.Context, workspaceID, userID uuid.UUID, v domain.Visibility, archived bool) ([]domain.Project, error) {
	rows, err := s.queries(ctx).ListProjects(ctx, gen.ListProjectsParams{UserID: userID, WorkspaceID: workspaceID, Archived: archived,
		SeesAll: v.All, SeesPublic: v.Public})
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	list := make([]domain.Project, len(rows))
	for i, r := range rows {
		if list[i], err = projectOf(gen.GetProjectRow(r)); err != nil {
			return nil, err
		}
	}
	return list, nil
}

// projectOf is the project a row of GetProject, or of ListProjects, holds.
func projectOf(r gen.GetProjectRow) (domain.Project, error) {
	var logo domain.LogoProps
	if err := json.Unmarshal(r.LogoProps, &logo); err != nil {
		return domain.Project{}, fmt.Errorf("read logo_props of project %s: %w", r.ID, err)
	}
	p := domain.Project{
		ID: r.ID, WorkspaceID: r.WorkspaceID, Name: r.Name, Description: r.Description, Identifier: r.Identifier,
		Network: domain.Network(r.Network), LeadID: r.ProjectLeadID, DefaultAssigneeID: r.DefaultAssigneeID,
		CycleView: r.CycleView, ModuleView: r.ModuleView, IssueViewsView: r.IssueViewsView, IntakeView: r.IntakeView,
		GuestViewAllFeatures: r.GuestViewAllFeatures, ArchiveIn: int(r.ArchiveIn), ArchivedAt: r.ArchivedAt, LogoProps: logo,
		Timezone: r.Timezone, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, SortOrder: r.SortOrder, MemberIDs: r.MemberIds,
	}
	if r.MemberRole != nil {
		role := shared.Role(*r.MemberRole)
		p.MemberRole = &role
	}
	return p, nil
}
