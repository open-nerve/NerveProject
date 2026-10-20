package shared_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ParseDate reads a day as YYYY-MM-DD, of a year from 0001 to 9999, and
// nothing else (M4 design 3.7): the first and the last days, the 29th of
// February of a leap year; not one of another year, nor a month's day it
// has not, nor a month 0 or 13, nor the year 0; not one digit for two, nor
// two for four; not another separator, a time or a zone after the day,
// white space around it, a sign, digits that are not ASCII, nor nothing.
func TestParseDate(t *testing.T) {
	for _, s := range []string{"2026-10-19", "0001-01-01", "9999-12-31", "2024-02-29"} {
		d, err := shared.ParseDate(s)
		if err != nil || d.String() != s {
			t.Errorf("ParseDate(%q) = %v, %v; want the day, written back as it was", s, d, err)
		}
	}
	for _, s := range []string{"2026-02-29", "2026-04-31", "2026-00-10", "2026-13-01", "0000-12-31", "2026-1-19", "2026-10-9", "26-10-19",
		"2026/10/19", "2026-10-19T00:00:00Z", "2026-10-19+08:00", " 2026-10-19", "2026-10-19\n", "+2026-10-19", "10000-01-01",
		"２０２６-10-19", ""} {
		if d, err := shared.ParseDate(s); err == nil {
			t.Errorf("ParseDate(%q) = %v, want an error", s, d)
		}
	}
}

// A day is at midnight UTC as a date column is written, and a moment is
// the day it falls on where it is: 23:30 at UTC-5 is that day, though it
// is the next one in UTC.
func TestDateAndTime(t *testing.T) {
	d, _ := shared.ParseDate("2026-10-19")
	if got, want := d.Time(), time.Date(2026, 10, 19, 0, 0, 0, 0, time.UTC); !got.Equal(want) || got.Location() != time.UTC {
		t.Errorf("Time() = %v, want %v", got, want)
	}
	late := time.Date(2026, 10, 19, 23, 30, 0, 0, time.FixedZone("UTC-5", -5*3600))
	if got := shared.DateOf(late); got != d {
		t.Errorf("DateOf(%v) = %v, want %v", late, got, d)
	}
}

// Days compare by year, then month, then day.
func TestDateCompare(t *testing.T) {
	day := func(s string) shared.Date {
		d, err := shared.ParseDate(s)
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	for _, tt := range []struct {
		a, b string
		want int
	}{
		{"2026-10-19", "2026-10-19", 0},
		{"2026-10-19", "2026-10-20", -1},
		{"2026-10-19", "2026-09-30", 1},
		{"2025-12-31", "2026-01-01", -1},
	} {
		if got := day(tt.a).Compare(day(tt.b)); got != tt.want {
			t.Errorf("%s.Compare(%s) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// In JSON a day is its string, read back by ParseDate's rule: a day the
// month has not, or a number, is an error, and null leaves a pointer nil.
func TestDateInJSON(t *testing.T) {
	type body struct {
		On  shared.Date  `json:"on"`
		Due *shared.Date `json:"due"`
	}
	d, _ := shared.ParseDate("2026-10-19")
	out, err := json.Marshal(body{On: d, Due: &d})
	if want := `{"on":"2026-10-19","due":"2026-10-19"}`; err != nil || string(out) != want {
		t.Errorf("Marshal = %s, %v; want %s", out, err, want)
	}
	var in body
	if err := json.Unmarshal([]byte(`{"on":"2026-10-19","due":null}`), &in); err != nil || in.On != d || in.Due != nil {
		t.Errorf("Unmarshal = %+v, %v; want %v and no due", in, err, d)
	}
	for _, text := range []string{`{"on":"2026-02-30"}`, `{"on":20261019}`} {
		if err := json.Unmarshal([]byte(text), new(body)); err == nil {
			t.Errorf("Unmarshal(%s) = nil, want an error", text)
		}
	}
}
