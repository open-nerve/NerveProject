package domain_test

import (
	"strings"
	"testing"
	"time"
	_ "time/tzdata"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func day(t *testing.T, s string) *shared.Date {
	t.Helper()
	d, err := shared.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

// A cycle has both dates or neither, the start not after the end (M6 design 3.1): a date alone is refused at the
// missing one, a start after the end at end_date.
func TestCheckNewDates(t *testing.T) {
	for _, tt := range []struct {
		name       string
		start, end string
		field      string
		code       string
	}{
		{"neither", "", "", "", ""},
		{"one day", "2026-11-07", "2026-11-07", "", ""},
		{"a range", "2026-11-07", "2026-11-20", "", ""},
		{"start alone", "2026-11-07", "", "end_date", shared.FieldRequired},
		{"end alone", "", "2026-11-07", "start_date", shared.FieldRequired},
		{"start after end", "2026-11-08", "2026-11-07", "end_date", shared.FieldNotAllowed},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var d domain.Dates
			if tt.start != "" {
				d.Start = day(t, tt.start)
			}
			if tt.end != "" {
				d.End = day(t, tt.end)
			}
			got := domain.CheckDates(d)
			switch {
			case tt.field == "" && got != nil:
				t.Fatalf("refused: %+v", got)
			case tt.field != "" && (got == nil || got.Field != tt.field || got.Code != tt.code):
				t.Fatalf("got %+v; want %s %s", got, tt.field, tt.code)
			}
		})
	}
}

// A name loses its surrounding white space; empty, longer than 255 characters or with NUL, it is refused.
func TestCleanName(t *testing.T) {
	if got, problem := domain.CleanName(" \tSprint 1\n"); got != "Sprint 1" || problem != nil {
		t.Fatalf("CleanName = %q, %+v", got, problem)
	}
	if got, problem := domain.CleanName(strings.Repeat("迭", 255)); got != strings.Repeat("迭", 255) || problem != nil {
		t.Fatalf("255 characters refused: %+v", problem)
	}
	for _, tt := range []struct{ name, in, code string }{
		{"empty", "", shared.FieldTooShort},
		{"blank", " \t\n", shared.FieldTooShort},
		{"256 characters", strings.Repeat("迭", 256), shared.FieldTooLong},
		{"NUL", "Spr\x00int", shared.FieldInvalidFormat},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, problem := domain.CleanName(tt.in); problem == nil || problem.Field != "name" || problem.Code != tt.code {
				t.Fatalf("got %+v; want name %s", problem, tt.code)
			}
		})
	}
}

// The status is read on the project's calendar day (M6 design 3.1): draft without dates, upcoming before the start,
// current from the start to the end, completed after the end. Each case sits on one side of a day boundary of the
// project's time zone where the UTC day is the other, so reading the day in UTC fails here.
func TestStatusOnTheProjectsDay(t *testing.T) {
	dated := domain.Dates{Start: day(t, "2026-11-07"), End: day(t, "2026-11-10")}
	for _, tt := range []struct {
		zone, now string
		dates     domain.Dates
		want      domain.Status
	}{
		{"Asia/Shanghai", "2026-11-06T15:59:59Z", dated, domain.StatusUpcoming},
		{"Asia/Shanghai", "2026-11-06T16:00:00Z", dated, domain.StatusCurrent},
		{"Asia/Shanghai", "2026-11-10T15:59:59Z", dated, domain.StatusCurrent},
		{"Asia/Shanghai", "2026-11-10T16:00:00Z", dated, domain.StatusCompleted},
		{"America/Los_Angeles", "2026-11-07T07:59:59Z", dated, domain.StatusUpcoming},
		{"America/Los_Angeles", "2026-11-07T08:00:00Z", dated, domain.StatusCurrent},
		{"America/Los_Angeles", "2026-11-11T07:59:59Z", dated, domain.StatusCurrent},
		{"America/Los_Angeles", "2026-11-11T08:00:00Z", dated, domain.StatusCompleted},
		{"Asia/Shanghai", "2026-11-08T00:00:00Z", domain.Dates{}, domain.StatusDraft},
	} {
		t.Run(tt.zone+" "+tt.now, func(t *testing.T) {
			now, err := time.Parse(time.RFC3339, tt.now)
			if err != nil {
				t.Fatal(err)
			}
			today, err := domain.Today(now, tt.zone)
			if err != nil {
				t.Fatal(err)
			}
			if got := domain.StatusOf(tt.dates, today); got != tt.want {
				t.Fatalf("status on %s in %s = %s; want %s", today, tt.zone, got, tt.want)
			}
		})
	}
	if _, err := domain.Today(time.Now(), "Not/AZone"); err == nil {
		t.Fatal("an unknown time zone was read")
	}
}

// Two dated cycles overlap when their closed ranges share a day; a cycle without dates overlaps none, and a cycle is
// not held against itself.
func TestOverlapping(t *testing.T) {
	self := uuid.NewV7()
	candidate := domain.Dates{Start: day(t, "2026-11-07"), End: day(t, "2026-11-10")}
	for _, tt := range []struct {
		name       string
		id         uuid.UUID
		start, end string
		want       bool
	}{
		{"starts on the end day", uuid.NewV7(), "2026-11-10", "2026-11-12", true},
		{"ends on the start day", uuid.NewV7(), "2026-11-01", "2026-11-07", true},
		{"inside", uuid.NewV7(), "2026-11-08", "2026-11-09", true},
		{"around", uuid.NewV7(), "2026-11-01", "2026-11-20", true},
		{"the day after", uuid.NewV7(), "2026-11-11", "2026-11-12", false},
		{"the day before", uuid.NewV7(), "2026-11-01", "2026-11-06", false},
		{"without dates", uuid.NewV7(), "", "", false},
		{"itself", self, "2026-11-07", "2026-11-10", false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			other := domain.DatedCycle{ID: tt.id}
			if tt.start != "" {
				other.Dates = domain.Dates{Start: day(t, tt.start), End: day(t, tt.end)}
			}
			if got := domain.Overlapping(self, candidate, []domain.DatedCycle{other}); got != tt.want {
				t.Fatalf("Overlapping = %t; want %t", got, tt.want)
			}
		})
	}
	all := []domain.DatedCycle{{ID: uuid.NewV7(), Dates: candidate}}
	if domain.Overlapping(self, domain.Dates{}, all) {
		t.Fatal("a cycle without dates overlapped")
	}
}
