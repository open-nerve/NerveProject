// Package app declares the asset use cases' consumer-owned dependencies.
package app

import (
	"context"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type UploadSigner interface {
	SignUpload(context.Context, domain.UploadGrant) (domain.SignedRequest, error)
}

type ReadSigner interface {
	SignRead(context.Context, domain.ReadGrant) (domain.SignedRequest, error)
}

type ObjectInspector interface {
	Inspect(context.Context, domain.ObjectKey) (domain.ObjectInfo, error)
}

// ObjectReader opens the expected version. The caller owns the returned reader;
// cancellation ends its reads, and Close releases its underlying resource.
type ObjectReader interface {
	Open(context.Context, domain.ObjectKey, string) (io.ReadCloser, error)
}

// ObjectSealer copies the expected source into a new immutable destination.
// Its result describes the bytes copied, not merely the source's old metadata.
type ObjectSealer interface {
	Seal(context.Context, domain.ObjectKey, domain.ObjectKey, string) (domain.ObjectInfo, error)
}

type ObjectDeleter interface {
	Delete(context.Context, domain.ObjectKey) error
}

// MAC is supplied for one explicit purpose; Verify compares tags in constant time.
type MAC interface {
	Tag([]byte) [16]byte
	Verify([]byte, [16]byte) bool
}
