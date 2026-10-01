package domain

import "testing"

// SortOrderFirst puts a new project before every other of the member's in
// the workspace, or at the default when he has none (M3 design 3.18): 10000
// before the least of his places, though it be past the default.
func TestSortOrderFirst(t *testing.T) {
	for _, tt := range []struct {
		lowest *float64
		want   float64
	}{{nil, 65535}, {ptr(65535.0), 55535}, {ptr(-2.5), -10002.5}, {ptr(0.0), -10000}, {ptr(70000.0), 60000}} {
		if got := SortOrderFirst(tt.lowest); got != tt.want {
			t.Errorf("SortOrderFirst(%v) = %v, want %v", tt.lowest, got, tt.want)
		}
	}
}
