package domain

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Dates are a cycle's start and end: days of the project's calendar, both or
// neither, the start not after the end, which the database's
// cycles_dates_check holds too (M6 design 3.1). The same day for both is in
// order.
type Dates struct {
	Start, End *shared.Date
}

// Dated reports whether d has its days.
func (d Dates) Dated() bool { return d.Start != nil && d.End != nil }

// CheckDates is the problem of d: a date alone is refused at the missing one,
// a start after the end at end_date; nil when d is in order.
func CheckDates(d Dates) *shared.FieldError {
	switch {
	case d.Start != nil && d.End == nil:
		return &shared.FieldError{Field: "end_date", Code: shared.FieldRequired, Message: "must be given with the start date"}
	case d.Start == nil && d.End != nil:
		return &shared.FieldError{Field: "start_date", Code: shared.FieldRequired, Message: "must be given with the end date"}
	case d.Dated() && d.Start.Compare(*d.End) > 0:
		return &shared.FieldError{Field: "end_date", Code: shared.FieldNotAllowed, Message: "must not be before the start date"}
	}
	return nil
}

// DatedCycle is another cycle of the project and its dates, as the overlap
// check reads it.
type DatedCycle struct {
	ID    uuid.UUID
	Dates Dates
}

// Overlapping reports whether d, the dates of the cycle self, share a day
// with the closed range of another of others (M6 design 3.1). A cycle without
// dates overlaps none, and self is not held against itself.
func Overlapping(self uuid.UUID, d Dates, others []DatedCycle) bool {
	if !d.Dated() {
		return false
	}
	for _, o := range others {
		if o.ID != self && o.Dates.Dated() && d.Start.Compare(*o.Dates.End) <= 0 && o.Dates.Start.Compare(*d.End) <= 0 {
			return true
		}
	}
	return false
}
