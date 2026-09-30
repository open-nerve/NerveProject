package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
)

// fakeDeleter is the projects' repository: it records each step with its
// arguments, and fails the step named in fail.
type fakeDeleter struct {
	calls []string
	fail  string
}

var errDisk = errors.New("disk full")

func (f *fakeDeleter) step(name string, workspaceID, by uuid.UUID, now time.Time) error {
	f.calls = append(f.calls, fmt.Sprintf("%s %s by %s at %s", name, workspaceID, by, now.Format(time.RFC3339Nano)))
	if name == f.fail {
		return errDisk
	}
	return nil
}

func (f *fakeDeleter) DeleteWorkspaceProjects(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceProjects", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceProjectMembers(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceProjectMembers", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceProjectPreferences(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceProjectPreferences", workspaceID, by, now)
}

func (f *fakeDeleter) DeleteWorkspaceStates(_ context.Context, workspaceID, by uuid.UUID, now time.Time) error {
	return f.step("DeleteWorkspaceStates", workspaceID, by, now)
}

// DeleteWorkspaceProjects runs the four steps in the order of M3 design
// 3.6, each with the caller's workspace, account and moment; a failing step
// comes back as itself and the steps after it do not run.
func TestDeleteWorkspaceProjects(t *testing.T) {
	workspace, by := uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
	var steps []string
	for _, name := range []string{"DeleteWorkspaceProjects", "DeleteWorkspaceProjectMembers", "DeleteWorkspaceProjectPreferences",
		"DeleteWorkspaceStates"} {
		steps = append(steps, fmt.Sprintf("%s %s by %s at %s", name, workspace, by, now.Format(time.RFC3339Nano)))
	}
	tests := []struct {
		fail    string
		wantErr error
		want    []string
	}{
		{"", nil, steps},
		{"DeleteWorkspaceProjects", errDisk, steps[:1]},
		{"DeleteWorkspaceProjectMembers", errDisk, steps[:2]},
		{"DeleteWorkspaceProjectPreferences", errDisk, steps[:3]},
		{"DeleteWorkspaceStates", errDisk, steps},
	}
	for _, tt := range tests {
		f := &fakeDeleter{fail: tt.fail}
		err := app.NewCascade(f).DeleteWorkspaceProjects(context.Background(), workspace, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.calls, tt.want) {
			t.Errorf("failing %q: DeleteWorkspaceProjects() = %v, the steps\n%q\nwant %v,\n%q", tt.fail, err, f.calls, tt.wantErr, tt.want)
		}
	}
}
