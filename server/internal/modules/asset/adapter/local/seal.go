package local

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) Seal(ctx context.Context, source, destination domain.ObjectKey, expectedETag string) (domain.ObjectInfo, error) {
	if destination.Kind() != 'o' {
		return domain.ObjectInfo{}, domain.ErrInvalidObjectKey
	}
	info, file, err := s.openObject(ctx, source)
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	defer func() { _ = file.Close() }()
	if info.ETag != expectedETag {
		return domain.ObjectInfo{}, domain.ErrObjectChanged
	}
	// Keep this exact opened file through the copy. The shared writer verifies
	// the copied bytes against the precondition before making them visible.
	return s.writeObject(ctx, destination, info.Size, info.ContentType, file, expectedETag)
}
