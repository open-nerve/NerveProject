package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// LockProject locks the undeleted project id FOR NO KEY UPDATE until the
// transaction ctx carries ends, and reads its workspace and whether it is
// archived; found is false when there is none, a project deleted while the
// lock waited too (app.ProjectLocker).
func (s *Store) LockProject(ctx context.Context, id uuid.UUID) (p app.LockedProject, found bool, err error) {
	r, err := s.queries(ctx).LockProject(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.LockedProject{}, false, nil
	case err != nil:
		return app.LockedProject{}, false, fmt.Errorf("lock project %s: %w", id, err)
	}
	return app.LockedProject{WorkspaceID: r.WorkspaceID, Archived: r.Archived}, true, nil
}

// UpdateProject changes the fields p gives of the project id, by the
// account by at now; the others keep their values. An identifier or a name
// another undeleted project of the workspace has is domain.ErrIdentifierTaken
// or domain.ErrNameTaken.
func (s *Store) UpdateProject(ctx context.Context, id uuid.UUID, p domain.ProjectPatch, by uuid.UUID, now time.Time) error {
	arg := gen.UpdateProjectParams{
		ID: id, Name: p.Name, Description: p.Description, Identifier: p.Identifier, SetLead: p.SetLead, ProjectLeadID: p.LeadID,
		SetDefaultAssignee: p.SetDefaultAssignee, DefaultAssigneeID: p.DefaultAssigneeID, CycleView: p.CycleView, ModuleView: p.ModuleView,
		IssueViewsView: p.IssueViewsView, IntakeView: p.IntakeView, GuestViewAllFeatures: p.GuestViewAllFeatures, Timezone: p.Timezone,
		UpdatedBy: by, Now: now,
	}
	if p.Network != nil {
		network := int16(*p.Network)
		arg.Network = &network
	}
	if p.ArchiveIn != nil {
		months := int32(*p.ArchiveIn)
		arg.ArchiveIn = &months
	}
	if p.LogoProps != nil {
		logo, err := json.Marshal(*p.LogoProps)
		if err != nil {
			return fmt.Errorf("update project %s: %w", id, err)
		}
		arg.LogoProps = logo
	}
	return taken(fmt.Sprintf("update project %s", id), s.queries(ctx).UpdateProject(ctx, arg))
}

// SetArchived archives the project id at now, or unarchives it, by the
// account by (app.ProjectArchiver).
func (s *Store) SetArchived(ctx context.Context, id uuid.UUID, archived bool, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).SetArchived(ctx, gen.SetArchivedParams{ID: id, Archived: archived, UpdatedBy: by, Now: now}); err != nil {
		return fmt.Errorf("set project %s archived %v: %w", id, archived, err)
	}
	return nil
}
