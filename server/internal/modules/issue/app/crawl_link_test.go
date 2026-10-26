package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

type crawlResult struct {
	ctx      context.Context
	url      string
	metadata domain.LinkMetadata
}

func (c *crawlResult) Crawl(ctx context.Context, url string) domain.LinkMetadata {
	c.ctx, c.url = ctx, url
	return c.metadata
}

type metadataWrite struct {
	ctx      context.Context
	id       uuid.UUID
	url      string
	metadata domain.LinkMetadata
	changed  bool
	err      error
}

func (w *metadataWrite) SetLinkMetadata(ctx context.Context, id uuid.UUID, url string, m domain.LinkMetadata) (bool, error) {
	w.ctx, w.id, w.url, w.metadata = ctx, id, url, m
	return w.changed, w.err
}

// Dropping a failed network result, treating a stale task as an error,
// swallowing a DB error, or losing the caller context changes these results.
func TestCrawlLink(t *testing.T) {
	title, rawURL, blocked, failed := "Crawled title", "http://169.254.169.254/task", "blocked", "failed"
	boom := errors.New("database unavailable")
	for _, tc := range []struct {
		name     string
		metadata domain.LinkMetadata
		changed  bool
		err      error
		canceled bool
	}{
		{"page success", domain.LinkMetadata{"title": &title, "url": &rawURL, "favicon": nil, "favicon_url": nil}, true, nil, false},
		{"blocked result completes", domain.LinkMetadata{"error": &blocked, "url": &rawURL}, true, nil, false},
		{"failed result completes", domain.LinkMetadata{"error": &failed, "url": &rawURL}, true, nil, false},
		{"stale task completes", domain.LinkMetadata{"error": &blocked, "url": &rawURL}, false, nil, false},
		{"DB failure retries", domain.LinkMetadata{"title": &title}, false, fmt.Errorf("write metadata: %w", boom), false},
		{"canceled context stays canceled", domain.LinkMetadata{"error": &failed, "url": &rawURL}, false, context.Canceled, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.canceled {
				cancel()
			}
			c := &crawlResult{metadata: tc.metadata}
			w := &metadataWrite{changed: tc.changed, err: tc.err}
			id := uuid.NewV7()
			err := app.NewCrawlLink(app.CrawlLinkDeps{Crawler: c, Links: w}).Execute(ctx, id, rawURL)
			if !errors.Is(err, tc.err) || c.ctx != ctx || w.ctx != ctx || c.url != rawURL || w.url != rawURL || w.id != id || !reflect.DeepEqual(w.metadata, tc.metadata) {
				t.Fatalf("crawl result err=%v crawler=(%v,%q) write=(%v,%s,%q,%#v); want original context, id, URL, metadata and error %v", err, c.ctx, c.url, w.ctx, w.id, w.url, w.metadata, tc.err)
			}
		})
	}
}
