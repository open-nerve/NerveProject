package navigation

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/navigation/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// Jobs declares the request-driven deletion of the views and the
// favorites of a deleted project, or of a deleted workspace's own.
func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.DeleteRowsJob(m.deleteRows, m.deleteBudget, m.logger)}
}

// EnqueuedKinds declares kinds that require a registered request-driven
// worker.
func EnqueuedKinds() []string {
	return []string{riveradapter.DeleteRowsKind}
}
