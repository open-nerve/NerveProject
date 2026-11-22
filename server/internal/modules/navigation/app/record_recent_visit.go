package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// RecordRecentVisitDeps are what the recording of one's recent visits
// needs: its locks and decision, the visits' writes and the clock.
type RecordRecentVisitDeps struct {
	Locks  RecentVisitLocks
	Visits RecentVisitWriter
	Clock  Clock
}

// RecordRecentVisit records one's visit of a work item or a project
// (recordRecentVisit).
type RecordRecentVisit struct{ d RecordRecentVisitDeps }

// NewRecordRecentVisit returns the use case over d.
func NewRecordRecentVisit(d RecordRecentVisitDeps) *RecordRecentVisit {
	return &RecordRecentVisit{d: d}
}

// Execute records the caller's visit of the target of kind with id, in the
// target's workspace (P3d spec 2.4, 2.5, 2.7, 2.8, R7). Its refusals, in
// order: a kind not one of the two, a 422 before any read
// (domain.CheckRecentVisitKind); then, before any lock
// (RecentVisitLocks.recording), and again under its locks
// (RecentVisitLocks.run), the target's own 404, issue.not_found or
// project.not_found, the one value, for a target not there, deleted, of a
// workspace he is not a member of, or one he does not see: a project he
// does not read, a work item the guest rule hides from him. An archived
// target, and a work item in its project's intake, he records (R5, R6).
// Under its last lock, the advisory one, it reads the moment once and
// writes his visit of the target at it, made and last changed by him: a new
// one in the target's workspace and project, or his undeleted visit of it
// revisited, its id and creation kept; then it deletes, for good, his
// undeleted visits in the workspace past the cap (domain.RecentVisitCap)
// by visit, the newest first, whether he sees their targets or not.
func (u *RecordRecentVisit) Execute(ctx context.Context, kind domain.RecentVisitKind, id uuid.UUID) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	if err := domain.CheckRecentVisitKind(kind); err != nil {
		return err
	}
	w, err := u.d.Locks.recording(ctx, actor, targetOf(kind, id))
	if err != nil {
		return err
	}
	return u.d.Locks.run(ctx, actor, w, func(ctx context.Context) error {
		now := u.d.Clock.Now()
		v := domain.RecentVisit{ID: uuid.NewV7(), WorkspaceID: w.place.workspaceID, ProjectID: w.place.projectID, UserID: actor.UserID, Kind: kind,
			EntityID: id, VisitedAt: now}
		if err := u.d.Visits.RecordRecentVisit(ctx, v, actor.UserID); err != nil {
			return err
		}
		return u.d.Visits.TrimRecentVisits(ctx, actor.UserID, w.place.workspaceID, domain.RecentVisitCap)
	})
}
