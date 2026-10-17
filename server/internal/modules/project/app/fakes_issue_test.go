package app_test

import (
	"context"
	"fmt"
	"time"
	"uuid"
)

// fakeIssues is the issue module's cascade (app.IssueCascade): it logs each
// call with its arguments, and " outside tx" when it ran outside the
// caller's transaction (callLog), and fails the method errs names, wrapped
// as the module wraps it.
type fakeIssues struct {
	log  *callLog
	errs map[string]error
}

func (f *fakeIssues) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeIssues) DeleteProjectIssues(ctx context.Context, projectID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DeleteProjectIssues", "%s by %s at %s", projectID, by, now.Format(timeFormat))
}

func (f *fakeIssues) DeleteWorkspaceIssues(ctx context.Context, projectIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DeleteWorkspaceIssues", "%v by %s at %s", projectIDs, by, now.Format(timeFormat))
}
