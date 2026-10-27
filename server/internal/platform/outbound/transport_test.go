package outbound

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type resolverFunc func(context.Context, string, string) ([]netip.Addr, error)

func (f resolverFunc) LookupNetIP(ctx context.Context, network, host string) ([]netip.Addr, error) {
	return f(ctx, network, host)
}

func fixtureDeps(t *testing.T, endpoint string) dependencies {
	t.Helper()
	return dependencies{
		resolver: resolverFunc(func(_ context.Context, network, host string) ([]netip.Addr, error) {
			if network != "ip" || host == "" {
				t.Errorf("unexpected DNS query: %q %q", network, host)
			}
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}),
		dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// Never pass a synthetic destination to the real network.
			return (&net.Dialer{}).DialContext(ctx, network, endpoint)
		},
		interfaceAddrs: func() ([]net.Addr, error) { return nil, nil },
	}
}
func fixtureClient(t *testing.T, deps dependencies) *Client {
	t.Helper()
	c, err := newClient(Options{Version: "test-version"}, deps)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
func testLimits() Limits {
	return Limits{ConnectTimeout: time.Second, Timeout: 3 * time.Second, MaxBytes: 1024}
}
func getFixture(t *testing.T, c *Client, target string) Response {
	t.Helper()
	r, err := c.Get(t.Context(), target, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestOutboundPinsEveryHop(t *testing.T) {
	t.Run("all_candidates_before_dial", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "unsafe") }))
		defer server.Close()
		for _, answers := range [][]netip.Addr{
			{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("127.0.0.1")},
			{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("8.8.8.8")},
			{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("fe80::1")},
		} {
			deps := fixtureDeps(t, server.Listener.Addr().String())
			var calls atomic.Int32
			deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) { return answers, nil })
			dial := deps.dial
			deps.dial = func(ctx context.Context, n, a string) (net.Conn, error) { calls.Add(1); return dial(ctx, n, a) }
			_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", testLimits())
			if !errors.Is(err, ErrBlocked) || errors.Is(err, ErrUnreachable) || calls.Load() != 0 {
				t.Fatalf("mixed DNS: error=%v dialCalls=%d, want blocked and zero", err, calls.Load())
			}
		}
	})
	t.Run("single_lookup_pinned_literal", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { _, _ = io.WriteString(w, r.Host) }))
		defer server.Close()
		deps := fixtureDeps(t, server.Listener.Addr().String())
		var lookups atomic.Int32
		addresses := make(chan string, 4)
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			if lookups.Add(1) > 1 {
				return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		})
		dial := deps.dial
		deps.dial = func(ctx context.Context, n, a string) (net.Conn, error) { addresses <- a; return dial(ctx, n, a) }
		r := getFixture(t, fixtureClient(t, deps), "http://preview.test:8123/page")
		if string(r.Body) != "preview.test:8123" || lookups.Load() != 1 {
			t.Fatalf("Host=%s lookups=%d", r.Body, lookups.Load())
		}
		if got := <-addresses; got != "8.8.8.8:8123" {
			t.Fatalf("dial=%q, want pinned literal", got)
		}
	})
	t.Run("same_host_redirect_rechecks", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				w.Header().Set("Location", "/next")
				w.WriteHeader(302)
				return
			}
			_, _ = io.WriteString(w, "unsafe")
		}))
		defer server.Close()
		deps := fixtureDeps(t, server.Listener.Addr().String())
		var lookups, dials atomic.Int32
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			if lookups.Add(1) == 1 {
				return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
			}
			return []netip.Addr{netip.MustParseAddr("127.0.0.1")}, nil
		})
		dial := deps.dial
		deps.dial = func(ctx context.Context, n, a string) (net.Conn, error) { dials.Add(1); return dial(ctx, n, a) }
		_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", testLimits())
		if !errors.Is(err, ErrBlocked) || errors.Is(err, ErrUnreachable) || lookups.Load() != 2 || dials.Load() != 1 {
			t.Fatalf("redirect: error=%v lookups=%d dials=%d", err, lookups.Load(), dials.Load())
		}
	})
	t.Run("https_host_sni_and_certificate", func(t *testing.T) {
		observed := make(chan string, 4)
		sni := make(chan string, 4)
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { observed <- r.Host; _, _ = io.WriteString(w, "secure") }))
		server.TLS = &tls.Config{MinVersion: tls.VersionTLS12, GetConfigForClient: func(hello *tls.ClientHelloInfo) (*tls.Config, error) { sni <- hello.ServerName; return nil, nil }}
		server.StartTLS()
		defer server.Close()
		roots := x509.NewCertPool()
		roots.AddCert(server.Certificate())
		deps := fixtureDeps(t, server.Listener.Addr().String())
		deps.roots = roots
		c := fixtureClient(t, deps)
		r := getFixture(t, c, "https://example.com:9443/")
		if string(r.Body) != "secure" || <-observed != "example.com:9443" || <-sni != "example.com" {
			t.Fatal("HTTPS lost original Host or SNI")
		}
		_, err := c.Get(t.Context(), "https://wrong.test/", testLimits())
		var certErr x509.HostnameError
		if !errors.Is(err, ErrUnreachable) || !errors.As(err, &certErr) {
			t.Fatalf("wrong certificate: %v", err)
		}
		if got := <-sni; got != "wrong.test" {
			t.Fatalf("SNI=%q", got)
		}
		deps.roots = nil
		_, err = fixtureClient(t, deps).Get(t.Context(), "https://example.com/", testLimits())
		var authority x509.UnknownAuthorityError
		if !errors.Is(err, ErrUnreachable) || !errors.As(err, &authority) {
			t.Fatalf("untrusted certificate: %v", err)
		}
	})
	t.Run("literal_skips_dns_and_fallback_pins", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "ok") }))
		defer server.Close()
		deps := fixtureDeps(t, server.Listener.Addr().String())
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			t.Error("literal resolved")
			return nil, errors.New("unexpected DNS")
		})
		if got := getFixture(t, fixtureClient(t, deps), "http://8.8.8.8/"); string(got.Body) != "ok" {
			t.Fatal(got)
		}
		deps.resolver = resolverFunc(func(context.Context, string, string) ([]netip.Addr, error) {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8"), netip.MustParseAddr("1.1.1.1")}, nil
		})
		addresses := make(chan string, 2)
		dial := deps.dial
		deps.dial = func(ctx context.Context, n, a string) (net.Conn, error) {
			addresses <- a
			if a == "8.8.8.8:80" {
				return nil, errors.New("first unreachable")
			}
			return dial(ctx, n, a)
		}
		getFixture(t, fixtureClient(t, deps), "http://preview.test/")
		if a, b := <-addresses, <-addresses; a != "8.8.8.8:80" || b != "1.1.1.1:80" {
			t.Fatalf("fallback dial=%s,%s", a, b)
		}
	})
}

func TestOutboundLocalSnapshot(t *testing.T) {
	deps := fixtureDeps(t, "")
	sentinel := errors.New("interface unavailable")
	deps.interfaceAddrs = func() ([]net.Addr, error) { return nil, sentinel }
	if _, err := newClient(Options{}, deps); !errors.Is(err, sentinel) {
		t.Fatalf("interface failure lost: %v", err)
	}
	var snapshots, dials atomic.Int32
	deps.interfaceAddrs = func() ([]net.Addr, error) {
		snapshots.Add(1)
		return []net.Addr{&net.IPNet{IP: net.ParseIP("8.8.8.8"), Mask: net.CIDRMask(24, 32)}}, nil
	}
	deps.dial = func(context.Context, string, string) (net.Conn, error) {
		dials.Add(1)
		return nil, errors.New("unexpected dial")
	}
	c := fixtureClient(t, deps)
	for range 2 {
		_, err := c.Get(t.Context(), "http://preview.test/", testLimits())
		if !errors.Is(err, ErrBlocked) {
			t.Fatalf("local public address: %v", err)
		}
	}
	if snapshots.Load() != 1 || dials.Load() != 0 {
		t.Fatalf("snapshots=%d dials=%d", snapshots.Load(), dials.Load())
	}
	deps.interfaceAddrs = func() ([]net.Addr, error) { return nil, nil }
	if _, err := newClient(Options{AllowCIDRs: []netip.Prefix{{}}}, deps); err == nil {
		t.Fatal("invalid allow accepted")
	}
}

type closedConn struct {
	net.Conn
	once   sync.Once
	closed chan struct{}
}

func (c *closedConn) Close() error {
	err := c.Conn.Close()
	c.once.Do(func() { close(c.closed) })
	return err
}
func awaitSignal(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(3 * time.Second):
		t.Fatal("fixture did not reach expected state")
	}
}

func TestOutboundClosesResources(t *testing.T) {
	for _, mode := range []string{"redirect", "final", "too_large"} {
		t.Run(mode, func(t *testing.T) {
			closed := make(chan struct{})
			finished := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/next" {
					select {
					case <-closed:
					default:
						t.Error("intermediate connection still open")
					}
					_, _ = io.WriteString(w, "ok")
					return
				}
				if mode == "redirect" {
					w.Header().Set("Location", "/next")
					w.WriteHeader(302)
					w.(http.Flusher).Flush()
					<-finished
					return
				}
				_, _ = io.WriteString(w, "abcdefgh")
				if mode == "too_large" {
					w.(http.Flusher).Flush()
					<-finished
				}
			}))
			defer server.Close()
			defer close(finished)
			deps := fixtureDeps(t, server.Listener.Addr().String())
			dial := deps.dial
			var calls atomic.Int32
			deps.dial = func(ctx context.Context, n, a string) (net.Conn, error) {
				conn, err := dial(ctx, n, a)
				if err == nil && calls.Add(1) == 1 {
					return &closedConn{Conn: conn, closed: closed}, nil
				}
				return conn, err
			}
			limits := testLimits()
			if mode == "too_large" {
				limits.MaxBytes = 4
			}
			_, err := fixtureClient(t, deps).Get(t.Context(), "http://preview.test/", limits)
			if mode == "too_large" {
				if !errors.Is(err, ErrTooLarge) {
					t.Fatal(err)
				}
			} else if err != nil {
				t.Fatal(err)
			}
			awaitSignal(t, closed)
		})
	}
}

func TestOutboundReason(t *testing.T) {
	for _, err := range []error{nil, errors.New("redirect_limit SECRET"), fmt.Errorf("redirect_limit: %w", ErrUnreachable)} {
		if got := Reason(err); got != "" {
			t.Fatalf("unsafe reason %q", got)
		}
	}
}

func TestOutboundHTTPSDowngrade(t *testing.T) {
	headers := make(chan http.Header, 1)
	plain := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headers <- r.Header.Clone()
		_, _ = io.WriteString(w, r.Host)
	}))
	defer plain.Close()
	secure := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Set-Cookie", "session=secret; Path=/")
		w.Header().Set("Location", "http://other.test/end")
		w.WriteHeader(302)
	}))
	defer secure.Close()
	roots := x509.NewCertPool()
	roots.AddCert(secure.Certificate())
	deps := fixtureDeps(t, secure.Listener.Addr().String())
	deps.roots = roots
	var queries atomic.Int32
	deps.resolver = resolverFunc(func(_ context.Context, _, host string) ([]netip.Addr, error) {
		queries.Add(1)
		if host == "example.com" {
			return []netip.Addr{netip.MustParseAddr("8.8.8.8")}, nil
		}
		if host != "other.test" {
			t.Errorf("unexpected redirect hostname %q", host)
		}
		return []netip.Addr{netip.MustParseAddr("1.1.1.1")}, nil
	})
	deps.dial = func(ctx context.Context, network, address string) (net.Conn, error) {
		endpoint := secure.Listener.Addr().String()
		switch address {
		case "8.8.8.8:443":
		case "1.1.1.1:80":
			endpoint = plain.Listener.Addr().String()
		default:
			return nil, fmt.Errorf("unexpected pinned address: %s", address)
		}
		return (&net.Dialer{}).DialContext(ctx, network, endpoint)
	}
	response := getFixture(t, fixtureClient(t, deps), "https://example.com/")
	if string(response.Body) != "other.test" || response.FinalURL != "http://other.test/end" || queries.Load() != 2 {
		t.Fatalf("downgrade response=%+v DNS=%d", response, queries.Load())
	}
	for key, values := range <-headers {
		switch key {
		case "Cookie", "Authorization", "Proxy-Authorization", "Referer":
			t.Errorf("downgrade forwarded %s=%v", key, values)
		}
	}
}

func idnaCertificate(t *testing.T) (tls.Certificate, *x509.CertPool) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		DNSNames:     []string{"xn--bcher-kva.test"},
		NotBefore:    time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		KeyUsage:    x509.KeyUsageDigitalSignature,
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	der, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certificate, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AddCert(certificate)
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}, roots
}
