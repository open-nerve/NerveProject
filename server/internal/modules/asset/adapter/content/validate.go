// Package content checks bounded file content without owning its reader.
package content

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"golang.org/x/image/webp"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

const (
	maxHeaderBytes = 1 << 20
	maxImageSide   = 16384
	maxImagePixels = 40000000
)

// Validator hashes the complete bounded stream and decodes only an image header.
// Readers supplied by ObjectReader must honor cancellation while a Read blocks.
type Validator struct{}

func (Validator) Validate(ctx context.Context, reader io.Reader, size int64, contentType string, imageRequired bool) (domain.ContentProof, error) {
	if err := ctx.Err(); err != nil {
		return domain.ContentProof{}, err
	}
	if size > domain.MaxObjectSize {
		return domain.ContentProof{}, domain.ErrAssetTooLarge
	}
	if size < 1 || !domain.ValidMediaType(contentType) {
		return domain.ContentProof{}, domain.ErrAssetContentInvalid
	}
	if imageRequired && !domain.RasterMediaType(contentType) {
		return domain.ContentProof{}, domain.ErrAssetTypeUnsupported
	}
	limited := io.LimitReader(reader, size+1)
	digest := sha256.New()
	buffer := make([]byte, 64<<10)
	var header []byte
	var total int64
	emptyReads := 0
	for {
		if err := ctx.Err(); err != nil {
			return domain.ContentProof{}, err
		}
		n, err := limited.Read(buffer)
		if n > 0 {
			emptyReads = 0
			total += int64(n)
			if total > size {
				return domain.ContentProof{}, domain.ErrAssetContentInvalid
			}
			_, _ = digest.Write(buffer[:n])
			if imageRequired && len(header) < maxHeaderBytes {
				header = append(header, buffer[:min(n, maxHeaderBytes-len(header))]...)
			}
		} else if err == nil {
			emptyReads++
			if emptyReads >= 100 {
				return domain.ContentProof{}, io.ErrNoProgress
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				return domain.ContentProof{}, err
			}
			break
		}
	}
	if err := ctx.Err(); err != nil {
		return domain.ContentProof{}, err
	}
	if total != size {
		return domain.ContentProof{}, domain.ErrAssetContentInvalid
	}
	proof := domain.ContentProof{SHA256: hex.EncodeToString(digest.Sum(nil))}
	if imageRequired {
		cfg, err := imageHeader(header, contentType)
		if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > maxImageSide || cfg.Height > maxImageSide || cfg.Width > maxImagePixels/cfg.Height {
			return domain.ContentProof{}, domain.ErrAssetContentInvalid
		}
		proof.Width, proof.Height = &cfg.Width, &cfg.Height
	}
	return proof, nil
}

func imageHeader(header []byte, contentType string) (image.Config, error) {
	reader := bytes.NewReader(header)
	switch contentType {
	case "image/png":
		return png.DecodeConfig(reader)
	case "image/jpeg":
		return jpeg.DecodeConfig(reader)
	case "image/gif":
		return gif.DecodeConfig(reader)
	case "image/webp":
		return webp.DecodeConfig(reader)
	default:
		return image.Config{}, domain.ErrAssetTypeUnsupported
	}
}
