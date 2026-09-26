package app_test

import (
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/instance/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
)

// The offsets are the clock's: Dublin is an hour east of UTC in summer.
func TestListTimezonesAtTheClocksNow(t *testing.T) {
	for _, tt := range []struct {
		at   time.Time
		want string
	}{
		{time.Date(2026, 1, 15, 12, 0, 0, 0, time.UTC), "+00:00"},
		{time.Date(2026, 7, 15, 12, 0, 0, 0, time.UTC), "+01:00"},
	} {
		zones, err := app.NewListTimezones(clocktest.At(tt.at)).Execute()
		if err != nil {
			t.Fatal(err)
		}
		var dublin string
		for _, z := range zones {
			if z.Name == "Europe/Dublin" {
				dublin = z.Offset
			}
		}
		if dublin != tt.want {
			t.Errorf("Dublin at %v: %q, want %s", tt.at, dublin, tt.want)
		}
	}
}
