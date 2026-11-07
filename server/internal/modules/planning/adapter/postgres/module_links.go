package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// GetModuleLink reads the undeleted module link id; found is false when
// there is none (module_links.sql). Its module is read apart (GetModule).
func (s *Store) GetModuleLink(ctx context.Context, id uuid.UUID) (domain.ModuleLink, bool, error) {
	r, err := s.queries(ctx).GetModuleLink(ctx, id)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return domain.ModuleLink{}, false, nil
	case err != nil:
		return domain.ModuleLink{}, false, fmt.Errorf("read module link %s: %w", id, err)
	}
	l, err := moduleLinkOf(r)
	return l, err == nil, err
}

// ListModuleLinks reads the module's undeleted links, the latest created
// first, then by id descending; none as an empty list.
func (s *Store) ListModuleLinks(ctx context.Context, module uuid.UUID) ([]domain.ModuleLink, error) {
	rows, err := s.queries(ctx).ListModuleLinks(ctx, module)
	if err != nil {
		return nil, fmt.Errorf("list the links of module %s: %w", module, err)
	}
	out := make([]domain.ModuleLink, len(rows))
	for i, r := range rows {
		if out[i], err = moduleLinkOf(gen.GetModuleLinkRow(r)); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// InsertModuleLink writes the new link l, its metadata empty, created and
// last changed by its creator at its creation. An address another undeleted
// link of the module has is domain.ErrModuleLinkExists (moduleLinkTaken).
func (s *Store) InsertModuleLink(ctx context.Context, l domain.ModuleLink) error {
	err := s.queries(ctx).InsertModuleLink(ctx, gen.InsertModuleLinkParams{ID: l.ID, WorkspaceID: l.WorkspaceID, ProjectID: l.ProjectID,
		ModuleID: l.ModuleID, Url: l.URL, Title: l.Title, By: l.CreatedByID, At: l.CreatedAt})
	if err != nil {
		return moduleLinkTaken(fmt.Sprintf("insert module link %s", l.ID), err)
	}
	return nil
}

// UpdateModuleLink writes l's address and title, last changed by by at its
// UpdatedAt, and answers the link as written: its metadata emptied when the
// address changes, else as stored. An address another undeleted link of the
// module has is domain.ErrModuleLinkExists (moduleLinkTaken).
func (s *Store) UpdateModuleLink(ctx context.Context, l domain.ModuleLink, by uuid.UUID) (domain.ModuleLink, error) {
	r, err := s.queries(ctx).UpdateModuleLink(ctx, gen.UpdateModuleLinkParams{ID: l.ID, Url: l.URL, Title: l.Title, By: &by, At: l.UpdatedAt})
	if err != nil {
		return domain.ModuleLink{}, moduleLinkTaken(fmt.Sprintf("update module link %s", l.ID), err)
	}
	return moduleLinkOf(gen.GetModuleLinkRow(r))
}

// DeleteModuleLink soft-deletes the link id, deleted and last changed by by
// at at; one deleted already keeps its deletion.
func (s *Store) DeleteModuleLink(ctx context.Context, id uuid.UUID, at time.Time, by uuid.UUID) error {
	if err := s.queries(ctx).DeleteModuleLink(ctx, gen.DeleteModuleLinkParams{ID: id, At: at, By: by}); err != nil {
		return fmt.Errorf("delete module link %s: %w", id, err)
	}
	return nil
}

// moduleLinkTaken is err, a write's of a module link, as an address another
// undeleted link of the module has: domain.ErrModuleLinkExists when it broke
// module_links_module_id_url_key, the exclusion of a second live address
// (SQLSTATE 23P01, as issue_links'); any other error is the write's.
func moduleLinkTaken(write string, err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23P01" && pg.ConstraintName == "module_links_module_id_url_key" {
		return domain.ErrModuleLinkExists
	}
	return fmt.Errorf("%s: %w", write, err)
}

// moduleLinkOf is the link a row of module_links.sql's reads holds.
func moduleLinkOf(r gen.GetModuleLinkRow) (domain.ModuleLink, error) {
	var metadata shared.LinkMetadata
	if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
		return domain.ModuleLink{}, fmt.Errorf("decode the metadata of module link %s: %w", r.ID, err)
	}
	return domain.ModuleLink{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, ModuleID: r.ModuleID, URL: r.Url, Title: r.Title,
		Metadata: metadata, CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, nil
}
