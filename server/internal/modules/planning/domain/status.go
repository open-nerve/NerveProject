package domain

import (
	"fmt"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Status is a cycle's status, read from its dates and the project's day,
// never stored (M6 design 3.1).
type Status string

const (
	StatusDraft     Status = "draft"
	StatusUpcoming  Status = "upcoming"
	StatusCurrent   Status = "current"
	StatusCompleted Status = "completed"
)

// Today is the day now falls on in the project's time zone, an IANA name the
// project module checked (M3 design 3.13). A name that does not load is an
// internal error (M6 P1 spec R8).
func Today(now time.Time, timezone string) (shared.Date, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return shared.Date{}, fmt.Errorf("load the project's time zone %q: %w", timezone, err)
	}
	return shared.DateOf(now.In(loc)), nil
}

// StatusOf is the status of a cycle of dates d on today, the project's day:
// draft without dates, upcoming before the start, current from the start to
// the end, completed after the end.
func StatusOf(d Dates, today shared.Date) Status {
	switch {
	case !d.Dated():
		return StatusDraft
	case today.Compare(*d.Start) < 0:
		return StatusUpcoming
	case today.Compare(*d.End) <= 0:
		return StatusCurrent
	}
	return StatusCompleted
}
