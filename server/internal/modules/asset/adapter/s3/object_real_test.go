package s3

import (
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
	"uuid"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func freshKey(t *testing.T, kind byte) domain.ObjectKey {
	t.Helper()
	key, err := domain.NewObjectKey(kind, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return key
}

func TestS3ObjectFailures(t *testing.T) {
	t.Run("real object preconditions", realObjectPreconditions)
	t.Run("bad metadata and backend status", metadataAndErrors)
	t.Run("copy embedded error", copyEmbeddedError)
	t.Run("read cancellation and close", readCancellation)
	t.Run("one missing object budget", missingObjectBudget)
}

func realObjectPreconditions(t *testing.T) {
	service := s3test.New(t)
	bucket := service.NewBucket(t)
	s := realStore(t, bucket)
	put := func(t *testing.T, key domain.ObjectKey, body string) domain.ObjectInfo {
		t.Helper()
		_, err := bucket.Client().PutObject(t.Context(), &awss3.PutObjectInput{Bucket: aws.String(bucket.Name), Key: aws.String(s.cfg.Prefix + string(key)), Body: strings.NewReader(body), ContentType: aws.String("text/plain")})
		if err != nil {
			t.Fatal(err)
		}
		info, err := s.Inspect(t.Context(), key)
		if err != nil {
			t.Fatal(err)
		}
		return info
	}
	read := func(t *testing.T, info domain.ObjectInfo) string {
		t.Helper()
		r, err := s.Open(t.Context(), info.Key, info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		body, err := io.ReadAll(r)
		closeErr := r.Close()
		if err != nil || closeErr != nil {
			t.Fatalf("read: %v %v", err, closeErr)
		}
		return string(body)
	}
	t.Run("missing objects and empty expected version", func(t *testing.T) {
		key := freshKey(t, 'u')
		if _, err := s.Inspect(t.Context(), key); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("missing HEAD = %v", err)
		}
		if _, err := s.Open(t.Context(), key, ""); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("missing empty-version open = %v", err)
		}
		if _, err := s.Seal(t.Context(), key, freshKey(t, 'o'), ""); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("missing empty-version copy = %v", err)
		}
		put(t, key, "abc")
		for _, expected := range []string{"", "*", "\"first\", \"second\"", "bad\r\nheader"} {
			if _, err := s.Open(t.Context(), key, expected); !errors.Is(err, domain.ErrObjectChanged) {
				t.Fatalf("unsafe read condition %q accepted: %v", expected, err)
			}
			if _, err := s.Seal(t.Context(), key, freshKey(t, 'o'), expected); !errors.Is(err, domain.ErrObjectChanged) {
				t.Fatalf("unsafe copy condition %q accepted: %v", expected, err)
			}
		}
	})
	t.Run("changed source and existing different final", func(t *testing.T) {
		key, target := freshKey(t, 'u'), freshKey(t, 'o')
		old := put(t, key, "AAA")
		put(t, key, "BBB")
		if _, err := s.Seal(t.Context(), key, target, old.ETag); !errors.Is(err, domain.ErrObjectChanged) {
			t.Fatalf("stale source copied: %v", err)
		}
		if _, err := s.Inspect(t.Context(), target); !errors.Is(err, domain.ErrObjectNotFound) {
			t.Fatalf("stale copy created target: %v", err)
		}
		now, err := s.Inspect(t.Context(), key)
		if err != nil {
			t.Fatal(err)
		}
		final := put(t, target, "CCC")
		if _, err := s.Seal(t.Context(), key, target, now.ETag); !errors.Is(err, domain.ErrObjectChanged) {
			t.Fatalf("existing target replaced: %v", err)
		}
		if got := read(t, final); got != "CCC" {
			t.Fatalf("same-size final changed: %q", got)
		}
	})
	t.Run("concurrent destination has one winner", func(t *testing.T) {
		first, second := put(t, freshKey(t, 'u'), "AAA"), put(t, freshKey(t, 'u'), "BBB")
		for range 8 {
			target := freshKey(t, 'o')
			start := make(chan struct{})
			var wg sync.WaitGroup
			type result struct {
				info domain.ObjectInfo
				err  error
				body string
			}
			results := make(chan result, 2)
			for i, source := range []domain.ObjectInfo{first, second} {
				wg.Add(1)
				go func() {
					defer wg.Done()
					<-start
					info, err := s.Seal(t.Context(), source.Key, target, source.ETag)
					body := "AAA"
					if i == 1 {
						body = "BBB"
					}
					results <- result{info, err, body}
				}()
			}
			close(start)
			wg.Wait()
			close(results)
			wins, conflicts := 0, 0
			want := ""
			for result := range results {
				if result.err == nil {
					wins++
					want = result.body
				} else if errors.Is(result.err, domain.ErrObjectChanged) {
					conflicts++
				} else {
					t.Fatalf("unexpected concurrent error: %v", result.err)
				}
			}
			if wins != 1 || conflicts != 1 {
				t.Fatalf("destination did not enforce one atomic winner: wins=%d conflicts=%d", wins, conflicts)
			}
			info, err := s.Inspect(t.Context(), target)
			if err != nil {
				t.Fatal(err)
			}
			if got := read(t, info); got != want {
				t.Fatalf("winner bytes=%q, want%q", got, want)
			}
		}
	})
	t.Run("deleted bucket never looks like a missing object", func(t *testing.T) {
		b := service.NewBucket(t)
		store := realStore(t, b)
		_, err := b.Client().DeleteBucket(t.Context(), &awss3.DeleteBucketInput{Bucket: aws.String(b.Name)})
		if err != nil {
			t.Fatal(err)
		}
		key := freshKey(t, 'u')
		checks := []func() error{func() error { _, err := store.Inspect(t.Context(), key); return err }, func() error { _, err := store.Open(t.Context(), key, "\"opaque\""); return err }, func() error { _, err := store.Seal(t.Context(), key, freshKey(t, 'o'), "\"opaque\""); return err }, func() error { return store.Delete(t.Context(), key) }}
		for i, check := range checks {
			if err := check(); err == nil || errors.Is(err, domain.ErrObjectNotFound) {
				t.Fatalf("deleted bucket operation %d treated as object absence: %v", i, err)
			}
		}
	})
}
