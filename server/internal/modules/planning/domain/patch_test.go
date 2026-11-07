package domain_test

import (
	"errors"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/planning/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func ptr[T any](v T) *T { return &v }

// dayPtr is the day s, YYYY-MM-DD.
func dayPtr(t *testing.T, s string) *shared.Date {
	t.Helper()
	d, err := shared.ParseDate(s)
	if err != nil {
		t.Fatal(err)
	}
	return &d
}

// The values a change carries are checked before anything is read (M6 P1 spec R2): the name cleaned, empty, too long
// or with NUL refused, a description with NUL refused, a sort order that is no finite number refused; every problem at
// once. Dates are held to the cycle's under the lock (MergeDates), not here.
func TestCheckCyclePatch(t *testing.T) {
	got, err := domain.CheckCyclePatch(domain.CyclePatch{Name: ptr("  Sprint 2 "), SortOrder: ptr(-1.5), SetStart: true})
	if err != nil || *got.Name != "Sprint 2" || *got.SortOrder != -1.5 || !got.SetStart {
		t.Fatalf("a clean change: %+v, %v", got, err)
	}
	_, err = domain.CheckCyclePatch(domain.CyclePatch{Name: ptr("  "), Description: ptr("a\x00b"), SortOrder: ptr(math.Inf(1))})
	var e *shared.Error
	if !errors.As(err, &e) || len(e.Fields) != 3 || e.Fields[0].Field != "name" || e.Fields[0].Code != shared.FieldTooShort ||
		e.Fields[1].Field != "description" || e.Fields[1].Code != shared.FieldInvalidFormat ||
		e.Fields[2].Field != "sort_order" || e.Fields[2].Code != shared.FieldInvalidFormat {
		t.Fatalf("every problem: %v", err)
	}
	if _, err := domain.CheckCyclePatch(domain.CyclePatch{Name: ptr(strings.Repeat("x", 256))}); err == nil {
		t.Fatal("a name of 256 characters accepted")
	}
	if _, err := domain.CheckCyclePatch(domain.CyclePatch{SortOrder: ptr(math.NaN())}); err == nil {
		t.Fatal("NaN accepted")
	}
}

// OnlySortOrder: all a completed cycle takes (R5) — a change of nothing, or of the sort order alone; any other field
// given, even as it is, is more.
func TestOnlySortOrder(t *testing.T) {
	for _, tt := range []struct {
		p    domain.CyclePatch
		want bool
	}{
		{domain.CyclePatch{}, true},
		{domain.CyclePatch{SortOrder: ptr(1.0)}, true},
		{domain.CyclePatch{Name: ptr("x")}, false},
		{domain.CyclePatch{Description: ptr("")}, false},
		{domain.CyclePatch{SetStart: true}, false},
		{domain.CyclePatch{SetEnd: true}, false},
	} {
		if got := tt.p.OnlySortOrder(); got != tt.want {
			t.Errorf("%+v.OnlySortOrder() = %t, want %t", tt.p, got, tt.want)
		}
	}
}

// Apply: the cycle as the change leaves it, and whether it changes anything: a field given as it is changes
// nothing. Its dates are the ones it gives, each else the cycle's, both or neither, the start not after the end
// (R4); a problem is refused at the date it gives alone, or, when it gives both, where CheckDates puts it.
func TestApply(t *testing.T) {
	dated := domain.Cycle{Name: "Sprint", Description: "Ship", SortOrder: 3,
		Dates: domain.Dates{Start: dayPtr(t, "2026-11-02"), End: dayPtr(t, "2026-11-13")}}
	draft := domain.Cycle{Name: "Draft", SortOrder: 3}
	for _, tt := range []struct {
		name    string
		c       domain.Cycle
		p       domain.CyclePatch
		want    domain.Cycle
		changed bool
	}{
		{"nothing", dated, domain.CyclePatch{}, dated, false},
		{"each as it is", dated, domain.CyclePatch{Name: ptr("Sprint"), Description: ptr("Ship"), SortOrder: ptr(3.0), SetStart: true,
			Start: dayPtr(t, "2026-11-02"), SetEnd: true, End: dayPtr(t, "2026-11-13")}, dated, false},
		{"each changed", dated, domain.CyclePatch{Name: ptr("Two"), Description: ptr(""), SortOrder: ptr(-7.5), SetStart: true,
			Start: dayPtr(t, "2026-11-01"), SetEnd: true, End: dayPtr(t, "2026-11-20")},
			domain.Cycle{Name: "Two", SortOrder: -7.5, Dates: domain.Dates{Start: dayPtr(t, "2026-11-01"), End: dayPtr(t, "2026-11-20")}}, true},
		{"the start alone", dated, domain.CyclePatch{SetStart: true, Start: dayPtr(t, "2026-11-13")},
			domain.Cycle{Name: "Sprint", Description: "Ship", SortOrder: 3, Dates: domain.Dates{Start: dayPtr(t, "2026-11-13"), End: dayPtr(t, "2026-11-13")}}, true},
		{"the end alone", dated, domain.CyclePatch{SetEnd: true, End: dayPtr(t, "2026-11-30")},
			domain.Cycle{Name: "Sprint", Description: "Ship", SortOrder: 3, Dates: domain.Dates{Start: dayPtr(t, "2026-11-02"), End: dayPtr(t, "2026-11-30")}}, true},
		{"both cleared", dated, domain.CyclePatch{SetStart: true, SetEnd: true},
			domain.Cycle{Name: "Sprint", Description: "Ship", SortOrder: 3}, true},
		{"a draft dated", draft, domain.CyclePatch{SetStart: true, Start: dayPtr(t, "2026-11-02"), SetEnd: true, End: dayPtr(t, "2026-11-02")},
			domain.Cycle{Name: "Draft", SortOrder: 3, Dates: domain.Dates{Start: dayPtr(t, "2026-11-02"), End: dayPtr(t, "2026-11-02")}}, true},
	} {
		got, changed, err := tt.p.Apply(tt.c)
		if err != nil || changed != tt.changed || !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s: %+v, %t, %v; want %+v, %t", tt.name, got, changed, err, tt.want, tt.changed)
		}
	}
	for _, tt := range []struct {
		name        string
		c           domain.Cycle
		p           domain.CyclePatch
		field, code string
	}{
		{"the start alone after the end", dated, domain.CyclePatch{SetStart: true, Start: dayPtr(t, "2026-11-14")}, "start_date", "not_allowed"},
		{"the end alone before the start", dated, domain.CyclePatch{SetEnd: true, End: dayPtr(t, "2026-11-01")}, "end_date", "not_allowed"},
		{"the start alone cleared", dated, domain.CyclePatch{SetStart: true}, "start_date", "not_allowed"},
		{"the end alone cleared", dated, domain.CyclePatch{SetEnd: true}, "end_date", "not_allowed"},
		{"a draft's start alone", draft, domain.CyclePatch{SetStart: true, Start: dayPtr(t, "2026-11-02")}, "start_date", "not_allowed"},
		{"a draft's end alone", draft, domain.CyclePatch{SetEnd: true, End: dayPtr(t, "2026-11-02")}, "end_date", "not_allowed"},
		{"both, reversed", dated, domain.CyclePatch{SetStart: true, Start: dayPtr(t, "2026-11-20"), SetEnd: true, End: dayPtr(t, "2026-11-19")},
			"end_date", "not_allowed"},
		{"both, the end cleared", dated, domain.CyclePatch{SetStart: true, Start: dayPtr(t, "2026-11-20"), SetEnd: true}, "end_date", "required"},
	} {
		_, _, err := tt.p.Apply(tt.c)
		var e *shared.Error
		if !errors.As(err, &e) || e.Code != shared.CodeValidationFailed || len(e.Fields) != 1 || e.Fields[0].Field != tt.field || string(e.Fields[0].Code) != tt.code {
			t.Errorf("%s: %v, want 422 at %s %s", tt.name, err, tt.field, tt.code)
		}
	}
}
