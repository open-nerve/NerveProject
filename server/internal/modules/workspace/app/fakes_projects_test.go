package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

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
