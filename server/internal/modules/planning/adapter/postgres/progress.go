package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CycleSnapshot reads the frozen progress of the undeleted cycle id, whole;
// nil when it is not frozen, or there is no such cycle (progress.sql).
func (s *Store) CycleSnapshot(ctx context.Context, id uuid.UUID) (*domain.Frozen, error) {
	raw, err := s.queries(ctx).CycleSnapshot(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the frozen progress of cycle %s: %w", id, err)
	case raw == nil:
		return nil, nil
	}
	f, err := domain.DecodeFrozen(raw)
	if err != nil {
		return nil, fmt.Errorf("cycle %s: %w", id, err)
	}
	return &f, nil
}

// CycleSnapshotCounts reads, of the cycles ids, the undeleted frozen ones'
// counts and moments, by cycle, in one statement; a cycle not frozen,
// deleted or missing is not among them (progress.sql).
func (s *Store) CycleSnapshotCounts(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]domain.FrozenCounts, error) {
	rows, err := s.queries(ctx).CycleSnapshotCounts(ctx, ids)
	if err != nil {
		return nil, fmt.Errorf("read the frozen counts of %d cycles: %w", len(ids), err)
	}
	out := make(map[uuid.UUID]domain.FrozenCounts, len(rows))
	for _, r := range rows {
		c, err := domain.DecodeFrozenCounts(r.Counts)
		if err != nil {
			return nil, fmt.Errorf("cycle %s: %w", r.ID, err)
		}
		out[r.ID] = c
	}
	return out, nil
}

// FreezeCycle stores f as the frozen progress of the undeleted cycle id
// when it has none, and answers whether it did: the first freeze stays. The
// cycle's last change is not a freeze's (progress.sql).
func (s *Store) FreezeCycle(ctx context.Context, id uuid.UUID, f domain.Frozen) (bool, error) {
	doc, err := domain.EncodeFrozen(f)
	if err != nil {
		return false, fmt.Errorf("encode the frozen progress of cycle %s: %w", id, err)
	}
	n, err := s.queries(ctx).FreezeCycle(ctx, gen.FreezeCycleParams{ID: id, Snapshot: doc})
	if err != nil {
		return false, fmt.Errorf("freeze cycle %s: %w", id, err)
	}
	return n == 1, nil
}

// FreezeDueProjects reads, in id order, the projects with an undeleted
// cycle not frozen, archived or not, that ends on utcToday or before it: the
// freeze job's candidates, read without a lock (progress.sql).
func (s *Store) FreezeDueProjects(ctx context.Context, utcToday shared.Date) ([]uuid.UUID, error) {
	ids, err := s.queries(ctx).FreezeDueProjects(ctx, utcToday.Time())
	if err != nil {
		return nil, fmt.Errorf("read the projects with a cycle to freeze on %s: %w", utcToday, err)
	}
	return ids, nil
}

// CyclesToFreeze reads, in id order, the project's undeleted cycles not
// frozen, archived or not, that ended before today, the project's day
// (progress.sql).
func (s *Store) CyclesToFreeze(ctx context.Context, project uuid.UUID, today shared.Date) ([]domain.Cycle, error) {
	rows, err := s.queries(ctx).CyclesToFreeze(ctx, gen.CyclesToFreezeParams{ProjectID: project, Today: today.Time()})
	if err != nil {
		return nil, fmt.Errorf("read the cycles of project %s to freeze on %s: %w", project, today, err)
	}
	out := make([]domain.Cycle, len(rows))
	for i, r := range rows {
		out[i] = cycleOf(gen.GetCycleRow(r))
	}
	return out, nil
}
