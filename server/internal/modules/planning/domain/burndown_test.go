package domain_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// spanned are the facts of items over start to end ("" for none) in zone on today.
func spanned(t *testing.T, zone, start, end, today string, items ...domain.WorkItem) domain.ProgressFacts {
	t.Helper()
	facts := domain.ProgressFacts{Items: items, Groups: stateGroups(), Timezone: zone, Today: *day(t, today)}
	if start != "" {
		facts.Start = day(t, start)
	}
	if end != "" {
		facts.End = day(t, end)
	}
	return facts
}

// points is a burndown's points as MM-DD=remaining, "-" for none.
func points(b *domain.Burndown) string {
	var out []string
	for _, p := range b.Points {
		remaining := "-"
		if p.Remaining != nil {
			remaining = fmt.Sprint(*p.Remaining)
		}
		out = append(out, p.Date.String()[5:]+"="+remaining)
	}
	return strings.Join(out, " ")
}

// The burndown has a point for each day of the span, on the project's calendar (M6 P4 spec 2.1, R4): its scope is the
// work items not cancelled; a day's remaining is the scope less those completed by the day's end, none after today.
// A work item is completed by its state's group, on the day its completed_at falls on in the project's time zone;
// without a completed_at, or completed before the start, from the first day. The completion moments sit on one side
// of a day boundary of the project's zone where the UTC day is the other, so reading the day in UTC fails here.
func TestProgressBurndown(t *testing.T) {
	const shanghai, losAngeles = "Asia/Shanghai", "America/Los_Angeles"
	for _, tt := range []struct {
		name              string
		zone              string
		start, end, today string
		items             []domain.WorkItem
		scope             int64
		points            string
	}{
		{"completed on the project's day, east of UTC", shanghai, "2026-11-01", "2026-11-05", "2026-11-04",
			[]domain.WorkItem{done(t, completed, "2026-11-02T17:30:00Z"), in(started, nil, nil)},
			2, "11-01=2 11-02=2 11-03=1 11-04=1 11-05=-"},
		{"completed on the project's day, west of UTC", losAngeles, "2026-11-01", "2026-11-05", "2026-11-04",
			[]domain.WorkItem{done(t, completed, "2026-11-03T05:00:00Z"), in(backlog, nil, nil)},
			2, "11-01=2 11-02=1 11-03=1 11-04=1 11-05=-"},
		{"the last moment of a day and the first of the next", shanghai, "2026-11-01", "2026-11-05", "2026-11-05",
			[]domain.WorkItem{done(t, completed, "2026-11-02T15:59:59Z"), done(t, completed2, "2026-11-02T16:00:00Z")},
			2, "11-01=2 11-02=1 11-03=0 11-04=0 11-05=0"},
		{"no remaining after today", shanghai, "2026-11-01", "2026-11-05", "2026-11-02",
			[]domain.WorkItem{done(t, completed, "2026-11-01T02:00:00Z"), in(started, nil, nil)},
			2, "11-01=1 11-02=1 11-03=- 11-04=- 11-05=-"},
		{"upcoming: no remaining", shanghai, "2026-11-01", "2026-11-05", "2026-10-31",
			[]domain.WorkItem{in(started, nil, nil)},
			1, "11-01=- 11-02=- 11-03=- 11-04=- 11-05=-"},
		{"ended: remaining every day", shanghai, "2026-11-01", "2026-11-05", "2026-11-20",
			[]domain.WorkItem{done(t, completed, "2026-11-03T04:00:00Z"), in(unstarted, nil, nil)},
			2, "11-01=2 11-02=2 11-03=1 11-04=1 11-05=1"},
		{"completed before the start: from the first day", shanghai, "2026-11-01", "2026-11-05", "2026-11-03",
			[]domain.WorkItem{done(t, completed, "2026-10-20T04:00:00Z"), in(started, nil, nil)},
			2, "11-01=1 11-02=1 11-03=1 11-04=- 11-05=-"},
		{"completed without completed_at: from the first day", shanghai, "2026-11-01", "2026-11-05", "2026-11-03",
			[]domain.WorkItem{done(t, completed2, ""), in(started, nil, nil)},
			2, "11-01=1 11-02=1 11-03=1 11-04=- 11-05=-"},
		{"a completed_at outside the completed group burns nothing", shanghai, "2026-11-01", "2026-11-05", "2026-11-05",
			[]domain.WorkItem{done(t, started, "2026-11-02T04:00:00Z"), done(t, backlog, "2026-10-20T04:00:00Z"),
				done(t, completed, "2026-11-03T04:00:00Z")},
			3, "11-01=3 11-02=3 11-03=2 11-04=2 11-05=2"},
		{"cancelled: out of the scope", shanghai, "2026-11-01", "2026-11-05", "2026-11-05",
			[]domain.WorkItem{in(cancelled, nil, nil), done(t, cancelled, "2026-11-02T04:00:00Z"),
				done(t, completed, "2026-11-02T04:00:00Z"), in(started, nil, nil)},
			2, "11-01=2 11-02=1 11-03=1 11-04=1 11-05=1"},
		{"completed after the end burns on no day", shanghai, "2026-11-01", "2026-11-05", "2026-11-10",
			[]domain.WorkItem{done(t, completed, "2026-11-07T04:00:00Z"), in(started, nil, nil)},
			2, "11-01=2 11-02=2 11-03=2 11-04=2 11-05=2"},
		{"a one-day span", shanghai, "2026-11-01", "2026-11-01", "2026-11-01",
			[]domain.WorkItem{done(t, completed, "2026-11-01T03:00:00Z"), in(started, nil, nil)},
			2, "11-01=1"},
		{"no work item", shanghai, "2026-11-01", "2026-11-05", "2026-11-03",
			nil,
			0, "11-01=0 11-02=0 11-03=0 11-04=- 11-05=-"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ProgressOf(spanned(t, tt.zone, tt.start, tt.end, tt.today, tt.items...))
			if err != nil {
				t.Fatal(err)
			}
			b := got.Burndown
			if b == nil {
				t.Fatal("no burndown")
			}
			if b.StartDate != *day(t, tt.start) || b.EndDate != *day(t, tt.end) || b.Scope != tt.scope {
				t.Fatalf("burndown %s to %s of scope %d; want %s to %s of %d", b.StartDate, b.EndDate, b.Scope,
					tt.start, tt.end, tt.scope)
			}
			if got := points(b); got != tt.points {
				t.Fatalf("points = %s; want %s", got, tt.points)
			}
		})
	}
}

// Without both days, a draft cycle or a module missing its start or its target date has no burndown (M6 P4 spec 2.1),
// and the counts and the distributions are as ever. Nor has a start after the end, which the cycles' and the modules'
// checks refuse.
func TestProgressBurndownNeedsBothDays(t *testing.T) {
	for _, tt := range []struct{ name, start, end string }{
		{"a draft", "", ""},
		{"a start alone", "2026-11-01", ""},
		{"an end alone", "", "2026-11-05"},
		{"a start after the end", "2026-11-03", "2026-11-01"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ProgressOf(spanned(t, "Asia/Shanghai", tt.start, tt.end, "2026-11-03", in(started, nil, nil)))
			if err != nil {
				t.Fatal(err)
			}
			if got.Burndown != nil {
				t.Fatalf("burndown %s", points(got.Burndown))
			}
			if got.Total != 1 || len(got.Assignees) != 1 || len(got.Labels) != 1 {
				t.Fatalf("progress %+v", got)
			}
		})
	}
}

// A span, the end less the start, of up to BurndownDays (3660 days) has a point each day, 3661 at most; a longer one
// has no burndown, the counts as ever (M6 P4 spec R4).
func TestProgressBurndownSpan(t *testing.T) {
	start := *day(t, "2026-01-01")
	end := func(days int) string { return shared.DateOf(start.Time().AddDate(0, 0, days)).String() }
	longest, err := domain.ProgressOf(spanned(t, "Asia/Shanghai", start.String(), end(domain.BurndownDays), "2026-11-03",
		in(started, nil, nil)))
	if err != nil {
		t.Fatal(err)
	}
	b := longest.Burndown
	if b == nil {
		t.Fatal("no burndown over 3660 days")
	}
	if n := len(b.Points); n != domain.BurndownDays+1 || b.Points[0].Date != start || b.Points[n-1].Date.String() != end(domain.BurndownDays) {
		t.Fatalf("%d points; want 3661 from %s to %s", n, start, end(domain.BurndownDays))
	}
	for i, p := range b.Points {
		if want := shared.DateOf(start.Time().AddDate(0, 0, i)); p.Date != want {
			t.Fatalf("point %d on %s; want %s", i, p.Date, want)
		}
	}
	tooLong, err := domain.ProgressOf(spanned(t, "Asia/Shanghai", start.String(), end(domain.BurndownDays+1), "2026-11-03",
		in(started, nil, nil)))
	if err != nil {
		t.Fatal(err)
	}
	if tooLong.Burndown != nil || tooLong.Total != 1 {
		t.Fatalf("over 3661 days: burndown %v, total %d", tooLong.Burndown != nil, tooLong.Total)
	}
}
