package app

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CountIntakeIssuesDeps supplies the read snapshot, the decision Locks
// make, the workspace directory, the projects the caller sees, which of
// them have their intake turned on, the counts' store and the clock.
type CountIntakeIssuesDeps struct {
	ReadTx     shared.TxManager
	Locks      Locks
	Workspaces WorkspaceDirectory
	Projects   VisibleProjects
	Intakes    IntakeProjects
	Counts     IntakeCounter
	Clock      Clock
}

// CountIntakeIssues counts the pending work items of the intakes of a
// workspace's projects: GET /api/v0/workspaces/{slug}/intake-counts (M7
// P2b spec 2.6).
type CountIntakeIssues struct{ d CountIntakeIssuesDeps }

// NewCountIntakeIssues returns the use case.
func NewCountIntakeIssues(d CountIntakeIssuesDeps) *CountIntakeIssues {
	return &CountIntakeIssues{d: d}
}

// Execute answers, in one read snapshot, the workspace slug names read and
// the decision on intake.count in it (Locks.seeWorkspace): a workspace not
// there or not seen is 404 workspace.not_found, a role the rule does not
// allow 403; then the projects the caller counts in (scope); the clock,
// read once; and each of them with its number of pending work items at
// that moment, a snooze run out among them, only his own where he sees
// only his own (IntakeCounter.CountPendingIntake): every one, none pending
// too, in id order. None when there is no project to count, nothing read
// after it.
func (u *CountIntakeIssues) Execute(ctx context.Context, slug string) ([]IntakeCount, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return nil, err
	}
	counts := []IntakeCount{}
	err = u.d.ReadTx.WithinTx(ctx, func(ctx context.Context) error {
		workspaceID, found, err := u.d.Workspaces.WorkspaceBySlug(ctx, slug)
		switch {
		case err != nil:
			return err
		case !found:
			return domain.ErrWorkspaceNotFound
		}
		if err := u.d.Locks.seeWorkspace(ctx, actor, domain.ActionCountIntake, workspaceID); err != nil {
			return err
		}
		scope, err := u.scope(ctx, workspaceID, actor.UserID)
		if err != nil || len(scope) == 0 {
			return err
		}
		counts, err = u.d.Counts.CountPendingIntake(ctx, actor.UserID, scope, u.d.Clock.Now())
		return err
	})
	if err != nil {
		return nil, err
	}
	return counts, nil
}

// scope are the projects of the workspace workspaceID whose pending work
// items the caller actor counts (spec 2.6, DP-20): those he sees, as his
// memberships give them, each with whether he sees only his own work items
// there (VisibleProjects.Visible: an active membership of an undeleted,
// unarchived project), that have their intake turned on
// (IntakeProjects.IntakeOn), in id order. No membership is read again; no
// project is asked about when he sees none.
func (u *CountIntakeIssues) scope(ctx context.Context, workspaceID, actor uuid.UUID) ([]VisibleProject, error) {
	visible, err := u.d.Projects.Visible(ctx, workspaceID, actor)
	if err != nil || len(visible) == 0 {
		return nil, err
	}
	on, err := u.d.Intakes.IntakeOn(ctx, projectIDs(visible))
	if err != nil {
		return nil, err
	}
	seen := make(map[uuid.UUID]VisibleProject, len(visible))
	for _, p := range visible {
		seen[p.ID] = p
	}
	scope := make([]VisibleProject, 0, len(on))
	for _, id := range on {
		p, ok := seen[id]
		if !ok {
			return nil, fmt.Errorf("project %s, not among those seen, answered as having its intake on", id)
		}
		scope = append(scope, p)
	}
	return scope, nil
}
