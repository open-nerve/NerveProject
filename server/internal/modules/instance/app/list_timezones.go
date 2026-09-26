package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// ListTimezones lists the time zones the web app offers:
// GET /api/v0/timezones.
type ListTimezones struct {
	zones domain.Timezones
	clock Clock
}

// NewListTimezones returns the use case over zones, loaded once when the
// module is built.
func NewListTimezones(zones domain.Timezones, clock Clock) *ListTimezones {
	return &ListTimezones{zones: zones, clock: clock}
}

// Execute returns the time zones with their offsets at the clock's now.
func (uc *ListTimezones) Execute() []domain.Timezone {
	return uc.zones.At(uc.clock.Now())
}
