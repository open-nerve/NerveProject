package s3

import (
	"context"
	"errors"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// SDK/url errors may contain signed URLs, credentials or untrusted messages.
// Only fixed categories, HTTP status and recognizable context causes escape.
func storageError(operation string, err error) error {
	if err == nil {
		return nil
	}
	for _, cause := range []error{context.Canceled, context.DeadlineExceeded} {
		if errors.Is(err, cause) {
			return fmt.Errorf("S3 %s: %w", operation, cause)
		}
	}
	var response interface{ HTTPStatusCode() int }
	if errors.As(err, &response) {
		return fmt.Errorf("S3 %s: backend HTTP status %d", operation, response.HTTPStatusCode())
	}
	return fmt.Errorf("S3 %s: transport or protocol failure", operation)
}

func (s *Store) objectError(ctx context.Context, operation string, err error) error {
	if cause := ctx.Err(); cause != nil {
		return cause
	}
	var response interface{ HTTPStatusCode() int }
	if !errors.As(err, &response) {
		return storageError(operation, err)
	}
	// An HTTP status also wraps SDK body read/XML failures. Only a completely
	// decoded API error is a business result; in particular an invalid 404
	// must not acknowledge deletion and lose the object's recovery record.
	var api interface{ ErrorCode() string }
	if !errors.As(err, &api) {
		return storageError(operation, err)
	}
	switch response.HTTPStatusCode() {
	case 409, 412:
		return domain.ErrObjectChanged
	case 404:
		if api.ErrorCode() == "NoSuchBucket" {
			return storageError(operation, err)
		}
		// HEAD has no error body; some compatible services also use NoSuchKey
		// for a deleted bucket. Confirm bucket existence in the SAME budget.
		_, bucketErr := s.client.HeadBucket(ctx, &awss3.HeadBucketInput{Bucket: aws.String(s.cfg.Bucket)})
		if bucketErr != nil {
			return storageError("confirm bucket after object404", bucketErr)
		}
		if cause := ctx.Err(); cause != nil {
			return cause
		}
		return domain.ErrObjectNotFound
	default:
		return storageError(operation, err)
	}
}
