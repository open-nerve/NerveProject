package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

type result struct {
	Size        int64  `json:"size,omitempty"`
	SHA256      string `json:"sha256,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	ETag        string `json:"etag,omitempty"`
	Exists      *bool  `json:"exists,omitempty"`
}

func execute(ctx context.Context, r request) (result, error) {
	transport := &http.Transport{Proxy: nil, DialContext: (&net.Dialer{Timeout: 10 * time.Second}).DialContext, MaxResponseHeaderBytes: 32 << 10}
	defer transport.CloseIdleConnections()
	client := s3.NewFromConfig(aws.Config{Region: r.Region, Credentials: credentials.NewStaticCredentialsProvider(r.AccessKeyID, r.SecretAccessKey, ""), RetryMaxAttempts: 1, HTTPClient: &http.Client{Transport: transport, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, func(o *s3.Options) {
		o.BaseEndpoint = &r.Endpoint
		o.UsePathStyle = true
		o.DisableS3ExpressSessionAuth = aws.Bool(true)
		o.DisableLogOutputChecksumValidationSkipped = true
	})
	switch r.Action {
	case "create-bucket":
		return result{}, createBucket(ctx, client, r)
	case "cors":
		_, err := client.PutBucketCors(ctx, &s3.PutBucketCorsInput{Bucket: &r.Bucket, CORSConfiguration: &types.CORSConfiguration{CORSRules: []types.CORSRule{{AllowedOrigins: []string{r.Origin}, AllowedMethods: []string{"PUT", "GET", "HEAD"}, AllowedHeaders: []string{"Content-Type"}, ExposeHeaders: []string{"ETag", "Content-Length", "Content-Type", "Content-Disposition"}}}}})
		return result{}, err
	case "delete":
		_, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &r.Bucket, Key: aws.String(r.Prefix + r.Key)})
		return result{}, err
	case "inspect":
		return inspect(ctx, client, r)
	case "delete-bucket":
		return result{}, deleteBucket(ctx, client, r)
	case "head-bucket":
		return headBucket(ctx, client, r)
	default:
		return result{}, errRequest
	}
}

func createBucket(ctx context.Context, client *s3.Client, r request) error {
	if _, err := client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &r.Bucket}); err != nil {
		return err
	}
	key := r.Prefix + "readiness"
	if _, err := client.PutObject(ctx, &s3.PutObjectInput{Bucket: &r.Bucket, Key: &key, Body: strings.NewReader("ready")}); err != nil {
		return err
	}
	object, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: &r.Bucket, Key: &key})
	if err != nil {
		return err
	}
	data, readErr := io.ReadAll(io.LimitReader(object.Body, 6))
	closeErr := object.Body.Close()
	if readErr != nil || closeErr != nil || string(data) != "ready" {
		return errOperation
	}
	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &r.Bucket, Key: &key})
	return err
}

func inspect(ctx context.Context, client *s3.Client, r request) (result, error) {
	object, err := client.GetObject(ctx, &s3.GetObjectInput{Bucket: &r.Bucket, Key: aws.String(r.Prefix + r.Key)})
	if err != nil {
		return result{}, err
	}
	defer func() { _ = object.Body.Close() }()
	const maxSize = 100 << 20
	if object.ContentLength == nil || *object.ContentLength < 1 || *object.ContentLength > maxSize {
		return result{}, errOperation
	}
	hash := sha256.New()
	n, err := io.Copy(hash, io.LimitReader(object.Body, maxSize+1))
	if err != nil || n != *object.ContentLength || n > maxSize {
		return result{}, errOperation
	}
	return result{Size: n, SHA256: hex.EncodeToString(hash.Sum(nil)), ContentType: aws.ToString(object.ContentType), ETag: aws.ToString(object.ETag)}, nil
}

// maxBucketKeys bounds one removal; an e2e world writes far fewer objects.
const maxBucketKeys = 10_000

// deleteBucket empties the owned bucket, every key whatever its prefix (the
// random bucket is the world's alone), then removes it, which also frees the
// service's volumes for it. A bucket already gone counts as removed.
func deleteBucket(ctx context.Context, client *s3.Client, r request) error {
	var token *string
	deleted := 0
	for {
		page, err := client.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: &r.Bucket, ContinuationToken: token})
		var gone *types.NoSuchBucket
		if errors.As(err, &gone) {
			return nil
		}
		if err != nil {
			return err
		}
		for _, object := range page.Contents {
			if deleted++; deleted > maxBucketKeys {
				return errOperation
			}
			if _, err := client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &r.Bucket, Key: object.Key}); err != nil {
				return err
			}
		}
		if !aws.ToBool(page.IsTruncated) {
			break
		}
		if token = page.NextContinuationToken; token == nil {
			return errOperation
		}
	}
	_, err := client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &r.Bucket})
	return err
}

// headBucket answers whether the owned bucket exists; only an exact 404 is absence.
func headBucket(ctx context.Context, client *s3.Client, r request) (result, error) {
	_, err := client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &r.Bucket})
	var absent *types.NotFound
	switch {
	case err == nil:
		return result{Exists: aws.Bool(true)}, nil
	case errors.As(err, &absent):
		return result{Exists: aws.Bool(false)}, nil
	default:
		return result{}, err
	}
}
