package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// Jobs declares request-driven crawls and scheduled automatic archival.
func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.AutoArchiveJob(m.autoArchive, m.autoArchiveInterval), riveradapter.CrawlLinkJob(m.crawlLink)}
}

// EnqueuedKinds declares kinds that require a registered request-driven worker.
func EnqueuedKinds() []string { return []string{riveradapter.CrawlLinkKind} }
