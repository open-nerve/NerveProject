package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

var errDisk = errors.New("disk full")

// deletionSteps are the steps of a deletion of project of workspace, "*"
// for every project of it, by by at at, in the order of M3 design 3.6.
func deletionSteps(workspace uuid.UUID, project string, by uuid.UUID, at time.Time) []string {
	var steps []string
	for _, name := range []string{"DeleteProjects", "DeleteProjectMembers", "DeleteProjectPreferences", "DeleteStates", "DeleteLabels"} {
		steps = append(steps, fmt.Sprintf("%s %s/%s by %s at %s", name, workspace, project, by, at.Format(timeFormat)))
	}
	return steps
}

// DeleteWorkspaceProjects runs the five steps in the order of M3 design
// 3.6, each on every project of the caller's workspace, then deletes the
// work-item side of the projects the caller locked through the issue
// module (M4 design 3.5), with the caller's account and moment, in the
// caller's transaction; a failing step comes back as itself and the steps
// after it do not run. Each step fails in turn.
func TestDeleteWorkspaceProjects(t *testing.T) {
	workspace, by := uuid.NewV7(), uuid.NewV7()
	projects := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	steps := append(deletionSteps(workspace, "*", by, clockNow),
		fmt.Sprintf("DeleteWorkspaceIssues %v by %s at %s", projects, by, clockNow.Format(timeFormat)))
	type run struct {
		fail    string
		wantErr error
		want    []string
	}
	tests := []run{{"", nil, steps}}
	for i, step := range steps {
		name, _, _ := strings.Cut(step, " ")
		tests = append(tests, run{name, errDisk, steps[:i+1]})
	}
	for _, tt := range tests {
		f := newWrites()
		f.store.errs, f.issues.errs = map[string]error{tt.fail: errDisk}, map[string]error{tt.fail: errDisk}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		err := app.NewCascade(f.store, &fakeLocker{}, &fakeDemoter{}, &fakeEnder{}, f.issues).DeleteWorkspaceProjects(inTx, workspace, projects, by, clockNow)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("failing %q: DeleteWorkspaceProjects() = %v, the steps\n%q\nwant %v,\n%q", tt.fail, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

// fakeLocker is the projects' lock of a workspace's deletion: it records
// each call with its arguments, and " outside tx" when it ran outside the
// caller's transaction (callLog), answers with locked, and fails with
// errDisk when fail is set.
type fakeLocker struct {
	locked []uuid.UUID
	log    callLog
	fail   bool
}

func (f *fakeLocker) LockWorkspaceProjects(ctx context.Context, workspaceID uuid.UUID) ([]uuid.UUID, error) {
	f.log.add(ctx, "LockWorkspaceProjects %s", workspaceID)
	if f.fail {
		return nil, errDisk
	}
	return f.locked, nil
}

// LockWorkspaceProjects locks the workspace's projects in the caller's
// transaction and answers the ids as the lock returned them; a failing
// lock comes back as itself.
func TestLockWorkspaceProjects(t *testing.T) {
	workspace := uuid.NewV7()
	projects := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	lock := []string{fmt.Sprintf("LockWorkspaceProjects %s", workspace)}
	tests := []struct {
		name    string
		locked  []uuid.UUID
		fail    bool
		wantErr error
		want    []uuid.UUID
	}{
		{"two projects", projects, false, nil, projects},
		{"none", nil, false, nil, nil},
		{"the lock failing", projects, true, errDisk, nil},
	}
	for _, tt := range tests {
		f := &fakeLocker{locked: tt.locked, fail: tt.fail}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		got, err := app.NewCascade(newWrites().store, f, &fakeDemoter{}, &fakeEnder{}, &fakeIssues{}).LockWorkspaceProjects(inTx, workspace)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(got, tt.want) || !slices.Equal(f.log.calls, lock) {
			t.Errorf("%s: LockWorkspaceProjects() = %v, %v, the calls %q; want %v, %v, %q", tt.name, got, err, f.log.calls, tt.want, tt.wantErr, lock)
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
		err := app.NewCascade(newWrites().store, &fakeLocker{}, f, &fakeEnder{}, &fakeIssues{}).DemoteToGuest(inTx, workspace, user, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("%s: DemoteToGuest() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}

// fakeEnder is the memberships' repository of an ending: it records each
// call with its arguments, and " outside tx" when it ran outside the
// caller's transaction (callLog), answers LockActiveMemberProjects with
// locked and SoleAdmin with sole, and fails the call named in fail.
type fakeEnder struct {
	locked []uuid.UUID
	sole   bool
	log    callLog
	fail   string
}

func (f *fakeEnder) call(ctx context.Context, name, format string, args ...any) error {
	f.log.add(ctx, name+" "+format, args...)
	if name == f.fail {
		return errDisk
	}
	return nil
}

func (f *fakeEnder) LockActiveMemberProjects(ctx context.Context, workspaceIDs []uuid.UUID, userID uuid.UUID) ([]uuid.UUID, error) {
	if err := f.call(ctx, "LockActiveMemberProjects", "%v %s", workspaceIDs, userID); err != nil {
		return nil, err
	}
	return f.locked, nil
}

func (f *fakeEnder) SoleAdmin(ctx context.Context, projectIDs []uuid.UUID, userID uuid.UUID) (bool, error) {
	if err := f.call(ctx, "SoleAdmin", "%v %s", projectIDs, userID); err != nil {
		return false, err
	}
	return f.sole, nil
}

func (f *fakeEnder) EndMemberships(ctx context.Context, projectIDs []uuid.UUID, userID, by uuid.UUID, now time.Time) error {
	return f.call(ctx, "EndMemberships", "%v %s by %s at %s", projectIDs, userID, by, now.Format(time.RFC3339Nano))
}

// EndMemberships locks the account's projects with an active membership in
// the workspaces, found when it is called (M3 design 3.6 convention 6);
// asks whether he is the only admin of one of those it locked that has
// other members; then ends his memberships of them, by the caller's account
// at the caller's moment; all in the caller's transaction. The projects
// pass on as the lock returned them, in an order that is no sort's. Were he
// the only admin, project.sole_admin, and nothing is written (3.7 rule 2);
// with none locked it neither asks nor writes. A failing call comes back as
// itself, and nothing runs after it.
func TestEndMemberships(t *testing.T) {
	workspaces := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}
	user, by := uuid.NewV7(), uuid.NewV7()
	now := time.Date(2026, 10, 2, 10, 0, 0, 123456000, time.UTC)
	first, second, third := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	projects := []uuid.UUID{second, first, third}
	lock := fmt.Sprintf("LockActiveMemberProjects %v %s", workspaces, user)
	ask := fmt.Sprintf("SoleAdmin %v %s", projects, user)
	end := fmt.Sprintf("EndMemberships %v %s by %s at %s", projects, user, by, now.Format(time.RFC3339Nano))
	tests := []struct {
		name    string
		locked  []uuid.UUID
		sole    bool
		fail    string
		wantErr error
		want    []string
	}{
		{"three projects", projects, false, "", nil, []string{lock, ask, end}},
		{"none", nil, false, "", nil, []string{lock}},
		{"the only admin of one", projects, true, "", domain.ErrSoleAdmin, []string{lock, ask}},
		{"the lock failing", projects, false, "LockActiveMemberProjects", errDisk, []string{lock}},
		{"the question failing", projects, false, "SoleAdmin", errDisk, []string{lock, ask}},
		{"the write failing", projects, false, "EndMemberships", errDisk, []string{lock, ask, end}},
	}
	for _, tt := range tests {
		f := &fakeEnder{locked: tt.locked, sole: tt.sole, fail: tt.fail}
		inTx := context.WithValue(context.Background(), inTxKey{}, true)
		err := app.NewCascade(newWrites().store, &fakeLocker{}, &fakeDemoter{}, f, &fakeIssues{}).EndMemberships(inTx, workspaces, user, by, now)
		if !errors.Is(err, tt.wantErr) || (tt.wantErr == nil) != (err == nil) || !slices.Equal(f.log.calls, tt.want) {
			t.Errorf("%s: EndMemberships() = %v, the calls\n%q\nwant %v,\n%q", tt.name, err, f.log.calls, tt.wantErr, tt.want)
		}
	}
}
