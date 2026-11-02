package httpadapter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
)

type failingTransferReader struct {
	TransferReader
	fail   atomic.Bool
	closed chan struct{}
}

type unavailableTransferReader struct{}

func (unavailableTransferReader) OpenTransfer(context.Context, domain.ObjectKey) (domain.ObjectInfo, io.ReadSeekCloser, error) {
	return domain.ObjectInfo{}, nil, io.ErrUnexpectedEOF
}

func TestLocalHTTPReadBackendFaultIsNotMissingOrInvalidUpload(t *testing.T) {
	w := newTransferWorld(t, func(d *Deps) { d.Reader = unavailableTransferReader{} })
	signed, err := w.store.SignRead(t.Context(), domain.ReadGrant{Key: transferKey(t, 'o'), Method: "GET", Name: "file.txt", ContentType: "text/plain", ExpiresAt: w.now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	response, data := request(t, w, "GET", signed.URL, nil, nil)
	if response.StatusCode != 500 || !strings.Contains(string(data), `"code":"internal_error"`) || strings.Contains(string(data), "unexpected EOF") || response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("backend failure = %d %s", response.StatusCode, data)
	}
}

func (r *failingTransferReader) OpenTransfer(ctx context.Context, key domain.ObjectKey) (domain.ObjectInfo, io.ReadSeekCloser, error) {
	info, reader, err := r.TransferReader.OpenTransfer(ctx, key)
	if err != nil || !r.fail.Swap(false) {
		return info, reader, err
	}
	return info, &interruptedReader{ReadSeekCloser: reader, remaining: 8192, closed: r.closed}, nil
}

type interruptedReader struct {
	io.ReadSeekCloser
	remaining int
	closed    chan struct{}
}

func (r *interruptedReader) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		return 0, io.ErrUnexpectedEOF
	}
	if len(p) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.ReadSeekCloser.Read(p)
	r.remaining -= n
	return n, err
}

func (r *interruptedReader) Close() error {
	err := r.ReadSeekCloser.Close()
	close(r.closed)
	return err
}

func TestLocalHTTPReadFailureAbortsAndReleasesReader(t *testing.T) {
	failure := &failingTransferReader{closed: make(chan struct{})}
	w := newTransferWorld(t, func(d *Deps) { failure.TransferReader = d.Reader; d.Reader = failure })
	data := strings.Repeat("a", 32768)
	source := w.upload(t, data, "application/octet-stream")
	sealed, err := w.store.Seal(t.Context(), source.Key, transferKey(t, 'o'), source.ETag)
	if err != nil {
		t.Fatal(err)
	}
	signed, err := w.store.SignRead(t.Context(), domain.ReadGrant{Key: sealed.Key, Method: "GET", Name: "data.bin", ContentType: "application/octet-stream", ExpiresAt: w.now().Add(time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	failure.fail.Store(true)
	request, err := http.NewRequestWithContext(t.Context(), "GET", w.server.URL+signed.URL, nil)
	if err != nil {
		t.Fatal(err)
	}
	response, err := w.server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if !errors.Is(err, io.ErrUnexpectedEOF) || string(got) != data[:8192] {
		t.Fatalf("interrupted body = %d bytes, %v", len(got), err)
	}
	select {
	case <-failure.closed:
	case <-time.After(time.Second):
		t.Fatal("failed response retained its reader")
	}
	response, err = w.server.Client().Do(request.Clone(t.Context()))
	if err != nil {
		t.Fatal(err)
	}
	got, err = io.ReadAll(response.Body)
	_ = response.Body.Close()
	if err != nil || string(got) != data || response.StatusCode != 200 {
		t.Fatalf("read after failure = %d, %d bytes, %v", response.StatusCode, len(got), err)
	}
}
