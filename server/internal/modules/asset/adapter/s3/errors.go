package s3

import (
	"context"
	"errors"
	"fmt"
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
