package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// A work item's start and target dates (M4 design 3.7): each a day, or
// none; the start not after the target when it has both, which the
// database's issues_dates_check holds too. The same day for both is in
// order.

// DateChange is a date a change sets: To, the new day, nil when it clears
// the date.
type DateChange struct {
	To *shared.Date
}

// outOfOrder reports whether start comes after target, either nil for
// none: a date missing orders nothing.
func outOfOrder(start, target *shared.Date) bool {
	return start != nil && target != nil && start.Compare(*target) > 0
}

// checkNewDates refuses the dates of a new work item, both as the request
// gives them, when the start comes after the target: at target_date.
func checkNewDates(start, target *shared.Date) *shared.FieldError {
	if outOfOrder(start, target) {
		return targetBeforeStart()
	}
	return nil
}

// DatesOutOfOrder is the problem of the dates the work item i has once p,
// a patch, changes it (M4 design 3.7): the dates p gives, each else i's.
// A start after the target is refused at target_date, or at start_date
// when p gives the start date alone; nil when they are in order. So a
// start date given alone is held to the target date the work item keeps,
// and one given with a target date to that date, not to the one it had.
func DatesOutOfOrder(i Issue, p IssuePatch) *shared.FieldError {
	start, target := i.StartDate, i.TargetDate
	if p.SetStartDate {
		start = p.StartDate
	}
	if p.SetTargetDate {
		target = p.TargetDate
	}
	switch {
	case !outOfOrder(start, target):
		return nil
	case p.SetStartDate && !p.SetTargetDate:
		return &shared.FieldError{Field: "start_date", Code: shared.FieldNotAllowed, Message: "must not be after the target date"}
	}
	return targetBeforeStart()
}

// targetBeforeStart is the problem of a target date before the start date.
func targetBeforeStart() *shared.FieldError {
	return &shared.FieldError{Field: "target_date", Code: shared.FieldNotAllowed, Message: "must not be before the start date"}
}

// dayText is d as an activity's value: YYYY-MM-DD, "" for none.
func dayText(d *shared.Date) string {
	if d == nil {
		return ""
	}
	return d.String()
}
