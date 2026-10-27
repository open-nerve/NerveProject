package crawler_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

func testSharedBudget(t *testing.T) {
	pageEntered, iconEntered, iconDone := make(chan struct{}), make(chan struct{}), make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			close(pageEntered)
			// Consume part of the total budget under a deadline, not a sleep used for ordering.
			phase, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()
			<-phase.Done()
			response(w, 200, "text/html", "<title>kept</title>")
			return
		}
		close(iconEntered)
		<-r.Context().Done()
		close(iconDone)
	}))
	defer s.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	result := make(chan domain.LinkMetadata, 1)
	c := newCrawler(t)
	start := time.Now()
	go func() { result <- c.Crawl(ctx, s.URL+"/") }()
	await(t, pageEntered)
	await(t, iconEntered)
	select {
	case got := <-result:
		elapsed := time.Since(start)
		if elapsed < 4500*time.Millisecond || elapsed > 6500*time.Millisecond {
			t.Errorf("shared budget elapsed=%v, want about 5s (fresh icon budget takes 8s)", elapsed)
		}
		wantMetadata(t, got, success(s.URL+"/", value("kept")))
	case <-ctx.Done():
		t.Fatal("crawl failed to honor its total five-second budget")
	}
	await(t, iconDone)
}

func testParentDeadline(t *testing.T) {
	entered, done := make(chan struct{}), make(chan struct{})
	s := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { close(entered); <-r.Context().Done(); close(done) }))
	defer s.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 200*time.Millisecond)
	defer cancel()
	result := make(chan domain.LinkMetadata, 1)
	c := newCrawler(t)
	go func() { result <- c.Crawl(ctx, s.URL) }()
	await(t, entered)
	select {
	case got := <-result:
		if ctx.Err() != context.DeadlineExceeded {
			t.Error("crawl returned before parent deadline")
		}
		wantMetadata(t, got, domain.LinkMetadata{"error": value("failed"), "url": value(s.URL)})
	case <-time.After(2 * time.Second):
		t.Fatal("shorter parent deadline ignored")
	}
	await(t, done)
}
