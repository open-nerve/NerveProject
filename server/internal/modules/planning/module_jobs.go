package planning

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/planning/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// LinkCrawler is the port the link crawl job fetches an address's preview
// through; bootstrap adapts the platform's link-preview crawler to it.
type LinkCrawler = app.LinkCrawler

// Jobs declares the request-driven crawl of a module link and deletion of a
// deleted project's planning rows, and the scheduled freeze of the cycles
// that have ended.
func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.CrawlLinkJob(m.crawlLink), riveradapter.DeleteRowsJob(m.deleteRows, m.deleteBudget, m.logger),
		riveradapter.FreezeCyclesJob(m.freezeCycles, m.freezeInterval)}
}

// EnqueuedKinds declares kinds that require a registered request-driven
// worker; the scheduled freeze is none.
func EnqueuedKinds() []string {
	return []string{riveradapter.CrawlLinkKind, riveradapter.DeleteRowsKind}
}
