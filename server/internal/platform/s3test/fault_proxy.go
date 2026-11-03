package s3test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"
	"time"
)

type HoldPoint int

const (
	BeforeRequest HoldPoint = iota
	BeforeLastUploadByte
	BeforeResponse
)

// FaultProxy only forwards to its owned fixture. It preserves the original
// signed Host/path/query and holds one exact request, never production traffic.
type FaultProxy struct {
	URL     string
	mu      sync.Mutex
	pending *HeldRequest
}

type HeldRequest struct {
	Reached      chan struct{}
	Result       chan error
	method, path string
	point        HoldPoint
	release      chan struct{}
	once         sync.Once
}

func (h *HeldRequest) Release() { h.once.Do(func() { close(h.release) }) }

func (h *HeldRequest) wait(ctx context.Context) error {
	close(h.Reached)
	select {
	case <-h.release:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (p *FaultProxy) HoldNext(t testing.TB, method, path string, point HoldPoint) *HeldRequest {
	t.Helper()
	h := &HeldRequest{Reached: make(chan struct{}), Result: make(chan error, 1), method: method, path: path, point: point, release: make(chan struct{})}
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.pending != nil {
		t.Fatal("s3test fault already pending")
	}
	p.pending = h
	t.Cleanup(h.Release)
	return h
}

func NewFaultProxy(t testing.TB, bucket *Bucket) *FaultProxy {
	t.Helper()
	target, err := url.Parse(bucket.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	life, cancel := context.WithCancel(context.Background())
	transport := &http.Transport{Proxy: nil, ExpectContinueTimeout: time.Second}
	p := &FaultProxy{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p.mu.Lock()
		h := p.pending
		if h != nil && h.method == r.Method && h.path == r.URL.Path {
			p.pending = nil
		} else {
			h = nil
		}
		p.mu.Unlock()
		ctx := r.Context()
		if h != nil {
			ctx = life
		}
		op, done := context.WithTimeout(ctx, 15*time.Second)
		defer done()
		forward := r.Clone(op)
		forward.URL.Scheme, forward.URL.Host = target.Scheme, target.Host
		forward.Host, forward.RequestURI = r.Host, ""
		if h != nil {
			body, readErr := io.ReadAll(io.LimitReader(r.Body, (1<<20)+1))
			if readErr != nil || len(body) > 1<<20 {
				h.Result <- errors.New("s3test held body invalid")
				w.WriteHeader(502)
				return
			}
			forward.Body = io.NopCloser(bytes.NewReader(body))
			if h.point == BeforeLastUploadByte {
				if len(body) < 2 {
					h.Result <- errors.New("s3test held upload needs two bytes")
					w.WriteHeader(502)
					return
				}
				forward.Header.Set("Expect", "100-continue")
				forward.Body = io.NopCloser(&heldUpload{data: body, held: h, ctx: op})
			}
			if h.point == BeforeRequest {
				if err := h.wait(op); err != nil {
					h.Result <- err
					w.WriteHeader(502)
					return
				}
			}
		}
		response, callErr := transport.RoundTrip(forward)
		if callErr != nil {
			if h != nil {
				h.Result <- errors.New("s3test provider transport failed")
			}
			w.WriteHeader(502)
			return
		}
		defer func() { _ = response.Body.Close() }()
		if h != nil {
			body, readErr := io.ReadAll(io.LimitReader(response.Body, (1<<20)+1))
			if readErr != nil || len(body) > 1<<20 {
				h.Result <- errors.New("s3test provider response invalid")
				w.WriteHeader(502)
				return
			}
			if h.point == BeforeResponse {
				if err := h.wait(op); err != nil {
					h.Result <- err
					w.WriteHeader(502)
					return
				}
			}
			var result error
			if response.StatusCode < 200 || response.StatusCode >= 300 || bytes.Contains(body, []byte("<Error>")) {
				result = fmt.Errorf("s3test provider HTTP status %d or embedded error", response.StatusCode)
			}
			h.Result <- result
			response.Body = io.NopCloser(bytes.NewReader(body))
		}
		for name, values := range response.Header {
			w.Header()[name] = values
		}
		w.WriteHeader(response.StatusCode)
		_, _ = io.Copy(w, response.Body)
	}))
	p.URL = srv.URL
	t.Cleanup(func() { cancel(); srv.Close(); transport.CloseIdleConnections() })
	return p
}

type heldUpload struct {
	data   []byte
	held   *HeldRequest
	ctx    context.Context
	waited bool
}

func (r *heldUpload) Read(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(r.data) == 0 {
		return 0, io.EOF
	}
	if len(r.data) == 1 && !r.waited {
		r.waited = true
		if err := r.held.wait(r.ctx); err != nil {
			return 0, err
		}
	}
	n := len(r.data)
	if n > 1 {
		n--
	}
	n = copy(p, r.data[:n])
	r.data = r.data[n:]
	return n, nil
}
