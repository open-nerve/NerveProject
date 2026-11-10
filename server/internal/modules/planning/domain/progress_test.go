package domain_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
)

// The project's states, one of each group a work item may be in, a second completed one, and a triage one.
var (
	backlog    = uuid.MustParse("5a000000-0000-7000-8000-000000000001")
	unstarted  = uuid.MustParse("5a000000-0000-7000-8000-000000000002")
	started    = uuid.MustParse("5a000000-0000-7000-8000-000000000003")
	completed  = uuid.MustParse("5a000000-0000-7000-8000-000000000004")
	completed2 = uuid.MustParse("5a000000-0000-7000-8000-000000000005")
	cancelled  = uuid.MustParse("5a000000-0000-7000-8000-000000000006")
	triage     = uuid.MustParse("5a000000-0000-7000-8000-000000000007")
)

func stateGroups() map[uuid.UUID]domain.StateGroup {
	return map[uuid.UUID]domain.StateGroup{
		backlog: domain.GroupBacklog, unstarted: domain.GroupUnstarted, started: domain.GroupStarted,
		completed: domain.GroupCompleted, completed2: domain.GroupCompleted, cancelled: domain.GroupCancelled,
	}
}

// in is a work item in state, with the given assignees and labels.
func in(state uuid.UUID, assignees, labels []uuid.UUID) domain.WorkItem {
	return domain.WorkItem{ID: uuid.NewV7(), StateID: state, AssigneeIDs: assignees, LabelIDs: labels}
}

// done is a work item in state with the completed_at moment at, RFC 3339, or none for "".
func done(t *testing.T, state uuid.UUID, at string) domain.WorkItem {
	t.Helper()
	item := in(state, nil, nil)
	if at != "" {
		moment, err := time.Parse(time.RFC3339, at)
		if err != nil {
			t.Fatal(err)
		}
		item.CompletedAt = &moment
	}
	return item
}

// undated are the facts of items without a span, in Asia/Shanghai on 2026-11-10.
func undated(t *testing.T, items ...domain.WorkItem) domain.ProgressFacts {
	return domain.ProgressFacts{Items: items, Groups: stateGroups(), Timezone: "Asia/Shanghai", Today: *day(t, "2026-11-10")}
}

// The counts are the work items by their state's group, two states of one group counting together, and the total is
// the sum of the five (M6 P4 spec 2.1).
func TestProgressCounts(t *testing.T) {
	for _, tt := range []struct {
		name   string
		states []uuid.UUID
		want   domain.Counts
	}{
		{"no work item", nil, domain.Counts{}},
		{"one of each group", []uuid.UUID{backlog, unstarted, started, completed, cancelled},
			domain.Counts{Total: 5, Backlog: 1, Unstarted: 1, Started: 1, Completed: 1, Cancelled: 1}},
		{"several in a group", []uuid.UUID{started, backlog, started, cancelled, started},
			domain.Counts{Total: 5, Backlog: 1, Started: 3, Cancelled: 1}},
		{"two states of one group", []uuid.UUID{completed, completed2, unstarted},
			domain.Counts{Total: 3, Unstarted: 1, Completed: 2}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var items []domain.WorkItem
			for _, s := range tt.states {
				items = append(items, in(s, nil, nil))
			}
			got, err := domain.ProgressOf(undated(t, items...))
			if err != nil {
				t.Fatal(err)
			}
			if got.Counts != tt.want {
				t.Fatalf("counts = %+v; want %+v", got.Counts, tt.want)
			}
			if c := got.Counts; c.Total != c.Backlog+c.Unstarted+c.Started+c.Completed+c.Cancelled {
				t.Fatalf("the total %d is not the sum of the groups: %+v", c.Total, c)
			}
		})
	}
}

// A state outside the five groups takes no work item (M6 P4 spec R1): one of the triage group, of an unknown group,
// or not among the project's states is a defect, answered as an error, never left out of the numbers. So is a time
// zone that does not load.
func TestProgressRefusesAStateOutsideTheGroups(t *testing.T) {
	for _, tt := range []struct {
		name  string
		group domain.StateGroup
		state uuid.UUID
	}{
		{"triage", "triage", triage},
		{"an unknown group", "archived", triage},
		{"an empty group", "", triage},
		{"not the project's state", "", uuid.NewV7()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			facts := undated(t, in(started, nil, nil), in(tt.state, nil, nil))
			if tt.state == triage {
				facts.Groups[triage] = tt.group
			}
			if got, err := domain.ProgressOf(facts); err == nil {
				t.Fatalf("answered %+v", got)
			}
		})
	}
	facts := undated(t, in(started, nil, nil))
	facts.Timezone = "Not/AZone"
	if _, err := domain.ProgressOf(facts); err == nil {
		t.Fatal("an unknown time zone was read")
	}
}

// The people and labels, by id: the order the work items name them is not the order of the rows.
var (
	alice = uuid.MustParse("0a000000-0000-7000-8000-000000000001")
	bob   = uuid.MustParse("0b000000-0000-7000-8000-000000000001")
	carol = uuid.MustParse("f0000000-0000-7000-8000-000000000001")
	dave  = uuid.MustParse("0c000000-0000-7000-8000-000000000001")
	ui    = uuid.MustParse("1a000000-0000-7000-8000-000000000001")
	bug   = uuid.MustParse("2b000000-0000-7000-8000-000000000001")
	docs  = uuid.MustParse("3c000000-0000-7000-8000-000000000001")
)

func of(u ...uuid.UUID) []uuid.UUID { return u }

func tally(total, completed, pending int64) domain.Tally {
	return domain.Tally{Total: total, Completed: completed, Pending: pending}
}

// The distributions leave the cancelled work items out (M6 P4 spec 2.1, R3): a work item counts in a row for each of
// its assignees or labels, in the row of none when it has none; completed by its state's group, pending in backlog,
// unstarted and started; only the rows with a work item, by id ascending, the row of none last.
func TestProgressDistribution(t *testing.T) {
	for _, tt := range []struct {
		name      string
		items     []domain.WorkItem
		assignees []domain.AssigneeDistribution
		labels    []domain.LabelDistribution
	}{
		{
			name: "several, none and cancelled",
			items: []domain.WorkItem{
				in(started, of(carol, alice), of(bug)),
				in(completed, of(alice), of(bug, ui)),
				in(backlog, nil, nil),
				in(cancelled, of(dave), of(docs)),
				in(cancelled, nil, nil),
				in(completed2, nil, of(ui)),
				in(unstarted, of(bob), nil),
			},
			assignees: []domain.AssigneeDistribution{
				{AssigneeID: &alice, Tally: tally(2, 1, 1)},
				{AssigneeID: &bob, Tally: tally(1, 0, 1)},
				{AssigneeID: &carol, Tally: tally(1, 0, 1)},
				{Tally: tally(2, 1, 1)},
			},
			labels: []domain.LabelDistribution{
				{LabelID: &ui, Tally: tally(2, 2, 0)},
				{LabelID: &bug, Tally: tally(2, 1, 1)},
				{Tally: tally(2, 0, 2)},
			},
		},
		{
			name:      "every one has some: no row of none",
			items:     []domain.WorkItem{in(started, of(bob), of(ui)), in(backlog, of(alice), of(ui))},
			assignees: []domain.AssigneeDistribution{{AssigneeID: &alice, Tally: tally(1, 0, 1)}, {AssigneeID: &bob, Tally: tally(1, 0, 1)}},
			labels:    []domain.LabelDistribution{{LabelID: &ui, Tally: tally(2, 0, 2)}},
		},
		{
			name:      "only cancelled: no row",
			items:     []domain.WorkItem{in(cancelled, of(alice), of(ui)), in(cancelled, nil, nil)},
			assignees: []domain.AssigneeDistribution{},
			labels:    []domain.LabelDistribution{},
		},
		{
			name:      "no work item",
			assignees: []domain.AssigneeDistribution{},
			labels:    []domain.LabelDistribution{},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ProgressOf(undated(t, tt.items...))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got.Assignees, tt.assignees) {
				t.Fatalf("assignees = %s; want %s", assigneeRows(got.Assignees), assigneeRows(tt.assignees))
			}
			if !reflect.DeepEqual(got.Labels, tt.labels) {
				t.Fatalf("labels = %s; want %s", labelRows(got.Labels), labelRows(tt.labels))
			}
		})
	}
}

// assigneeRows are the rows of an assignee distribution, as row writes them.
func assigneeRows(rows []domain.AssigneeDistribution) string {
	var out []string
	for _, r := range rows {
		out = append(out, row(r.AssigneeID, r.Tally))
	}
	return "[" + strings.Join(out, " ") + "]"
}

// labelRows are the rows of a label distribution, as row writes them.
func labelRows(rows []domain.LabelDistribution) string {
	var out []string
	for _, r := range rows {
		out = append(out, row(r.LabelID, r.Tally))
	}
	return "[" + strings.Join(out, " ") + "]"
}

// row is a distribution's row as its id's first byte and its total/completed/pending.
func row(id *uuid.UUID, t domain.Tally) string {
	name := "none"
	if id != nil {
		name = id.String()[:2]
	}
	return fmt.Sprintf("%s:%d/%d/%d", name, t.Total, t.Completed, t.Pending)
}
