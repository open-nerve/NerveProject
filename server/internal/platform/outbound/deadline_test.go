package outbound

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync/atomic"
	"testing"
	"time"
)

func TestOutboundDeadlines(t *testing.T) {
	for _, stage := range []string{"resolve", "dial", "headers", "body", "tls", "tls_connect"} {
		t.Run(stage, func(t *testing.T) {
			entered := make(chan struct{})
			done := make(chan struct{})
			var endpoint string
			if stage == "headers" || stage == "body" {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if stage == "body" {
						w.WriteHeader(200)
						w.(http.Flusher).Flush()
					}
					close(entered)
					<-r.Context().Done()
					close(done)
				}))
				defer server.Close()
				endpoint = server.Listener.Addr().String()
			}
			if stage == "tls" || stage == "tls_connect" {
				listener, err := net.Listen("tcp", "127.0.0.1:0")
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = listener.Close() }()
				endpoint = listener.Addr().String()
				go func() {
					defer close(done)
					conn, err := listener.Accept()
					if err != nil {
						return
					}
					defer func() { _ = conn.Close() }()
					close(entered)
					_, _ = io.Copy(io.Discard, conn)
				}()
			}
			deps := fixtureDeps(t, endpoint)
			if stage == "resolve" {
				deps.resolver = resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
					close(entered)
					<-ctx.Done()
					close(done)
					return nil, ctx.Err()
				})
			}
			if stage == "dial" {
				var total time.Time
				deps.resolver = resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
					total, _ = ctx.Deadline()
					return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
				})
				deps.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
					deadline, ok := ctx.Deadline()
					if !ok || deadline.After(total) {
						t.Error("dial outlives total GET deadline")
					}
					close(entered)
					<-ctx.Done()
					close(done)
					return nil, ctx.Err()
				}
			}
			limits := testLimits()
			limits.Timeout = 200 * time.Millisecond
			if stage == "tls_connect" {
				limits.ConnectTimeout = 80 * time.Millisecond
				limits.Timeout = time.Second
			}
			target := "http://preview.test/"
			if stage == "tls" || stage == "tls_connect" {
				target = "https://preview.test/"
			}
			c := fixtureClient(t, deps)
			result := make(chan error, 1)
			started := time.Now()
			go func() { _, err := c.Get(t.Context(), target, limits); result <- err }()
			awaitSignal(t, entered)
			select {
			case err := <-result:
				if stage == "tls_connect" && time.Since(started) >= 500*time.Millisecond {
					t.Error("TLS handshake used total rather than connect budget")
				}
				if !errors.Is(err, ErrTimeout) {
					t.Fatalf("%s deadline: %v", stage, err)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("total deadline did not cover stage")
			}
			awaitSignal(t, done)
		})
	}
	t.Run("parent_cancel_body", func(t *testing.T) {
		entered := make(chan struct{})
		done := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			close(entered)
			<-r.Context().Done()
			close(done)
		}))
		defer server.Close()
		ctx, cancel := context.WithCancel(t.Context())
		defer cancel()
		c := fixtureClient(t, fixtureDeps(t, server.Listener.Addr().String()))
		result := make(chan error, 1)
		go func() { _, err := c.Get(ctx, "http://preview.test/", testLimits()); result <- err }()
		awaitSignal(t, entered)
		cancel()
		select {
		case err := <-result:
			if !errors.Is(err, ErrTimeout) || !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation chain: %v", err)
			}
		case <-time.After(time.Second):
			t.Fatal("caller cancellation ignored")
		}
		awaitSignal(t, done)
	})
	t.Run("parent_shorter_than_total", func(t *testing.T) {
		deps := fixtureDeps(t, "")
		ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
		defer cancel()
		expected, _ := ctx.Deadline()
		deps.resolver = resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
			got, _ := ctx.Deadline()
			if !got.Equal(expected) {
				t.Errorf("parent deadline replaced: got=%v want=%v", got, expected)
			}
			<-ctx.Done()
			return nil, ctx.Err()
		})
		_, err := fixtureClient(t, deps).Get(ctx, "http://preview.test/", testLimits())
		if !errors.Is(err, ErrTimeout) || !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal(err)
		}
	})
	t.Run("connect_budget", func(t *testing.T) {
		deps := fixtureDeps(t, "")
		limits := testLimits()
		limits.ConnectTimeout = 80 * time.Millisecond
		var total time.Time
		deps.resolver = resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
			total, _ = ctx.Deadline()
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		})
		deps.dial = func(ctx context.Context, _, _ string) (net.Conn, error) {
			deadline, ok := ctx.Deadline()
			if !ok || !deadline.Before(total) {
				t.Error("connect has no shorter deadline")
			}
			<-ctx.Done()
			return nil, ctx.Err()
		}
		_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", limits)
		if !errors.Is(err, ErrTimeout) {
			t.Fatal(err)
		}
	})
	t.Run("redirect_shares_deadline", func(t *testing.T) {
		finalDone := make(chan struct{})
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Location", "/next")
				w.WriteHeader(302)
				return
			}
			w.WriteHeader(200)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			close(finalDone)
		}))
		defer server.Close()
		deps := fixtureDeps(t, server.Listener.Addr().String())
		var first time.Time
		var calls atomic.Int32
		deps.resolver = resolverFunc(func(ctx context.Context, _, _ string) ([]netip.Addr, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Error("missing GET deadline")
			}
			if calls.Add(1) == 1 {
				first = deadline
			} else if !deadline.Equal(first) {
				t.Errorf("redirect reset deadline: first=%v second=%v", first, deadline)
			}
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		})
		limits := testLimits()
		limits.Timeout = 200 * time.Millisecond
		_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", limits)
		if !errors.Is(err, ErrTimeout) || calls.Load() != 2 {
			t.Fatalf("redirect deadline: %v calls=%d", err, calls.Load())
		}
		awaitSignal(t, finalDone)
	})
}

func TestOutboundLateDialClose(t *testing.T) {
	entered, release, closed := make(chan struct{}), make(chan struct{}), make(chan struct{})
	caller, peer := net.Pipe()
	defer func() { _ = caller.Close(); _ = peer.Close() }()
	deps := fixtureDeps(t, "")
	deps.dial = func(context.Context, string, string) (net.Conn, error) {
		close(entered)
		<-release
		return &closedConn{Conn: caller, closed: closed}, nil
	}
	c := fixtureClient(t, deps)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	result := make(chan error, 1)
	go func() { _, err := c.Get(ctx, "http://bücher.test/", testLimits()); result <- err }()
	awaitSignal(t, entered)
	cancel()
	select {
	case err := <-result:
		if !errors.Is(err, ErrTimeout) || !errors.Is(err, context.Canceled) {
			t.Errorf("cancellation chain: %v", err)
		}
	case <-time.After(time.Second):
		t.Error("GET waited for late dial")
	}
	close(release)
	awaitSignal(t, closed)
}
