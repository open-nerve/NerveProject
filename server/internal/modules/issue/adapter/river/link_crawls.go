package riveradapter

import (
	"context"
	"uuid"
)

// LinkCrawls adapts link events to transactional River inserts.
type LinkCrawls struct{ inserter Inserter }

func NewLinkCrawls(i Inserter) *LinkCrawls { return &LinkCrawls{inserter: i} }

func (c *LinkCrawls) Enqueue(ctx context.Context, id uuid.UUID, url string) error {
	args := CrawlLinkArgs{LinkID: id, URL: url}
	opts := args.InsertOpts()
	return c.inserter.InsertTx(ctx, args, &opts)
}
