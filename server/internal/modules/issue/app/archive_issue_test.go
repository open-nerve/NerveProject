package app_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// cancelled is web's state of the cancelled group, which newArchive adds.
var cancelled = app.StateRef{ID: uuid.NewV7(), Name: "Cancelled", Group: domain.GroupCancelled}

func (f *fakeIssues) SetArchivedAt(ctx context.Context, a app.IssueArchival) error {
	if err := f.log.add(ctx, "SetArchivedAt", a.ID); err != nil {
		return err
	}
	if f.issue == nil || a.ID != f.issue.ID {
		return fmt.Errorf("SetArchivedAt: %+v is not the work item %+v", a, f.issue)
	}
	f.archival, f.archivedAt = &a, a.ArchivedAt
	return nil
}

// archiveOver is archiveIssue and unarchiveIssue over w's locks, work
// items and states as they are when it is called, their clock at
// clockNow: after gated rebuilds w's locks, they are over the gated ones.
func archiveOver(w *world) (*app.ArchiveIssue, *app.UnarchiveIssue) {
	d := app.ArchiveIssueDeps{Locks: w.locks, Issues: w.issues, States: w.project, Clock: clockAt{at: clockNow, log: w.log}}
	return app.NewArchiveIssue(d), app.NewUnarchiveIssue(d)
}

// newArchive is archiveIssue and unarchiveIssue over updateIssue's world
// (newUpdate), web's states with Cancelled, its work item 7 moved into
// Done: made by carol at earlier, not archived.
func newArchive() (*app.ArchiveIssue, *app.UnarchiveIssue, *world, uuid.UUID) {
	_, w, id := newUpdate()
	w.project.states = append(w.project.states, cancelled)
	w.issues.issue.StateID = done.ID
	archive, unarchive := archiveOver(w)
	return archive, unarchive, w, id
}

// stored is newArchive's work item as a read answers it, archived at
// archivedAt, or not for nil, last written by by at at.
func stored(id uuid.UUID, state uuid.UUID, archivedAt *time.Time, by uuid.UUID, at time.Time) domain.Issue {
	return domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: state, Priority: domain.PriorityHigh,
		SortOrder: 65535, ArchivedAt: archivedAt, AssigneeIDs: []uuid.UUID{bob, out}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol,
		UpdatedByID: &by, CreatedAt: earlier, UpdatedAt: at}
}

// An archiving, in the order of M4 design 3.5 and 3.8: Locks' calls on
// issue.archive (lockedAs); the work item's state read under the lock; the
// clock; the work item archived at the moment, by alice; its activity's
// row 14, alice's at the moment; the answer read back, its sub-items
// counted (Locks.seen). A work item in Cancelled is archived as one in
// Done.
func TestArchiveIssueArchivesTheWorkItem(t *testing.T) {
	for _, state := range []app.StateRef{done, cancelled} {
		t.Run(state.Name, func(t *testing.T) {
			uc, _, w, id := newArchive()
			w.issues.issue.StateID = state.ID
			got, err := uc.Execute(as(alice), id)
			if want := stored(id, state.ID, &clockNow, alice, clockNow); err != nil || !reflect.DeepEqual(got, want) {
				t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
			}
			calls := append(lockedAs(domain.ActionArchive, id), line("States", web), "Now", line("SetArchivedAt", id), line("InsertActivities", 1),
				line("GetIssue", id), counted(id))
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if want := (app.IssueArchival{ID: id, ArchivedAt: &clockNow, By: alice, At: clockNow}); w.issues.archival == nil ||
				!reflect.DeepEqual(*w.issues.archival, want) {
				t.Errorf("the archiving written %+v, want %+v", w.issues.archival, want)
			}
			wantActivity(t, w, id, []domain.Activity{domain.ArchiveActivity()})
		})
	}
}

// An unarchiving, as an archiving, without the state, which it does not
// check: the work item, archived in Backlog, unarchived at the moment, by
// alice; its activity's row 16.
func TestUnarchiveIssueUnarchivesTheWorkItem(t *testing.T) {
	_, uc, w, id := newArchive()
	w.issues.issue.StateID, w.issues.archivedAt = backlog.ID, &earlier
	got, err := uc.Execute(as(alice), id)
	if want := stored(id, backlog.ID, nil, alice, clockNow); err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	calls := append(lockedAs(domain.ActionArchive, id), "Now", line("SetArchivedAt", id), line("InsertActivities", 1), line("GetIssue", id),
		counted(id))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if want := (app.IssueArchival{ID: id, By: alice, At: clockNow}); w.issues.archival == nil || !reflect.DeepEqual(*w.issues.archival, want) {
		t.Errorf("the unarchiving written %+v, want %+v", w.issues.archival, want)
	}
	wantActivity(t, w, id, []domain.Activity{domain.UnarchiveActivity()})
}

// Archiving an archived work item, in Backlog too, whose state is not
// read, and unarchiving one that is not archived: the precheck, the lock
// and the decisions run as for any; nothing is written, not even the
// clock read, and the answer is the work item as read under the lock, its
// last writer and moment as they were, its sub-items counted (M4 design
// 3.8, 3.9).
func TestArchivingAsItIsWritesNothing(t *testing.T) {
	for _, tt := range []struct {
		name       string
		archive    bool
		archivedAt *time.Time
	}{{"archiving an archived work item", true, &earlier}, {"unarchiving a work item not archived", false, nil}} {
		t.Run(tt.name, func(t *testing.T) {
			archive, unarchive, w, id := newArchive()
			w.issues.issue.StateID, w.issues.archivedAt = backlog.ID, tt.archivedAt
			execute := unarchive.Execute
			if tt.archive {
				execute = archive.Execute
			}
			got, err := execute(as(alice), id)
			if want := stored(id, backlog.ID, tt.archivedAt, carol, earlier); err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
			}
			if calls := append(lockedAs(domain.ActionArchive, id), counted(id)); !slices.Equal(w.log.calls, calls) || w.issues.archival != nil ||
				w.issues.activities != nil {
				t.Errorf("calls:\n%q\nwant\n%q; written %+v and %+v, want nothing", w.log.calls, calls, w.issues.archival, w.issues.activities)
			}
		})
	}
}

// What refuses an archiving or an unarchiving: an archived project, under
// the lock, before the work item is looked at, an archived one too (409
// project.archived); a work item in Backlog, its state read, when it is
// archived (409 issue.not_archivable); a role that may not, in the
// precheck; a read, a write or a count that fails, its failure as itself;
// a request without a caller, unauthenticated before any read.
func TestArchiveIssueRefusals(t *testing.T) {
	for _, tt := range []struct {
		name    string
		archive bool
		set     func(w *world)
		ctx     context.Context
		want    error
		calls   int // of the archiving's calls (TestArchiveIssueArchivesTheWorkItem), the work item's state read in Done
	}{
		{"an archived project", true, func(w *world) { w.project.locked.Archived = true }, as(alice), domain.ErrProjectArchived, 6},
		{"an archived project, the work item archived", true, func(w *world) {
			w.project.locked.Archived, w.issues.archivedAt = true, &earlier
		}, as(alice), domain.ErrProjectArchived, 6},
		{"an archived project, unarchiving", false, func(w *world) {
			w.project.locked.Archived, w.issues.archivedAt = true, &earlier
		}, as(alice), domain.ErrProjectArchived, 6},
		{"a work item in Backlog", true, func(w *world) { w.issues.issue.StateID = backlog.ID }, as(alice), domain.ErrNotArchivable, 7},
		{"a role that may not", true, func(w *world) { w.auth.answers = []error{shared.Forbidden()} }, as(alice), shared.Forbidden(), 2},
		{"the state's read fails", true, func(w *world) { w.log.fail = "States" }, as(alice), errDisk, 7},
		{"the archiving's write fails", true, func(w *world) { w.log.fail = "SetArchivedAt" }, as(alice), errDisk, 9},
		{"the activity's write fails", true, func(w *world) { w.log.fail = "InsertActivities" }, as(alice), errDisk, 10},
		{"the count fails", true, func(w *world) { w.log.fail = "SubIssueCounts" }, as(alice), errDisk, 12},
		{"no caller", true, func(*world) {}, context.Background(), shared.Unauthenticated(), 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			archive, unarchive, w, id := newArchive()
			tt.set(w)
			execute := unarchive.Execute
			if tt.archive {
				execute = archive.Execute
			}
			got, err := execute(tt.ctx, id)
			var se, wantSE *shared.Error
			if !errors.Is(err, tt.want) && (!errors.As(err, &se) || !errors.As(tt.want, &wantSE) || se.Code != wantSE.Code) ||
				got.ID != (uuid.UUID{}) {
				t.Errorf("Execute() = %+v, %v; want no work item and %v", got, err, tt.want)
			}
			calls := slices.Concat(lockedAs(domain.ActionArchive, id), []string{line("States", web), "Now", line("SetArchivedAt", id),
				line("InsertActivities", 1), line("GetIssue", id), counted(id)})[:tt.calls]
			if !slices.Equal(w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
		})
	}
}

// The states read under the lock answer what the archiving asked: a work
// item in a state that is none of its project's, as the store answers
// them, is an error, not a refusal (409 issue.not_archivable), and nothing
// is written after the read.
func TestArchivingAWorkItemWhoseStateIsNotItsProjectsFails(t *testing.T) {
	archive, _, w, id := newArchive()
	w.issues.issue.StateID = uuid.NewV7()
	_, err := archive.Execute(as(alice), id)
	var se *shared.Error
	if err == nil || errors.As(err, &se) {
		t.Errorf("Execute() = %v; want an error that is no refusal", err)
	}
	if calls := slices.Concat(lockedAs(domain.ActionArchive, id), []string{line("States", web)}); !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
}
