package outbound

import (
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
)

func TestOutboundLimitsAndHeaders(t *testing.T) {
	t.Run("redirect_limit", func(t *testing.T) {
		for _, redirects := range []int{5, 6} {
			t.Run(strconv.Itoa(redirects), func(t *testing.T) {
				var calls atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					step, _ := strconv.Atoi(strings.TrimPrefix(r.URL.Path, "/"))
					if step < redirects {
						w.Header().Set("Location", fmt.Sprintf("/%d", step+1))
						w.WriteHeader(302)
						return
					}
					w.Header().Set("Content-Type", "text/plain")
					_, _ = io.WriteString(w, "done")
				}))
				defer server.Close()
				c := fixtureClient(t, fixtureDeps(t, server.Listener.Addr().String()))
				r, err := c.Get(t.Context(), "http://preview.test/0", testLimits())
				if calls.Load() != 6 {
					t.Fatalf("GET calls=%d, want 6", calls.Load())
				}
				if redirects == 5 {
					if err != nil || string(r.Body) != "done" || r.FinalURL != "http://preview.test/5" {
						t.Fatalf("five redirects: %+v %v", r, err)
					}
				} else if !errors.Is(err, ErrUnreachable) || Reason(fmt.Errorf("wrapped: %w", err)) != "redirect_limit" {
					t.Fatalf("sixth redirect: %v reason=%q", err, Reason(err))
				}
			})
		}
	})
	t.Run("redirect_targets_blocked", func(t *testing.T) {
		for _, location := range []string{"http://user:secret@next.test/", "http://[fe80::1%25en0]/", "ftp://next.test/", "http://127.0.0.1/", "http://next.test:65536/"} {
			t.Run(location, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.Header().Set("Location", location); w.WriteHeader(302) }))
				defer server.Close()
				deps := fixtureDeps(t, server.Listener.Addr().String())
				var dials atomic.Int32
				dial := deps.dial
				deps.dial = func(ctx context.Context, n, a string) (net.Conn, error) { dials.Add(1); return dial(ctx, n, a) }
				_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", testLimits())
				if !errors.Is(err, ErrBlocked) || dials.Load() != 1 {
					t.Fatalf("unsafe redirect: %v dials=%d", err, dials.Load())
				}
			})
		}
	})
	t.Run("all_redirect_statuses", func(t *testing.T) {
		for _, code := range []int{301, 302, 303, 307, 308} {
			t.Run(strconv.Itoa(code), func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/dir/start" {
						w.Header().Set("Location", "../final")
						w.WriteHeader(code)
						return
					}
					_, _ = io.WriteString(w, r.Method+" "+r.URL.Path)
				}))
				defer server.Close()
				r := getFixture(t, fixtureClient(t, fixtureDeps(t, server.Listener.Addr().String())), "http://preview.test/dir/start")
				if string(r.Body) != "GET /final" {
					t.Fatal(string(r.Body))
				}
			})
		}
	})
	t.Run("proxy_and_credentials", func(t *testing.T) {
		var proxyCalls atomic.Int32
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { proxyCalls.Add(1); w.WriteHeader(502) }))
		defer proxy.Close()
		for _, key := range []string{"HTTP_PROXY", "HTTPS_PROXY", "http_proxy", "https_proxy"} {
			t.Setenv(key, proxy.URL)
		}
		t.Setenv("NO_PROXY", "")
		t.Setenv("no_proxy", "")
		var calls atomic.Int32
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			for _, key := range []string{"Cookie", "Authorization", "Proxy-Authorization", "Referer"} {
				if got := r.Header.Get(key); got != "" {
					t.Errorf("credential header %s=%q", key, got)
				}
			}
			if r.Method != "GET" || r.UserAgent() != "Nerve/test-version (+link preview)" || r.URL.IsAbs() {
				t.Errorf("origin request method=%s UA=%q URI=%q", r.Method, r.UserAgent(), r.RequestURI)
			}
			if r.URL.Path == "/" {
				w.Header().Set("Set-Cookie", "secret=leaked; Path=/")
				w.Header().Set("Location", "http://another.test/end")
				w.WriteHeader(302)
				return
			}
			_, _ = io.WriteString(w, "ok")
		}))
		defer server.Close()
		r := getFixture(t, fixtureClient(t, fixtureDeps(t, server.Listener.Addr().String())), "http://preview.test/")
		if string(r.Body) != "ok" || proxyCalls.Load() != 0 || calls.Load() != 2 {
			t.Fatalf("body=%q proxy=%d origin=%d", r.Body, proxyCalls.Load(), calls.Load())
		}
	})
	t.Run("body_limit", func(t *testing.T) {
		for _, mode := range []string{"exact", "over", "chunked", "gzip"} {
			t.Run(mode, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
					body := strings.Repeat("x", 64)
					if mode != "exact" {
						body += "x"
					}
					if mode == "gzip" {
						w.Header().Set("Content-Encoding", "gzip")
						z := gzip.NewWriter(w)
						_, _ = io.WriteString(z, body)
						_ = z.Close()
						return
					}
					if mode == "chunked" {
						w.(http.Flusher).Flush()
					}
					_, _ = io.WriteString(w, body)
				}))
				defer server.Close()
				limits := testLimits()
				limits.MaxBytes = 64
				r, err := fixtureClient(t, fixtureDeps(t, server.Listener.Addr().String())).Get(t.Context(), "http://preview.test/", limits)
				if mode == "exact" {
					if err != nil || len(r.Body) != 64 {
						t.Fatalf("exact: %d %v", len(r.Body), err)
					}
				} else if !errors.Is(err, ErrTooLarge) || len(r.Body) != 0 {
					t.Fatalf("oversize %s: len=%d err=%v", mode, len(r.Body), err)
				}
			})
		}
	})
	t.Run("invalid_limits_and_url", func(t *testing.T) {
		deps := fixtureDeps(t, "")
		var calls atomic.Int32
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			calls.Add(1)
			return nil, errors.New("unexpected resolve")
		})
		c := fixtureClient(t, deps)
		limits := testLimits()
		for _, bad := range []Limits{{}, {ConnectTimeout: -1, Timeout: limits.Timeout, MaxBytes: 1}, {ConnectTimeout: limits.ConnectTimeout, Timeout: -1, MaxBytes: 1}, {ConnectTimeout: limits.ConnectTimeout, Timeout: limits.Timeout, MaxBytes: -1}, {ConnectTimeout: limits.ConnectTimeout, Timeout: limits.Timeout, MaxBytes: math.MaxInt64}} {
			if _, err := c.Get(t.Context(), "http://preview.test/", bad); err == nil {
				t.Fatal("invalid limits accepted")
			}
		}
		for _, bad := range []string{"ftp://preview.test/", "http://user:pw@preview.test/", "http://[fe80::1%25en0]/", "http:///path", "http://preview.test:/", "http://preview.test:abc/", "http://preview.test:65536/", "http://preview.test/\x00", "http://[preview.test]/"} {
			if _, err := c.Get(t.Context(), bad, limits); !errors.Is(err, ErrBlocked) {
				t.Errorf("URL %q: %v", bad, err)
			}
		}
		if calls.Load() != 0 {
			t.Fatalf("invalid input reached DNS %d times", calls.Load())
		}
	})
	t.Run("status_and_network_errors", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(503)
			_, _ = io.WriteString(w, "unavailable")
		}))
		defer server.Close()
		deps := fixtureDeps(t, server.Listener.Addr().String())
		r := getFixture(t, fixtureClient(t, deps), "http://preview.test/")
		if r.StatusCode != 503 || r.ContentType != "text/plain; charset=utf-8" || string(r.Body) != "unavailable" {
			t.Fatalf("response=%+v", r)
		}
		sentinel := errors.New("DNS private details")
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return nil, sentinel })
		_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", testLimits())
		if !errors.Is(err, ErrUnreachable) || !errors.Is(err, sentinel) {
			t.Fatal(err)
		}
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return nil, nil })
		if _, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", testLimits()); !errors.Is(err, ErrUnreachable) {
			t.Fatal(err)
		}
	})
}
