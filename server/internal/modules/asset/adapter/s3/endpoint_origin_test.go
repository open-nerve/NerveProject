package s3

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/s3test"
)

func TestS3EndpointCanonicalOrigin(t *testing.T) {
	for _, tc := range []struct{ raw, host string }{
		{"http://storage.invalid:80/", "storage.invalid"},
		{"http://STORAGE.invalid/", "storage.invalid"},
		{"http://storage.invalid:00080/", "storage.invalid"},
		{"https://STORAGE.invalid:00443/", "storage.invalid"},
		{"http://STORAGE.invalid:09443/", "storage.invalid:9443"},
		{"https://storage.invalid:9443/", "storage.invalid:9443"},
	} {
		t.Run(tc.raw, func(t *testing.T) {
			u, err := url.Parse(tc.raw)
			if err != nil {
				t.Fatal(err)
			}
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Host != tc.host {
					t.Errorf("noncanonical signed host: %q", r.Host)
				}
				versionReply(w, r, "")
			})
			var srv *httptest.Server
			transport := newTransport()
			if u.Scheme == "https" {
				srv = httptest.NewTLSServer(handler)
				transport.TLSClientConfig = srv.Client().Transport.(*http.Transport).TLSClientConfig.Clone()
				transport.TLSClientConfig.ServerName = "127.0.0.1"
			} else {
				srv = httptest.NewServer(handler)
			}
			defer srv.Close()
			transport.DialContext = func(ctx context.Context, network, _ string) (net.Conn, error) {
				return (&net.Dialer{}).DialContext(ctx, network, srv.Listener.Addr().String())
			}
			store, err := newStore(t.Context(), testConfig(tc.raw), Deps{RuntimeNow: time.Now}, transport)
			if err != nil {
				t.Fatalf("equivalent endpoint rejected: %v", err)
			}
			defer func() { _ = store.Close() }()
			if store.Origin() != u.Scheme+"://"+tc.host {
				t.Errorf("noncanonical origin: %q", store.Origin())
			}
			grant, err := store.SignUpload(t.Context(), domain.UploadGrant{Key: freshKey(t, 'u'), Size: 3, ContentType: "text/plain", ExpiresAt: time.Now().Add(time.Minute)})
			if err != nil {
				t.Fatal(err)
			}
			signed, err := url.Parse(grant.URL)
			if err != nil {
				t.Fatal(err)
			}
			if signed.Host != tc.host {
				t.Errorf("browser would rewrite signed authority: %q", signed.Host)
			}
		})
	}
}

// A browser removes port leading zeros before making a request. The provider
// must authenticate that same authority, with no change to an already signed URL.
func canonicalEndpointTransfer(t *testing.T, bucket *s3test.Bucket) {
	t.Helper()
	u, err := url.Parse(bucket.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	alias := *bucket
	alias.Endpoint = u.Scheme + "://" + u.Hostname() + ":0" + u.Port()
	store := realStore(t, &alias)
	key := freshKey(t, 'u')
	grant, err := store.SignUpload(t.Context(), domain.UploadGrant{Key: key, Size: 3, ContentType: "text/plain", ExpiresAt: time.Now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(grant.URL, bucket.Endpoint+"/") {
		t.Fatal("real signed URL authority would change in the browser")
	}
	resp, _ := transfer(t, grant, "abc")
	if resp.StatusCode != 200 {
		t.Fatalf("canonical authority PUT rejected: %d", resp.StatusCode)
	}
	info, err := store.Inspect(t.Context(), key)
	if err != nil || info.Size != 3 || info.ContentType != "text/plain" {
		t.Fatalf("canonical endpoint lost actual bytes: %+v %v", info, err)
	}
	if err := store.Delete(t.Context(), key); err != nil {
		t.Fatal(err)
	}
}
