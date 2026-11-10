package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// StateGroup is the group of a work item's state, as the project module's
// states.group stores it (M3 design 4.9).
type StateGroup string

// The five groups a cycle's or a module's work item is in (M6 P4 spec R1);
// the triage group takes no work item.
const (
	GroupBacklog   StateGroup = "backlog"
	GroupUnstarted StateGroup = "unstarted"
	GroupStarted   StateGroup = "started"
	GroupCompleted StateGroup = "completed"
	GroupCancelled StateGroup = "cancelled"
)

// BurndownDays is the longest span of a burndown, its end less its start in
// days (M6 P4 spec R4): about ten years, 3661 points. A longer span has no
// burndown.
const BurndownDays = 3660

// WorkItem is a work item of a cycle or a module as its progress reads it,
// from the issue module (M6 P4 spec R2): its state by id, the moment it was
// put in a completed state, if any, and its undeleted assignees and labels.
type WorkItem struct {
	ID, StateID           uuid.UUID
	CompletedAt           *time.Time
	AssigneeIDs, LabelIDs []uuid.UUID
}

// StateCount is how many of the work items of a cycle or a module,
// TargetID's, are in the state StateID, as a list row counts them, from
// the issue module (M6 P4 spec R2, R9); the state's group is the project's.
type StateCount struct {
	TargetID, StateID uuid.UUID
	Issues            int64
}

// CycleAssignee is an account assigned to a work item of a cycle, as a
// cycle's list row reads its assignees, from the issue module (M6 P4 spec
// R9).
type CycleAssignee struct {
	CycleID, AssigneeID uuid.UUID
}

// Counts are the work items by the group of their state; the total is the
// sum of the five.
type Counts struct {
	Total     int64 `json:"total_issues"`
	Backlog   int64 `json:"backlog_issues"`
	Unstarted int64 `json:"unstarted_issues"`
	Started   int64 `json:"started_issues"`
	Completed int64 `json:"completed_issues"`
	Cancelled int64 `json:"cancelled_issues"`
}

// Tally is a row of a distribution: its work items, the cancelled ones left
// out, as the completed ones and the pending ones (backlog, unstarted,
// started).
type Tally struct {
	Total     int64 `json:"total_issues"`
	Completed int64 `json:"completed_issues"`
	Pending   int64 `json:"pending_issues"`
}

// AssigneeDistribution is the tally of an assignee's work items, or of
// those without an assignee when AssigneeID is nil.
type AssigneeDistribution struct {
	AssigneeID *uuid.UUID `json:"assignee_id"`
	Tally
}

// LabelDistribution is the tally of a label's work items, or of those
// without a label when LabelID is nil.
type LabelDistribution struct {
	LabelID *uuid.UUID `json:"label_id"`
	Tally
}

// Burndown is the work left on each day of a span, on the project's
// calendar: of its scope, the work items not cancelled.
type Burndown struct {
	StartDate shared.Date     `json:"start_date"`
	EndDate   shared.Date     `json:"end_date"`
	Scope     int64           `json:"scope"`
	Points    []BurndownPoint `json:"points"`
}

// BurndownPoint is a day of a burndown and the work items of its scope not
// completed by the day's end; none for a day after today.
type BurndownPoint struct {
	Date      shared.Date `json:"date"`
	Remaining *int64      `json:"remaining"`
}

// Progress is the numbers of a cycle or a module (M6 P4 spec 2.1): the
// counts, the distributions by assignee and by label, and the burndown,
// none without a span to draw.
type Progress struct {
	Counts
	Assignees []AssigneeDistribution `json:"assignees"`
	Labels    []LabelDistribution    `json:"labels"`
	Burndown  *Burndown              `json:"burndown"`
}

// Frozen is a cycle's progress as it was when the cycle ended, and the
// moment it was taken: a cycle's progress_snapshot (M6 P4 spec 2.4).
type Frozen struct {
	Progress
	FrozenAt time.Time `json:"frozen_at"`
}

// CycleProgress is a cycle's status and its progress as its reader is
// answered it (M6 P4 spec 2.3): live, of the work items he sees, and as
// frozen, nil while it is not, and for a reader who sees only his own work
// items, whom it would show others' (R5).
type CycleProgress struct {
	Status   Status
	Live     Progress
	Snapshot *Frozen
}

// FrozenCounts are the counts of a frozen progress and its moment, as a
// cycle's list row answers them (M6 P4 spec R5).
type FrozenCounts struct {
	Counts
	FrozenAt time.Time `json:"frozen_at"`
}

// ProgressFacts are what a progress is computed from: the work items, the
// group of each of the project's states, the span of the burndown, either
// day nil for none, and the project's time zone, an IANA name, and today in
// it.
type ProgressFacts struct {
	Items      []WorkItem
	Groups     map[uuid.UUID]StateGroup
	Start, End *shared.Date
	Timezone   string
	Today      shared.Date
}

// grouped is a work item and the group of its state.
type grouped struct {
	WorkItem
	group StateGroup
}

// ProgressOf is the progress of f's work items (M6 P4 spec 2.1). A work item
// in a state not among f's groups, or in one outside the five groups, is an
// error, a defect rather than a number to leave out (R1); so is a time zone
// that does not load.
func ProgressOf(f ProgressFacts) (Progress, error) {
	loc, err := time.LoadLocation(f.Timezone)
	if err != nil {
		return Progress{}, fmt.Errorf("load the project's time zone %q: %w", f.Timezone, err)
	}
	var counts Counts
	items := make([]grouped, len(f.Items))
	for i, item := range f.Items {
		group, ok := f.Groups[item.StateID]
		if !ok {
			return Progress{}, fmt.Errorf("work item %s is in state %s, not one of the project's", item.ID, item.StateID)
		}
		if err := counts.add(group); err != nil {
			return Progress{}, fmt.Errorf("work item %s: %w", item.ID, err)
		}
		items[i] = grouped{item, group}
	}
	return Progress{
		Counts: counts,
		Assignees: distribution(items, func(w WorkItem) []uuid.UUID { return w.AssigneeIDs },
			func(id *uuid.UUID, t Tally) AssigneeDistribution { return AssigneeDistribution{id, t} }),
		Labels: distribution(items, func(w WorkItem) []uuid.UUID { return w.LabelIDs },
			func(id *uuid.UUID, t Tally) LabelDistribution { return LabelDistribution{id, t} }),
		Burndown: burndownOf(items, f.Start, f.End, loc, f.Today),
	}, nil
}

// add counts a work item in a state of group, one of the five.
func (c *Counts) add(group StateGroup) error {
	switch group {
	case GroupBacklog:
		c.Backlog++
	case GroupUnstarted:
		c.Unstarted++
	case GroupStarted:
		c.Started++
	case GroupCompleted:
		c.Completed++
	case GroupCancelled:
		c.Cancelled++
	default:
		return fmt.Errorf("its state's group %q takes no work item", group)
	}
	c.Total++
	return nil
}

// distribution is the rows, as row makes them, of the work items not
// cancelled (M6 P4 spec R3): a work item counts in the row of each id that
// ids gives it, or in the row of none, id nil, when it gives none. A row has
// a work item at least; the rows are by id ascending, in the byte order of
// the ids, which is that of their text and Postgres's, and the row of none
// is last.
func distribution[R any](items []grouped, ids func(WorkItem) []uuid.UUID, row func(*uuid.UUID, Tally) R) []R {
	tallies := map[uuid.UUID]*Tally{}
	var none Tally
	for _, item := range items {
		if item.group == GroupCancelled {
			continue
		}
		of := ids(item.WorkItem)
		if len(of) == 0 {
			none.add(item.group)
		}
		for _, id := range of {
			t, ok := tallies[id]
			if !ok {
				t = &Tally{}
				tallies[id] = t
			}
			t.add(item.group)
		}
	}
	rows := make([]R, 0, len(tallies)+1)
	for _, id := range slices.SortedFunc(maps.Keys(tallies), uuid.UUID.Compare) {
		rows = append(rows, row(&id, *tallies[id]))
	}
	if none.Total > 0 {
		rows = append(rows, row(nil, none))
	}
	return rows
}

// add counts a work item in a state of group, not cancelled: completed in
// the completed group, pending in the others.
func (t *Tally) add(group StateGroup) {
	t.Total++
	if group == GroupCompleted {
		t.Completed++
	} else {
		t.Pending++
	}
}

// burndownOf is the burndown of items from start to end, the days of the
// project in loc, on today (M6 P4 spec 2.1, R4); nil without both days, with
// the start after the end, which the cycles' and the modules' checks refuse,
// or over a span longer than BurndownDays. A work item is completed in the
// completed group, on the day its completed_at falls on in loc, or from the
// start without one; a completed_at in another group is stale and burns
// nothing.
func burndownOf(items []grouped, start, end *shared.Date, loc *time.Location, today shared.Date) *Burndown {
	if start == nil || end == nil {
		return nil
	}
	span := daysFrom(*start, *end)
	if span < 0 || span > BurndownDays {
		return nil
	}
	var scope int64
	var completions []shared.Date
	for _, item := range items {
		if item.group == GroupCancelled {
			continue
		}
		scope++
		if item.group != GroupCompleted {
			continue
		}
		on := *start
		if item.CompletedAt != nil {
			on = shared.DateOf(item.CompletedAt.In(loc))
		}
		completions = append(completions, on)
	}
	slices.SortFunc(completions, shared.Date.Compare)
	points := make([]BurndownPoint, 0, span+1)
	completed := 0
	for i := range span + 1 {
		d := shared.DateOf(start.Time().AddDate(0, 0, int(i)))
		for completed < len(completions) && completions[completed].Compare(d) <= 0 {
			completed++
		}
		point := BurndownPoint{Date: d}
		if d.Compare(today) <= 0 {
			remaining := scope - int64(completed)
			point.Remaining = &remaining
		}
		points = append(points, point)
	}
	return &Burndown{StartDate: *start, EndDate: *end, Scope: scope, Points: points}
}

// daysFrom is the days from a to b, negative when b is before a.
func daysFrom(a, b shared.Date) int64 {
	return (b.Time().Unix() - a.Time().Unix()) / (24 * 60 * 60)
}

// EncodeFrozen is f as a cycle's progress_snapshot stores it: the API's
// FrozenProgress, by its field names (M6 P4 spec 2.3, 2.4).
func EncodeFrozen(f Frozen) ([]byte, error) {
	return json.Marshal(f)
}

// DecodeFrozen reads a cycle's progress_snapshot. The column is the module's
// own record, so a document other than EncodeFrozen writes is a defect,
// refused rather than read as zeros or as nothing: a key unknown, missing or
// written in another case, a list null.
func DecodeFrozen(raw []byte) (Frozen, error) {
	var f Frozen
	if err := decodeExact(raw, &f); err != nil {
		return Frozen{}, fmt.Errorf("read a cycle's frozen progress: %w", err)
	}
	if f.Assignees == nil || f.Labels == nil || (f.Burndown != nil && f.Burndown.Points == nil) {
		return Frozen{}, errors.New("read a cycle's frozen progress: a list is null")
	}
	return f, nil
}

// DecodeFrozenCounts reads the counts and the moment of a cycle's
// progress_snapshot as a list row reads them (M6 P4 spec R5): a document of
// those keys alone, as EncodeFrozen writes them. The whole document is
// refused, so that a list never reads the distributions and the burndown.
func DecodeFrozenCounts(raw []byte) (FrozenCounts, error) {
	var c FrozenCounts
	if err := decodeExact(raw, &c); err != nil {
		return FrozenCounts{}, fmt.Errorf("read the counts of a cycle's frozen progress: %w", err)
	}
	return c, nil
}

// decodeExact reads raw into v and holds raw to be v's encoding, the same
// keys and values: json.Unmarshal alone leaves a missing key zero and
// matches a key in another case.
func decodeExact(raw []byte, v any) error {
	if err := json.Unmarshal(raw, v); err != nil {
		return err
	}
	encoded, err := json.Marshal(v)
	if err != nil {
		return err
	}
	read, err := jsonValues(raw)
	if err != nil {
		return err
	}
	written, err := jsonValues(encoded)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(read, written) {
		return errors.New("a key is unknown, missing or written otherwise")
	}
	return nil
}

// jsonValues are the values of the JSON document doc, its numbers as written.
func jsonValues(doc []byte) (any, error) {
	d := json.NewDecoder(bytes.NewReader(doc))
	d.UseNumber()
	var v any
	err := d.Decode(&v)
	return v, err
}
