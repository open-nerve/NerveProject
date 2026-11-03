package s3

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) List(ctx context.Context, token string, limit int) (app.ObjectPage, error) {
	invalid := errors.New("invalid S3 inventory page or cursor")
	if limit < 1 || limit > 1000 || len(token) > 8192 {
		return app.ObjectPage{}, invalid
	}
	op, done, err := s.operation(ctx)
	if err != nil {
		return app.ObjectPage{}, err
	}
	defer done()
	// Bound XML before SDK deserialization, including unknown elements and
	// chunked bodies. Object downloads do not carry this metadata-only budget.
	op = context.WithValue(op, metadataBudgetKey{}, int64(64<<10+limit*domain.MaxMetadataBytes))
	input := &awss3.ListObjectsV2Input{Bucket: &s.cfg.Bucket, Prefix: &s.cfg.Prefix, MaxKeys: aws.Int32(int32(limit))}
	if token != "" {
		input.ContinuationToken = &token
	}
	out, err := s.client.ListObjectsV2(op, input)
	if err != nil {
		return app.ObjectPage{}, storageError("list objects", err)
	}
	if err := op.Err(); err != nil {
		return app.ObjectPage{}, err
	}
	next := aws.ToString(out.NextContinuationToken)
	if out.IsTruncated == nil || len(out.Contents) > limit || len(out.CommonPrefixes) != 0 || len(next) > 8192 ||
		(*out.IsTruncated && (next == "" || next == token || len(out.Contents) == 0)) || (!*out.IsTruncated && next != "") ||
		(out.Prefix != nil && *out.Prefix != s.cfg.Prefix) || (out.Name != nil && *out.Name != s.cfg.Bucket) ||
		(out.KeyCount != nil && int(*out.KeyCount) != len(out.Contents)) || (out.MaxKeys != nil && (*out.MaxKeys < 1 || int(*out.MaxKeys) > limit)) {
		return app.ObjectPage{}, invalid
	}
	page := app.ObjectPage{Next: next, Objects: make([]domain.ObjectInfo, 0, len(out.Contents))}
	var last domain.ObjectKey
	for _, object := range out.Contents {
		raw, ok := strings.CutPrefix(aws.ToString(object.Key), s.cfg.Prefix)
		key, err := domain.ParseObjectKey(raw)
		if !ok || err != nil || key <= last || !validObjectMetadata(object.Size, object.ETag, object.LastModified) {
			return app.ObjectPage{}, invalid
		}
		page.Objects = append(page.Objects, domain.ObjectInfo{Key: key, Size: *object.Size, ETag: *object.ETag, ModifiedAt: *object.LastModified})
		last = key
	}
	return page, nil
}

func (s *Store) SweepTemporary(ctx context.Context, cutoff time.Time, limit int, token string) (string, int, error) {
	op, done, err := s.operation(ctx)
	if err != nil {
		return "", 0, err
	}
	defer done()
	if cutoff.IsZero() || limit < 1 || limit > 1000 || token != "" {
		return "", 0, errors.New("invalid S3 temporary inventory parameters")
	}
	return "", 0, op.Err()
}
