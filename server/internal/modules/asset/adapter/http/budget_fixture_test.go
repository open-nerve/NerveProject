package httpadapter

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/local"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/storagetest"
	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
)

type budgetWorld struct {
	store         *local.Store
	root, address string
	client        *http.Client
	started       chan struct{}
	finished      chan string
	cancellations chan context.CancelFunc
}

type observingUploader struct {
	Uploader
	started chan struct{}
}

func (u observingUploader) Put(ctx context.Context, g domain.UploadGrant, r io.Reader) error {
	return u.Uploader.Put(ctx, g, &enteredBodyReader{Reader: r, started: u.started})
}

type enteredBodyReader struct {
	io.Reader
	started chan struct{}
	once    sync.Once
}

func (r *enteredBodyReader) Read(p []byte) (int, error) {
	r.once.Do(func() { r.started <- struct{}{} })
	return r.Reader.Read(p)
}

// This wrapper deliberately makes ResponseController unsupported on one request.
type opaqueResponseWriter struct{ http.ResponseWriter }

func budgetTestFiles() config.FilesConfig {
	return config.FilesConfig{SizeLimit: 1 << 20, MinTransferBytesPerSecond: 1 << 30, TransferGrace: 2 * time.Second, TransferIdleTimeout: 500 * time.Millisecond, MaxTransferDuration: 3 * time.Second, MaxConcurrentTransfers: 1}
}

func newBudgetWorld(t *testing.T, files config.FilesConfig, globalRead, globalWrite time.Duration) *budgetWorld {
	t.Helper()
	w := &budgetWorld{root: t.TempDir(), client: &http.Client{Timeout: 5 * time.Second}, started: make(chan struct{}, 32), finished: make(chan string, 32), cancellations: make(chan context.CancelFunc, 32)}
	s, err := local.New(local.Config{Root: w.root, SizeLimit: domain.MaxObjectSize}, local.Deps{UploadMAC: storagetest.MAC("budget-upload"), ReadMAC: storagetest.MAC("budget-read"), RuntimeNow: time.Now})
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
	transfer, err := NewTransfer(Deps{UploadVerifier: s, ReadVerifier: s, Uploader: observingUploader{s, w.started}, Reader: s, Files: files, RuntimeNow: time.Now, Errors: httpserver.NewAPIErrors(logger)})
	if err != nil {
		t.Fatal(err)
	}
	router := httpserver.NewRouter(logger)
	transfer.Register(router, func(h http.Handler) http.Handler { return h })
	wrapped := http.HandlerFunc(func(response http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithCancel(r.Context())
		defer cancel()
		w.cancellations <- cancel
		defer func() { w.finished <- r.URL.Path }()
		if r.Header.Get("X-Test-Opaque-Writer") == "yes" {
			response = opaqueResponseWriter{response}
		}
		router.ServeHTTP(response, r.WithContext(ctx))
	})
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	w.address = listener.Addr().String()
	server := httpserver.NewServer(config.ServerConfig{ReadHeaderTimeout: time.Second, ReadTimeout: globalRead, WriteTimeout: globalWrite, ShutdownTimeout: time.Second}, wrapped, logger)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- server.Serve(ctx, listener) }()
	t.Cleanup(func() {
		cancel()
		select {
		case err := <-done:
			if err != nil {
				t.Error(err)
			}
		case <-time.After(3 * time.Second):
			t.Error("budget server did not stop")
		}
		w.client.CloseIdleConnections()
	})
	return w
}

func (w *budgetWorld) grant(t *testing.T, size int64) (domain.UploadGrant, domain.SignedRequest) {
	t.Helper()
	g := domain.UploadGrant{Key: transferKey(t, 'u'), Size: size, ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(time.Minute)}
	signed, err := w.store.SignUpload(t.Context(), g)
	if err != nil {
		t.Fatal(err)
	}
	return g, signed
}

func (w *budgetWorld) beginUpload(t *testing.T, signed domain.SignedRequest, size int64) *net.TCPConn {
	t.Helper()
	connection, err := net.DialTimeout("tcp", w.address, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	conn := connection.(*net.TCPConn)
	t.Cleanup(func() { _ = conn.Close() })
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if _, err := fmt.Fprintf(conn, "PUT %s HTTP/1.1\r\nHost: %s\r\nContent-Length: %d\r\nContent-Type: application/octet-stream\r\nConnection: close\r\n\r\n", signed.URL, w.address, size); err != nil {
		t.Fatal(err)
	}
	select {
	case <-w.started:
	case <-time.After(time.Second):
		t.Fatal("upload did not enter the storage consumer")
	}
	return conn
}

func wireResponse(t *testing.T, conn net.Conn) (*http.Response, []byte) {
	t.Helper()
	response, err := http.ReadResponse(bufio.NewReader(conn), nil)
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if err != nil || closeErr != nil {
		t.Fatalf("response body = %v, close = %v", err, closeErr)
	}
	return response, body
}

func (w *budgetWorld) put(t *testing.T, signed domain.SignedRequest, body string, opaque bool) int {
	t.Helper()
	r, err := http.NewRequestWithContext(t.Context(), "PUT", "http://"+w.address+signed.URL, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	r.Header.Set("Content-Type", "application/octet-stream")
	if opaque {
		r.Header.Set("X-Test-Opaque-Writer", "yes")
	}
	response, err := w.client.Do(r)
	if err != nil {
		t.Fatal(err)
	}
	data, readErr := io.ReadAll(response.Body)
	closeErr := response.Body.Close()
	if readErr != nil || closeErr != nil {
		t.Fatalf("PUT response = %v / %v", readErr, closeErr)
	}
	if response.StatusCode == 503 && (response.Header.Get("Retry-After") != "1" || response.Header.Get("Cache-Control") != "no-store" || !strings.Contains(string(data), `"code":"server_busy"`)) {
		t.Fatalf("busy response = %s %v", data, response.Header)
	}
	return response.StatusCode
}

func (w *budgetWorld) emptyTemporaryDirectory(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(w.root, "tmp"))
	if err != nil || len(entries) != 0 {
		t.Fatalf("temporary directory = %v, %v", entries, err)
	}
}

func (w *budgetWorld) nextUploadWorks(t *testing.T) {
	t.Helper()
	_, signed := w.grant(t, 3)
	if status := w.put(t, signed, "new", false); status != 204 {
		t.Fatalf("next transfer = %d", status)
	}
	w.emptyTemporaryDirectory(t)
}

func transferKeyFromRequest(t *testing.T, request domain.SignedRequest) domain.ObjectKey {
	t.Helper()
	path, _, _ := strings.Cut(request.URL, "?")
	key, err := domain.ParseObjectKey(strings.TrimPrefix(path, "/static/local/"))
	if err != nil {
		t.Fatal(err)
	}
	return key
}
