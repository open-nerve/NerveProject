package s3

import (
	"context"
	"errors"
	"time"
	"unicode/utf8"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) Inspect(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, error) {
	if _, err := domain.ParseObjectKey(string(key)); err != nil {
		return domain.ObjectInfo{}, err
	}
	op, done, err := s.operation(ctx)
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	defer done()
	return s.inspect(op, key)
}

func (s *Store) inspect(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, error) {
	out, err := s.client.HeadObject(ctx, &awss3.HeadObjectInput{Bucket: aws.String(s.cfg.Bucket), Key: aws.String(s.cfg.Prefix + string(key))})
	if err != nil {
		return domain.ObjectInfo{}, s.objectError(ctx, "head object", err)
	}
	if err := ctx.Err(); err != nil {
		return domain.ObjectInfo{}, err
	}
	return objectInfo(key, out.ContentLength, out.ContentType, out.ETag, out.LastModified, out.Metadata)
}

func objectInfo(key domain.ObjectKey, size *int64, media, tag *string, modified *time.Time, metadata map[string]string) (domain.ObjectInfo, error) {
	if !validObjectMetadata(size, tag, modified) || media == nil || !domain.ValidMediaType(*media) {
		return domain.ObjectInfo{}, errors.New("invalid S3 object metadata")
	}
	// The fixed scalar fields fit in 64 bytes; bound all variable fields,
	// including custom provider metadata even though Nerve does not persist it.
	bytes := 64 + len(key) + len(*media) + len(*tag)
	for k, v := range metadata {
		bytes += len(k) + len(v)
	}
	if bytes > domain.MaxMetadataBytes {
		return domain.ObjectInfo{}, errors.New("S3 object metadata exceeds limit")
	}
	return domain.ObjectInfo{Key: key, Size: *size, ContentType: *media, ETag: *tag, ModifiedAt: *modified}, nil
}

func validObjectMetadata(size *int64, tag *string, modified *time.Time) bool {
	return size != nil && *size >= 1 && *size <= domain.MaxObjectSize && tag != nil && validETag(*tag) && modified != nil && !modified.IsZero()
}

// Require one strong opaque HTTP entity tag. A wildcard or a list would turn
// an expected-version comparison into a weaker condition. No hash is assumed.
func validETag(tag string) bool {
	if len(tag) < 2 || len(tag) > 1024 || tag[0] != '"' || tag[len(tag)-1] != '"' || !utf8.ValidString(tag) {
		return false
	}
	for _, b := range []byte(tag[1 : len(tag)-1]) {
		if b < '!' || b == '"' || b == 127 {
			return false
		}
	}
	return true
}
