package s3

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func wireStore(t *testing.T, handler http.HandlerFunc) *Store {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/nerve-test" {
			versionReply(w, r, "")
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func objectHeaders(h http.Header) {
	h.Set("Content-Length", "3")
	h.Set("Content-Type", "text/plain")
	h.Set("ETag", "\"opaque-version\"")
	h.Set("Last-Modified", "Tue, 03 Nov 2026 00:00:00 GMT")
}

func metadataAndErrors(t *testing.T) {
	t.Run("opaque metadata stays opaque", func(t *testing.T) {
		s := wireStore(t, func(w http.ResponseWriter, r *http.Request) { objectHeaders(w.Header()); w.WriteHeader(200) })
		info, err := s.Inspect(t.Context(), objectKey(t, 'u'))
		if err != nil {
			t.Fatal(err)
		}
		if info.ETag != "\"opaque-version\"" || info.Size != 3 || info.ContentType != "text/plain" || !info.ModifiedAt.Equal(time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)) {
			t.Fatalf("metadata changed: %+v", info)
		}
	})
	for _, tc := range []struct{ name, header, value string }{
		{"missing size", "Content-Length", ""}, {"negative size", "Content-Length", "-1"}, {"empty object", "Content-Length", "0"}, {"oversize object", "Content-Length", "104857601"},
		{"missing type", "Content-Type", ""}, {"noncanonical type", "Content-Type", "text/plain; charset=utf-8"}, {"oversize metadata", "Content-Type", strings.Repeat("a", 17000) + "/plain"},
		{"oversized response headers", "X-Remote-Metadata", strings.Repeat("x", 33<<10)},
		{"missing tag", "ETag", ""}, {"large tag", "ETag", "\"" + strings.Repeat("x", 1023) + "\""}, {"wildcard tag", "ETag", "*"}, {"tag list", "ETag", "\"first\", \"second\""}, {"weak tag", "ETag", "W/\"weak\""},
		{"missing time", "Last-Modified", ""}, {"bad time", "Last-Modified", "not-a-time"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := wireStore(t, func(w http.ResponseWriter, r *http.Request) {
				objectHeaders(w.Header())
				if tc.value == "" {
					w.Header().Del(tc.header)
				} else {
					w.Header().Set(tc.header, tc.value)
				}
				w.WriteHeader(200)
			})
			if _, err := s.Inspect(t.Context(), objectKey(t, 'u')); err == nil || errors.Is(err, domain.ErrObjectNotFound) {
				t.Fatalf("malformed metadata accepted or treated as missing: %v", err)
			}
		})
	}
	for _, status := range []int{403, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			s := wireStore(t, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(status)
				_, _ = io.WriteString(w, `<Error><Code>NoSuchKey</Code><Message>test-secret-sentinel test-token-sentinel</Message></Error>`)
			})
			key := objectKey(t, 'u')
			checks := []func() error{func() error { _, err := s.Inspect(t.Context(), key); return err }, func() error { _, err := s.Open(t.Context(), key, "\"opaque\""); return err }, func() error { _, err := s.Seal(t.Context(), key, objectKey(t, 'o'), "\"opaque\""); return err }, func() error { return s.Delete(t.Context(), key) }}
			for i, check := range checks {
				err := check()
				if err == nil || errors.Is(err, domain.ErrObjectNotFound) {
					t.Fatalf("backend failure operation%d mapped to absence: %v", i, err)
				}
				assertSafeError(t, err)
			}
		})
	}
	t.Run("unsafe keys never reach transport", func(t *testing.T) {
		var calls atomic.Int32
		s := wireStore(t, func(w http.ResponseWriter, r *http.Request) { calls.Add(1); w.WriteHeader(500) })
		bad := domain.ObjectKey("../neighbor")
		if _, err := s.Inspect(t.Context(), bad); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("bad inspect key: %v", err)
		}
		if _, err := s.Open(t.Context(), bad, "\"opaque\""); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("bad open key: %v", err)
		}
		if _, err := s.Seal(t.Context(), objectKey(t, 'u'), objectKey(t, 'u'), "\"opaque\""); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("bad destination kind: %v", err)
		}
		if err := s.Delete(t.Context(), bad); !errors.Is(err, domain.ErrInvalidObjectKey) {
			t.Fatalf("bad deletion key: %v", err)
		}
		if calls.Load() != 0 {
			t.Fatal("unsafe key dispatched")
		}
	})
}

func copyEmbeddedError(t *testing.T) {
	var copies, heads atomic.Int32
	s := wireStore(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "PUT" {
			copies.Add(1)
			if r.Header.Get("X-Amz-Copy-Source-If-Match") != "\"opaque\"" || r.Header.Get("If-None-Match") != "*" {
				t.Error("copy did not include both preconditions")
			}
			if r.Header.Get("X-Amz-Copy-Source") != "nerve-test/test/"+string(objectKey(t, 'u')) {
				t.Errorf("copy source = %q", r.Header.Get("X-Amz-Copy-Source"))
			}
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(200)
			_, _ = io.WriteString(w, `<Error><Code>InternalError</Code><Message>test-secret-sentinel</Message><RequestId>fixture</RequestId></Error>`)
			return
		}
		heads.Add(1)
		objectHeaders(w.Header())
		w.WriteHeader(200)
	})
	_, err := s.Seal(t.Context(), objectKey(t, 'u'), objectKey(t, 'o'), "\"opaque\"")
	if err == nil || errors.Is(err, domain.ErrObjectNotFound) {
		t.Fatalf("200 embedded COPY error accepted: %v", err)
	}
	assertSafeError(t, err)
	if copies.Load() != 1 || heads.Load() != 0 {
		t.Fatalf("embedded error retried or published: copy%d head%d", copies.Load(), heads.Load())
	}
}
