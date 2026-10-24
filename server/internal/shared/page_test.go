package shared

import (
	"errors"
	"reflect"
	"testing"
)

func TestPageSize(t *testing.T) {
	for _, tt := range []struct {
		limit *int
		want  int
	}{{nil, 50}, {pageLimit(1), 1}, {pageLimit(100), 100}} {
		if got, err := PageSize(tt.limit); err != nil || got != tt.want {
			t.Errorf("PageSize(%v) = %d, %v; want %d", tt.limit, got, err, tt.want)
		}
	}
	for _, limit := range []int{0, 101, -1} {
		got, err := PageSize(&limit)
		var problem *Error
		want := &Error{Kind: KindInvalid, Code: "validation_failed", Detail: "The request has invalid values.", Fields: []FieldError{{Field: "limit", Code: "out_of_range", Message: "must be between 1 and 100"}}}
		if got != 0 || !errors.As(err, &problem) || !reflect.DeepEqual(problem, want) || problem.ProblemStatus() != 422 {
			t.Errorf("PageSize(%d) = %d, %#v; want 0, %#v (422)", limit, got, err, want)
		}
	}
}

func pageLimit(v int) *int { return &v }
