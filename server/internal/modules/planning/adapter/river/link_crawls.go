package riveradapter

import (
	"context"
	"uuid"
)

// LinkCrawls inserts a module link's crawl in the transaction ctx carries
// (app.LinkCrawls).
type LinkCrawls struct{ inserter Inserter }

func NewLinkCrawls(i Inserter) *LinkCrawls { return &LinkCrawls{inserter: i} }

func (c *LinkCrawls) Enqueue(ctx context.Context, id uuid.UUID, url string) error {
	args := CrawlLinkArgs{LinkID: id, URL: url}
	opts := args.InsertOpts()
	return c.inserter.InsertTx(ctx, args, &opts)
}
