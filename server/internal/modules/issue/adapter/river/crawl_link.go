package riveradapter

import (
	"context"
	"uuid"

	"github.com/riverqueue/river"

	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// CrawlLinkKind identifies metadata fetches for a link's task URL.
const CrawlLinkKind = "issue.crawl_link"

// CrawlLinkMaxAttempts bounds retries of database failures.
const CrawlLinkMaxAttempts = 3

// CrawlLinkArgs retains the exact URL against which metadata may be written.
type CrawlLinkArgs struct {
	LinkID uuid.UUID `json:"link_id"`
	URL    string    `json:"url"`
}

func (CrawlLinkArgs) Kind() string { return CrawlLinkKind }

// InsertOpts keeps request-driven crawls on the default queue.
func (CrawlLinkArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{Queue: river.QueueDefault, MaxAttempts: CrawlLinkMaxAttempts}
}

// CrawlLinkUseCase runs a crawl without depending on River.
type CrawlLinkUseCase interface {
	Execute(context.Context, uuid.UUID, string) error
}

// CrawlLinkWorker delegates task data and the running job context unchanged.
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

// CrawlLinkJob declares a worker without a periodic schedule.
func CrawlLinkJob(uc CrawlLinkUseCase) jobs.Job {
	return jobs.Job{Worker: jobs.Register(NewCrawlLinkWorker(uc))}
}
