package domain

import (
	"strings"
	"testing"
)

// A zone that does not load fails the loading and is named in the error:
// the instance module stops startup with it, no request meets it.
func TestLoadTimezonesFailsOnAnUnknownZone(t *testing.T) {
	zones, err := loadTimezones([][2]string{{"Beijing", "Asia/Shanghai"}, {"Nowhere", "Nowhere/Nothing"}})

	if err == nil || !strings.HasPrefix(err.Error(), "time zone Nowhere/Nothing: ") || zones.places != nil {
		t.Errorf("loadTimezones() = %+v, %v; want no zones and an error naming Nowhere/Nothing", zones, err)
	}
}
