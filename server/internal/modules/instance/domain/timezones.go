package domain

import (
	"cmp"
	"fmt"
	"slices"
	"time"
)

// Timezone is one of the time zones the web app offers (M2 design 5.3).
type Timezone struct {
	Label  string // a place in the zone, e.g. "Beijing"
	Name   string // its IANA name, e.g. "Asia/Shanghai"
	Offset string // from UTC at the time asked, e.g. "+08:00" or "-09:30"
}

// timezoneLocations are Plane's places and their zones
// (plane/apps/api/plane/app/views/timezone/base.py:30-178), in its order.
var timezoneLocations = [][2]string{
	{"Midway Island", "Pacific/Midway"},
	{"American Samoa", "Pacific/Pago_Pago"},
	{"Hawaii", "Pacific/Honolulu"},
	{"Aleutian Islands", "America/Adak"},
	{"Marquesas Islands", "Pacific/Marquesas"},
	{"Alaska", "America/Anchorage"},
	{"Gambier Islands", "Pacific/Gambier"},
	{"Pacific Time (US and Canada)", "America/Los_Angeles"},
	{"Baja California", "America/Tijuana"},
	{"Mountain Time (US and Canada)", "America/Denver"},
	{"Arizona", "America/Phoenix"},
	{"Chihuahua, Mazatlan", "America/Chihuahua"},
	{"Central Time (US and Canada)", "America/Chicago"},
	{"Saskatchewan", "America/Regina"},
	{"Guadalajara, Mexico City, Monterrey", "America/Mexico_City"},
	{"Tegucigalpa, Honduras", "America/Tegucigalpa"},
	{"Costa Rica", "America/Costa_Rica"},
	{"Eastern Time (US and Canada)", "America/New_York"},
	{"Lima", "America/Lima"},
	{"Bogota", "America/Bogota"},
	{"Quito", "America/Guayaquil"},
	{"Chetumal", "America/Cancun"},
	{"Caracas (Old Venezuela Time)", "America/Caracas"},
	{"Atlantic Time (Canada)", "America/Halifax"},
	{"Caracas", "America/Caracas"},
	{"Santiago", "America/Santiago"},
	{"La Paz", "America/La_Paz"},
	{"Manaus", "America/Manaus"},
	{"Georgetown", "America/Guyana"},
	{"Bermuda", "Atlantic/Bermuda"},
	{"Newfoundland Time (Canada)", "America/St_Johns"},
	{"Buenos Aires", "America/Argentina/Buenos_Aires"},
	{"Brasilia", "America/Sao_Paulo"},
	{"Greenland", "America/Godthab"},
	{"Montevideo", "America/Montevideo"},
	{"Falkland Islands", "Atlantic/Stanley"},
	{"South Georgia and the South Sandwich Islands", "Atlantic/South_Georgia"},
	{"Azores", "Atlantic/Azores"},
	{"Cape Verde Islands", "Atlantic/Cape_Verde"},
	{"Dublin", "Europe/Dublin"},
	{"Reykjavik", "Atlantic/Reykjavik"},
	{"Lisbon", "Europe/Lisbon"},
	{"Monrovia", "Africa/Monrovia"},
	{"Casablanca", "Africa/Casablanca"},
	{"Central European Time (Berlin, Rome, Paris)", "Europe/Paris"},
	{"West Central Africa", "Africa/Lagos"},
	{"Algiers", "Africa/Algiers"},
	{"Lagos", "Africa/Lagos"},
	{"Tunis", "Africa/Tunis"},
	{"Eastern European Time (Cairo, Helsinki, Kyiv)", "Europe/Kyiv"},
	{"Athens", "Europe/Athens"},
	{"Jerusalem", "Asia/Jerusalem"},
	{"Johannesburg", "Africa/Johannesburg"},
	{"Harare, Pretoria", "Africa/Harare"},
	{"Moscow Time", "Europe/Moscow"},
	{"Baghdad", "Asia/Baghdad"},
	{"Nairobi", "Africa/Nairobi"},
	{"Kuwait, Riyadh", "Asia/Riyadh"},
	{"Tehran", "Asia/Tehran"},
	{"Abu Dhabi", "Asia/Dubai"},
	{"Baku", "Asia/Baku"},
	{"Yerevan", "Asia/Yerevan"},
	{"Astrakhan", "Europe/Astrakhan"},
	{"Tbilisi", "Asia/Tbilisi"},
	{"Mauritius", "Indian/Mauritius"},
	{"Kabul", "Asia/Kabul"},
	{"Islamabad", "Asia/Karachi"},
	{"Karachi", "Asia/Karachi"},
	{"Tashkent", "Asia/Tashkent"},
	{"Yekaterinburg", "Asia/Yekaterinburg"},
	{"Maldives", "Indian/Maldives"},
	{"Chagos", "Indian/Chagos"},
	{"Chennai", "Asia/Kolkata"},
	{"Kolkata", "Asia/Kolkata"},
	{"Mumbai", "Asia/Kolkata"},
	{"New Delhi", "Asia/Kolkata"},
	{"Sri Jayawardenepura", "Asia/Colombo"},
	{"Kathmandu", "Asia/Kathmandu"},
	{"Dhaka", "Asia/Dhaka"},
	{"Almaty", "Asia/Almaty"},
	{"Bishkek", "Asia/Bishkek"},
	{"Thimphu", "Asia/Thimphu"},
	{"Yangon (Rangoon)", "Asia/Yangon"},
	{"Cocos Islands", "Indian/Cocos"},
	{"Bangkok", "Asia/Bangkok"},
	{"Hanoi", "Asia/Ho_Chi_Minh"},
	{"Jakarta", "Asia/Jakarta"},
	{"Novosibirsk", "Asia/Novosibirsk"},
	{"Krasnoyarsk", "Asia/Krasnoyarsk"},
	{"Beijing", "Asia/Shanghai"},
	{"Singapore", "Asia/Singapore"},
	{"Perth", "Australia/Perth"},
	{"Hong Kong", "Asia/Hong_Kong"},
	{"Ulaanbaatar", "Asia/Ulaanbaatar"},
	{"Palau", "Pacific/Palau"},
	{"Eucla", "Australia/Eucla"},
	{"Tokyo", "Asia/Tokyo"},
	{"Seoul", "Asia/Seoul"},
	{"Yakutsk", "Asia/Yakutsk"},
	{"Adelaide", "Australia/Adelaide"},
	{"Darwin", "Australia/Darwin"},
	{"Sydney", "Australia/Sydney"},
	{"Brisbane", "Australia/Brisbane"},
	{"Guam", "Pacific/Guam"},
	{"Vladivostok", "Asia/Vladivostok"},
	{"Tahiti", "Pacific/Tahiti"},
	{"Lord Howe Island", "Australia/Lord_Howe"},
	{"Solomon Islands", "Pacific/Guadalcanal"},
	{"Magadan", "Asia/Magadan"},
	{"Norfolk Island", "Pacific/Norfolk"},
	{"Bougainville Island", "Pacific/Bougainville"},
	{"Chokurdakh", "Asia/Srednekolymsk"},
	{"Auckland", "Pacific/Auckland"},
	{"Wellington", "Pacific/Auckland"},
	{"Fiji Islands", "Pacific/Fiji"},
	{"Anadyr", "Asia/Anadyr"},
	{"Chatham Islands", "Pacific/Chatham"},
	{"Nuku'alofa", "Pacific/Tongatapu"},
	{"Samoa", "Pacific/Apia"},
	{"Kiritimati Island", "Pacific/Kiritimati"},
}

// Timezones are Plane's places with their zones loaded: LoadTimezones
// loads them once, and At gives their offsets at any time.
type Timezones struct {
	places []place
}

// place is a place the web app offers and its loaded zone.
type place struct {
	label, name string
	loc         *time.Location
}

// LoadTimezones loads the zone of each of Plane's places. A zone that does
// not load is an error, for the instance module to stop startup with,
// rather than to answer a request with.
func LoadTimezones() (Timezones, error) {
	return loadTimezones(timezoneLocations)
}

// loadTimezones loads the zones of places, pairs of a label and an IANA name.
func loadTimezones(places [][2]string) (Timezones, error) {
	z := Timezones{places: make([]place, len(places))}
	for i, p := range places {
		loc, err := time.LoadLocation(p[1])
		if err != nil {
			return Timezones{}, fmt.Errorf("time zone %s: %w", p[1], err)
		}
		z.places[i] = place{label: p[0], name: p[1], loc: loc}
	}
	return z, nil
}

// At returns the time zones with their offsets at t, sorted by offset and
// then by label, as Plane sorts them (base.py:209). Plane writes a negative
// offset that is not a whole hour an hour too far west, -09:30 as -10:30
// (base.py:192); here it is -09:30.
func (z Timezones) At(t time.Time) []Timezone {
	type zone struct {
		Timezone
		seconds int
	}
	zones := make([]zone, len(z.places))
	for i, p := range z.places {
		_, seconds := t.In(p.loc).Zone()
		zones[i] = zone{Timezone{Label: p.label, Name: p.name, Offset: offset(seconds)}, seconds}
	}
	slices.SortFunc(zones, func(a, b zone) int {
		return cmp.Or(cmp.Compare(a.seconds, b.seconds), cmp.Compare(a.Label, b.Label))
	})
	out := make([]Timezone, len(zones))
	for i, z := range zones {
		out[i] = z.Timezone
	}
	return out
}

// offset writes seconds east of UTC as ±hh:mm.
func offset(seconds int) string {
	sign := '+'
	if seconds < 0 {
		sign, seconds = '-', -seconds
	}
	return fmt.Sprintf("%c%02d:%02d", sign, seconds/3600, seconds%3600/60)
}
