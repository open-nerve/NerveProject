package shared_test

import (
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestValidTimezone(t *testing.T) {
	for name, want := range map[string]bool{
		"UTC":              true,
		"Asia/Shanghai":    true,
		"America/New_York": true,
		"Mars/Olympus":     false,
		"":                 false, // LoadLocation takes it for UTC
		"Local":            false, // the host's zone
	} {
		if got := shared.ValidTimezone(name); got != want {
			t.Errorf("ValidTimezone(%q) = %v, want %v", name, got, want)
		}
	}
}
