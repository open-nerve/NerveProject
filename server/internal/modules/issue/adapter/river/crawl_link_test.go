package riveradapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/riverqueue/river"

	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
)

type crawlCall struct {
	ctx context.Context
	id  uuid.UUID
	url string
	err error
}

func (c *crawlCall) Execute(ctx context.Context, id uuid.UUID, url string) error {
	c.ctx, c.id, c.url = ctx, id, url
	return c.err
}

func TestCrawlLinkWorker(t *testing.T) {
	id := uuid.MustParse("018e0000-0000-7000-8000-000000000001")
	args := riveradapter.CrawlLinkArgs{LinkID: id, URL: "http://169.254.169.254/task"}
	data, err := json.Marshal(args)
	if err != nil || string(data) != `{"link_id":"018e0000-0000-7000-8000-000000000001","url":"http://169.254.169.254/task"}` {
		t.Fatalf("args=%s err=%v", data, err)
	}
	opts := args.InsertOpts()
	if args.Kind() != "issue.crawl_link" || opts.Queue != "default" || opts.MaxAttempts != 3 || riveradapter.CrawlLinkMaxAttempts != 3 {
		t.Fatalf("kind=%s opts=%+v", args.Kind(), opts)
	}
	boom := errors.New("database unavailable")
	for _, tc := range []struct {
		name string
		err  error
	}{{"network result completed", nil}, {"database failure", fmt.Errorf("write metadata: %w", boom)}, {"cancellation", context.Canceled}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if errors.Is(tc.err, context.Canceled) {
				cancel()
			}
			uc := &crawlCall{err: tc.err}
			err := riveradapter.NewCrawlLinkWorker(uc).Work(ctx, &river.Job[riveradapter.CrawlLinkArgs]{Args: args})
			if !errors.Is(err, tc.err) || uc.ctx != ctx || uc.id != id || uc.url != args.URL {
				t.Fatalf("worker err=%v ctx=%v id=%s url=%q; want original call and error %v", err, uc.ctx, uc.id, uc.url, tc.err)
			}
			job := riveradapter.CrawlLinkJob(uc)
			if job.Worker == nil || job.Periodic != nil {
				t.Fatalf("crawl job=%+v; want worker without periodic schedule", job)
			}
		})
	}
}
