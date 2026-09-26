package app

import "github.com/open-nerve/NerveProject/server/internal/modules/instance/domain"

// ListTimezones lists the time zones the web app offers:
// GET /api/v0/timezones.
type ListTimezones struct {
	clock Clock
}

// NewListTimezones returns the use case.
func NewListTimezones(clock Clock) *ListTimezones {
	return &ListTimezones{clock: clock}
}

// Execute returns the time zones with their offsets at the clock's now.
func (uc *ListTimezones) Execute() ([]domain.Timezone, error) {
	return domain.Timezones(uc.clock.Now())
}
