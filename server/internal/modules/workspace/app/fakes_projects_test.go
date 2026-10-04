package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// projectSoleAdmin is the project module's ErrSoleAdmin, which this module
// does not import: its kind, code and detail, the cascade's refusal of the
// only active admin of a project with other active members.
var projectSoleAdmin = shared.NewError(shared.KindConflict, "project.sole_admin",
	"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
		"other active members. It must first be given another admin, or be deleted.")

// fakeProjects is the project module's cascade (app.ProjectCascade): it
// logs each call with its arguments, and fails a method with the error set
// for it, wrapped as the module wraps it.
type fakeProjects struct {
	log  *callLog
	errs map[string]error // by method, e.g. "DeleteWorkspaceProjects"
}

func (f *fakeProjects) DeleteWorkspaceProjects(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DeleteWorkspaceProjects %s by %s at %s", workspaceID, by, now.Format(time.RFC3339Nano))
	if err := f.errs["DeleteWorkspaceProjects"]; err != nil {
		return fmt.Errorf("delete the workspace's projects: %w", err)
	}
	return nil
}

func (f *fakeProjects) DemoteToGuest(ctx context.Context, workspaceID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "DemoteToGuest %s %s by %s at %s", workspaceID, userID, by, now.Format(time.RFC3339Nano))
	if err := f.errs["DemoteToGuest"]; err != nil {
		return fmt.Errorf("demote the member's project memberships: %w", err)
	}
	return nil
}

func (f *fakeProjects) EndMemberships(ctx context.Context, workspaceIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	f.log.add(ctx, "EndMemberships %v %s by %s at %s", workspaceIDs, userID, by, now.Format(time.RFC3339Nano))
	if err := f.errs["EndMemberships"]; err != nil {
		return fmt.Errorf("end the member's project memberships: %w", err)
	}
	return nil
}

// fakeCounts is the project module's ProjectMembershipCounts: it logs each
// call and answers the count it holds, or fails with its error, wrapped as
// the module wraps it.
type fakeCounts struct {
	log *callLog
	n   int
	err error
}

func (f *fakeCounts) CountInactive(ctx context.Context, workspaceID, userID uuid.UUID) (int, error) {
	f.log.add(ctx, "CountInactive %s %s", workspaceID, userID)
	if f.err != nil {
		return 0, fmt.Errorf("count the ended project memberships: %w", f.err)
	}
	return f.n, nil
}
