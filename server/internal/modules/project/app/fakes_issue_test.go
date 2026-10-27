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
// as the module wraps it. The states in inUse have work items.
type fakeIssues struct {
	log   *callLog
	errs  map[string]error
	inUse map[uuid.UUID]bool
}

func (f *fakeIssues) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if err := f.errs[name]; err != nil {
		return fmt.Errorf("%s: %w", name, err)
	}
	return nil
}

func (f *fakeIssues) StateInUse(ctx context.Context, stateID uuid.UUID) (bool, error) {
	if err := f.call(ctx, "StateInUse", "%s", stateID); err != nil {
		return false, err
	}
	return f.inUse[stateID], nil
}

func (f *fakeIssues) DetachLabel(ctx context.Context, labelIDs []uuid.UUID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DetachLabel", "%v by %s at %s", labelIDs, by, now.Format(timeFormat))
}
