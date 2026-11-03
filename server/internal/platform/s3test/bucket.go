package s3test

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

type Bucket struct {
	Endpoint, Region, Name, AccessKeyID, SecretAccessKey string
	client                                               *s3.Client
}

// A service can host many independent test worlds; none shares a bucket.
// Removing the owned container also removes all bucket versions after tests.
func (s *Service) NewBucket(t testing.TB) *Bucket {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 90*time.Second)
	defer cancel()
	b, err := s.newBucket(ctx)
	if err != nil {
		t.Fatalf("s3test bucket: %v", err)
	}
	return b
}

func (b *Bucket) Client() *s3.Client { return b.client }

func (s *Service) newBucket(ctx context.Context) (*Bucket, error) {
	b := &Bucket{Endpoint: s.endpoint, Region: "us-east-1", Name: "nerve-" + strings.ToLower(rand.Text()), AccessKeyID: s.access, SecretAccessKey: s.secret, client: s.client}
	if err := retryReady(ctx, func() error {
		_, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: aws.String(b.Name)})
		return err
	}); err != nil {
		return nil, err
	}
	// HTTP 403 from the root is only transport readiness. Prove that this
	// bucket has a writable volume, return actual bytes, then remove the probe.
	if err := retryReady(ctx, func() error { return b.probe(ctx) }); err != nil {
		return nil, err
	}
	return b, nil
}

func (b *Bucket) probe(ctx context.Context) error {
	key := "readiness"
	_, err := b.client.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(b.Name), Key: &key, Body: strings.NewReader("ready")})
	if err != nil {
		return err
	}
	object, err := b.client.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(b.Name), Key: &key})
	if err != nil {
		return err
	}
	body, readErr := io.ReadAll(io.LimitReader(object.Body, 6))
	closeErr := object.Body.Close()
	if readErr != nil {
		return readErr
	}
	if closeErr != nil {
		return closeErr
	}
	if string(body) != "ready" {
		return fmt.Errorf("s3test readiness bytes differ")
	}
	_, err = b.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(b.Name), Key: &key})
	return err
}

func ready(ctx context.Context, s *Service) error {
	b, err := s.newBucket(ctx)
	if err != nil {
		return err
	}
	_, err = s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: aws.String(b.Name)})
	return err
}

func retryReady(ctx context.Context, attempt func() error) error {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := attempt(); err == nil {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
