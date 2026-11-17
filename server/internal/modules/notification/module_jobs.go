package notification

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/notification/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/platform/jobs"
)

// Jobs declares the request-driven fan-out of an event's notifications and
// deletion of a deleted project's notifications.
func (m *Module) Jobs() []jobs.Job {
	return []jobs.Job{riveradapter.FanoutJob(m.fanout, m.logger), riveradapter.DeleteRowsJob(m.deleteRows, m.deleteBudget, m.logger)}
}

// EnqueuedKinds declares kinds that require a registered request-driven
// worker.
func EnqueuedKinds() []string {
	return []string{riveradapter.FanoutKind, riveradapter.DeleteRowsKind}
}
