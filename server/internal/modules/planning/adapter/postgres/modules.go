package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
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
	return domain.Module{ID: r.ID, ProjectID: r.ProjectID, WorkspaceID: r.WorkspaceID, Name: r.Name, Description: r.Description,
		StartDate: dayOf(r.StartDate), TargetDate: dayOf(r.TargetDate), Status: domain.ModuleStatus(r.Status), LeadID: r.LeadID,
		MemberIDs: r.MemberIds, SortOrder: r.SortOrder, ArchivedAt: r.ArchivedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt,
		CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID}, true, nil
}
