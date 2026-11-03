// Package s3 implements asset storage using a private, never-versioned bucket.
package s3

import (
	"context"
	"errors"
	"io/fs"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

type Deps struct{ RuntimeNow func() time.Time }
type Store struct {
	cfg       Config
	deps      Deps
	client    *awss3.Client
	origin    string
	transport *http.Transport
	life      context.Context
	cancel    context.CancelFunc
	closed    atomic.Bool
	closeOnce sync.Once
}

func New(ctx context.Context, cfg Config, deps Deps) (*Store, error) {
	return newStore(ctx, cfg, deps, newTransport())
}

// The unexported transport argument lets tests use their own TLS/DNS endpoints.
// Production has no TLS, redirect or dial bypass setting.
func newStore(ctx context.Context, cfg Config, deps Deps, transport *http.Transport) (*Store, error) {
	if err := cfg.validate(); err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	endpoint, err := canonicalEndpoint(cfg.Endpoint)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	cfg.Endpoint = endpoint
	if deps.RuntimeNow == nil {
		transport.CloseIdleConnections()
		return nil, errors.New("S3 storage lacks runtime clock")
	}
	if err := ctx.Err(); err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	resolved, err := awss3.NewDefaultEndpointResolverV2().ResolveEndpoint(ctx, awss3.EndpointParameters{
		Bucket: aws.String(cfg.Bucket), Region: aws.String(cfg.Region), Endpoint: aws.String(cfg.Endpoint),
		ForcePathStyle: aws.Bool(cfg.UsePathStyle), DisableS3ExpressSessionAuth: aws.Bool(true),
	})
	if err != nil {
		transport.CloseIdleConnections()
		return nil, storageError("resolve endpoint", err)
	}
	origin, err := urlOrigin(&resolved.URI)
	if err != nil {
		transport.CloseIdleConnections()
		return nil, err
	}
	client := awss3.NewFromConfig(aws.Config{
		Region: cfg.Region, Credentials: credentials.NewStaticCredentialsProvider(cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken),
		HTTPClient:       &http.Client{Transport: originTransport{origin: origin, next: transport}, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		RetryMaxAttempts: 1,
	}, func(o *awss3.Options) {
		o.BaseEndpoint = aws.String(cfg.Endpoint)
		o.UsePathStyle = cfg.UsePathStyle
		o.DisableS3ExpressSessionAuth = aws.Bool(true)
	})
	life, cancel := context.WithCancel(context.Background())
	s := &Store{cfg: cfg, deps: deps, client: client, origin: origin, transport: transport, life: life, cancel: cancel}
	if err := s.checkBucket(ctx); err != nil {
		_ = s.Close()
		return nil, err
	}
	return s, nil
}

// Both read-only checks share this one operation budget.
func (s *Store) checkBucket(ctx context.Context) error {
	op, done, err := s.operation(ctx)
	if err != nil {
		return err
	}
	defer done()
	_, err = s.client.HeadBucket(op, &awss3.HeadBucketInput{Bucket: aws.String(s.cfg.Bucket)})
	if err != nil {
		return storageError("head bucket", err)
	}
	version, err := s.client.GetBucketVersioning(op, &awss3.GetBucketVersioningInput{Bucket: aws.String(s.cfg.Bucket)})
	if err != nil {
		return storageError("get bucket versioning", err)
	}
	if version.Status != "" {
		return errors.New("S3 storage requires a never-versioned bucket")
	}
	return nil
}

func (s *Store) Origin() string { return s.origin }
func (s *Store) Close() error {
	s.closeOnce.Do(func() { s.closed.Store(true); s.cancel(); s.transport.CloseIdleConnections() })
	return nil
}
func (s *Store) operation(ctx context.Context) (context.Context, func(), error) {
	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}
	if s.closed.Load() {
		return nil, nil, fs.ErrClosed
	}
	op, cancel := context.WithTimeout(ctx, s.cfg.OperationTimeout)
	stop := context.AfterFunc(s.life, cancel)
	return op, func() { stop(); cancel() }, nil
}
