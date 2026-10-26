package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// Jobs declares the automatic archival worker and its periodic schedule.
func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.AutoArchiveJob(m.autoArchive, m.autoArchiveInterval)}
}

// EnqueuedKinds is empty: requests do not enqueue jobs in P6.
func EnqueuedKinds() []string { return nil }
