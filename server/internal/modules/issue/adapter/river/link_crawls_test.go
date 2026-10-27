package riveradapter

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/riverqueue/river"
)

type crawlInsertCall func(context.Context, river.JobArgs, *river.InsertOpts) error

func (f crawlInsertCall) InsertTx(ctx context.Context, args river.JobArgs, opts *river.InsertOpts) error {
	return f(ctx, args, opts)
}

func TestLinkCrawlsEnqueue(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	id := uuid.NewV7()
	url := "http://169.254.169.254/Exact?x=%2F"
	failure := errors.New("transaction insert failed")
	calls := 0
	crawls := NewLinkCrawls(crawlInsertCall(func(got context.Context, args river.JobArgs, opts *river.InsertOpts) error {
		calls++
		if got != ctx || !reflect.DeepEqual(args, CrawlLinkArgs{LinkID: id, URL: url}) {
			t.Errorf("lost caller context or args: %v %+v", got == ctx, args)
		}
		if opts == nil || opts.Queue != "default" || opts.MaxAttempts != 3 || !reflect.DeepEqual(opts.UniqueOpts, river.UniqueOpts{}) {
			t.Errorf("insert opts=%+v; want default/3 without deduplication", opts)
		}
		return failure
	}))
	for range 2 {
		if err := crawls.Enqueue(ctx, id, url); !errors.Is(err, failure) {
			t.Errorf("enqueue=%v; want original error", err)
		}
	}
	if calls != 2 {
		t.Fatalf("InsertTx calls=%d; want independent insert for each event", calls)
	}
}
