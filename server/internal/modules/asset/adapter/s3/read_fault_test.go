package s3

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

func readCancellation(t *testing.T) {
	for _, mode := range []string{"cancel", "close", "deadline"} {
		t.Run(mode, func(t *testing.T) {
			disconnected := make(chan struct{}, 1)
			s := wireStore(t, func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" || r.Header.Get("If-Match") != "\"opaque\"" {
					t.Error("read was not conditional GET")
				}
				objectHeaders(w.Header())
				w.Header().Set("ETag", "\"opaque\"")
				w.Header().Set("Content-Length", "1048576")
				w.WriteHeader(200)
				_, _ = io.WriteString(w, "abc")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
				disconnected <- struct{}{}
			})
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			r, err := s.Open(ctx, objectKey(t, 'o'), "\"opaque\"")
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = r.Close() }()
			buf := make([]byte, 3)
			if _, err := io.ReadFull(r, buf); err != nil || string(buf) != "abc" {
				t.Fatalf("reader closed before return: %q %v", buf, err)
			}
			switch mode {
			case "cancel":
				cancel()
			case "close":
				if err := r.Close(); err != nil {
					t.Fatal(err)
				}
			}
			_, err = r.Read(make([]byte, 1))
			want := context.Canceled
			if mode == "deadline" {
				want = context.DeadlineExceeded
			}
			if !errors.Is(err, want) {
				t.Fatalf("stream did not preserve %v: %v", want, err)
			}
			select {
			case <-disconnected:
			case <-time.After(2 * time.Second):
				t.Fatal("canceled/closed reader retained network connection")
			}
			if err := r.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
	t.Run("read condition checked against returned metadata", func(t *testing.T) {
		s := wireStore(t, func(w http.ResponseWriter, r *http.Request) {
			objectHeaders(w.Header())
			w.WriteHeader(200)
			_, _ = io.WriteString(w, "abc")
		})
		r, err := s.Open(t.Context(), objectKey(t, 'o'), "\"different\"")
		if r != nil {
			_ = r.Close()
		}
		if !errors.Is(err, domain.ErrObjectChanged) {
			t.Fatalf("provider ignored If-Match without detection: %v", err)
		}
	})
}

func missingObjectBudget(t *testing.T) {
	var started atomic.Bool
	wait := func(r *http.Request) bool {
		timer := time.NewTimer(800 * time.Millisecond)
		defer timer.Stop()
		select {
		case <-timer.C:
			return true
		case <-r.Context().Done():
			return false
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !started.Load() {
			versionReply(w, r, "")
			return
		}
		if !wait(r) {
			return
		}
		if r.URL.Path == "/nerve-test" {
			w.WriteHeader(200)
		} else {
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()
	s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = s.Close() }()
	started.Store(true)
	begin := time.Now()
	_, err = s.Inspect(t.Context(), objectKey(t, 'u'))
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ambiguous404 fallback did not share timeout: %v", err)
	}
	if elapsed := time.Since(begin); elapsed > 1500*time.Millisecond {
		t.Fatalf("missing-object classification exceeded one budget: %v", elapsed)
	}
}
