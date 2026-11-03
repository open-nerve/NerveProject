package app

import (
	"context"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// ContentValidator consumes a cancellable reader; its caller owns Close.
type ContentValidator interface {
	Validate(context.Context, io.Reader, int64, string, bool) (domain.ContentProof, error)
}

// SealingDeps are the byte-operation ports; none acquires database resources.
type SealingDeps struct {
	Inspector ObjectInspector
	Reader    ObjectReader
	Sealer    ObjectSealer
	Validator ContentValidator
}
