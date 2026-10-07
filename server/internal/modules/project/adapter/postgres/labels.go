package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// labelOf is a label row as the domain has it. The queries that read a
// label each read the same columns, so their rows convert to
// gen.LabelByIDRow.
func labelOf(r gen.LabelByIDRow) domain.Label {
	return domain.Label{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, ParentID: r.ParentID, Name: r.Name, Color: r.Color,
		SortOrder: r.SortOrder, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
}

// labelTaken is err, a write's of a label, as a name another undeleted
// label of the project has, in any case: domain.ErrLabelNameTaken; any
// other error is the write's.
func labelTaken(write string, err error) error {
	if postgres.UniqueViolation(err, "labels_project_id_name_key") {
		return domain.ErrLabelNameTaken
	}
	return fmt.Errorf("%s: %w", write, err)
}

// CreateLabel inserts r and returns it as stored (app.LabelCreator). A
// name another undeleted label of the project has, in any case, is
// domain.ErrLabelNameTaken. The domain and the use case checked every
// value, so a CHECK or a foreign key's violation is a bug: an internal
// error, not a domain error.
func (s *Store) CreateLabel(ctx context.Context, r app.LabelRow) (domain.Label, error) {
	row, err := s.queries(ctx).CreateLabel(ctx, gen.CreateLabelParams{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID,
		ParentID: r.ParentID, Name: r.Name, Color: r.Color, SortOrder: r.SortOrder, CreatedBy: &r.CreatedBy, Now: r.Now})
	if err != nil {
		return domain.Label{}, labelTaken(fmt.Sprintf("create label %q", r.Name), err)
	}
	return labelOf(gen.LabelByIDRow(row)), nil
}

// LabelByID is the undeleted label id; found is false when there is none
// (app.LabelFinder).
func (s *Store) LabelByID(ctx context.Context, id uuid.UUID) (l domain.Label, found bool, err error) {
	r, err := s.queries(ctx).LabelByID(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.Label{}, false, nil
	case err != nil:
		return domain.Label{}, false, fmt.Errorf("read label %s: %w", id, err)
	}
	return labelOf(r), true, nil
}

// ListLabels lists projectID's undeleted labels, by sort order, then id
// (app.LabelLister).
func (s *Store) ListLabels(ctx context.Context, projectID uuid.UUID) ([]domain.Label, error) {
	rows, err := s.queries(ctx).ListLabels(ctx, projectID)
	if err != nil {
		return nil, fmt.Errorf("list labels of project %s: %w", projectID, err)
	}
	out := make([]domain.Label, len(rows))
	for i, r := range rows {
		out[i] = labelOf(gen.LabelByIDRow(r))
	}
	return out, nil
}

// GreatestSortOrder is the greatest sort order of projectID's undeleted
// labels, nil when it has none (app.LabelCreator).
func (s *Store) GreatestSortOrder(ctx context.Context, projectID uuid.UUID) (*float64, error) {
	greatest, err := s.queries(ctx).GreatestSortOrder(ctx, projectID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return nil, nil
	case err != nil:
		return nil, fmt.Errorf("read the greatest sort order of project %s: %w", projectID, err)
	}
	return &greatest, nil
}

// HasChildren reports whether an undeleted label has the label id as its
// parent (app.LabelUpdater).
func (s *Store) HasChildren(ctx context.Context, id uuid.UUID) (bool, error) {
	has, err := s.queries(ctx).HasChildren(ctx, id)
	if err != nil {
		return false, fmt.Errorf("read the children of label %s: %w", id, err)
	}
	return has, nil
}

// UpdateLabel changes the fields p gives of the undeleted label id, by the
// account by at now, and returns it as stored (app.LabelUpdater). A name
// another undeleted label of the project has, in any case, is
// domain.ErrLabelNameTaken; a deleted label is an error, not written.
func (s *Store) UpdateLabel(ctx context.Context, id uuid.UUID, p domain.LabelPatch, by uuid.UUID, now time.Time) (domain.Label, error) {
	r, err := s.queries(ctx).UpdateLabel(ctx, gen.UpdateLabelParams{ID: id, Name: p.Name, Color: p.Color, SetParent: p.SetParent,
		ParentID: p.ParentID, SortOrder: p.SortOrder, UpdatedBy: by, Now: now})
	if err != nil {
		return domain.Label{}, labelTaken(fmt.Sprintf("update label %s", id), err)
	}
	return labelOf(gen.LabelByIDRow(r)), nil
}

// DeleteLabel deletes the undeleted label id and the undeleted labels
// under it, by the account by at now, in one statement (app.LabelDeleter).
func (s *Store) DeleteLabel(ctx context.Context, id, by uuid.UUID, now time.Time) error {
	if err := s.queries(ctx).DeleteLabel(ctx, gen.DeleteLabelParams{ID: id, DeletedBy: by, Now: now}); err != nil {
		return fmt.Errorf("delete label %s: %w", id, err)
	}
	return nil
}
