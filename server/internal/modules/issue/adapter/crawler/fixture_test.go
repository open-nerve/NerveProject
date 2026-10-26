package crawler_test

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/crawler"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
)

var _ app.LinkCrawler = (*crawler.Crawler)(nil)

func localClient(t *testing.T) *outbound.Client {
	t.Helper()
	c, err := outbound.New(outbound.Options{AllowCIDRs: []netip.Prefix{netip.MustParsePrefix("127.0.0.1/32")}, Version: "crawler-test"})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func newCrawler(t *testing.T) *crawler.Crawler {
	t.Helper()
	return crawler.New(localClient(t), slog.New(slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.Level(math.MinInt)})))
}

func response(w http.ResponseWriter, status int, mime, body string) {
	w.Header().Set("Content-Type", mime)
	w.WriteHeader(status)
	_, _ = io.WriteString(w, body)
}

func value(s string) *string { return &s }

func success(target string, title *string) domain.LinkMetadata {
	return domain.LinkMetadata{"title": title, "favicon": nil, "url": value(target), "favicon_url": nil}
}

func wantMetadata(t *testing.T, got, want domain.LinkMetadata) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		// Keep oversized data URLs out of assertion output.
		actual, _ := json.Marshal(got)
		expected, _ := json.Marshal(want)
		if len(actual) > 512 {
			actual = actual[:512]
		}
		if len(expected) > 512 {
			expected = expected[:512]
		}
		t.Errorf("metadata mismatch: got %q want %q", actual, expected)
	}
}

func await(t *testing.T, signal <-chan struct{}) {
	t.Helper()
	select {
	case <-signal:
	case <-time.After(10 * time.Second):
		t.Fatal("fixture condition did not occur before deadline")
	}
}

type observedListener struct {
	net.Listener
	accepted atomic.Int32
}

func (l *observedListener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err == nil {
		l.accepted.Add(1)
	}
	return c, err
}

func dangerousFixture(t *testing.T) (*httptest.Server, *observedListener, *atomic.Int32) {
	t.Helper()
	hits := new(atomic.Int32)
	s := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		hits.Add(1)
		response(w, 200, "image/png", "danger")
	}))
	listener := &observedListener{Listener: s.Listener}
	s.Listener = listener
	s.Start()
	t.Cleanup(s.Close)
	// Positive control proves this real listener is reachable and both counters work.
	r, err := localClient(t).Get(context.Background(), s.URL, outbound.Limits{ConnectTimeout: time.Second, Timeout: time.Second, MaxBytes: 1024})
	if err != nil || string(r.Body) != "danger" || listener.accepted.Load() != 1 || hits.Load() != 1 {
		t.Fatalf("danger fixture control: err=%v sockets=%d requests=%d", err, listener.accepted.Load(), hits.Load())
	}
	return s, listener, hits
}
