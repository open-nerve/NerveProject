package bodyshape

import (
	"errors"
	"fmt"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// At most maxProblems problems come back, and the same ones every time: the
// first in the body for the ambiguities, the first by name for the
// structure (the members are written in reverse order here).
func TestCheckListsAtMostSixteenProblems(t *testing.T) {
	var twice, undeclared []string
	var wantTwice, wantUndeclared []FieldError
	for i := range 20 {
		name := fmt.Sprintf("x%02d", i)
		twice = append(twice, `"`+name+`":1,"`+name+`":2`)
		undeclared = slices.Insert(undeclared, 0, `"`+name+`":1`)
		if i < 16 {
			wantTwice = append(wantTwice, FieldError{name, "duplicate"})
			wantUndeclared = append(wantUndeclared, FieldError{name, "not_allowed"})
		}
	}
	const valid = `"name":"a","nested":{"a":"y"},`
	tests := []struct {
		name string
		body string
		want []FieldError
	}{
		{"twenty names twice", `{` + valid + strings.Join(twice, ",") + `}`, wantTwice},
		{"twenty undeclared names", `{` + valid + strings.Join(undeclared, ",") + `}`, wantUndeclared},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var shape *Error
			if err := things().Check(pattern, []byte(tt.body)); !errors.As(err, &shape) {
				t.Fatalf("Check = %v, want a *Error", err)
			}
			if !slices.Equal(shape.Fields, tt.want) {
				t.Errorf("Check = %v, want %v", shape.Fields, tt.want)
			}
		})
	}
}

// A body can nest values as deep as json.Valid allows, and have a problem
// every few bytes under a long name. Check keeps the path as a stack and
// lists at most maxProblems problems, so what it allocates grows with the
// body. A path string for every value costs the square of the depth, and
// every problem with its path the number of problems times the name: each
// body here makes such code allocate hundreds of megabytes, and gigabytes at
// the 1 MiB body limit (M2 design 3.11). Not parallel: TotalAlloc counts the
// whole process.
func TestCheckCostsAboutTheBody(t *testing.T) {
	// An open map of closed objects: a client's key is part of every path under it.
	groups := &Table{
		Nodes: []Node{
			{Types: Object, Extra: 1, Items: Open, Props: map[string]int{}},
			{Types: Object, Extra: Closed, Items: Open, Props: map[string]int{}},
		},
		Roots: map[string]int{pattern: 0},
	}
	long := strings.Repeat("n", 1<<16)
	var members []string
	for i := range 8000 {
		members = append(members, fmt.Sprintf(`"m%04d":1`, i))
	}
	tests := []struct {
		name  string
		table *Table
		body  string
	}{
		{"objects nested as deep as JSON allows", things(),
			strings.Repeat(`{"nnnnnnnnnnnnnnnn":`, 10000) + "1" + strings.Repeat("}", 10000)},
		{"arrays nested as deep as JSON allows", things(), strings.Repeat("[", 10000) + strings.Repeat("]", 10000)},
		{"strings that are not UTF-8 under a long name", things(),
			`{"` + long + `":[` + strings.Repeat("\"\xff\",", 7999) + "\"\xff\"]}"},
		{"undeclared members under a long map key", groups, `{"` + long + `":{` + strings.Join(members, ",") + `}}`},
	}
	for _, tt := range tests {
		runtime.GC()
		var before, after runtime.MemStats
		runtime.ReadMemStats(&before)
		err := tt.table.Check(pattern, []byte(tt.body))
		runtime.ReadMemStats(&after)

		var shape *Error
		if !errors.As(err, &shape) {
			t.Fatalf("%s: Check = %v, want a *Error", tt.name, err)
		}
		if got := after.TotalAlloc - before.TotalAlloc; got > 64<<20 {
			t.Errorf("%s: Check of %d bytes allocated %d MiB, want at most 64", tt.name, len(tt.body), got>>20)
		}
		t.Logf("%s: %d bytes, %d problems, allocated %d KiB", tt.name, len(tt.body), len(shape.Fields),
			(after.TotalAlloc-before.TotalAlloc)>>10)
	}
}
