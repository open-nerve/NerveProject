package domain

import (
	"testing"
	"time"
	"uuid"
)

func TestDescriptionVersionMergeWindow(t *testing.T) {
	owner := uuid.NewV7()
	at := time.Date(2026, 10, 25, 1, 0, 0, 0, time.UTC)
	v := DescriptionVersion{OwnedByID: owner, CreatedAt: at.Add(-time.Hour), LastSavedAt: at}
	for _, tt := range []struct {
		name  string
		actor uuid.UUID
		delta time.Duration
		want  bool
	}{
		{"same instant", owner, 0, true},
		{"599 seconds", owner, 599 * time.Second, true},
		{"inclusive bound", owner, 600 * time.Second, true},
		{"outside window", owner, 600*time.Second + time.Microsecond, false},
		{"different owner", uuid.NewV7(), time.Second, false},
		{"future saved at", owner, -time.Microsecond, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := CanMergeDescriptionVersion(v, tt.actor, at.Add(tt.delta)); got != tt.want {
				t.Fatalf("CanMergeDescriptionVersion(delta=%s) = %v, want %v", tt.delta, got, tt.want)
			}
		})
	}
	v.LastSavedAt = at.Add(599 * time.Second)
	if !CanMergeDescriptionVersion(v, owner, at.Add(1199*time.Second)) {
		t.Fatal("second save must slide the window from last_saved_at, not created_at")
	}
}
