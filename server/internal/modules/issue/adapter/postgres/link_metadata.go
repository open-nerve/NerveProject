package postgresadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// SetLinkMetadata writes only metadata; a stale or deleted link is a no-op.
func (s *Store) SetLinkMetadata(ctx context.Context, id uuid.UUID, url string, metadata domain.LinkMetadata) (bool, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return false, fmt.Errorf("encode link metadata: %w", err)
	}
	rows, err := s.queries(ctx).SetLinkMetadata(ctx, gen.SetLinkMetadataParams{ID: id, Url: url, Metadata: data})
	if err != nil {
		return false, fmt.Errorf("set link metadata: %w", err)
	}
	return rows > 0, nil
}
