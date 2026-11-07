package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// GetModule reads the undeleted module id, archived or not, with the
// accounts of its undeleted members in id order; found is false when there
// is none (modules.sql).
func (s *Store) GetModule(ctx context.Context, id uuid.UUID) (domain.Module, bool, error) {
	r, err := s.queries(ctx).GetModule(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Module{}, false, nil
	case err != nil:
		return domain.Module{}, false, fmt.Errorf("read module %s: %w", id, err)
	}
	return moduleOf(r), true, nil
}

// ListModules lists the project's undeleted modules, each with the accounts
// of its undeleted members in id order: the unarchived ones by sort order,
// creation and id, or, when archived, the archived ones alone, the latest
// archived first, then by id (modules.sql).
func (s *Store) ListModules(ctx context.Context, project uuid.UUID, archived bool) ([]domain.Module, error) {
	var rows []gen.GetModuleRow
	if archived {
		listed, err := s.queries(ctx).ListArchivedModules(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("list the archived modules of project %s: %w", project, err)
		}
		for _, r := range listed {
			rows = append(rows, gen.GetModuleRow(r))
		}
	} else {
		listed, err := s.queries(ctx).ListModules(ctx, project)
		if err != nil {
			return nil, fmt.Errorf("list the modules of project %s: %w", project, err)
		}
		for _, r := range listed {
			rows = append(rows, gen.GetModuleRow(r))
		}
	}
	out := make([]domain.Module, len(rows))
	for i, r := range rows {
		out[i] = moduleOf(r)
	}
	return out, nil
}

// moduleOf is the module a row of modules.sql's reads holds.
func moduleOf(r gen.GetModuleRow) domain.Module {
	return domain.Module{ID: r.ID, ProjectID: r.ProjectID, WorkspaceID: r.WorkspaceID, Name: r.Name, Description: r.Description,
		StartDate: dayOf(r.StartDate), TargetDate: dayOf(r.TargetDate), Status: domain.ModuleStatus(r.Status), LeadID: r.LeadID,
		MemberIDs: r.MemberIds, SortOrder: r.SortOrder, ArchivedAt: r.ArchivedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID}
}

// LowestModuleSortOrder reads the lowest sort order of the project's
// undeleted modules, archived ones too; nil when it has none.
func (s *Store) LowestModuleSortOrder(ctx context.Context, project uuid.UUID) (*float64, error) {
	lowest, err := s.queries(ctx).LowestModuleSortOrder(ctx, project)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the lowest module sort order of project %s: %w", project, err)
	}
	return &lowest, nil
}

// CreateModule writes the new module m, not its members' rows
// (InsertModuleMembers). A name another undeleted module of the project
// has, archived or not, in any case, is domain.ErrModuleNameTaken (M6 P2
// spec R8). The domain and the use case checked every other value, so a
// CHECK's or a foreign key's violation is a bug: an internal error.
func (s *Store) CreateModule(ctx context.Context, m domain.Module) error {
	err := s.queries(ctx).CreateModule(ctx, gen.CreateModuleParams{ID: m.ID, WorkspaceID: m.WorkspaceID, ProjectID: m.ProjectID, Name: m.Name,
		Description: m.Description, StartDate: columnOf(m.StartDate), TargetDate: columnOf(m.TargetDate), Status: string(m.Status),
		LeadID: m.LeadID, SortOrder: m.SortOrder, CreatedAt: m.CreatedAt, UpdatedAt: m.UpdatedAt, CreatedByID: m.CreatedByID,
		UpdatedByID: m.UpdatedByID})
	switch {
	case postgres.UniqueViolation(err, "modules_project_id_name_key"):
		return domain.ErrModuleNameTaken
	case err != nil:
		return fmt.Errorf("create module %s: %w", m.ID, err)
	}
	return nil
}
