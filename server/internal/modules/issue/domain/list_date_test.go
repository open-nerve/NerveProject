package domain

import "testing"

// Catch dropped range bounds, reversed bounds, treating none as a date,
// and accepting a range with no endpoints.
func TestParseListDateFilter(t *testing.T) {
	for _, tc := range []struct {
		value, from, to string
		none            bool
	}{
		{"2026-10-25", "2026-10-25", "2026-10-25", false},
		{"2026-01-01..2026-12-31", "2026-01-01", "2026-12-31", false},
		{"2026-10-25..2026-10-25", "2026-10-25", "2026-10-25", false},
		{"..2026-10-25", "", "2026-10-25", false},
		{"2026-10-25..", "2026-10-25", "", false},
		{"0001-01-01..9999-12-31", "0001-01-01", "9999-12-31", false},
		{"none", "", "", true},
	} {
		t.Run(tc.value, func(t *testing.T) {
			got, err := ParseListDateFilter(tc.value)
			if err != nil {
				t.Fatal(err)
			}
			from, to := "", ""
			if got.From != nil {
				from = got.From.String()
			}
			if got.To != nil {
				to = got.To.String()
			}
			if got.None != tc.none || from != tc.from || to != tc.to {
				t.Errorf("got none=%v from=%q to=%q, want none=%v from=%q to=%q", got.None, from, to, tc.none, tc.from, tc.to)
			}
		})
	}
	for _, value := range []string{"", "..", "2026-12-31..2026-01-01", "2026-10-25..2026-10-26..2026-10-27", "none..2026-10-25", "2026-10-25..none", "2026-02-30", "0000-01-01", "10000-01-01", "2026-1-01", "2026-01-1", "2026-10-25T00:00:00Z", " 2026-10-25", "2026-10-25 ", "NONE"} {
		t.Run("reject_"+value, func(t *testing.T) {
			if got, err := ParseListDateFilter(value); err == nil {
				t.Errorf("must reject %q, got %+v", value, got)
			}
		})
	}
}
