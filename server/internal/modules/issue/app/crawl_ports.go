package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// LinkCrawler produces bounded metadata for one task URL.
type LinkCrawler interface {
	Crawl(context.Context, string) shared.LinkMetadata
}
