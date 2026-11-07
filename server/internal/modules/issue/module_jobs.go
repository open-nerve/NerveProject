package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// LinkCrawler is the port the link crawl job fetches a URL's preview through;
// bootstrap adapts the platform's link-preview crawler to it.
type LinkCrawler = app.LinkCrawler

// Jobs declares request-driven crawls and deletions and scheduled archival and description retention.
func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.TrimVersionsJob(m.trimVersions, m.trimVersionsInterval), riveradapter.AutoArchiveJob(m.autoArchive, m.autoArchiveInterval), riveradapter.CrawlLinkJob(m.crawlLink), riveradapter.DeleteRowsJob(m.deleteRows, m.deleteBudget, m.logger)}
}

// EnqueuedKinds declares kinds that require a registered request-driven worker.
func EnqueuedKinds() []string {
	return []string{riveradapter.CrawlLinkKind, riveradapter.DeleteRowsKind}
}
