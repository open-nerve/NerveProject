package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"net/http/httptrace"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

// A real socket streams one byte and then stalls. A single-connection
// transport makes leaked readers and canceled connection waiters observable.
func TestS3Resources(t *testing.T) {
	t.Run("bounded protocol metadata", resourceMetadataEnvelopes)
	for _, mode := range []string{"caller cancel", "operation timeout", "store close"} {
		t.Run(mode, func(t *testing.T) {
			var sockets, heads atomic.Int32
			var stall atomic.Bool
			stall.Store(true)
			disconnected := make(chan struct{}, 1)
			srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/nerve-test" {
					versionReply(w, r, "")
					return
				}
				objectHeaders(w.Header())
				if r.Method == "HEAD" {
					heads.Add(1)
					return
				}
				if stall.Swap(false) {
					_, _ = io.WriteString(w, "a")
					w.(http.Flusher).Flush()
					<-r.Context().Done()
					disconnected <- struct{}{}
					return
				}
				_, _ = io.WriteString(w, "abc")
			}))
			srv.Config.ConnState = func(_ net.Conn, state http.ConnState) {
				if state == http.StateNew {
					sockets.Add(1)
				}
				if state == http.StateClosed {
					sockets.Add(-1)
				}
			}
			srv.Start()
			t.Cleanup(srv.Close)
			transport := newTransport()
			transport.MaxConnsPerHost = 1
			cfg := testConfig(srv.URL)
			if mode != "operation timeout" {
				cfg.OperationTimeout = 10 * time.Second
			}
			store, err := newStore(t.Context(), cfg, Deps{RuntimeNow: time.Now}, transport)
			if err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = store.Close() })
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			reader, err := store.Open(ctx, objectKey(t, 'o'), `"opaque-version"`)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = reader.Close() }()
			first := make([]byte, 1)
			if _, err := io.ReadFull(reader, first); err != nil || string(first) != "a" {
				t.Fatalf("missing first network byte: %q %v", first, err)
			}
			readDone := make(chan error, 1)
			go func() { _, err := io.Copy(io.Discard, reader); readDone <- err }()
			var waiting <-chan error
			if mode != "operation timeout" {
				queued, cancelWait := context.WithCancel(t.Context())
				canceled := resourceWaiter(t, store, queued)
				cancelWait()
				if err := resourceResult(t, canceled); !errors.Is(err, context.Canceled) {
					t.Fatalf("queued caller was not canceled: %v", err)
				}
				if heads.Load() != 0 {
					t.Fatal("canceled connection waiter dispatched while the stream owned the only connection")
				}
				waiting = resourceWaiter(t, store, t.Context())
			}
			start := time.Now()
			switch mode {
			case "caller cancel":
				cancel()
			case "store close":
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
			}
			want := context.Canceled
			if mode == "operation timeout" {
				want = context.DeadlineExceeded
			}
			if err := resourceResult(t, readDone); !errors.Is(err, want) {
				t.Fatalf("slow reader lost %v: %v", want, err)
			}
			if time.Since(start) > 2*time.Second {
				t.Fatal("slow reader did not end within its actual deadline/cancellation bound")
			}
			select {
			case <-disconnected:
			case <-time.After(2 * time.Second):
				t.Fatal("active socket survived cancellation")
			}
			if mode == "store close" {
				if err := resourceResult(t, waiting); !errors.Is(err, context.Canceled) {
					t.Fatalf("store close left queued work: %v", err)
				}
				if _, err := store.Inspect(t.Context(), objectKey(t, 'o')); !errors.Is(err, fs.ErrClosed) {
					t.Fatalf("closed store accepted work: %v", err)
				}
			} else {
				if waiting != nil {
					if err := resourceResult(t, waiting); err != nil {
						t.Fatalf("released connection was not reusable: %v", err)
					}
				}
				next, err := store.Open(t.Context(), objectKey(t, 'o'), `"opaque-version"`)
				if err != nil {
					t.Fatal(err)
				}
				data, err := io.ReadAll(next)
				closeErr := next.Close()
				if err != nil || closeErr != nil || string(data) != "abc" {
					t.Fatalf("recovery stream=%q %v %v", data, err, closeErr)
				}
				if err := store.Close(); err != nil {
					t.Fatal(err)
				}
			}
			deadline := time.NewTimer(2 * time.Second)
			defer deadline.Stop()
			tick := time.NewTicker(5 * time.Millisecond)
			defer tick.Stop()
			for sockets.Load() != 0 {
				select {
				case <-tick.C:
				case <-deadline.C:
					t.Fatalf("store close retained %d active or idle sockets", sockets.Load())
				}
			}
		})
	}
}

func resourceWaiter(t *testing.T, store *Store, ctx context.Context) <-chan error {
	t.Helper()
	entered := make(chan struct{}, 1)
	done := make(chan error, 1)
	traced := httptrace.WithClientTrace(ctx, &httptrace.ClientTrace{GetConn: func(string) {
		select {
		case entered <- struct{}{}:
		default:
		}
	}})
	go func() { _, err := store.Inspect(traced, objectKey(t, 'o')); done <- err }()
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("request did not reach the connection waiter")
	}
	return done
}
func resourceResult(t *testing.T, result <-chan error) error {
	t.Helper()
	select {
	case err := <-result:
		return err
	case <-time.After(3 * time.Second):
		t.Fatal("resource operation did not terminate")
		return nil
	}
}

func resourceMetadataEnvelopes(t *testing.T) {
	t.Run("incomplete error is not a business result", resourceIncompleteErrors)
	for _, operation := range []string{"startup", "copy", "object error"} {
		for _, chunked := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/chunked=%v", operation, chunked), func(t *testing.T) {
				padding := "<Ignored>" + strings.Repeat("x", 96<<10) + "</Ignored>"
				writeXML := func(w http.ResponseWriter, status int, body string) {
					w.Header().Set("Content-Type", "application/xml")
					if !chunked {
						w.Header().Set("Content-Length", fmt.Sprint(len(body)))
					}
					w.WriteHeader(status)
					if chunked {
						w.(http.Flusher).Flush()
					}
					_, _ = io.WriteString(w, body)
				}
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.URL.Path == "/nerve-test" {
						if operation == "startup" && r.URL.Query().Has("versioning") {
							writeXML(w, 200, "<VersioningConfiguration>"+padding+"</VersioningConfiguration>")
							return
						}
						versionReply(w, r, "")
						return
					}
					if operation == "copy" && r.Method == "PUT" {
						writeXML(w, 200, "<CopyObjectResult><ETag>\"opaque-version\"</ETag><LastModified>2026-11-03T00:00:00Z</LastModified>"+padding+"</CopyObjectResult>")
						return
					}
					if operation == "object error" && r.Method == "GET" {
						writeXML(w, 404, "<Error><Code>NoSuchKey</Code>"+padding+"</Error>")
						return
					}
					objectHeaders(w.Header())
					if r.Method == "GET" {
						_, _ = io.WriteString(w, "abc")
					}
				}))
				defer srv.Close()
				store, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
				if operation == "startup" {
					if store != nil {
						_ = store.Close()
					}
					if err == nil {
						t.Fatal("oversized startup metadata accepted")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = store.Close() }()
				if operation == "copy" {
					if _, err := store.Seal(t.Context(), freshKey(t, 'u'), freshKey(t, 'o'), `"opaque-version"`); err == nil {
						t.Fatal("oversized copy metadata accepted")
					}
				} else {
					r, err := store.Open(t.Context(), freshKey(t, 'o'), `"opaque-version"`)
					if r != nil {
						_ = r.Close()
					}
					if err == nil || errors.Is(err, domain.ErrObjectNotFound) {
						t.Fatalf("oversized object error treated as trusted absence: %v", err)
					}
				}
			})
		}
	}
}

func resourceIncompleteErrors(t *testing.T) {
	for _, status := range []int{404, 409, 412} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			store := wireStore(t, func(w http.ResponseWriter, _ *http.Request) {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(status)
				_, _ = io.WriteString(w, "<Error><Code>")
			})
			for _, operation := range []string{"open", "copy", "delete"} {
				t.Run(operation, func(t *testing.T) {
					var err error
					switch operation {
					case "open":
						var reader io.ReadCloser
						reader, err = store.Open(t.Context(), freshKey(t, 'o'), `"opaque-version"`)
						if reader != nil {
							_ = reader.Close()
						}
					case "copy":
						_, err = store.Seal(t.Context(), freshKey(t, 'u'), freshKey(t, 'o'), `"opaque-version"`)
					case "delete":
						err = store.Delete(t.Context(), freshKey(t, 'o'))
					}
					if err == nil || errors.Is(err, domain.ErrObjectNotFound) || errors.Is(err, domain.ErrObjectChanged) {
						t.Fatalf("incomplete protocol error treated as a business result: %v", err)
					}
					assertSafeError(t, err)
				})
			}
		})
	}
}
