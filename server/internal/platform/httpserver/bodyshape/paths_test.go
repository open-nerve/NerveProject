package bodyshape

import (
	"slices"
	"testing"
)

// Paths sort byte by byte, but an index against an index by its number
// (M2 handoff 4): a[1].z before a[2].a, though the rest of the path would
// order them the other way, and a[9].z before a[10], before a[10].name; a
// member before an index of the same value ('.' before '['), the shorter
// path first; a name with a '[' and digits in it is ordered as an index
// is. The order is the same from any order of the paths.
func TestComparePaths(t *testing.T) {
	want := []string{"", "[2]", "[10]", "a", "a.b", "a[0]", "a[1].z", "a[2].a", "a[9].z", "a[10]", "a[10].name", "a[10][3]", "a[10][20]",
		"k[9", "k[10", "z"}
	reversed := slices.Clone(want)
	slices.Reverse(reversed)
	for _, from := range [][]string{slices.Clone(want), reversed, {"a[10]", "a[2].a", "z", "k[10", "", "a[0]", "[10]", "a.b", "a[10][20]",
		"a[9].z", "k[9", "a", "a[1].z", "a[10][3]", "[2]", "a[10].name"}} {
		got := slices.SortedFunc(slices.Values(from), comparePaths)
		if !slices.Equal(got, want) {
			t.Errorf("sorted from %q = %q, want %q", from, got, want)
		}
	}
	for _, p := range want {
		if c := comparePaths(p, p); c != 0 {
			t.Errorf("comparePaths(%q, %q) = %d, want 0", p, p, c)
		}
	}
}
