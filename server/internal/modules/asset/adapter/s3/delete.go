package s3

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) Delete(ctx context.Context, key domain.ObjectKey) error {
	if _, err := domain.ParseObjectKey(string(key)); err != nil {
		return err
	}
	op, done, err := s.operation(ctx)
	if err != nil {
		return err
	}
	defer done()
	_, err = s.client.DeleteObject(op, &awss3.DeleteObjectInput{Bucket: aws.String(s.cfg.Bucket), Key: aws.String(s.cfg.Prefix + string(key))})
	if err != nil {
		converted := s.objectError(op, "delete object", err)
		if errors.Is(converted, domain.ErrObjectNotFound) {
			return nil
		}
		return converted
	}
	return op.Err()
}
