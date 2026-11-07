package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetCycle reads the undeleted cycle id, archived or not; found is false
// when there is none (cycles.sql).
func (s *Store) GetCycle(ctx context.Context, id uuid.UUID) (domain.Cycle, bool, error) {
	r, err := s.queries(ctx).GetCycle(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Cycle{}, false, nil
	case err != nil:
		return domain.Cycle{}, false, fmt.Errorf("read cycle %s: %w", id, err)
	}
	return domain.Cycle{ID: r.ID, ProjectID: r.ProjectID, WorkspaceID: r.WorkspaceID, Name: r.Name, Description: r.Description,
		Dates: datesOf(r.StartDate, r.EndDate), OwnedByID: r.OwnedByID, SortOrder: r.SortOrder, ArchivedAt: r.ArchivedAt,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID}, true, nil
}

// ProjectCycleDates reads the dates of the project's undeleted cycles,
// archived ones too.
func (s *Store) ProjectCycleDates(ctx context.Context, project uuid.UUID) ([]domain.DatedCycle, error) {
	rows, err := s.queries(ctx).ProjectCycleDates(ctx, project)
	if err != nil {
		return nil, fmt.Errorf("read the cycle dates of project %s: %w", project, err)
	}
	out := make([]domain.DatedCycle, 0, len(rows))
	for _, r := range rows {
		out = append(out, domain.DatedCycle{ID: r.ID, Dates: datesOf(r.StartDate, r.EndDate)})
	}
	return out, nil
}

// LowestCycleSortOrder reads the lowest sort order of the project's undeleted
// cycles, archived ones too; nil when it has none.
func (s *Store) LowestCycleSortOrder(ctx context.Context, project uuid.UUID) (*float64, error) {
	lowest, err := s.queries(ctx).LowestCycleSortOrder(ctx, project)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the lowest cycle sort order of project %s: %w", project, err)
	}
	return &lowest, nil
}

// CreateCycle writes the new cycle c.
func (s *Store) CreateCycle(ctx context.Context, c domain.Cycle) error {
	start, end := columns(c.Dates)
	err := s.queries(ctx).CreateCycle(ctx, gen.CreateCycleParams{ID: c.ID, WorkspaceID: c.WorkspaceID, ProjectID: c.ProjectID, Name: c.Name,
		Description: c.Description, StartDate: start, EndDate: end, OwnedByID: c.OwnedByID, SortOrder: c.SortOrder,
		CreatedAt: c.CreatedAt, UpdatedAt: c.UpdatedAt, CreatedByID: c.CreatedByID, UpdatedByID: c.UpdatedByID})
	if err != nil {
		return fmt.Errorf("create cycle %s: %w", c.ID, err)
	}
	return nil
}

// datesOf are the days of two date columns, as the driver reads them at
// midnight UTC.
func datesOf(start, end *time.Time) domain.Dates {
	return domain.Dates{Start: dayOf(start), End: dayOf(end)}
}

func dayOf(t *time.Time) *shared.Date {
	if t == nil {
		return nil
	}
	d := shared.DateOf(*t)
	return &d
}

// columns are d's days as date columns are written: midnight UTC.
func columns(d domain.Dates) (start, end *time.Time) {
	return columnOf(d.Start), columnOf(d.End)
}

func columnOf(d *shared.Date) *time.Time {
	if d == nil {
		return nil
	}
	t := d.Time()
	return &t
}
