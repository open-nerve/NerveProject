package app

import (
	"context"
	"errors"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type verifiedObject struct {
	info  domain.ObjectInfo
	proof domain.ContentProof
}

// Both complete streams belong to the same deadline. No database, gate or
// transaction is available to this operation; every opened reader is closed.
func sealContent(ctx context.Context, ports SealingDeps, source domain.ObjectRecord, destination domain.ObjectKey, imageRequired bool, limit int64) (verifiedObject, error) {
	original, err := inspectContent(ctx, ports, source.Key, source.Size, source.ContentType, imageRequired, limit)
	if err != nil {
		return verifiedObject{}, err
	}
	if source.ETag != "" && source.ETag != original.info.ETag || source.SHA256 != "" && source.SHA256 != original.proof.SHA256 {
		return verifiedObject{}, domain.ErrObjectChanged
	}
	sealed, err := ports.Sealer.Seal(ctx, source.Key, destination, original.info.ETag)
	if err != nil {
		return verifiedObject{}, err
	}
	target, err := inspectContent(ctx, ports, destination, source.Size, source.ContentType, imageRequired, limit)
	if err != nil {
		return verifiedObject{}, err
	}
	if sealed.Key != target.info.Key || sealed.ETag != target.info.ETag || sealed.Size != target.info.Size || sealed.ContentType != target.info.ContentType || original.proof.SHA256 != target.proof.SHA256 {
		return verifiedObject{}, domain.ErrObjectChanged
	}
	return target, nil
}

func inspectContent(ctx context.Context, ports SealingDeps, key domain.ObjectKey, size int64, contentType string, imageRequired bool, limit int64) (verifiedObject, error) {
	if err := ctx.Err(); err != nil {
		return verifiedObject{}, err
	}
	info, err := ports.Inspector.Inspect(ctx, key)
	if err != nil {
		return verifiedObject{}, err
	}
	if info.Size > limit || info.Size > domain.MaxObjectSize {
		return verifiedObject{}, domain.ErrAssetTooLarge
	}
	if info.Key != key || info.Size != size || info.ContentType != contentType {
		return verifiedObject{}, domain.ErrAssetContentInvalid
	}
	if info.ETag == "" || len(info.ETag) > 1024 {
		return verifiedObject{}, domain.ErrObjectChanged
	}
	reader, err := ports.Reader.Open(ctx, key, info.ETag)
	if err != nil {
		return verifiedObject{}, err
	}
	proof, readErr := ports.Validator.Validate(ctx, reader, size, contentType, imageRequired)
	closeErr := reader.Close()
	if readErr != nil {
		return verifiedObject{}, readErr
	}
	if closeErr != nil {
		return verifiedObject{}, closeErr
	}
	if err := ctx.Err(); err != nil {
		return verifiedObject{}, err
	}
	return verifiedObject{info: info, proof: proof}, nil
}

// A request cancellation remains cancellation. An independent storage timeout
// is retryable, never evidence that an object is missing. Backend messages may
// contain paths or signed URLs, so they do not enter the public error/log path.
func publicationError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	switch {
	case errors.Is(err, domain.ErrObjectNotFound):
		return domain.ErrAssetObjectMissing
	case errors.Is(err, domain.ErrObjectChanged), errors.Is(err, domain.ErrObjectExists):
		return domain.ErrAssetObjectChanged
	case errors.Is(err, domain.ErrAssetContentInvalid), errors.Is(err, domain.ErrAssetTooLarge), errors.Is(err, domain.ErrAssetTypeUnsupported):
		return err
	default:
		return domain.AssetStorageUnavailable()
	}
}
