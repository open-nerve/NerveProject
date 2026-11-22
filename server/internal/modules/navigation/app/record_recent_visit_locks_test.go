package app_test

import (
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A recording decides under its locks (spec 2.7, 2.8): what another writer committed while its check ran and its locks
// waited, the reads under them find. A workspace deleted, she no longer its member, the project deleted, the target
// deleted, moved or no longer seen (she left its project, or sees only her own now) is the target's 404, the one value:
// the transaction ends where the refusal is, before the advisory lock, the clock and the write. A work item deleted
// meanwhile is found deleted twice, by its place read again and by what she sees of it read again (two layers, each
// alone enough); a project deleted meanwhile, by its FOR SHARE and by the projects she reads.
func TestARecordingDecidesUnderItsLocks(t *testing.T) {
	gone := func(r *recents) {
		r.underLock = map[uuid.UUID]*app.ProjectFacts{web: nil}
		delete(r.projectTargets, web)
		r.grants[acme] = []app.ProjectGrant{{ProjectID: ops}, {ProjectID: attic}}
	}
	left := func(r *recents) { r.grants[acme] = []app.ProjectGrant{{ProjectID: ops}, {ProjectID: attic}} }
	issue := []string{"issue places task"}
	for _, tt := range []struct {
		name       string
		kind       domain.RecentVisitKind
		id         uuid.UUID
		meanwhile  func(r *recents)
		stop       string
		place, see []string
	}{
		{"a work item's workspace deleted", domain.RecentVisitIssue, task, func(r *recents) { r.workspaceGone = map[uuid.UUID]bool{acme: true} },
			"share workspace acme", issue, []string{"readable alice acme", "issue summaries task"}},
		{"she left the workspace", domain.RecentVisitIssue, task, func(r *recents) {
			r.decide = func(_ shared.Action, _ shared.Target, inTx bool) (shared.Grant, error) {
				if inTx {
					return shared.Grant{}, shared.ErrNotVisible
				}
				return member, nil
			}
		}, "authorize alice recent_visit.write acme/-", issue, []string{"readable alice acme", "issue summaries task"}},
		{"a work item's project deleted", domain.RecentVisitIssue, task, gone, "share project web", issue,
			[]string{"readable alice acme", "issue summaries task"}},
		{"the work item deleted", domain.RecentVisitIssue, task, func(r *recents) { delete(r.issues, task) }, "issue places task", issue,
			[]string{"readable alice acme", "issue summaries task"}},
		{"the work item in another project", domain.RecentVisitIssue, task, func(r *recents) {
			i := r.issues[task]
			i.place.ProjectID = ops
			r.issues[task] = i
		}, "issue places task", issue, []string{"readable alice acme", "issue summaries task"}},
		{"she left the work item's project", domain.RecentVisitIssue, task, left, "issue summaries task", issue,
			[]string{"readable alice acme", "issue summaries task"}},
		{"she sees only her own now", domain.RecentVisitIssue, task, func(r *recents) {
			r.grants[acme] = []app.ProjectGrant{{ProjectID: web, OwnOnly: true}}
		}, "issue summaries task", issue, []string{"readable alice acme", "issue summaries task"}},
		{"the project deleted", domain.RecentVisitProject, web, gone, "share project web", []string{"project targets web"},
			[]string{"readable alice acme"}},
		{"she left the project", domain.RecentVisitProject, web, left, "readable alice acme", []string{"project targets web"},
			[]string{"readable alice acme"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			r := newRecents()
			r.meanwhile = func(*shelf) { tt.meanwhile(r) }
			err := r.recordVisit().Execute(as(t), tt.kind, tt.id)
			wantSame(t, err, map[domain.RecentVisitKind]error{domain.RecentVisitIssue: domain.ErrIssueNotFound,
				domain.RecentVisitProject: domain.ErrProjectNotFound}[tt.kind])
			wroteNoVisit(t, r)
			full := recordCalls(tt.place, tt.see, tt.kind, r.name(tt.id))
			begin := slices.Index(full, "begin write")
			stop := begin + 1 + slices.Index(full[begin+1:], tt.stop)
			wantCalls(t, r.world, append(full[:stop+1:stop+1], "end write")...)
		})
	}
}
