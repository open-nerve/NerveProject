package domain

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

// showState writes each of s's times in RFC 3339 with its zone, "-" for
// none, so a comparison sees the instant, the precision and the zone.
func showState(s State) string {
	show := func(at *time.Time) string {
		if at == nil {
			return "-"
		}
		return at.Format(time.RFC3339Nano)
	}
	return "{read " + show(s.ReadAt) + " archived " + show(s.ArchivedAt) + " snoozed " + show(s.SnoozedTill) + "}"
}

// A patch (spec 2.7, steps 6–7; R7) changes each of the three states on
// its own, only the ones it gives. read true stamps read_at with now when
// it is not read, and keeps when it was read when it is; false clears it;
// archived the same on archived_at. snoozed_till is set to the time it
// gives, in UTC and at the database's microsecond, or cleared by null; the
// time it is set to already is no change. The empty patch changes nothing.
// now is taken in UTC at the microsecond too.
func TestPatchApply(t *testing.T) {
	now := time.Date(2026, 11, 18, 9, 0, 0, 123456000, time.UTC)
	earlier, later, evenLater := now.Add(-time.Hour), now.Add(time.Hour), now.Add(2*time.Hour)
	nextMicro := now.Add(time.Microsecond)
	cst := time.FixedZone("CST", 8*3600)
	tests := []struct {
		name    string
		state   State
		patch   Patch
		now     time.Time
		want    State
		changed bool
	}{
		{"nothing on a new row", State{}, Patch{}, now, State{}, false},
		{"nothing on a row in every state", State{&earlier, &earlier, &later}, Patch{}, now, State{&earlier, &earlier, &later}, false},
		{"read", State{}, Patch{Read: ptr(true)}, now, State{ReadAt: &now}, true},
		{"read again keeps when", State{ReadAt: &earlier}, Patch{Read: ptr(true)}, now, State{ReadAt: &earlier}, false},
		{"unread", State{ReadAt: &earlier}, Patch{Read: ptr(false)}, now, State{}, true},
		{"unread again", State{}, Patch{Read: ptr(false)}, now, State{}, false},
		{"archive", State{}, Patch{Archived: ptr(true)}, now, State{ArchivedAt: &now}, true},
		{"archive again keeps when", State{ArchivedAt: &earlier}, Patch{Archived: ptr(true)}, now, State{ArchivedAt: &earlier}, false},
		{"unarchive", State{ArchivedAt: &earlier}, Patch{Archived: ptr(false)}, now, State{}, true},
		{"unarchive again", State{}, Patch{Archived: ptr(false)}, now, State{}, false},
		{"snooze", State{}, Patch{SetSnoozed: true, SnoozedTill: &later}, now, State{SnoozedTill: &later}, true},
		{"snooze a microsecond after now", State{}, Patch{SetSnoozed: true, SnoozedTill: &nextMicro}, now, State{SnoozedTill: &nextMicro}, true},
		{"snooze at the microsecond", State{}, Patch{SetSnoozed: true, SnoozedTill: ptr(later.Add(999))}, now, State{SnoozedTill: &later}, true},
		{"snooze in another zone", State{}, Patch{SetSnoozed: true, SnoozedTill: ptr(later.In(cst))}, now, State{SnoozedTill: &later}, true},
		{"snooze till the same time", State{SnoozedTill: &later}, Patch{SetSnoozed: true, SnoozedTill: ptr(later)}, now,
			State{SnoozedTill: &later}, false},
		{"snooze till the same microsecond", State{SnoozedTill: &later}, Patch{SetSnoozed: true, SnoozedTill: ptr(later.Add(300))}, now,
			State{SnoozedTill: &later}, false},
		{"snooze till another time", State{SnoozedTill: &later}, Patch{SetSnoozed: true, SnoozedTill: &evenLater}, now,
			State{SnoozedTill: &evenLater}, true},
		{"unsnooze", State{SnoozedTill: &later}, Patch{SetSnoozed: true}, now, State{}, true},
		{"unsnooze an expired snooze", State{SnoozedTill: &earlier}, Patch{SetSnoozed: true}, now, State{}, true},
		{"unsnooze again", State{}, Patch{SetSnoozed: true}, now, State{}, false},
		{"snooze not given", State{SnoozedTill: &later}, Patch{SnoozedTill: &earlier}, now, State{SnoozedTill: &later}, false},
		{"read leaves the others", State{ArchivedAt: &earlier, SnoozedTill: &later}, Patch{Read: ptr(true)}, now,
			State{&now, &earlier, &later}, true},
		{"unread leaves the others", State{&earlier, &earlier, &later}, Patch{Read: ptr(false)}, now,
			State{ArchivedAt: &earlier, SnoozedTill: &later}, true},
		{"archive leaves the others", State{ReadAt: &earlier, SnoozedTill: &later}, Patch{Archived: ptr(true)}, now,
			State{&earlier, &now, &later}, true},
		{"unarchive leaves the others", State{&earlier, &earlier, &later}, Patch{Archived: ptr(false)}, now,
			State{ReadAt: &earlier, SnoozedTill: &later}, true},
		{"snooze leaves the others", State{ReadAt: &earlier, ArchivedAt: &earlier}, Patch{SetSnoozed: true, SnoozedTill: &later}, now,
			State{&earlier, &earlier, &later}, true},
		{"unsnooze leaves the others", State{&earlier, &earlier, &later}, Patch{SetSnoozed: true}, now,
			State{ReadAt: &earlier, ArchivedAt: &earlier}, true},
		{"all three", State{}, Patch{Read: ptr(true), Archived: ptr(true), SetSnoozed: true, SnoozedTill: &later}, now,
			State{&now, &now, &later}, true},
		{"all three as they are", State{ReadAt: &earlier}, Patch{Read: ptr(true), Archived: ptr(false), SetSnoozed: true}, now,
			State{ReadAt: &earlier}, false},
		{"now at the microsecond in UTC", State{}, Patch{Read: ptr(true), Archived: ptr(true)}, now.Add(700).In(cst), State{ReadAt: &now, ArchivedAt: &now},
			true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := tt.patch.Apply(tt.state, tt.now)
			if err != nil || showState(got) != showState(tt.want) || changed != tt.changed {
				t.Errorf("Apply() = %s, %v, %v; want %s, %v", showState(got), changed, err, showState(tt.want), tt.changed)
			}
		})
	}
}

// A time given for snoozed_till must be after now, both at the microsecond
// the database keeps (spec 2.7 step 6, CheckAPIToken's rule): else one 422
// must_be_future on snoozed_till, whatever else the patch gives, and
// nothing changes. A snooze till a time that has passed is refused even
// when the row is snoozed till it already.
func TestPatchApplyRefusesASnoozeNotInTheFuture(t *testing.T) {
	now := time.Date(2026, 11, 18, 9, 0, 0, 123456000, time.UTC)
	earlier := now.Add(-time.Hour)
	tests := []struct {
		name  string
		state State
		patch Patch
		now   time.Time
	}{
		{"now", State{}, Patch{SetSnoozed: true, SnoozedTill: ptr(now)}, now},
		{"the past", State{}, Patch{SetSnoozed: true, SnoozedTill: &earlier}, now},
		{"within now's microsecond", State{}, Patch{SetSnoozed: true, SnoozedTill: ptr(now.Add(999))}, now},
		{"after now within its microsecond", State{}, Patch{SetSnoozed: true, SnoozedTill: ptr(now.Add(200))}, now.Add(100)},
		{"now in another zone", State{}, Patch{SetSnoozed: true, SnoozedTill: ptr(now.In(time.FixedZone("CST", 8*3600)))}, now},
		{"the past it is snoozed till", State{SnoozedTill: &earlier}, Patch{SetSnoozed: true, SnoozedTill: &earlier}, now},
		{"with the others", State{}, Patch{Read: ptr(true), Archived: ptr(true), SetSnoozed: true, SnoozedTill: &earlier}, now},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed, err := tt.patch.Apply(tt.state, tt.now)
			if fields := fieldCodes(t, err); !slices.Equal(fields, []string{"snoozed_till:must_be_future"}) || changed ||
				!reflect.DeepEqual(got, State{}) {
				t.Errorf("Apply() = %s, %v with %v; want nothing with snoozed_till:must_be_future", showState(got), changed, fields)
			}
		})
	}
}
