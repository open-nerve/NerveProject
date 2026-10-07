package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newDelete is DeleteProject over newWrites' fakes, its clock logged.
func newDelete() (*app.DeleteProject, *writeFixture) {
	f := newWrites()
	return app.NewDeleteProject(f.store, f.locks(), f.tx, clockAt{clockNow, f.log}), f
}

// deleted are the calls of bob's deletion of project: the locks, the
// decision, the clock, the five steps on that project alone.
func deleted(project uuid.UUID) []string {
	return slices.Concat(lockedDecision(bob, project, domain.ActionDelete), []string{"Now"},
		deletionSteps(acme.ID, project.String(), bob, clockNow))
}

// DeleteProject, in one transaction and in the order of M3 design 3.6,
// takes the project's locks, decides, reads the clock, then runs the steps a
// workspace's deletion runs, on the project alone, of the workspace its
// lock read, by the caller at that one moment. An archived project is
// deleted as any other.
func TestDeleteProject(t *testing.T) {
	for _, project := range []uuid.UUID{webID, opsID} {
		uc, f := newDelete()
		if err := uc.Execute(as(bob), project); err != nil || !slices.Equal(f.log.calls, deleted(project)) {
			t.Errorf("Execute(%s) = %v, calls\n%q\nwant\n%q", project, err, f.log.calls, deleted(project))
		}
	}
}

// Refusals, each in its place, and nothing deleted: no caller, before the
// transaction; a project not there, or not visible: 404; a project member:
// the Authorizer's 403.
func TestDeleteProjectRefuses(t *testing.T) {
	tests := []struct {
		name  string
		ctx   context.Context
		id    uuid.UUID
		want  error
		calls []string
	}{
		{"no caller", context.Background(), webID, shared.Unauthenticated(), nil},
		{"no project", as(bob), uuid.Nil(), domain.ErrNotFound, noProject},
		{"not seen", as(erin), webID, domain.ErrNotFound, lockedDecision(erin, webID, domain.ActionDelete)},
		{"a project member", as(alice), webID, shared.Forbidden(), lockedDecision(alice, webID, domain.ActionDelete)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newDelete()
			if err := uc.Execute(tt.ctx, tt.id); !sameError(err, tt.want) || !slices.Equal(f.log.calls, tt.calls) {
				t.Errorf("Execute() = %v, calls %q; want %v, calls %q", err, f.log.calls, tt.want, tt.calls)
			}
		})
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after.
func TestDeleteProjectReturnsEachFailure(t *testing.T) {
	all := deleted(webID)
	tests := []struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of all's ran
	}{
		{"the lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": errDisk} }, 4},
		{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: errDisk} }, 5},
		{"the projects", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteProjects": errDisk} }, 7},
		{"the members", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteProjectMembers": errDisk} }, 8},
		{"the display settings", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteProjectPreferences": errDisk} }, 9},
		{"the states", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteStates": errDisk} }, 10},
		{"the labels", func(f *writeFixture) { f.store.errs = map[string]error{"DeleteLabels": errDisk} }, 11},
		{"the commit", func(f *writeFixture) { f.tx.commitErr = errDisk }, 11},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newDelete()
			tt.fail(f)
			if err := uc.Execute(as(bob), webID); !errors.Is(err, errDisk) || !slices.Equal(f.log.calls, all[:tt.calls]) {
				t.Errorf("Execute() = %v, calls %q; want %v after %q", err, f.log.calls, errDisk, all[:tt.calls])
			}
		})
	}
}
