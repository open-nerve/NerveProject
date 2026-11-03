package s3

import (
	"context"
	"errors"
	"mime"
	"net/url"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func (s *Store) SignUpload(ctx context.Context, g domain.UploadGrant) (domain.SignedRequest, error) {
	op, done, err := s.operation(ctx)
	if err != nil {
		return domain.SignedRequest{}, err
	}
	defer done()
	now := s.deps.RuntimeNow()
	if err := domain.ValidateUpload(g, s.cfg.SizeLimit, now); err != nil {
		return domain.SignedRequest{}, err
	}
	g.ExpiresAt = g.ExpiresAt.UTC().Truncate(time.Second)
	if !g.ExpiresAt.After(now) {
		return domain.SignedRequest{}, domain.ErrInvalidUpload
	}
	input := &awss3.PutObjectInput{Bucket: aws.String(s.cfg.Bucket), Key: aws.String(s.cfg.Prefix + string(g.Key)), ContentLength: aws.Int64(g.Size), ContentType: aws.String(g.ContentType)}
	// Body is deliberately nil: the browser supplies the actual bytes, while
	// ContentLength/ContentType and Host remain authenticated by SigV4.
	raw, err := awss3.NewPresignClient(s.client).PresignPutObject(op, input, exactDeadline(g.ExpiresAt, domain.MaxUploadTTL, domain.ErrInvalidUpload))
	signed, err := s.finishSignature(op, raw, g.ExpiresAt, domain.ErrInvalidUpload, err)
	if err != nil {
		return domain.SignedRequest{}, err
	}
	signed.Headers = map[string]string{"Content-Type": g.ContentType}
	return signed, nil
}

func (s *Store) SignRead(ctx context.Context, g domain.ReadGrant) (domain.SignedRequest, error) {
	op, done, err := s.operation(ctx)
	if err != nil {
		return domain.SignedRequest{}, err
	}
	defer done()
	now := s.deps.RuntimeNow()
	if err := domain.ValidateRead(g, now); err != nil {
		return domain.SignedRequest{}, err
	}
	g.ExpiresAt = g.ExpiresAt.UTC().Truncate(time.Second)
	if !g.ExpiresAt.After(now) {
		return domain.SignedRequest{}, domain.ErrInvalidRead
	}
	kind := "attachment"
	if g.Inline {
		kind = "inline"
	}
	disposition := mime.FormatMediaType(kind, map[string]string{"filename": g.Name})
	bucket, key := aws.String(s.cfg.Bucket), aws.String(s.cfg.Prefix+string(g.Key))
	client := awss3.NewPresignClient(s.client)
	option := exactDeadline(g.ExpiresAt, domain.MaxReadTTL, domain.ErrInvalidRead)
	var raw *v4.PresignedHTTPRequest
	if g.Method == "HEAD" {
		raw, err = client.PresignHeadObject(op, &awss3.HeadObjectInput{Bucket: bucket, Key: key, ResponseContentType: aws.String(g.ContentType), ResponseContentDisposition: &disposition, ResponseCacheControl: aws.String("private, no-store")}, option)
	} else {
		raw, err = client.PresignGetObject(op, &awss3.GetObjectInput{Bucket: bucket, Key: key, ResponseContentType: aws.String(g.ContentType), ResponseContentDisposition: &disposition, ResponseCacheControl: aws.String("private, no-store")}, option)
	}
	return s.finishSignature(op, raw, g.ExpiresAt, domain.ErrInvalidRead, err)
}

func (s *Store) finishSignature(ctx context.Context, raw *v4.PresignedHTTPRequest, expires time.Time, invalid, errorIn error) (domain.SignedRequest, error) {
	if err := ctx.Err(); err != nil {
		return domain.SignedRequest{}, err
	}
	if errors.Is(errorIn, invalid) {
		return domain.SignedRequest{}, invalid
	}
	if errorIn != nil {
		return domain.SignedRequest{}, storageError("presign", errorIn)
	}
	if raw == nil || !expires.After(s.deps.RuntimeNow()) {
		return domain.SignedRequest{}, invalid
	}
	u, err := url.Parse(raw.URL)
	if err != nil {
		return domain.SignedRequest{}, invalid
	}
	origin, err := urlOrigin(u)
	if err != nil || origin != s.origin {
		return domain.SignedRequest{}, invalid
	}
	return domain.SignedRequest{Method: raw.Method, URL: raw.URL, ExpiresAt: expires}, nil
}
