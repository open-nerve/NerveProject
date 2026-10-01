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
	var logo domain.LogoProps
	if err := json.Unmarshal(r.LogoProps, &logo); err != nil {
		return domain.Project{}, false, fmt.Errorf("read logo_props of project %s: %w", id, err)
	}
	p = domain.Project{
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
	return p, true, nil
}
