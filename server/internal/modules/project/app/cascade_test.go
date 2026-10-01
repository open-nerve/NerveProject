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

var errDisk = errors.New("disk full")

// deletionSteps are the steps of a deletion of project of workspace, "*"
// for every project of it, by by at at, in the order of M3 design 3.6.
func deletionSteps(workspace uuid.UUID, project string, by uuid.UUID, at time.Time) []string {
	var steps []string
	for _, name := range []string{"DeleteProjects", "DeleteProjectMembers", "DeleteProjectPreferences", "DeleteStates"} {
		steps = append(steps, fmt.Sprintf("%s %s/%s by %s at %s", name, workspace, project, by, at.Format(timeFormat)))
	}
	return steps
}

// DeleteWorkspaceProjects runs the four steps in the order of M3 design
// 3.6, each on every project of the caller's workspace, with the caller's
// account and moment, in the caller's transaction; a failing step comes
// back as itself and the steps after it do not run.
func TestDeleteWorkspaceProjects(t *testing.T) {
	workspace, by := uuid.NewV7(), uuid.NewV7()
	steps := deletionSteps(workspace, "*", by, clockNow)
	tests := []struct {
		fail    string
		wantErr error
		want    []string
	}{
		{"", nil, steps},
		{"DeleteProjects", errDisk, steps[:1]},
		{"DeleteProjectMembers", errDisk, steps[:2]},
		{"DeleteProjectPreferences", errDisk, steps[:3]},
		{"DeleteStates", errDisk, steps},
	}
	for _, tt := range tests {
		f := newWrites()
		f.store.errs = map[string]error{tt.fail: errDisk}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		err := app.NewCascade(f.store, &fakeDemoter{}).DeleteWorkspaceProjects(inTx, workspace, by, clockNow)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("failing %q: DeleteWorkspaceProjects() = %v, the steps\n%q\nwant %v,\n%q", tt.fail, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

// fakeDemoter is the memberships' repository: it records each call with its
// arguments, and " outside tx" when it ran outside the caller's
// transaction (callLog), answers LockMemberProjects with locked, and fails
// the call named in fail.
type fakeDemoter struct {
	locked []uuid.UUID
	log    callLog
	fail   string
}

func (f *fakeDemoter) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if name == f.fail {
		return errDisk
	}
	return nil
}

func (f *fakeDemoter) LockMemberProjects(ctx context.Context, workspaceID, userID uuid.UUID) ([]uuid.UUID, error) {
	if err := f.call(ctx, "LockMemberProjects", "%s %s", workspaceID, userID); err != nil {
		return nil, err
	}
	return f.locked, nil
}

func (f *fakeDemoter) DemoteMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "DemoteMemberships", "%v %s by %s at %s", projectIDs, userID, by, now.Format(time.RFC3339Nano))
}

// DemoteToGuest locks the account's projects in the workspace, then
// demotes his memberships of the ones locked, by the caller's account at
// the caller's moment, both in the caller's transaction; with none locked
// it writes nothing. A failing call comes back as itself, and nothing runs
// after it.
func TestDemoteToGuest(t *testing.T) {
	workspace, user, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 1, 10, 0, 0, 123456000, time.UTC)
	projects := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	lock := fmt.Sprintf("LockMemberProjects %s %s", workspace, user)
	demote := fmt.Sprintf("DemoteMemberships %v %s by %s at %s", projects, user, by, now.Format(time.RFC3339Nano))
	tests := []struct {
		name    string
		locked  []uuid.UUID
		fail    string
		wantErr error
		want    []string
	}{
		{"two projects", projects, "", nil, []string{lock, demote}},
		{"none", nil, "", nil, []string{lock}},
		{"the lock failing", projects, "LockMemberProjects", errDisk, []string{lock}},
		{"the write failing", projects, "DemoteMemberships", errDisk, []string{lock, demote}},
	}
	for _, tt := range tests {
		f := &fakeDemoter{locked: tt.locked, fail: tt.fail}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		err := app.NewCascade(newWrites().store, f).DemoteToGuest(inTx, workspace, user, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("%s: DemoteToGuest() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}
