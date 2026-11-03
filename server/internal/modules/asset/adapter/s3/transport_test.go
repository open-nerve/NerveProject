package s3

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
)

// A redirect or proxy must not receive signed credentials for this storage.
func TestS3TransportBoundary(t *testing.T) {
	t.Run("same origin redirect is also rejected", func(t *testing.T) {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			calls.Add(1)
			if r.URL.Path != "/moved" {
				w.Header().Set("Location", "/moved")
				w.WriteHeader(307)
				return
			}
			versionReply(w, r, "")
		}))
		defer srv.Close()
		s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
		if err == nil {
			_ = s.Close()
			t.Error("same-origin redirect accepted")
		}
		if calls.Load() != 1 {
			t.Fatalf("redirect caused %d requests, want one", calls.Load())
		}
	})
	t.Run("redirect never follows", func(t *testing.T) {
		var escaped atomic.Int32
		target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { escaped.Add(1); versionReply(w, r, "") }))
		defer target.Close()
		for _, status := range []int{301, 307} {
			source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Location", target.URL+"/?X-Amz-Signature=secret")
				w.WriteHeader(status)
			}))
			_, err := New(t.Context(), testConfig(source.URL), Deps{RuntimeNow: time.Now})
			source.Close()
			if err == nil {
				t.Fatal("redirect accepted")
			}
			assertSafeError(t, err)
		}
		if escaped.Load() != 0 {
			t.Fatalf("credentials sent across origin: %d", escaped.Load())
		}
	})
	t.Run("proxy environment is ignored", func(t *testing.T) {
		var proxied atomic.Int32
		proxy := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { proxied.Add(1); w.WriteHeader(502) }))
		defer proxy.Close()
		t.Setenv("HTTP_PROXY", proxy.URL)
		t.Setenv("HTTPS_PROXY", proxy.URL)
		t.Setenv("NO_PROXY", "")
		t.Setenv("AWS_ACCESS_KEY_ID", "unwanted-environment-access")
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.Contains(r.Header.Get("Authorization"), "Credential=explicit-test-access/") {
				t.Error("ambient credentials used")
			}
			versionReply(w, r, "")
		}))
		defer srv.Close()
		transport := newTransport()
		routeTo(transport, srv.Listener.Addr().String())
		s, err := newStore(t.Context(), testConfig("http://storage.invalid"), Deps{RuntimeNow: time.Now}, transport)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = s.Close() }()
		if proxied.Load() != 0 {
			t.Fatal("storage used ambient proxy")
		}
	})
	t.Run("official addressing matches actual requests", func(t *testing.T) {
		for _, tc := range []struct {
			name, bucket       string
			pathStyle, tls     bool
			wantHost, wantPath string
		}{
			{"path", "nerve-test", true, false, "storage.invalid", "/nerve-test"},
			{"virtual", "nerve-test", false, false, "nerve-test.storage.invalid", "/"},
			{"dotted http", "nerve.test", false, false, "nerve.test.storage.invalid", "/"},
			{"dotted https", "nerve.test", false, true, "storage.invalid", "/nerve.test"},
		} {
			t.Run(tc.name, func(t *testing.T) {
				var calls atomic.Int32
				srv := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					calls.Add(1)
					if r.Host != tc.wantHost+":9443" || r.URL.Path != tc.wantPath {
						t.Errorf("request = %s%s", r.Host, r.URL.Path)
					}
					versionReply(w, r, "")
				}))
				transport := newTransport()
				scheme := "http"
				if tc.tls {
					srv.StartTLS()
					scheme = "https"
					transport.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
					transport.TLSClientConfig.ServerName = "127.0.0.1"
				} else {
					srv.Start()
				}
				defer srv.Close()
				routeTo(transport, srv.Listener.Addr().String())
				cfg := testConfig(scheme + "://storage.invalid:9443/")
				cfg.Bucket = tc.bucket
				cfg.UsePathStyle = tc.pathStyle
				s, err := newStore(t.Context(), cfg, Deps{RuntimeNow: time.Now}, transport)
				if err != nil {
					t.Fatal(err)
				}
				defer func() { _ = s.Close() }()
				if s.Origin() != scheme+"://"+tc.wantHost+":9443" || calls.Load() != 2 {
					t.Fatalf("origin %q calls %d", s.Origin(), calls.Load())
				}
			})
		}
	})
	t.Run("outgoing host and URL are both guarded", func(t *testing.T) {
		var calls atomic.Int32
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls.Add(1); _, _ = io.WriteString(w, "leak") }))
		defer srv.Close()
		transport := newTransport()
		defer transport.CloseIdleConnections()
		guard := originTransport{origin: srv.URL, next: transport}
		for _, tc := range []struct{ raw, host string }{{srv.URL, "other.invalid"}, {srv.URL + "@other.invalid", ""}, {"https://other.invalid/", ""}} {
			req, err := http.NewRequestWithContext(t.Context(), "GET", tc.raw, nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Host = tc.host
			resp, err := guard.RoundTrip(req)
			if resp != nil {
				_ = resp.Body.Close()
			}
			if err == nil {
				t.Fatal("unapproved destination accepted")
			}
		}
		if calls.Load() != 0 {
			t.Fatal("guard dispatched invalid request")
		}
	})
	t.Run("close ends active requests and rejects new operations", func(t *testing.T) {
		entered := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/nerve-test/block" {
				close(entered)
				<-r.Context().Done()
				return
			}
			versionReply(w, r, "")
		}))
		defer srv.Close()
		s, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
		if err != nil {
			t.Fatal(err)
		}
		op, cancel, err := s.operation(t.Context())
		if err != nil {
			t.Fatal(err)
		}
		defer cancel()
		finished := make(chan error, 1)
		go func() {
			_, err := s.client.HeadObject(op, &awss3.HeadObjectInput{Bucket: aws.String("nerve-test"), Key: aws.String("block")})
			finished <- err
		}()
		<-entered
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
		if err := <-finished; !errors.Is(err, context.Canceled) {
			t.Fatalf("active request not canceled: %v", err)
		}
		if _, _, err := s.operation(t.Context()); !errors.Is(err, fs.ErrClosed) {
			t.Fatalf("closed store accepted operation: %v", err)
		}
	})
}

func routeTo(transport *http.Transport, destination string) {
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		host, _, err := net.SplitHostPort(address)
		if err != nil {
			return nil, err
		}
		if strings.HasSuffix(host, "storage.invalid") {
			address = destination
		}
		return (&net.Dialer{}).DialContext(ctx, network, address)
	}
}

// Construction must reject an untrusted TLS certificate without a runtime bypass.
func TestS3TLS(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { versionReply(w, r, "") }))
	defer srv.Close()
	_, err := New(t.Context(), testConfig(srv.URL), Deps{RuntimeNow: time.Now})
	if err == nil {
		t.Fatal("untrusted TLS accepted")
	}
	assertSafeError(t, err)
}
