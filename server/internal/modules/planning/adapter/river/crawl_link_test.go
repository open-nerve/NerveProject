package riveradapter

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"uuid"

	"github.com/riverqueue/river"
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

// The job carries the link and the exact address its metadata may be written against, as planning.crawl_link on the
// default queue with three attempts (M6 P2 spec R11). The worker hands the running job's context, the link and the
// address to the use case and answers its error unchanged, so a database failure is retried and a cancellation
// stays one; the job has no schedule.
func TestCrawlLinkWorker(t *testing.T) {
	id := uuid.MustParse("018e0000-0000-7000-8000-000000000001")
	args := CrawlLinkArgs{LinkID: id, URL: "http://169.254.169.254/task"}
	data, err := json.Marshal(args)
	if err != nil || string(data) != `{"link_id":"018e0000-0000-7000-8000-000000000001","url":"http://169.254.169.254/task"}` {
		t.Fatalf("args=%s err=%v", data, err)
	}
	if opts := args.InsertOpts(); args.Kind() != "planning.crawl_link" || !reflect.DeepEqual(opts, river.InsertOpts{Queue: "default", MaxAttempts: 3}) {
		t.Fatalf("kind=%s opts=%+v", args.Kind(), opts)
	}
	boom := errors.New("database unavailable")
	for _, tc := range []struct {
		name string
		err  error
	}{{"metadata written or stale", nil}, {"a database failure", fmt.Errorf("write metadata: %w", boom)}, {"a cancellation", context.Canceled}} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()
			if errors.Is(tc.err, context.Canceled) {
				cancel()
			}
			uc := &crawlCall{err: tc.err}
			err := NewCrawlLinkWorker(uc).Work(ctx, &river.Job[CrawlLinkArgs]{Args: args})
			if !errors.Is(err, tc.err) || uc.ctx != ctx || uc.id != id || uc.url != args.URL {
				t.Fatalf("worker err=%v ctx=%v id=%s url=%q; want the original call and error %v", err, uc.ctx, uc.id, uc.url, tc.err)
			}
			if job := CrawlLinkJob(uc); job.Worker == nil || job.Periodic != nil {
				t.Fatalf("crawl job=%+v; want a worker without a schedule", job)
			}
		})
	}
}

// Each enqueue inserts one job, in the caller's context (its transaction), with the link and the address as given,
// on the default queue with three attempts and no deduplication: an address a link had before is crawled again. The
// insert's failure is the enqueue's.
func TestLinkCrawlsEnqueue(t *testing.T) {
	ctx := t.Context()
	id := uuid.NewV7()
	url := "http://169.254.169.254/Exact?x=%2F"
	failure := errors.New("insert failed")
	calls := 0
	crawls := NewLinkCrawls(insertCall(func(got context.Context, args river.JobArgs, opts *river.InsertOpts) error {
		calls++
		if got != ctx || !reflect.DeepEqual(args, CrawlLinkArgs{LinkID: id, URL: url}) {
			t.Errorf("lost the caller's context or the args: %t %+v", got == ctx, args)
		}
		if opts == nil || !reflect.DeepEqual(*opts, river.InsertOpts{Queue: "default", MaxAttempts: 3}) {
			t.Errorf("opts=%+v; want default/3 without deduplication", opts)
		}
		return failure
	}))
	for range 2 {
		if err := crawls.Enqueue(ctx, id, url); !errors.Is(err, failure) {
			t.Errorf("Enqueue = %v; want the insert's failure", err)
		}
	}
	if calls != 2 {
		t.Fatalf("InsertTx calls=%d; want one for each enqueue", calls)
	}
}
