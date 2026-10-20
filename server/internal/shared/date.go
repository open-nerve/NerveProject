package shared

import (
	"cmp"
	"fmt"
	"time"
)

// Date is a day of the calendar, without a time of day or a time zone: a
// work item's start and target dates (M4 design 3.7, 6.1). Its text is
// YYYY-MM-DD, of a day from 0001-01-01 to 9999-12-31; in JSON, a string.
// The zero Date is no day.
type Date struct {
	year  int
	month time.Month
	day   int
}

// ParseDate reads s, a day as YYYY-MM-DD: four digits of a year from 0001,
// two of a month, and two of a day that month has. Anything else is an
// error: another layout, a time or a zone after the day, white space
// around it. The platform's body check reads a date by the same rule
// (bodyshape's FormatDate), which may not import this package.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse(time.DateOnly, s)
	if err != nil || t.Year() < 1 {
		return Date{}, fmt.Errorf("%q is not a day as YYYY-MM-DD", s)
	}
	return DateOf(t), nil
}

// DateOf is the day t falls on, where t is: a date column, as the database
// driver reads it at midnight UTC.
func DateOf(t time.Time) Date {
	y, m, d := t.Date()
	return Date{year: y, month: m, day: d}
}

// Time is d's midnight UTC, as a date column is written.
func (d Date) Time() time.Time {
	return time.Date(d.year, d.month, d.day, 0, 0, 0, 0, time.UTC)
}

// String is d as YYYY-MM-DD.
func (d Date) String() string {
	return fmt.Sprintf("%04d-%02d-%02d", d.year, d.month, d.day)
}

// Compare is -1 when d is before e, +1 when it is after, and 0 when they are
// the same day.
func (d Date) Compare(e Date) int {
	return cmp.Or(cmp.Compare(d.year, e.year), cmp.Compare(d.month, e.month), cmp.Compare(d.day, e.day))
}

// MarshalText writes d as YYYY-MM-DD, its JSON string.
func (d Date) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

// UnmarshalText reads text as ParseDate does.
func (d *Date) UnmarshalText(text []byte) error {
	parsed, err := ParseDate(string(text))
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}
