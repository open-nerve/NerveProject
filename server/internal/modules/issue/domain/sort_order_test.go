package domain

import (
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestIssueSortOrderPatch(t *testing.T) {
	for _, value := range []float64{0, math.Copysign(0, -1), -12.5, math.MaxFloat64, math.SmallestNonzeroFloat64, math.Nextafter(1, 2)} {
		p := IssuePatch{SortOrder: &value}
		got, err := CheckIssuePatch(p)
		if err != nil || !reflect.DeepEqual(got, p) {
			t.Fatalf("finite %g: %+v, %v", value, got, err)
		}
		c := ChangesOf(Issue{SortOrder: 65535}, p)
		if c.SortOrder == nil || *c.SortOrder != (Changed[float64]{Old: 65535, New: value}) || c.None() {
			t.Fatalf("lost sort diff: %+v", c)
		}
		if c := ChangesOf(Issue{SortOrder: value}, p); !c.None() || c.SortOrder != nil {
			t.Fatalf("same sort is a change: %+v", c)
		}
	}
	if c := ChangesOf(Issue{SortOrder: 19}, IssuePatch{}); !c.None() || c.SortOrder != nil {
		t.Fatal("omission changes sort", c)
	}
	if c := ChangesOf(Issue{SortOrder: 0}, IssuePatch{SortOrder: ptr(math.Copysign(0, -1))}); !c.None() {
		t.Fatal("signed zero changed", c)
	}
	if (Changes{SortOrder: &Changed[float64]{Old: 1, New: 0}}).None() {
		t.Fatal("sort alone must count as change")
	}
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := CheckIssuePatch(IssuePatch{SortOrder: &value, Name: ptr("")})
		var p *shared.Error
		if !errors.As(err, &p) || p.Code != "validation_failed" || len(p.Fields) != 2 {
			t.Fatalf("want both invalid name and sort for %g: %v", value, err)
		}
		found := false
		for _, f := range p.Fields {
			if f.Field == "sort_order" && f.Code == shared.FieldInvalidFormat {
				found = true
			}
		}
		if !found {
			t.Fatal("missing finite error", p.Fields)
		}
	}
}
