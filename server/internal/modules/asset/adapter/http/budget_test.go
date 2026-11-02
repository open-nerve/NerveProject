package httpadapter

import (
	"bufio"
	"context"
	"errors"
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

func TestLocalTransferDeadlines(t *testing.T) {
	t.Run("progress extends both inherited socket deadlines", func(t *testing.T) {
		w := newBudgetWorld(t, budgetTestFiles(), 200*time.Millisecond, 200*time.Millisecond)
		g, signed := w.grant(t, 512<<10)
		conn := w.beginUpload(t, signed, g.Size)
		chunk := strings.Repeat("a", 64<<10)
		ticker := time.NewTicker(100 * time.Millisecond)
		defer ticker.Stop()
		start := time.Now()
		for range 8 {
			<-ticker.C
			if _, err := io.WriteString(conn, chunk); err != nil {
				t.Fatal(err)
			}
		}
		response, body := wireResponse(t, conn)
		if response.StatusCode != 204 || len(body) != 0 || time.Since(start) < 600*time.Millisecond {
			t.Fatalf("progress response = %d %q after %s", response.StatusCode, body, time.Since(start))
		}
		if response.Header.Get(httpserver.HeaderRequestID) == "" || response.Header.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("platform headers = %v", response.Header)
		}
		info, err := w.store.Inspect(t.Context(), g.Key)
		if err != nil || info.Size != g.Size {
			t.Fatalf("completed object = %+v, %v", info, err)
		}
		w.nextUploadWorks(t)
	})
	t.Run("idle read ends the socket and releases the slot", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferIdleTimeout = 200 * time.Millisecond
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		g, signed := w.grant(t, 1024)
		conn := w.beginUpload(t, signed, g.Size)
		if _, err := conn.Write([]byte("a")); err != nil {
			t.Fatal(err)
		}
		select {
		case <-w.finished:
		case <-time.After(time.Second):
			t.Fatal("idle upload retained its socket")
		}
		response, _ := wireResponse(t, conn)
		if response.StatusCode != 408 {
			t.Fatalf("idle status = %d", response.StatusCode)
		}
		if _, err := w.store.Inspect(t.Context(), g.Key); err != domain.ErrObjectNotFound {
			t.Fatalf("idle upload published: %v", err)
		}
		w.nextUploadWorks(t)
	})
	t.Run("progress cannot extend the total budget", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferGrace = 50 * time.Millisecond
		files.MaxTransferDuration = 1050 * time.Millisecond
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		g, signed := w.grant(t, 1024)
		conn := w.beginUpload(t, signed, g.Size)
		stop, writerDone := make(chan struct{}), make(chan struct{})
		go func() {
			defer close(writerDone)
			ticker := time.NewTicker(100 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-stop:
					return
				case <-ticker.C:
					if _, err := conn.Write([]byte("a")); err != nil {
						return
					}
				}
			}
		}()
		defer func() { close(stop); <-writerDone }()
		select {
		case <-w.finished:
		case <-time.After(2 * time.Second):
			t.Fatal("progress bypassed the total transfer budget")
		}
		response, _ := wireResponse(t, conn)
		if response.StatusCode != 408 {
			t.Fatalf("total status = %d", response.StatusCode)
		}
		w.nextUploadWorks(t)
	})
	t.Run("parent cancellation interrupts a blocked body", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferIdleTimeout = 2 * time.Second
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		g, signed := w.grant(t, 1024)
		conn := w.beginUpload(t, signed, g.Size)
		(<-w.cancellations)()
		select {
		case <-w.finished:
		case <-time.After(time.Second):
			t.Fatal("cancellation left the body read blocked")
		}
		response, _ := wireResponse(t, conn)
		if response.StatusCode != 499 {
			t.Fatalf("cancel status = %d", response.StatusCode)
		}
		w.nextUploadWorks(t)
	})
	t.Run("busy and all early exits release their slots", func(t *testing.T) {
		w := newBudgetWorld(t, budgetTestFiles(), 5*time.Second, 5*time.Second)
		g, first := w.grant(t, 3)
		conn := w.beginUpload(t, first, g.Size)
		_, second := w.grant(t, 3)
		if status := w.put(t, second, "new", false); status != 503 {
			t.Fatalf("busy = %d", status)
		}
		if _, err := conn.Write([]byte("old")); err != nil {
			t.Fatal(err)
		}
		if response, _ := wireResponse(t, conn); response.StatusCode != 204 {
			t.Fatalf("first upload = %d", response.StatusCode)
		}
		if status := w.put(t, first, "old", false); status != 409 {
			t.Fatalf("conflict = %d", status)
		}
		if status := w.put(t, second, "new", true); status != 500 {
			t.Fatalf("unsupported controller = %d", status)
		}
		if _, err := w.store.Inspect(t.Context(), transferKeyFromRequest(t, second)); err != domain.ErrObjectNotFound {
			t.Fatalf("unsupported transfer published: %v", err)
		}
		tmp := filepath.Join(w.root, "tmp")
		if err := os.Remove(tmp); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(tmp, []byte("cannot create temporary directory"), 0600); err != nil {
			t.Fatal(err)
		}
		if status := w.put(t, second, "new", false); status != 500 {
			t.Fatalf("storage failure = %d", status)
		}
		if err := os.Remove(tmp); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(tmp, 0700); err != nil {
			t.Fatal(err)
		}
		if status := w.put(t, second, "new", false); status != 204 {
			t.Fatalf("retry after faults = %d", status)
		}
		w.emptyTemporaryDirectory(t)
	})
	t.Run("stalled download ends its write and releases the slot", func(t *testing.T) {
		files := budgetTestFiles()
		files.TransferIdleTimeout = 200 * time.Millisecond
		w := newBudgetWorld(t, files, 5*time.Second, 5*time.Second)
		data := strings.Repeat("a", 8<<20)
		grant, _ := w.grant(t, int64(len(data)))
		if err := w.store.Put(t.Context(), grant, strings.NewReader(data)); err != nil {
			t.Fatal(err)
		}
		info, err := w.store.Inspect(t.Context(), grant.Key)
		if err != nil {
			t.Fatal(err)
		}
		sealed, err := w.store.Seal(t.Context(), grant.Key, transferKey(t, 'o'), info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		signed, err := w.store.SignRead(t.Context(), domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "large.bin", ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		connection, err := net.DialTimeout("tcp", w.address, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		conn := connection.(*net.TCPConn)
		defer func() { _ = conn.Close() }()
		if err := conn.SetReadBuffer(1024); err != nil {
			t.Fatal(err)
		}
		if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", signed.URL, w.address); err != nil {
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 200 || response.ContentLength != int64(len(data)) {
			t.Fatalf("download did not start: %d, length %d", response.StatusCode, response.ContentLength)
		}
		// Do not drain the receive buffer. The real TCP write must time out well
		// before the five-second global deadline and the three-second total.
		select {
		case <-w.finished:
		case <-time.After(1500 * time.Millisecond):
			t.Fatal("stalled client retained the write/socket")
		}
		if err := conn.SetReadBuffer(4 << 20); err != nil {
			t.Fatal(err)
		}
		partial, err := io.ReadAll(response.Body)
		if !errors.Is(err, io.ErrUnexpectedEOF) || len(partial) >= len(data) || string(partial) != data[:len(partial)] {
			t.Fatalf("stalled response was not truncated: %d bytes, %v", len(partial), err)
		}
		w.nextUploadWorks(t)
	})
	t.Run("active download is not canceled by an idle request side", func(t *testing.T) {
		w := newBudgetWorld(t, budgetTestFiles(), 200*time.Millisecond, 200*time.Millisecond)
		data := strings.Repeat("a", 8<<20)
		grant, _ := w.grant(t, int64(len(data)))
		if err := w.store.Put(t.Context(), grant, strings.NewReader(data)); err != nil {
			t.Fatal(err)
		}
		info, err := w.store.Inspect(t.Context(), grant.Key)
		if err != nil {
			t.Fatal(err)
		}
		sealed, err := w.store.Seal(t.Context(), grant.Key, transferKey(t, 'o'), info.ETag)
		if err != nil {
			t.Fatal(err)
		}
		signed, err := w.store.SignRead(t.Context(), domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "paced.bin", ContentType: "application/octet-stream", ExpiresAt: time.Now().Add(time.Minute)})
		if err != nil {
			t.Fatal(err)
		}
		connection, err := net.DialTimeout("tcp", w.address, time.Second)
		if err != nil {
			t.Fatal(err)
		}
		conn := connection.(*net.TCPConn)
		defer func() { _ = conn.Close() }()
		if err := conn.SetReadBuffer(64 << 10); err != nil {
			t.Fatal(err)
		}
		if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
			t.Fatal(err)
		}
		if _, err := fmt.Fprintf(conn, "GET %s HTTP/1.1\r\nHost: %s\r\nConnection: close\r\n\r\n", signed.URL, w.address); err != nil {
			t.Fatal(err)
		}
		response, err := http.ReadResponse(bufio.NewReader(conn), nil)
		if err != nil {
			t.Fatal(err)
		}
		defer func() { _ = response.Body.Close() }()
		if response.StatusCode != 200 {
			t.Fatalf("download status = %d", response.StatusCode)
		}
		ticker := time.NewTicker(50 * time.Millisecond)
		defer ticker.Stop()
		chunk := make([]byte, 256<<10)
		for range 32 {
			<-ticker.C
			n, err := io.ReadFull(response.Body, chunk)
			if err != nil || n != len(chunk) || string(chunk) != data[:len(chunk)] {
				t.Fatalf("active download interrupted: %d bytes, %v", n, err)
			}
		}
		if n, err := response.Body.Read(chunk[:1]); n != 0 || err != io.EOF {
			t.Fatalf("download end = %d, %v", n, err)
		}
		w.nextUploadWorks(t)
	})
}

func transferKeyFromRequest(t *testing.T, request domain.SignedRequest) domain.ObjectKey {
	t.Helper()
	path, _, _ := strings.Cut(request.URL, "?")
	key, err := domain.ParseObjectKey(strings.TrimPrefix(path, "/storage/local/"))
	if err != nil {
		t.Fatal(err)
	}
	return key
}
