package s3

import (
	"context"
	"net/url"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) Seal(ctx context.Context, source, target domain.ObjectKey, expectedETag string) (domain.ObjectInfo, error) {
	if _, err := domain.ParseObjectKey(string(source)); err != nil {
		return domain.ObjectInfo{}, err
	}
	if target.Kind() != 'o' {
		return domain.ObjectInfo{}, domain.ErrInvalidObjectKey
	}
	op, done, err := s.operation(ctx)
	if err != nil {
		return domain.ObjectInfo{}, err
	}
	defer done()
	if expectedETag == "" {
		if _, err := s.inspect(op, source); err != nil {
			return domain.ObjectInfo{}, err
		}
		return domain.ObjectInfo{}, domain.ErrObjectChanged
	}
	if !validETag(expectedETag) {
		return domain.ObjectInfo{}, domain.ErrObjectChanged
	}
	copySource := (&url.URL{Path: s.cfg.Bucket + "/" + s.cfg.Prefix + string(source)}).EscapedPath()
	_, err = s.client.CopyObject(op, &awss3.CopyObjectInput{Bucket: aws.String(s.cfg.Bucket), Key: aws.String(s.cfg.Prefix + string(target)), CopySource: &copySource, CopySourceIfMatch: &expectedETag, IfNoneMatch: aws.String("*")})
	if err != nil {
		return domain.ObjectInfo{}, s.objectError(op, "copy object", err)
	}
	return s.inspect(op, target)
}
