package domain

import (
	"math"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CyclePatch is a change of a cycle (updateCycle's body): each field given
// or not; a date given with SetStart or SetEnd, nil when it is cleared.
type CyclePatch struct {
	Name, Description *string
	SetStart, SetEnd  bool
	Start, End        *shared.Date
	SortOrder         *float64
}

// CheckCyclePatch is p with its name cleaned, or every problem of its values
// that needs no read, as validation_failed (M6 P1 spec R2): the name, a
// description with NUL, a sort order that is no finite number. The dates it
// gives are held to the cycle's under the lock (Apply).
func CheckCyclePatch(p CyclePatch) (CyclePatch, error) {
	var fields []shared.FieldError
	if p.Name != nil {
		name, problem := CleanName(*p.Name)
		if problem != nil {
			fields = append(fields, *problem)
		}
		p.Name = &name
	}
	if p.Description != nil {
		if problem := checkDescription(*p.Description); problem != nil {
			fields = append(fields, *problem)
		}
	}
	if p.SortOrder != nil && (math.IsNaN(*p.SortOrder) || math.IsInf(*p.SortOrder, 0)) {
		fields = append(fields, shared.FieldError{Field: "sort_order", Code: shared.FieldInvalidFormat, Message: "must be a finite number"})
	}
	if len(fields) > 0 {
		return CyclePatch{}, shared.Invalid(fields...)
	}
	return p, nil
}

// OnlySortOrder reports whether p gives no field but the sort order: all a
// completed cycle takes (M6 P1 spec R5). A field given as it is counts.
func (p CyclePatch) OnlySortOrder() bool {
	return p.Name == nil && p.Description == nil && !p.SetStart && !p.SetEnd
}

// Apply is c as p leaves it, and whether p changes anything: a field given
// as it is changes nothing. Its dates are the ones p gives, each else c's
// (M6 P1 spec R4), refused as validation_failed when they are not both or
// neither with the start not after the end (mergedDates).
func (p CyclePatch) Apply(c Cycle) (Cycle, bool, error) {
	next := c
	if p.Name != nil {
		next.Name = *p.Name
	}
	if p.Description != nil {
		next.Description = *p.Description
	}
	if p.SortOrder != nil {
		next.SortOrder = *p.SortOrder
	}
	dates, problem := p.mergedDates(c.Dates)
	if problem != nil {
		return Cycle{}, false, shared.Invalid(*problem)
	}
	next.Dates = dates
	changed := next.Name != c.Name || next.Description != c.Description || next.SortOrder != c.SortOrder || !next.Dates.Equal(c.Dates)
	return next, changed, nil
}

// mergedDates are the dates p gives, each else d's, or their problem: where
// CheckDates puts it when p gives both; at the date p gives alone when it
// gives one, since the other is the cycle's.
func (p CyclePatch) mergedDates(d Dates) (Dates, *shared.FieldError) {
	merged := d
	if p.SetStart {
		merged.Start = p.Start
	}
	if p.SetEnd {
		merged.End = p.End
	}
	problem := CheckDates(merged)
	switch {
	case problem == nil || p.SetStart == p.SetEnd:
		return merged, problem
	case p.SetStart:
		return Dates{}, aloneProblem("start_date", "end date", merged.Start, merged.End, "must not be after the end date")
	}
	return Dates{}, aloneProblem("end_date", "start date", merged.End, merged.Start, "must not be before the start date")
}

// aloneProblem is the problem of a date given alone, field, against the
// other date the cycle keeps, other: given while the cycle has no other,
// cleared while it has one, or out of order with it.
func aloneProblem(field, other string, given, kept *shared.Date, order string) *shared.FieldError {
	message := order
	switch {
	case given != nil && kept == nil:
		message = "must not be given alone: the cycle has no " + other
	case given == nil && kept != nil:
		message = "must not be cleared alone: the cycle keeps its " + other
	}
	return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: message}
}
