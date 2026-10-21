package app_test

import (
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// october is the day d of October 2026, a work item's date.
func october(d int) *shared.Date {
	day := shared.DateOf(time.Date(2026, time.October, d, 0, 0, 0, 0, time.UTC))
	return &day
}

// An update writes the dates the work item then has (M4 design 3.7, 3.13):
// each date given, the other kept; each date it changes is a row of its
// activity, from and to YYYY-MM-DD or "" for none (rows 8, 9), alice's at
// the moment, after the clock and the work item's fields; the answer is the
// work item read back, its sub-items counted. Web's work item 7 runs from
// the 12th to the 19th: both dates moved past the target date it had, which
// a start date given with the target date is not held to; the start date
// cleared, and a target date before the start date it had; a start date
// alone, on the target date it keeps; the target date cleared.
func TestUpdateIssueWritesTheDates(t *testing.T) {
	for _, tt := range []struct {
		name          string
		p             domain.IssuePatch
		start, target *shared.Date
		rows          domain.Update
	}{
		{"both past the target date", domain.IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(25)},
			october(20), october(25), domain.Update{StartDate: &domain.Changed[*shared.Date]{Old: october(12), New: october(20)},
				TargetDate: &domain.Changed[*shared.Date]{Old: october(19), New: october(25)}}},
		{"the start date cleared, the target date before it", domain.IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: october(10)},
			nil, october(10), domain.Update{StartDate: &domain.Changed[*shared.Date]{Old: october(12)},
				TargetDate: &domain.Changed[*shared.Date]{Old: october(19), New: october(10)}}},
		{"a start date alone, on the target date", domain.IssuePatch{SetStartDate: true, StartDate: october(19)}, october(19), october(19),
			domain.Update{StartDate: &domain.Changed[*shared.Date]{Old: october(12), New: october(19)}}},
		{"the target date cleared", domain.IssuePatch{SetTargetDate: true}, october(12), nil,
			domain.Update{TargetDate: &domain.Changed[*shared.Date]{Old: october(19)}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.issues.issue.StartDate, w.issues.issue.TargetDate = october(12), october(19)
			got, err := uc.Execute(as(alice), id, tt.p)
			if err != nil || !reflect.DeepEqual(got.StartDate, tt.start) || !reflect.DeepEqual(got.TargetDate, tt.target) {
				t.Fatalf("Execute() = %+v, %v; want it from %v to %v", got, err, tt.start, tt.target)
			}
			rows := domain.UpdateActivity(tt.rows)
			if calls := append(locked(id), "Now", line("UpdateIssue", id), line("InsertActivities", len(rows)), line("GetIssue", id),
				counted(id)); !slices.Equal(
				w.log.calls, calls) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
			}
			if fields := (app.IssueFields{ID: id, Name: "Fix", StateID: backlog.ID, Priority: domain.PriorityHigh, StartDate: tt.start,
				TargetDate: tt.target, By: alice, At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, fields) {
				t.Errorf("the fields written %+v, want %+v", w.issues.changed, fields)
			}
			wantActivity(t, w, id, rows)
		})
	}
}

// The dates an update leaves the work item with are held in order under the
// lock, merged with those it keeps (M4 design 3.7): of web's work item 7,
// from the 12th to the 19th, a start date alone after the target date kept
// is refused at start_date; a target date alone before the start date kept
// at target_date; both given out of order at target_date, though each is
// in order with the date it had. The dates come after the dependent rows'
// problems, in the one 422, and nothing is written.
func TestUpdateIssueHoldsTheMergedDatesInOrder(t *testing.T) {
	other := uuid.NewV7() // after every id of the world
	start := shared.FieldError{Field: "start_date", Code: "not_allowed", Message: "must not be after the target date"}
	target := shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	for _, tt := range []struct {
		name  string
		p     domain.IssuePatch
		want  []shared.FieldError
		calls []string // after locked's
	}{
		{"a start date alone after the target date", domain.IssuePatch{SetStartDate: true, StartDate: october(20)}, []shared.FieldError{start}, nil},
		{"a target date alone before the start date", domain.IssuePatch{SetTargetDate: true, TargetDate: october(11)}, []shared.FieldError{target},
			nil},
		{"both out of order, each in order with the date it had",
			domain.IssuePatch{SetStartDate: true, StartDate: october(15), SetTargetDate: true, TargetDate: october(14)}, []shared.FieldError{target},
			nil},
		{"a label not the project's, and a start date alone after the target date",
			domain.IssuePatch{LabelIDs: set(bug, other), SetStartDate: true, StartDate: october(20)},
			[]shared.FieldError{domain.UnknownLabel(1), start}, []string{line("Labels", web, []uuid.UUID{other})}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.issues.issue.StartDate, w.issues.issue.TargetDate = october(12), october(19)
			_, err := uc.Execute(as(alice), id, tt.p)
			unchanged(t, w, err, shared.Invalid(tt.want...), append(locked(id), tt.calls...))
		})
	}
}
