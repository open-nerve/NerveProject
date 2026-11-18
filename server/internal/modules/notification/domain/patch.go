package domain

import (
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// State is a notification's three states (P1c spec 2.7, R7), each of them
// changed on its own, by no rule of the others: when it was read and when
// it was archived, nil for not; till when it is snoozed, nil for not.
type State struct {
	ReadAt      *time.Time
	ArchivedAt  *time.Time
	SnoozedTill *time.Time
}

// Patch is a receiver's change of his notification's states (spec 2.7):
// Read and Archived nil when not given; SnoozedTill only when SetSnoozed,
// nil to clear it.
type Patch struct {
	Read        *bool
	Archived    *bool
	SetSnoozed  bool
	SnoozedTill *time.Time
}

// Apply is what s becomes under p at now (spec 2.7, steps 6 and 7), and
// whether that is a change. A time for snoozed_till must be after now,
// both in UTC at the microsecond the database keeps (identity's
// CheckAPIToken), else it is one 422 must_be_future on snoozed_till and
// nothing changes; it is set to that time, and null clears it. read true
// stamps read_at with now when it is not set, and keeps it when it is;
// false clears it; archived the same with archived_at. The state s is
// already in is no change.
func (p Patch) Apply(s State, now time.Time) (State, bool, error) {
	now = now.UTC().Truncate(time.Microsecond)
	next := s
	if p.SetSnoozed {
		next.SnoozedTill = nil
		if p.SnoozedTill != nil {
			till := p.SnoozedTill.UTC().Truncate(time.Microsecond)
			if !till.After(now) {
				return State{}, false, shared.Invalid(shared.FieldError{Field: "snoozed_till", Code: shared.FieldMustBeFuture,
					Message: "must be in the future"})
			}
			next.SnoozedTill = &till
		}
	}
	if p.Read != nil {
		next.ReadAt = stamped(s.ReadAt, *p.Read, now)
	}
	if p.Archived != nil {
		next.ArchivedAt = stamped(s.ArchivedAt, *p.Archived, now)
	}
	changed := !sameTime(next.ReadAt, s.ReadAt) || !sameTime(next.ArchivedAt, s.ArchivedAt) || !sameTime(next.SnoozedTill, s.SnoozedTill)
	return next, changed, nil
}

// stamped is a state's time at, set on or not: kept when it is set and on,
// now when it is not set and on, nil when off.
func stamped(at *time.Time, on bool, now time.Time) *time.Time {
	switch {
	case !on:
		return nil
	case at != nil:
		return at
	}
	return &now
}

// sameTime tells whether a and b are both nil, or the same instant.
func sameTime(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
