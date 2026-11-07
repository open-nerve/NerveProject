package bootstrap

import (
	"context"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound"
	"github.com/open-nerve/NerveProject/server/internal/platform/outbound/linkpreview"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// One link-preview crawler serves the link crawl of every module that has
// links (M6 design 3.2, P2 spec R12).
var _ issue.LinkCrawler = linkPreviews{}

// linkPreviews hands the platform crawler's Metadata to the modules as the
// shared kernel's LinkMetadata: the platform does not import shared.
type linkPreviews struct {
	crawler *linkpreview.Crawler
}

func newLinkPreviews(client *outbound.Client, logger *slog.Logger) linkPreviews {
	return linkPreviews{crawler: linkpreview.New(client, logger)}
}

func (p linkPreviews) Crawl(ctx context.Context, url string) shared.LinkMetadata {
	return shared.LinkMetadata(p.crawler.Crawl(ctx, url))
}
