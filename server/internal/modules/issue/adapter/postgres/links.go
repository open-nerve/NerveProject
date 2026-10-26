package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// GetLink reads a live link without deciding visibility of its work item.
func (s *Store) GetLink(ctx context.Context, id uuid.UUID) (domain.Link, bool, error) {
	r, err := s.queries(ctx).GetLink(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Link{}, false, nil
	}
	if err != nil {
		return domain.Link{}, false, fmt.Errorf("read link %s: %w", id, err)
	}
	link, err := linkOf(gen.LinksRow(r))
	return link, err == nil, err
}

// Links reads only this issue's live links, newest first with an id tie break.
func (s *Store) Links(ctx context.Context, issueID uuid.UUID) ([]domain.Link, error) {
	rows, err := s.queries(ctx).Links(ctx, issueID)
	if err != nil {
		return nil, fmt.Errorf("read links of work item %s: %w", issueID, err)
	}
	links := make([]domain.Link, len(rows))
	for n, row := range rows {
		links[n], err = linkOf(row)
		if err != nil {
			return nil, err
		}
	}
	return links, nil
}

// LinkURLExists compares the stored original URL, excluding this link id.
func (s *Store) LinkURLExists(ctx context.Context, issueID, excludeID uuid.UUID, url string) (bool, error) {
	exists, err := s.queries(ctx).LinkURLExists(ctx, gen.LinkURLExistsParams{IssueID: issueID, ExcludeID: excludeID, Url: url})
	if err != nil {
		return false, fmt.Errorf("check link URL of work item %s: %w", issueID, err)
	}
	return exists, nil
}

// InsertLink creates a link with pending metadata and one creation audit stamp.
func (s *Store) InsertLink(ctx context.Context, w app.LinkWrite) error {
	l := w.Link
	err := s.queries(ctx).InsertLink(ctx, gen.InsertLinkParams{ID: l.ID, WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID, IssueID: l.IssueID, Url: l.URL, Title: l.Title, By: &w.By, At: w.At})
	if err != nil {
		return fmt.Errorf("insert link %s: %w", l.ID, linkConflict(err))
	}
	return nil
}

// UpdateLink lets SQL preserve current metadata unless the URL changes, and
// returns the complete row produced by that write rather than the input copy.
func (s *Store) UpdateLink(ctx context.Context, w app.LinkWrite) (domain.Link, error) {
	r, err := s.queries(ctx).UpdateLink(ctx, gen.UpdateLinkParams{ID: w.Link.ID, Url: w.Link.URL, Title: w.Link.Title, By: &w.By, At: w.At})
	if err != nil {
		return domain.Link{}, fmt.Errorf("update link %s: %w", w.Link.ID, linkConflict(err))
	}
	return linkOf(gen.LinksRow(r))
}

// DeleteLink preserves prior deletion stamps by updating only live rows.
func (s *Store) DeleteLink(ctx context.Context, d app.LinkDeletion) error {
	err := s.queries(ctx).DeleteLink(ctx, gen.DeleteLinkParams{ID: d.ID, By: &d.By, At: d.At})
	if err != nil {
		return fmt.Errorf("delete link %s: %w", d.ID, err)
	}
	return nil
}

func linkOf(r gen.LinksRow) (domain.Link, error) {
	var metadata domain.LinkMetadata
	if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
		return domain.Link{}, fmt.Errorf("decode metadata of link %s: %w", r.ID, err)
	}
	return domain.Link{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, IssueID: r.IssueID, URL: r.Url, Title: r.Title, Metadata: metadata, CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}
