package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newArchive is ArchiveProject, or UnarchiveProject when archive is false,
// over newWrites' fakes, its clock logged.
func newArchive(archive bool) (*app.ArchiveProject, *writeFixture) {
	f := newWrites()
	d := app.ArchiveProjectDeps{Projects: f.store, Locks: f.locks(), Tx: f.tx, Clock: clockAt{clockNow, f.log}, Caller: imageCaller{f.log}, Images: noProjectImages{},
		Events: f.events}
	if archive {
		return app.NewArchiveProject(d), f
	}
	return app.NewUnarchiveProject(d), f
}

// archived are the calls of bob's archive of project, or unarchive: the
// locks, the decision, the clock, the change, the archive's event (an
// archive's alone), the answer.
func archived(project uuid.UUID, archive bool) []string {
	action := domain.ActionUnarchive
	if archive {
		action = domain.ActionArchive
	}
	calls := append(withImageCaller(bob, lockedDecision(bob, project, action)), "Now",
		fmt.Sprintf("SetArchived %s %v by %s at %s", project, archive, bob, clockNow.Format(timeFormat)))
	if archive {
		calls = append(calls, fmt.Sprintf("Publish project.archived %s by %s at %s", project, bob, clockNow.Format(timeFormat)))
	}
	return append(calls, fmt.Sprintf("GetProject %s for %s", project, bob))
}

// ArchiveProject, in one transaction and in the order of M3 design 3.6,
// takes the project's locks, decides, reads the clock and archives it, by the
// caller, then publishes the archive at that moment (M7 P3c spec 2.6, R3);
// UnarchiveProject the same, and clears the time, and publishes nothing. An
// archived project archives again, and an unarchived one unarchives: each is
// the same change, as Plane's, so an archive again publishes again. The
// answer is the project read back as stored.
func TestArchiveProject(t *testing.T) {
	tests := []struct {
		name     string
		project  uuid.UUID
		archive  bool
		archived bool
	}{
		{"archive web", webID, true, true},
		{"archive ops, archived already", opsID, true, true},
		{"unarchive ops", opsID, false, false},
		{"unarchive web, not archived", webID, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc, f := newArchive(tt.archive)

			got, err := uc.Execute(as(bob), tt.project)

			if want := archived(tt.project, tt.archive); err != nil || !slices.Equal(f.log.calls, want) {
				t.Errorf("Execute() = %v, calls\n%q\nwant\n%q", err, f.log.calls, want)
			}
			if got.ID != tt.project || (got.ArchivedAt != nil) != tt.archived || (tt.archived && !got.ArchivedAt.Equal(now)) {
				t.Errorf("Execute() = %+v; want the project, archived %v at %v", got, tt.archived, now)
			}
			if !tt.archive {
				if len(f.events.events) != 0 {
					t.Errorf("published %+v; an unarchive publishes nothing", f.events.events)
				}
				return
			}
			wantProjectArchive(t, f.events.events, tt.project, clockNow)
		})
	}
}

// Refusals, each in its place, and nothing changed: no caller, before the
// transaction; a project not there, or not visible: 404; a project member:
// the Authorizer's 403.
func TestArchiveProjectRefuses(t *testing.T) {
	for _, archive := range []bool{true, false} {
		action := domain.ActionUnarchive
		if archive {
			action = domain.ActionArchive
		}
		tests := []struct {
			name    string
			ctx     context.Context
			project uuid.UUID
			want    error
			calls   []string
		}{
			{"no caller", context.Background(), webID, shared.Unauthenticated(), nil},
			{"no project", as(bob), uuid.Nil(), domain.ErrNotFound, withImageCaller(bob, noProject)},
			{"not seen", as(erin), webID, domain.ErrNotFound, withImageCaller(erin, lockedDecision(erin, webID, action))},
			{"a project member", as(alice), webID, shared.Forbidden(), withImageCaller(alice, lockedDecision(alice, webID, action))},
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("%s %s", action, tt.name), func(t *testing.T) {
				uc, f := newArchive(archive)

				got, err := uc.Execute(tt.ctx, tt.project)

				if !sameError(err, tt.want) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, tt.calls) || len(f.events.events) != 0 {
					t.Errorf("Execute() = %+v, %v, calls %q, published %+v; want %v, calls %q", got, err, f.log.calls, f.events.events, tt.want, tt.calls)
				}
			})
		}
	}
}

// Every port's failure comes back as itself, the commit's too, after the
// calls before it and none after; a project the store cannot read back
// after the change is an internal error, not a 404. An archive's event is
// one more call, after the change: its failure fails the archive.
func TestArchiveProjectReturnsEachFailure(t *testing.T) {
	failure := errors.New("disk full")
	type failing struct {
		name  string
		fail  func(f *writeFixture)
		calls int // how many of all's ran
	}
	for _, archive := range []bool{true, false} {
		all := archived(webID, archive)
		event := 0
		if archive {
			event = 1
		}
		tests := []failing{
			{"the lock", func(f *writeFixture) { f.store.errs = map[string]error{"LockProject": failure} }, 4},
			{"the decision", func(f *writeFixture) { f.auth.errs = map[grantKey]error{{bob, acme.ID}: failure} }, 5},
			{"the change", func(f *writeFixture) { f.store.errs = map[string]error{"SetArchived": failure} }, 7},
			{"the answer", func(f *writeFixture) { f.store.errs = map[string]error{"GetProject": failure} }, 8 + event},
			{"the commit", func(f *writeFixture) { f.tx.commitErr = failure }, 8 + event},
		}
		if archive {
			tests = append(tests, failing{"the event", func(f *writeFixture) { f.events.err = failure }, 8})
		}
		for _, tt := range tests {
			t.Run(fmt.Sprintf("archive %v, %s", archive, tt.name), func(t *testing.T) {
				uc, f := newArchive(archive)
				tt.fail(f)
				got, err := uc.Execute(as(bob), webID)
				if !errors.Is(err, failure) || got.ID != (uuid.UUID{}) || !slices.Equal(f.log.calls, all[:tt.calls+2]) {
					t.Errorf("Execute() = %+v, %v, calls %q; want %v after %q", got, err, f.log.calls, failure, all[:tt.calls+2])
				}
			})
		}
		uc, f := newArchive(archive)
		f.store.missing = true
		var e *shared.Error
		if got, err := uc.Execute(as(bob), webID); err == nil || errors.As(err, &e) || got.ID != (uuid.UUID{}) {
			t.Errorf("Execute() with the project gone = %+v, %v; want an internal error", got, err)
		}
	}
}
