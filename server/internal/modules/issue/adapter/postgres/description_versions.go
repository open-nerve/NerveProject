package postgresadapter

import (
	"context"
	"errors"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// LatestDescriptionVersion reads the latest live saved snapshot, using its id to break ties.
func (s *Store) LatestDescriptionVersion(ctx context.Context, id uuid.UUID) (domain.DescriptionVersion, bool, error) {
	r, err := s.queries(ctx).LatestDescriptionVersion(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.DescriptionVersion{}, false, nil
	}
	if err != nil {
		return domain.DescriptionVersion{}, false, err
	}
	v := domain.DescriptionVersion{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, OwnedByID: r.OwnedByID,
		HTML: r.DescriptionHtml, LastSavedAt: r.LastSavedAt, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}
	if r.DescriptionStripped != nil {
		v.Text = *r.DescriptionStripped
	}
	if r.CreatedByID != nil {
		v.CreatedByID = *r.CreatedByID
	}
	if r.UpdatedByID != nil {
		v.UpdatedByID = *r.UpdatedByID
	}
	return v, true, nil
}

// InsertDescriptionVersion records a new version attributed to this writer.
func (s *Store) InsertDescriptionVersion(ctx context.Context, v domain.DescriptionVersion) error {
	return s.queries(ctx).InsertDescriptionVersion(ctx, gen.InsertDescriptionVersionParams{ID: v.ID, WorkspaceID: v.WorkspaceID, ProjectID: v.ProjectID, IssueID: v.IssueID, Html: v.HTML, Text: v.Text, LastSavedAt: v.LastSavedAt, OwnedByID: v.OwnedByID, CreatedByID: v.CreatedByID, UpdatedByID: v.UpdatedByID, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt})
}

// UpdateDescriptionVersion advances a snapshot without changing its owner or origin.
func (s *Store) UpdateDescriptionVersion(ctx context.Context, v domain.DescriptionVersion) error {
	return s.queries(ctx).UpdateDescriptionVersion(ctx, gen.UpdateDescriptionVersionParams{ID: v.ID, Html: v.HTML, Text: v.Text, LastSavedAt: v.LastSavedAt, UpdatedAt: v.UpdatedAt, UpdatedByID: v.UpdatedByID})
}
