package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestInspectStreamsAndBounds(t *testing.T) {
	const size = 5 << 20
	for _, n := range []int64{size, (100 << 20) + 1} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasPrefix(r.Header.Get("Authorization"), "AWS4-HMAC-SHA256 ") {
					t.Error("no SDK signature")
				}
				w.Header().Set("Content-Length", fmt.Sprint(n))
				w.Header().Set("Content-Type", "text/plain")
				w.Header().Set("ETag", `"opaque"`)
				if n == size {
					_, _ = io.CopyN(w, zeroReader{}, n)
				}
			}))
			defer s.Close()
			r := testRequest(s.URL)
			data, _ := json.Marshal(r)
			var out bytes.Buffer
			err := run(context.Background(), bytes.NewReader(data), &out)
			if n > 100<<20 {
				if err == nil {
					t.Fatal("oversized object accepted")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var got result
			if err := json.Unmarshal(out.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			h := sha256.New()
			_, _ = io.CopyN(h, zeroReader{}, n)
			if got.Size != n || got.SHA256 != hex.EncodeToString(h.Sum(nil)) || got.ContentType != "text/plain" || got.ETag != `"opaque"` {
				t.Fatalf("wrong evidence: %+v", got)
			}
			if out.Len() > 1024 {
				t.Fatalf("file bytes leaked into output (%d)", out.Len())
			}
		})
	}
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func TestHelperCORSAndRedirect(t *testing.T) {
	t.Run("exact CORS", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			body, _ := io.ReadAll(r.Body)
			text := string(body)
			for _, want := range []string{"<AllowedOrigin>http://127.0.0.1:4567</AllowedOrigin>", "<AllowedMethod>PUT</AllowedMethod>", "<AllowedMethod>GET</AllowedMethod>", "<AllowedMethod>HEAD</AllowedMethod>", "<AllowedHeader>Content-Type</AllowedHeader>", "<ExposeHeader>ETag</ExposeHeader>", "<ExposeHeader>Content-Length</ExposeHeader>", "<ExposeHeader>Content-Type</ExposeHeader>", "<ExposeHeader>Content-Disposition</ExposeHeader>"} {
				if !strings.Contains(text, want) {
					t.Errorf("CORS missing %s", want)
				}
			}
			if strings.Contains(text, "*") || strings.Contains(text, "Credentials") {
				t.Error("CORS widened")
			}
			if r.Method != "PUT" || !r.URL.Query().Has("cors") {
				t.Error("not PutBucketCors")
			}
		}))
		defer s.Close()
		r := testRequest(s.URL)
		r.Action = "cors"
		r.Key = ""
		r.Origin = "http://127.0.0.1:4567"
		data, _ := json.Marshal(r)
		if err := run(context.Background(), bytes.NewReader(data), io.Discard); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("no redirect or error leak", func(t *testing.T) {
		second := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { t.Error("credentials reached second origin") }))
		defer second.Close()
		first := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Location", second.URL+"/secret-value")
			w.WriteHeader(307)
		}))
		defer first.Close()
		r := testRequest(first.URL)
		data, _ := json.Marshal(r)
		err := run(context.Background(), bytes.NewReader(data), io.Discard)
		if err == nil || strings.Contains(err.Error(), "secret") || strings.Contains(err.Error(), "127.0.0.1") {
			t.Fatalf("unsafe error: %v", err)
		}
	})
	t.Run("caller cancellation", func(t *testing.T) {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
		defer s.Close()
		r := testRequest(s.URL)
		data, _ := json.Marshal(r)
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if err := run(ctx, bytes.NewReader(data), io.Discard); err == nil {
			t.Fatal("canceled request succeeded")
		}
	})
}
