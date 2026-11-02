package httpadapter

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

type transferWorld struct {
	store  *local.Store
	server *httptest.Server
	clock  atomic.Int64
}

func transferTestFiles() config.FilesConfig {
	return config.FilesConfig{SizeLimit: 5242880, MinTransferBytesPerSecond: 131072, TransferGrace: 15 * time.Second, TransferIdleTimeout: 30 * time.Second, MaxTransferDuration: 15 * time.Minute, MaxConcurrentTransfers: 16}
}

func newTransferWorld(t *testing.T, configure ...func(*Deps)) *transferWorld {
	t.Helper()
	w := &transferWorld{}
	w.clock.Store(time.Date(2026, 11, 2, 12, 0, 0, 0, time.UTC).Unix())
	s, err := local.New(local.Config{Root: t.TempDir(), SizeLimit: 5242880}, local.Deps{
		UploadMAC: storagetest.MAC("upload-http-test"), ReadMAC: storagetest.MAC("read-http-test"), RuntimeNow: w.now,
	})
	if err != nil {
		t.Fatal(err)
	}
	w.store = s
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	logger := slog.New(slog.DiscardHandler)
	deps := Deps{UploadVerifier: s, ReadVerifier: s, Uploader: s, Reader: s, Files: transferTestFiles(), RuntimeNow: w.now, Errors: httpserver.NewAPIErrors(logger)}
	for _, change := range configure {
		change(&deps)
	}
	h, err := NewTransfer(deps)
	if err != nil {
		t.Fatal(err)
	}
	router := httpserver.NewRouter(logger)
	h.Register(router, func(h http.Handler) http.Handler { return h })
	router.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "SPA") })
	w.server = httptest.NewServer(router)
	t.Cleanup(w.server.Close)
	return w
}

func TestLocalHTTPUploadUsesCurrentSizePolicy(t *testing.T) {
	w := newTransferWorld(t)
	// A grant issued before an administrator lowered the limit remains
	// cryptographically valid; its request must reach the explicit 413 policy.
	old, err := local.New(local.Config{Root: t.TempDir(), SizeLimit: 104857600}, local.Deps{UploadMAC: storagetest.MAC("upload-http-test"), ReadMAC: storagetest.MAC("read-http-test"), RuntimeNow: w.now})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := old.Close(); err != nil {
			t.Error(err)
		}
	})
	signed, err := old.SignUpload(t.Context(), domain.UploadGrant{Key: transferKey(t, 'u'), Size: 5242881, ContentType: "text/plain", ExpiresAt: w.now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	response, data := request(t, w, "PUT", signed.URL, strings.NewReader("x"), signed.Headers)
	if response.StatusCode != 413 || !bytes.Contains(data, []byte(`"code":"payload_too_large"`)) {
		t.Fatalf("old signed grant after limit reduction = %d %s", response.StatusCode, data)
	}
}

func (w *transferWorld) now() time.Time { return time.Unix(w.clock.Load(), 0).UTC() }

func transferKey(t *testing.T, kind byte) domain.ObjectKey {
	t.Helper()
	k, err := domain.NewObjectKey(kind, uuid.New(), uuid.New())
	if err != nil {
		t.Fatal(err)
	}
	return k
}

func (w *transferWorld) upload(t *testing.T, body string, contentType string) domain.ObjectInfo {
	t.Helper()
	g := domain.UploadGrant{Key: transferKey(t, 'u'), Size: int64(len(body)), ContentType: contentType, ExpiresAt: w.now().Add(time.Minute)}
	signed, err := w.store.SignUpload(t.Context(), g)
	if err != nil {
		t.Fatal(err)
	}
	response, data := request(t, w, signed.Method, signed.URL, strings.NewReader(body), signed.Headers)
	if response.StatusCode != 204 || len(data) != 0 {
		t.Fatalf("upload = %d %s", response.StatusCode, data)
	}
	info, err := w.store.Inspect(t.Context(), g.Key)
	if err != nil {
		t.Fatal(err)
	}
	return info
}

func request(t *testing.T, w *transferWorld, method, path string, body io.Reader, headers map[string]string) (*http.Response, []byte) {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), method, w.server.URL+path, body)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		r.Header.Set(k, v)
	}
	response, err := w.server.Client().Do(r)
	if err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("body = %v, close = %v", err, closeErr)
	}
	return response, data
}

func TestTransferRequiresDependencies(t *testing.T) {
	w := newTransferWorld(t)
	valid := Deps{UploadVerifier: w.store, ReadVerifier: w.store, Uploader: w.store, Reader: w.store, Files: transferTestFiles(), RuntimeNow: w.now, Errors: httpserver.NewAPIErrors(slog.New(slog.DiscardHandler))}
	var typedNil *local.Store
	for _, change := range []func(*Deps){
		func(d *Deps) { d.UploadVerifier = nil }, func(d *Deps) { d.ReadVerifier = nil }, func(d *Deps) { d.Uploader = nil }, func(d *Deps) { d.Reader = nil }, func(d *Deps) { d.Errors = nil }, func(d *Deps) { d.RuntimeNow = nil },
		func(d *Deps) { d.Reader = typedNil }, func(d *Deps) { d.Files.SizeLimit = 0 }, func(d *Deps) { d.Files.SizeLimit = 104857601 },
	} {
		d := valid
		change(&d)
		if _, err := NewTransfer(d); err == nil {
			t.Fatal("invalid transfer dependencies accepted")
		}
	}
}

func TestLocalHTTPUploadContract(t *testing.T) {
	w := newTransferWorld(t)
	for _, tc := range []struct {
		name, body, contentType string
		size                    int64
		status                  int
	}{
		{"complete", "abc", "text/plain", 3, 204},
		{"short declared body", "ab", "text/plain", 3, 400},
		{"excess declared body", "abcd", "text/plain", 3, 400},
		{"type mismatch", "abc", "image/png", 3, 400},
		{"type parameters", "abc", "text/plain; charset=utf-8", 3, 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := domain.UploadGrant{Key: transferKey(t, 'u'), Size: tc.size, ContentType: "text/plain", ExpiresAt: w.now().Add(time.Minute)}
			signed, err := w.store.SignUpload(t.Context(), g)
			if err != nil {
				t.Fatal(err)
			}
			response, data := request(t, w, "PUT", signed.URL, strings.NewReader(tc.body), map[string]string{"Content-Type": tc.contentType, "Authorization": "Bearer ignored", "Cookie": "ignored=value"})
			if response.StatusCode != tc.status {
				t.Fatalf("status = %d %s; want %d", response.StatusCode, data, tc.status)
			}
			if response.Header.Get("Cache-Control") != "no-store" {
				t.Fatalf("upload cache = %q", response.Header.Get("Cache-Control"))
			}
			if tc.status == 204 {
				response, data = request(t, w, "PUT", signed.URL, strings.NewReader(tc.body), signed.Headers)
				if response.StatusCode != 409 || !bytes.Contains(data, []byte(`"code":"asset.object_exists"`)) {
					t.Fatalf("replay = %d %s", response.StatusCode, data)
				}
			} else if _, err := w.store.Inspect(t.Context(), g.Key); err != domain.ErrObjectNotFound {
				t.Fatalf("invalid upload published: %v", err)
			}
		})
	}
	t.Run("chunked body has no declared size", func(t *testing.T) {
		g := domain.UploadGrant{Key: transferKey(t, 'u'), Size: 3, ContentType: "text/plain", ExpiresAt: w.now().Add(time.Minute)}
		signed, err := w.store.SignUpload(t.Context(), g)
		if err != nil {
			t.Fatal(err)
		}
		response, data := request(t, w, "PUT", signed.URL, struct{ io.Reader }{strings.NewReader("abc")}, signed.Headers)
		if response.StatusCode != 400 {
			t.Fatalf("chunked = %d %s", response.StatusCode, data)
		}
	})
	t.Run("actual short body never publishes", func(t *testing.T) {
		g := domain.UploadGrant{Key: transferKey(t, 'u'), Size: 3, ContentType: "text/plain", ExpiresAt: w.now().Add(time.Minute)}
		signed, err := w.store.SignUpload(t.Context(), g)
		if err != nil {
			t.Fatal(err)
		}
		u, err := url.Parse(w.server.URL)
		if err != nil {
			t.Fatal(err)
		}
		conn, err := net.DialTimeout("tcp", u.Host, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = conn.Close() }()
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(conn, "PUT %s HTTP/1.1\r\nHost: %s\r\nContent-Type: text/plain\r\nContent-Length: 3\r\nConnection: close\r\n\r\nab", signed.URL, u.Host); err != nil {
			t.Fatal(err)
		}
		if err := conn.(*net.TCPConn).CloseWrite(); err != nil {
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(response.Body)
		_ = response.Body.Close()
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != 400 {
			t.Fatalf("short stream = %d %s", response.StatusCode, data)
		}
		if _, err := w.store.Inspect(t.Context(), g.Key); err != domain.ErrObjectNotFound {
			t.Fatalf("short stream published: %v", err)
		}
	})
	t.Run("invalid signature never reveals existence", func(t *testing.T) {
		info := w.upload(t, "abc", "text/plain")
		_, exists := request(t, w, "PUT", "/static/local/"+string(info.Key)+"?payload=wrong&tag=wrong", strings.NewReader("abc"), nil)
		response, missing := request(t, w, "PUT", "/static/local/"+string(transferKey(t, 'u'))+"?payload=wrong&tag=wrong", strings.NewReader("abc"), nil)
		if response.StatusCode != 403 || !bytes.Equal(exists, missing) || !bytes.Contains(missing, []byte(`"code":"asset.invalid_signature"`)) {
			t.Fatalf("signature response: %d %s / %s", response.StatusCode, exists, missing)
		}
	})
	for _, path := range []string{"/static", "/static/", "/static/unknown/key", "/static/local/key/extra"} {
		response, data := request(t, w, "GET", path, nil, nil)
		if response.StatusCode != 404 || response.Header.Get("Content-Type") != httpserver.ContentTypeProblem || bytes.Contains(data, []byte("SPA")) {
			t.Fatalf("fallback %s = %d %s", path, response.StatusCode, data)
		}
	}
}
