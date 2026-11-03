package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func testConfig(endpoint string) Config {
	return Config{Endpoint: endpoint, Region: "us-east-1", Bucket: "nerve-test", Prefix: "test/",
		UsePathStyle: true, AccessKeyID: "explicit-test-access", SecretAccessKey: "test-secret-sentinel",
		SessionToken: "test-token-sentinel", SizeLimit: 5 << 20, OperationTimeout: time.Second}
}

func versionReply(w http.ResponseWriter, r *http.Request, status string) {
	if r.Method == http.MethodHead {
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Content-Type", "application/xml")
	_, _ = fmt.Fprintf(w, `<VersioningConfiguration xmlns="http://s3.amazonaws.com/doc/2006-03-01/">%s</VersioningConfiguration>`, status)
}

// Allowing a versioned bucket would lose old bytes behind a delete marker.
func TestS3Startup(t *testing.T) {
	for _, status := range []string{"", "Enabled", "Suspended", "unexpected"} {
		t.Run("version="+status, func(t *testing.T) {
			var mu sync.Mutex
			var requests []string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				requests = append(requests, r.Method+" "+r.URL.RequestURI())
				mu.Unlock()
				if !strings.Contains(r.Header.Get("Authorization"), "Credential=explicit-test-access/") || r.Header.Get("X-Amz-Security-Token") != "test-token-sentinel" {
					t.Error("request did not use explicit credentials")
				}
				body := ""
				if status != "" {
					body = "<Status>" + status + "</Status>"
				}
				versionReply(w, r, body)
			}))
			defer srv.Close()
			s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
			if status == "" {
				if err != nil {
					t.Fatal(err)
				}
				if s.Origin() != srv.URL {
					t.Fatalf("origin = %q", s.Origin())
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
				if err := s.Close(); err != nil {
					t.Fatal(err)
				}
			} else if err == nil {
				_ = s.Close()
				t.Fatal("versioned/unknown bucket accepted")
			}
			mu.Lock()
			defer mu.Unlock()
			if fmt.Sprint(requests) != "[HEAD /nerve-test GET /nerve-test?versioning=]" {
				t.Fatalf("startup must only read bucket state: %v", requests)
			}
		})
	}
	t.Run("forbidden and backend errors are sanitized without retries", func(t *testing.T) {
		for _, status := range []int{403, 500} {
			var calls atomic.Int32
			closed := make(chan struct{}, 2)
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls.Add(1)
				if r.Method == http.MethodHead {
					w.WriteHeader(http.StatusOK)
					return
				}
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `<Error><Code>AccessDenied</Code><Message>test-secret-sentinel test-token-sentinel Authorization X-Amz-Signature=secret</Message></Error>`)
			}))
			srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				if state == http.StateClosed {
					select {
					case closed <- struct{}{}:
					default:
					}
				}
			}
			srv.Start()
			_, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
			select {
			case <-closed:
			case <-time.After(time.Second):
				t.Error("failed startup retained its idle connection")
			}
			srv.Close()
			if err == nil || !strings.Contains(err.Error(), fmt.Sprint(status)) {
				t.Fatalf("missing safe HTTP status: %v", err)
			}
			assertSafeError(t, err)
			if calls.Load() != 2 {
				t.Fatalf("requests = %d, want one HEAD and one failed GET", calls.Load())
			}
		}
	})
	t.Run("one startup deadline", func(t *testing.T) {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.Method == http.MethodHead {
				select {
				case <-time.After(800 * time.Millisecond):
					versionReply(w, r, "")
				case <-r.Context().Done():
				}
				return
			}
			<-r.Context().Done()
		}))
		defer srv.Close()
		begin := time.Now()
		_, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("deadline cause lost: %v", err)
		}
		if elapsed := time.Since(begin); elapsed > 1500*time.Millisecond {
			t.Fatalf("two calls exceeded one budget: %v", elapsed)
		}
		if calls.Load() != 2 {
			t.Fatalf("did not exercise second read: %d", calls.Load())
		}
	})
	t.Run("caller cancellation", func(t *testing.T) {
		entered := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done() }))
		defer srv.Close()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		go func() { <-entered; cancel() }()
		_, err := New(ctx, testConfig(srv.URL), Deps{RuntimeNow: time.Now})
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation cause lost: %v", err)
		}
	})
	t.Run("real private bucket and versioned old bytes", realStartup)
}

func assertSafeError(t *testing.T, err error) {
	t.Helper()
	for _, secret := range []string{"test-secret-sentinel", "test-token-sentinel", "explicit-test-access", "Authorization", "X-Amz-Signature", "http://", "https://"} {
		if strings.Contains(err.Error(), secret) {
			t.Errorf("storage error leaked %q: %v", secret, err)
		}
	}
}

func realStartup(t *testing.T) {
	service := s3test.New(t)
	bucket := service.NewBucket(t)
	cfg := testConfig(bucket.Endpoint)
	cfg.Bucket, cfg.AccessKeyID, cfg.SecretAccessKey, cfg.SessionToken = bucket.Name, bucket.AccessKeyID, bucket.SecretAccessKey, ""
	cfg.OperationTimeout = 10 * time.Second
	s, err := New(t.Context(), cfg, Deps{RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	client := bucket.Client()
	for _, mode := range []types.BucketVersioningStatus{types.BucketVersioningStatusEnabled, types.BucketVersioningStatusSuspended} {
		_, err := client.PutBucketVersioning(t.Context(), &awss3.PutBucketVersioningInput{Bucket: aws.String(bucket.Name), VersioningConfiguration: &types.VersioningConfiguration{Status: mode}})
		if err != nil {
			t.Fatal(err)
		}
		if mode == types.BucketVersioningStatusEnabled {
			_, err = client.PutObject(t.Context(), &awss3.PutObjectInput{Bucket: aws.String(bucket.Name), Key: aws.String("old-bytes"), Body: strings.NewReader("persisted")})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.DeleteObject(t.Context(), &awss3.DeleteObjectInput{Bucket: aws.String(bucket.Name), Key: aws.String("old-bytes")})
			if err != nil {
				t.Fatal(err)
			}
			versions, err := client.ListObjectVersions(t.Context(), &awss3.ListObjectVersionsInput{Bucket: aws.String(bucket.Name)})
			if err != nil {
				t.Fatal(err)
			}
			if len(versions.Versions) != 1 || aws.ToInt64(versions.Versions[0].Size) != 9 || len(versions.DeleteMarkers) != 1 {
				t.Fatalf("versioned delete did not retain expected 9 bytes: %+v", versions)
			}
		}
		unexpected, err := New(t.Context(), cfg, Deps{RuntimeNow: time.Now})
		if err == nil {
			_ = unexpected.Close()
			t.Fatalf("%s bucket accepted despite retained old bytes", mode)
		}
	}
}
