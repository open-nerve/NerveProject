package domain

import (
	"errors"
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A work item under a work item of an ancestry that holds it would be
// under itself (M4 design 3.9): the parent itself, or one above it, but
// not one beside or below it.
func TestLoops(t *testing.T) {
	item, parent, grandparent, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	ancestry := []uuid.UUID{parent, grandparent}
	for _, tt := range []struct {
		name string
		id   uuid.UUID
		want bool
	}{{"the parent", parent, true}, {"one above it", grandparent, true}, {"another", other, false}, {"the work item", item, false}} {
		if got := Loops(tt.id, ancestry); got != tt.want {
			t.Errorf("%s: Loops() = %v, want %v", tt.name, got, tt.want)
		}
	}
	if Loops(item, nil) {
		t.Error("Loops() under no ancestry = true, want false")
	}
}

// An addition of sub-items names one work item at least and a hundred at
// most, each once (M4 design 3.9): none is refused at sub_issue_ids, a
// hundred and one too, alone, and each work item named again at its index;
// one and a hundred are accepted.
func TestCheckSubIssueIDs(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	for _, tt := range []struct {
		name string
		ids  []uuid.UUID
		want []shared.FieldError
	}{
		{"one", []uuid.UUID{a}, nil},
		{"a hundred", ids(MaxSubIssues), nil},
		{"none", nil, []shared.FieldError{{Field: "sub_issue_ids", Code: "too_short", Message: "must name a work item"}}},
		{"a hundred and one, one twice", append(ids(MaxSubIssues), a, a)[1:],
			[]shared.FieldError{{Field: "sub_issue_ids", Code: "too_long", Message: "must name at most 100"}}},
		{"two named again", []uuid.UUID{a, b, a, b}, []shared.FieldError{
			{Field: "sub_issue_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "sub_issue_ids[3]", Code: "duplicate", Message: "is listed before"}}},
	} {
		err := CheckSubIssueIDs(tt.ids)
		var se *shared.Error
		if tt.want == nil && err != nil || tt.want != nil && (!errors.As(err, &se) || se.Code != "validation_failed" ||
			!reflect.DeepEqual(se.Fields, tt.want)) {
			t.Errorf("%s: CheckSubIssueIDs() = %v, want %+v", tt.name, err, tt.want)
		}
	}
	if got, want := SubIssueNotAllowed(4), (shared.FieldError{Field: "sub_issue_ids[4]", Code: "not_allowed",
		Message: "must be a work item of the project that is not archived, and neither this one nor above it"}); got != want {
		t.Errorf("SubIssueNotAllowed(4) = %+v, want %+v", got, want)
	}
}
