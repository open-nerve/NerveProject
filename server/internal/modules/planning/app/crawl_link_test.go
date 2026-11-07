package app

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// crawled is a LinkCrawler that records what it is asked and answers metadata.
type crawled struct {
	ctx      context.Context
	url      string
	metadata shared.LinkMetadata
}

func (c *crawled) Crawl(ctx context.Context, url string) shared.LinkMetadata {
	c.ctx, c.url = ctx, url
	return c.metadata
}

// metadataWrite is a LinkMetadataWriter that records what it is asked and answers whether the link still matched,
// and err.
type metadataWrite struct {
	ctx      context.Context
	id       uuid.UUID
	url      string
	metadata shared.LinkMetadata
	matched  bool
	err      error
}

func (w *metadataWrite) SetModuleLinkMetadata(ctx context.Context, id uuid.UUID, url string, m shared.LinkMetadata) (bool, error) {
	w.ctx, w.id, w.url, w.metadata = ctx, id, url, m
	return w.matched, w.err
}

// A crawl fetches the job's address and writes what it found against the link and that same address, in the job's
// context, without a lock of the project nor an audit (M6 P2 spec R11): a page, and a blocked or failed fetch, are
// written as they are and complete the job; a link whose address changed or that is deleted since is not written,
// and completes it too; the store's failure is the crawl's, its chain kept, so the job is retried; a cancellation
// stays one.
func TestCrawlLink(t *testing.T) {
	title, rawURL, blocked, failed := "Crawled title", "http://169.254.169.254/task", "blocked", "failed"
	boom := errors.New("database unavailable")
	for _, tc := range []struct {
		name     string
		metadata shared.LinkMetadata
		matched  bool
		err      error
		canceled bool
	}{
		{"a page", shared.LinkMetadata{"title": &title, "url": &rawURL, "favicon": nil, "favicon_url": nil}, true, nil, false},
		{"a blocked fetch", shared.LinkMetadata{"error": &blocked, "url": &rawURL}, true, nil, false},
		{"a failed fetch", shared.LinkMetadata{"error": &failed, "url": &rawURL}, true, nil, false},
		{"a link changed or deleted since", shared.LinkMetadata{"error": &blocked, "url": &rawURL}, false, nil, false},
		{"a database failure", shared.LinkMetadata{"title": &title}, false, fmt.Errorf("write metadata: %w", boom), false},
		{"a cancellation", shared.LinkMetadata{"error": &failed, "url": &rawURL}, false, context.Canceled, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if tc.canceled {
				cancel()
			}
			c := &crawled{metadata: tc.metadata}
			w := &metadataWrite{matched: tc.matched, err: tc.err}
			id := uuid.NewV7()
			err := NewCrawlLink(CrawlLinkDeps{Crawler: c, Links: w}).Execute(ctx, id, rawURL)
			if !errors.Is(err, tc.err) || c.ctx != ctx || w.ctx != ctx || c.url != rawURL || w.url != rawURL || w.id != id ||
				!reflect.DeepEqual(w.metadata, tc.metadata) {
				t.Fatalf("crawl err=%v crawler=(%v,%q) write=(%v,%s,%q,%#v); want the job's context, link, address, metadata and error %v",
					err, c.ctx, c.url, w.ctx, w.id, w.url, w.metadata, tc.err)
			}
		})
	}
}
