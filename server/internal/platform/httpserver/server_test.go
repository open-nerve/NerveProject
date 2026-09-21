package httpserver

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
)

var client = &http.Client{Timeout: 5 * time.Second}

// startServer runs a Server on a random local port. Cancelling the returned
// context starts the shutdown; done yields the result of Serve.
func startServer(t *testing.T, shutdownTimeout time.Duration, h http.Handler) (string, context.CancelFunc, <-chan error) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.ServerConfig{Addr: ln.Addr().String(), ReadHeaderTimeout: time.Second, ShutdownTimeout: shutdownTimeout}
	srv := NewServer(cfg, h, slog.New(slog.DiscardHandler))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(ctx, ln) }()
	return "http://" + ln.Addr().String(), cancel, done
}

func wait(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("Serve did not return")
		return nil
	}
}

func TestServeAppliesMiddlewareAndStopsOnCancel(t *testing.T) {
	url, cancel, done := startServer(t, time.Second, NewMux(slog.New(slog.DiscardHandler)))

	resp, err := client.Get(url + "/healthz")
	if err != nil {
		t.Fatal(err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get(HeaderRequestID) == "" {
		t.Errorf("GET /healthz = %d, X-Request-Id %q; want 200 with a request ID", resp.StatusCode, resp.Header.Get(HeaderRequestID))
	}

	cancel()
	if err := wait(t, done); err != nil {
		t.Errorf("Serve() = %v, want nil after a clean shutdown", err)
	}
}

func TestShutdownDrainsInFlightRequests(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	url, cancel, done := startServer(t, 5*time.Second, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = io.WriteString(w, "finished")
	}))
	type result struct {
		body string
		err  error
	}
	got := make(chan result, 1)
	go func() {
		resp, err := client.Get(url + "/slow")
		if err != nil {
			got <- result{err: err}
			return
		}
		defer func() { _ = resp.Body.Close() }()
		body, err := io.ReadAll(resp.Body)
		got <- result{string(body), err}
	}()
	<-started

	cancel()
	addr := strings.TrimPrefix(url, "http://")
	for deadline := time.Now().Add(2 * time.Second); ; {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			break // the listener is closed: no new connections
		}
		_ = conn.Close()
		if time.Now().After(deadline) {
			t.Fatal("server still accepts connections after shutdown began")
		}
		time.Sleep(10 * time.Millisecond)
	}
	close(release)

	if r := <-got; r.err != nil || r.body != "finished" {
		t.Errorf("in-flight request = %q, %v; want it to finish", r.body, r.err)
	}
	if err := wait(t, done); err != nil {
		t.Errorf("Serve() = %v, want nil", err)
	}
}

func TestShutdownGivesUpAfterTimeout(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() { close(release) })
	url, cancel, done := startServer(t, 100*time.Millisecond, http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		close(started)
		<-release
	}))
	go func() {
		if resp, err := client.Get(url + "/stuck"); err == nil {
			_ = resp.Body.Close()
		}
	}()
	<-started

	begin := time.Now()
	cancel()
	err := wait(t, done)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Serve() = %v, want a shutdown deadline error", err)
	}
	if elapsed := time.Since(begin); elapsed > 2*time.Second {
		t.Errorf("shutdown took %s, want about the 100ms timeout", elapsed)
	}
}

func TestListenAndServeReportsListenError(t *testing.T) {
	taken, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = taken.Close() }()
	cfg := config.ServerConfig{Addr: taken.Addr().String(), ReadHeaderTimeout: time.Second, ShutdownTimeout: time.Second}

	err = NewServer(cfg, http.NotFoundHandler(), slog.New(slog.DiscardHandler)).ListenAndServe(context.Background())

	if err == nil || !strings.Contains(err.Error(), "listen on "+taken.Addr().String()) {
		t.Errorf("ListenAndServe() = %v, want a listen error", err)
	}
}
