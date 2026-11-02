package local

import (
	"context"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) OpenTransfer(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, io.ReadSeekCloser, error) {
	info, file, err := s.openObject(ctx, key)
	if err != nil {
		return domain.ObjectInfo{}, nil, err
	}
	return info, s.ownReader(ctx, file), nil
}
