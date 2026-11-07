package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LinkCrawler fetches an address's preview, bounded: a page's, or what kept
// it from being fetched, as metadata (M6 P2 spec R12: the one crawler of
// every module that has links, which bootstrap adapts).
type LinkCrawler interface {
	Crawl(ctx context.Context, url string) shared.LinkMetadata
}

// LinkMetadataWriter writes a crawl's metadata to the module link id only
// while it is undeleted and has the address url, as it is spelt; matched
// reports whether it did.
type LinkMetadataWriter interface {
	SetModuleLinkMetadata(ctx context.Context, id uuid.UUID, url string, metadata shared.LinkMetadata) (matched bool, err error)
}

// CrawlLinkDeps are CrawlLink's ports.
type CrawlLinkDeps struct {
	Crawler LinkCrawler
	Links   LinkMetadataWriter
}

// CrawlLink runs a module link's crawl, planning.crawl_link (M6 P2 spec
// R11).
type CrawlLink struct{ d CrawlLinkDeps }

// NewCrawlLink returns the use case over d.
func NewCrawlLink(d CrawlLinkDeps) *CrawlLink { return &CrawlLink{d: d} }

// Execute fetches url and writes what it found against the link id at that
// same address, without the project's lock nor an audit: a link deleted or
// given another address since is not written, and the crawl is done, as it
// is when the fetch was blocked or failed, which its metadata says. The
// store's failure is the crawl's, its chain kept, so the job is retried.
func (u *CrawlLink) Execute(ctx context.Context, id uuid.UUID, url string) error {
	_, err := u.d.Links.SetModuleLinkMetadata(ctx, id, url, u.d.Crawler.Crawl(ctx, url))
	return err
}
