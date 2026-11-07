package postgresadapter

import (
	"context"
	"encoding/json"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// SetModuleLinkMetadata writes a crawl's metadata alone to the link id only
// while it is undeleted and has the address url, as it is spelt
// (link_metadata.sql); matched reports whether it did.
func (s *Store) SetModuleLinkMetadata(ctx context.Context, id uuid.UUID, url string, metadata shared.LinkMetadata) (bool, error) {
	data, err := json.Marshal(metadata)
	if err != nil {
		return false, fmt.Errorf("encode the metadata of module link %s: %w", id, err)
	}
	rows, err := s.queries(ctx).SetModuleLinkMetadata(ctx, gen.SetModuleLinkMetadataParams{ID: id, Url: url, Metadata: data})
	if err != nil {
		return false, fmt.Errorf("set the metadata of module link %s: %w", id, err)
	}
	return rows > 0, nil
}
