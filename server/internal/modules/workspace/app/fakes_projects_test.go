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
