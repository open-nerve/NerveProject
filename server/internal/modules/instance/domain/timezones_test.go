package domain_test

import (
	"cmp"
	"testing"
	"time"
	_ "time/tzdata" // as in the binary: a zone the host lacks loads from the embedded database

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"
)

// minutes reads an offset ±hh:mm.
func minutes(t *testing.T, offset string) int {
	t.Helper()
	at, err := time.Parse("-07:00", offset)
	if err != nil {
		t.Fatalf("offset %q is not ±hh:mm: %v", offset, err)
	}
	_, seconds := at.Zone()
	return seconds / 60
}

// loaded is Plane's places with their zones loaded.
func loaded(t *testing.T) domain.Timezones {
	t.Helper()
	zones, err := domain.LoadTimezones()
	if err != nil {
		t.Fatal(err)
	}
	return zones
}

// Every zone loads; the list is sorted by offset, then by label.
func TestTimezonesAreSorted(t *testing.T) {
	got := loaded(t).At(time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC))

	if len(got) != 120 {
		t.Fatalf("At() = %d zones, want Plane's 120", len(got))
	}
	for i := 1; i < len(got); i++ {
		a, b := got[i-1], got[i]
		if cmp.Or(cmp.Compare(minutes(t, a.Offset), minutes(t, b.Offset)), cmp.Compare(a.Label, b.Label)) >= 0 {
			t.Errorf("%+v comes before %+v", a, b)
		}
	}
	first, last := domain.Timezone{Label: "American Samoa", Name: "Pacific/Pago_Pago", Offset: "-11:00"},
		domain.Timezone{Label: "Kiritimati Island", Name: "Pacific/Kiritimati", Offset: "+14:00"}
	if got[0] != first || got[len(got)-1] != last {
		t.Errorf("first %+v, last %+v; want %+v, %+v", got[0], got[len(got)-1], first, last)
	}
}

// The offsets are the zones' at the time asked, daylight saving time
// included; a negative offset that is not a whole hour keeps its hour.
func TestTimezoneOffsets(t *testing.T) {
	january, july := time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		label, january, july string
	}{
		{"Marquesas Islands", "-09:30", "-09:30"},
		{"Newfoundland Time (Canada)", "-03:30", "-02:30"},
		{"Pacific Time (US and Canada)", "-08:00", "-07:00"},
		{"Reykjavik", "+00:00", "+00:00"},
		{"Dublin", "+00:00", "+01:00"},
		{"Kolkata", "+05:30", "+05:30"},
		{"Kathmandu", "+05:45", "+05:45"},
		{"Beijing", "+08:00", "+08:00"},
		{"Chatham Islands", "+13:45", "+12:45"},
	}
	zones := loaded(t)
	offsets := func(at time.Time) map[string]string {
		m := map[string]string{}
		for _, z := range zones.At(at) {
			m[z.Label] = z.Offset
		}
		return m
	}
	inJanuary, inJuly := offsets(january), offsets(july)
	for _, tt := range tests {
		if inJanuary[tt.label] != tt.january || inJuly[tt.label] != tt.july {
			t.Errorf("%s: %q in January, %q in July; want %q, %q", tt.label, inJanuary[tt.label], inJuly[tt.label], tt.january, tt.july)
		}
	}
}
