package domain

import (
	"reflect"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// october is the day d of October 2026, a work item's date.
func october(d int) *shared.Date {
	day := shared.DateOf(time.Date(2026, time.October, d, 0, 0, 0, 0, time.UTC))
	return &day
}

// The dates a work item has once a patch changes it are held in order (M4
// design 3.7): each date given, else the work item's. A start date given
// alone is held to the target date the work item keeps, and refused at
// start_date; given with a target date, to that date, not to the one it
// had, and refused at target_date, as a target date given alone is. A date
// cleared, or missing, orders nothing; the same day for both is in order.
func TestDatesOutOfOrder(t *testing.T) {
	start := &shared.FieldError{Field: "start_date", Code: "not_allowed", Message: "must not be after the target date"}
	target := &shared.FieldError{Field: "target_date", Code: "not_allowed", Message: "must not be before the start date"}
	from12To19 := Issue{StartDate: october(12), TargetDate: october(19)}
	tests := []struct {
		name string
		i    Issue
		p    IssuePatch
		want *shared.FieldError
	}{
		{"nothing given", from12To19, IssuePatch{}, nil},
		{"a start date alone, on the target date kept", from12To19, IssuePatch{SetStartDate: true, StartDate: october(19)}, nil},
		{"a start date alone, after the target date kept", from12To19, IssuePatch{SetStartDate: true, StartDate: october(20)}, start},
		{"a target date alone, on the start date kept", from12To19, IssuePatch{SetTargetDate: true, TargetDate: october(12)}, nil},
		{"a target date alone, before the start date kept", from12To19, IssuePatch{SetTargetDate: true, TargetDate: october(11)}, target},
		{"both in order, past the target date it had", from12To19,
			IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(25)}, nil},
		{"both out of order, each in order with the date it had", from12To19,
			IssuePatch{SetStartDate: true, StartDate: october(15), SetTargetDate: true, TargetDate: october(14)}, target},
		{"a start date after the target date cleared", from12To19, IssuePatch{SetStartDate: true, StartDate: october(30), SetTargetDate: true}, nil},
		{"the start date cleared, a target date before it", from12To19, IssuePatch{SetStartDate: true, SetTargetDate: true, TargetDate: october(1)},
			nil},
		{"a start date alone, no target date", Issue{}, IssuePatch{SetStartDate: true, StartDate: october(30)}, nil},
		{"a target date alone, no start date", Issue{TargetDate: october(19)}, IssuePatch{SetTargetDate: true, TargetDate: october(1)}, nil},
	}
	for _, tt := range tests {
		if got := DatesOutOfOrder(tt.i, tt.p); !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: DatesOutOfOrder() = %+v, want %+v", tt.name, got, tt.want)
		}
	}
}
