package riveradapter

import (
	"context"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// CrawlLinkKind identifies a module link's crawl (M6 P2 spec R11).
const CrawlLinkKind = "planning.crawl_link"

// CrawlLinkMaxAttempts bounds the retries of a crawl's database failures,
// as the work items' crawl.
const CrawlLinkMaxAttempts = 3

// CrawlLinkArgs are the link and the exact address its metadata may be
// written against.
type CrawlLinkArgs struct {
	LinkID uuid.UUID `json:"link_id"`
	URL    string    `json:"url"`
}

func (CrawlLinkArgs) Kind() string { return CrawlLinkKind }

// InsertOpts keeps the request-driven crawls on the default queue, without
// deduplication: an address a link had before is crawled again.
func (CrawlLinkArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: CrawlLinkMaxAttempts}
}

// CrawlLinkUseCase keeps River out of the application layer.
type CrawlLinkUseCase interface {
	Execute(ctx context.Context, id uuid.UUID, url string) error
}

// CrawlLinkWorker hands a job's link and address to the use case, in the
// running job's context, and answers its error unchanged.
type CrawlLinkWorker struct {
	river.WorkerDefaults[CrawlLinkArgs]
	uc CrawlLinkUseCase
}

func NewCrawlLinkWorker(uc CrawlLinkUseCase) *CrawlLinkWorker {
	return &CrawlLinkWorker{uc: uc}
}

func (w *CrawlLinkWorker) Work(ctx context.Context, job *river.Job[CrawlLinkArgs]) error {
	return w.uc.Execute(ctx, job.Args.LinkID, job.Args.URL)
}

// CrawlLinkJob declares the request-driven worker, without a schedule.
func CrawlLinkJob(uc CrawlLinkUseCase) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewCrawlLinkWorker(uc))}
}
