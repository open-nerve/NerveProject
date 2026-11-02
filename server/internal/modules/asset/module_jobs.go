package asset

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/asset/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.CleanupJob(m.cleanup, m.cleanupInterval, m.cleanupBudget), riveradapter.DeleteScopeJob(m.deleteScope, m.cleanupBudget)}
}
func EnqueuedKinds() []string { return []string{riveradapter.DeleteScopeKind} }
