package postgresadapter_test

import (
	"context"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

func purgeAll(p *app.Purger, ctx context.Context, cutoff time.Time, limit int) (int, error) {
	var cursor app.PurgeCursor
	total := 0
	for {
		r, err := p.Page(ctx, cutoff, limit, cursor)
		total += r.Deleted
		if err != nil || r.Done {
			return total, err
		}
		cursor = r.Cursor
	}
}
