package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

var (
	_ app.DraftReader = (*Store)(nil)
	_ app.DraftLister = (*Store)(nil)
	_ app.DraftWriter = (*Store)(nil)
)

// GetDraft reads a live draft without preempting workspace or owner decisions.
func (s *Store) GetDraft(ctx context.Context, id uuid.UUID) (domain.Draft, bool, error) {
	r, err := s.queries(ctx).GetDraft(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Draft{}, false, nil
	}
	if err != nil {
		return domain.Draft{}, false, fmt.Errorf("read draft %s: %w", id, err)
	}
	return draftOf(gen.GetDraftRow(r)), true, nil
}

// LockDraft takes the draft row lock after the caller's workspace/project locks.
func (s *Store) LockDraft(ctx context.Context, id uuid.UUID) (domain.Draft, bool, error) {
	r, err := s.queries(ctx).LockDraft(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Draft{}, false, nil
	}
	if err != nil {
		return domain.Draft{}, false, fmt.Errorf("lock draft %s: %w", id, err)
	}
	return draftOf(gen.GetDraftRow(r)), true, nil
}

// ListDrafts filters before LIMIT and counts the complete scope independently
// of the cursor. The caller's read transaction keeps both queries in one snapshot.
func (s *Store) ListDrafts(ctx context.Context, r app.DraftListRead) ([]domain.Draft, int64, error) {
	q := s.queries(ctx)
	p := gen.ListDraftRowsParams{WorkspaceID: r.WorkspaceID, ActorID: r.ActorID, ProjectIds: r.ProjectIDs, PageLimit: int32(r.Limit)}
	if r.After != nil {
		p.HasAfter, p.AfterAt, p.AfterID = true, r.After.CreatedAt, r.After.ID
	}
	rows, err := q.ListDraftRows(ctx, p)
	if err != nil {
		return nil, 0, fmt.Errorf("list drafts in workspace %s: %w", r.WorkspaceID, err)
	}
	total, err := q.CountDraftRows(ctx, gen.CountDraftRowsParams{WorkspaceID: r.WorkspaceID, ActorID: r.ActorID, ProjectIds: r.ProjectIDs})
	if err != nil {
		return nil, 0, fmt.Errorf("count drafts in workspace %s: %w", r.WorkspaceID, err)
	}
	out := make([]domain.Draft, len(rows))
	for n, row := range rows {
		out[n] = draftOf(gen.GetDraftRow(row))
	}
	return out, total, nil
}

// InsertDraft writes the supplied payload and one creation audit stamp.
func (s *Store) InsertDraft(ctx context.Context, w app.DraftWrite) (domain.Draft, error) {
	r, err := s.queries(ctx).InsertDraft(ctx, gen.InsertDraftParams{ID: w.ID, WorkspaceID: w.WorkspaceID, ProjectID: w.ProjectID, Payload: w.Payload, By: &w.By, At: w.At})
	if err != nil {
		return domain.Draft{}, fmt.Errorf("insert draft %s: %w", w.ID, err)
	}
	return draftOf(gen.GetDraftRow(r)), nil
}

// UpdateDraft returns the actual stored row. SQL skips equal project/JSONB
// values entirely, preserving audit stamps even when object key order changes.
func (s *Store) UpdateDraft(ctx context.Context, w app.DraftWrite) (domain.Draft, error) {
	r, err := s.queries(ctx).UpdateDraft(ctx, gen.UpdateDraftParams{ID: w.ID, ProjectID: w.ProjectID, Payload: w.Payload, By: w.By, At: w.At})
	if err != nil {
		return domain.Draft{}, fmt.Errorf("update draft %s: %w", w.ID, err)
	}
	return draftOf(gen.GetDraftRow(r)), nil
}

// DeleteDraft soft-deletes only live rows, preserving prior deletion stamps.
func (s *Store) DeleteDraft(ctx context.Context, d app.DraftDeletion) error {
	if err := s.queries(ctx).DeleteDraft(ctx, gen.DeleteDraftParams{ID: d.ID, By: d.By, At: d.At}); err != nil {
		return fmt.Errorf("delete draft %s: %w", d.ID, err)
	}
	return nil
}

func draftOf(r gen.GetDraftRow) domain.Draft {
	return domain.Draft{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, Payload: r.Payload, CreatedByID: r.CreatedByID, UpdatedByID: r.UpdatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}
