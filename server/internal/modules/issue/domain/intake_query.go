package domain

import (
	"maps"
	"slices"
	"strconv"
	"strings"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// IntakeOrder is an intake list's ordering word (M7 P2b spec 2.3): the
// work item's created_at, updated_at or sequence_id, with a leading minus
// for descending; equal keys go by the work item's id the same way. It is
// no ListOrder: sequence_id orders no other list, nor a view's settings
// (DP-13).
type IntakeOrder string

// IntakeOrderAllowed accepts the six intake ordering words.
func IntakeOrderAllowed(o IntakeOrder) bool {
	switch o.Key() {
	case "created_at", "updated_at", "sequence_id":
		return true
	}
	return false
}

// Key is the field o orders by, without its minus.
func (o IntakeOrder) Key() string { return strings.TrimPrefix(string(o), "-") }

// Descending reports whether o orders from the greatest key down.
func (o IntakeOrder) Descending() bool { return strings.HasPrefix(string(o), "-") }

// InstantRange is a half-open range of instants in UTC (spec 2.3, R8):
// From in it, To after it, a nil end open. Its zero value has no
// condition.
type InstantRange struct {
	From, To *time.Time
}

// IntakeFilters are an intake list's filters (spec 2.3): alternatives
// within each field, every field conjunctive. Statuses are effective ones:
// pending takes a snooze whose time has come, snoozed one whose time has
// not. Created and Updated range over the work item's created_at and
// updated_at.
type IntakeFilters struct {
	Statuses          []IntakeStatus
	Priorities        []Priority
	Labels, Assignees IDFilter
	CreatorIDs        []uuid.UUID
	Created, Updated  InstantRange
}

// IntakeQuery is a validated intake list request, independent of HTTP
// binding.
type IntakeQuery struct {
	Filters IntakeFilters
	Order   IntakeOrder
	Limit   int
	Cursor  *IntakeCursor
}

// ParseIntakeQuery reads an intake list's query (spec 2.3), each
// parameter once, else 422 not_allowed on it, in the parameters' name
// order: status, the five names comma-separated; priority, label_id,
// assignee_id and created_by_id as the project list reads them, me the
// caller; created_at and updated_at a day, a..b, a.. or ..b, not none, each
// the half-open range from its first day's midnight UTC to the midnight
// after its last; order_by one of IntakeOrder's, -created_at by default;
// limit as the lists'; and a cursor of the order. No referenced id is
// looked up.
func ParseIntakeQuery(raw map[string][]string, actorID uuid.UUID) (IntakeQuery, error) {
	q := IntakeQuery{Order: "-created_at"}
	var atoms ListFilters
	var limit *int
	for _, field := range slices.Sorted(maps.Keys(raw)) {
		values := raw[field]
		if len(values) != 1 {
			return IntakeQuery{}, listNotAllowed(field)
		}
		value := values[0]
		var err error
		switch field {
		case "status":
			q.Filters.Statuses, err = parseIntakeStatuses(value)
		case "priority", "label_id", "assignee_id", "created_by_id":
			err = atoms.parse(field, values, actorID)
		case "created_at":
			q.Filters.Created, err = parseInstantRange(field, value)
		case "updated_at":
			q.Filters.Updated, err = parseInstantRange(field, value)
		case "order_by":
			if !IntakeOrderAllowed(IntakeOrder(value)) {
				return IntakeQuery{}, listNotAllowed(field)
			}
			q.Order = IntakeOrder(value)
		case "limit":
			parsed, convErr := strconv.Atoi(value)
			if convErr != nil {
				return IntakeQuery{}, listBindingError(field, shared.FieldInvalidFormat)
			}
			limit = &parsed
		case "cursor": // Decode once the order is known.
		default:
			return IntakeQuery{}, listNotAllowed(field)
		}
		if err != nil {
			return IntakeQuery{}, err
		}
	}
	q.Filters.Priorities, q.Filters.Labels, q.Filters.Assignees, q.Filters.CreatorIDs = atoms.Priorities, atoms.Labels, atoms.Assignees, atoms.CreatorIDs
	var err error
	if q.Limit, err = shared.PageSize(limit); err != nil {
		return IntakeQuery{}, err
	}
	if values, ok := raw["cursor"]; ok {
		cursor, err := DecodeIntakeCursor(values[0], q.Order)
		if err != nil {
			return IntakeQuery{}, err
		}
		q.Cursor = &cursor
	}
	return q, nil
}

// parseIntakeStatuses reads status's comma-separated names, each one of
// the five as spelt.
func parseIntakeStatuses(value string) ([]IntakeStatus, error) {
	var statuses []IntakeStatus
	for _, name := range strings.Split(value, ",") {
		if !slices.Contains(intakeStatuses, IntakeStatus(name)) {
			return nil, listNotAllowed("status")
		}
		statuses = append(statuses, IntakeStatus(name))
	}
	return statuses, nil
}

// parseInstantRange reads a day condition of field as the half-open
// range of instants it covers in UTC (R8): from the first day's midnight,
// in it, to the midnight after the last day, out of it, so that no
// instant of the last day is missed. none is refused: every work item has
// the two times.
func parseInstantRange(field, value string) (InstantRange, error) {
	days, err := ParseListDateFilter(value)
	if err != nil || days.None {
		return InstantRange{}, listNotAllowed(field)
	}
	var r InstantRange
	if days.From != nil {
		from := days.From.Time()
		r.From = &from
	}
	if days.To != nil {
		to := days.To.Time().AddDate(0, 0, 1)
		r.To = &to
	}
	return r, nil
}
