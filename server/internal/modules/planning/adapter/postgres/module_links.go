package postgresadapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

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
	var metadata shared.LinkMetadata
	if err := json.Unmarshal(r.Metadata, &metadata); err != nil {
		return domain.ModuleLink{}, false, fmt.Errorf("decode the metadata of module link %s: %w", id, err)
	}
	return domain.ModuleLink{ID: r.ID, WorkspaceID: r.WorkspaceID, ProjectID: r.ProjectID, ModuleID: r.ModuleID, URL: r.Url, Title: r.Title,
		Metadata: metadata, CreatedByID: r.CreatedByID, CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt}, true, nil
}
