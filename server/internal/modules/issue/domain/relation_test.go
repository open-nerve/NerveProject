package domain

import (
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each type reads from a relation's other end as the page shows it there
// (M4 design 3.10): blocked_by and blocking each as the other, relates_to
// and duplicate as themselves.
func TestRelationTypeReversed(t *testing.T) {
	for t1, t2 := range map[RelationType]RelationType{BlockedBy: Blocking, Blocking: BlockedBy, RelatesTo: RelatesTo, Duplicate: Duplicate} {
		if got := t1.Reversed(); got != t2 {
			t.Errorf("%s.Reversed() = %s, want %s", t1, got, t2)
		}
	}
}

// An addition of relations names one of the four types and one work item
// at least, twenty at most, each once (M4 design 3.10): each type and one
// work item, or twenty, are accepted; Plane's start_before, which the page
// does not show, is refused at relation_type, no work item at issue_ids,
// twenty-one too, alone, and each work item named again at its index;
// every problem in one 422.
func TestCheckNewRelations(t *testing.T) {
	a, b := uuid.NewV7(), uuid.NewV7()
	for _, kind := range []RelationType{RelatesTo, Duplicate, BlockedBy, Blocking} {
		if err := CheckNewRelations(kind, []uuid.UUID{a}); err != nil {
			t.Errorf("CheckNewRelations(%s, one) = %v, want nil", kind, err)
		}
	}
	unknown := shared.FieldError{Field: "relation_type", Code: "invalid_format", Message: "must be one of relates_to, duplicate, blocked_by, blocking"}
	for _, tt := range []struct {
		name string
		kind RelationType
		ids  []uuid.UUID
		want []shared.FieldError
	}{
		{"twenty", BlockedBy, ids(MaxRelatedIssues), nil},
		{"start_before, none", "start_before", nil, []shared.FieldError{unknown,
			{Field: "issue_ids", Code: "too_short", Message: "must name a work item"}}},
		{"twenty-one, one twice", RelatesTo, append(ids(MaxRelatedIssues), a, a)[1:],
			[]shared.FieldError{{Field: "issue_ids", Code: "too_long", Message: "must name at most 20"}}},
		{"blocking in another case, two named again", "Blocking", []uuid.UUID{a, b, a, b}, []shared.FieldError{unknown,
			{Field: "issue_ids[2]", Code: "duplicate", Message: "is listed before"},
			{Field: "issue_ids[3]", Code: "duplicate", Message: "is listed before"}}},
	} {
		err := CheckNewRelations(tt.kind, tt.ids)
		var se *shared.Error
		if tt.want == nil && err != nil || tt.want != nil && (!errors.As(err, &se) || se.Code != "validation_failed" ||
			!reflect.DeepEqual(se.Fields, tt.want)) {
			t.Errorf("%s: CheckNewRelations() = %v, want %+v", tt.name, err, tt.want)
		}
	}
	for got, want := range map[shared.FieldError]shared.FieldError{
		RelatedNotAllowed(3): {Field: "issue_ids[3]", Code: "not_allowed",
			Message: "must be another work item of the workspace that you see, not archived, in a project that is not archived"},
		RelatedAlready(0): {Field: "issue_ids[0]", Code: "duplicate", Message: "relates to this work item already"},
	} {
		if got != want {
			t.Errorf("the problem %+v, want %+v", got, want)
		}
	}
}

// A relation is stored one row a pair (M4 design 3.10): asked for from a
// work item of Web to one of Ops, relates_to, duplicate and blocked_by are
// the row of the work item asked from, of Web, at its issue_id end; blocking
// is the row of the other, of Ops, its blocked_by, the work item asked
// from at its related end. Each end reads the type as asked from it, the
// other end its reverse; each end's other is the other work item.
func TestNewRelation(t *testing.T) {
	acme, web, ops, by := uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	from, to := Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web}, Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: ops}
	at := time.Date(2026, 10, 21, 10, 0, 0, 0, time.UTC)
	for _, kind := range []RelationType{RelatesTo, Duplicate, BlockedBy, Blocking} {
		id := uuid.NewV7()
		r := NewRelation(id, from, to, kind, by, at)
		want := Relation{ID: id, WorkspaceID: acme, ProjectID: web, IssueID: from.ID, RelatedID: to.ID, Type: kind, CreatedByID: &by, CreatedAt: at}
		if kind == Blocking {
			want.ProjectID, want.IssueID, want.RelatedID, want.Type = ops, to.ID, from.ID, BlockedBy
		}
		if !reflect.DeepEqual(r, want) {
			t.Errorf("NewRelation(%s) = %+v, want %+v", kind, r, want)
		}
		if r.TypeFrom(from.ID) != kind || r.TypeFrom(to.ID) != kind.Reversed() || r.Other(from.ID) != to.ID || r.Other(to.ID) != from.ID {
			t.Errorf("%s: from the work item asked from %s and %s, from the other %s and %s; want %s and %s, %s and %s", kind, r.TypeFrom(from.ID),
				r.Other(from.ID), r.TypeFrom(to.ID), r.Other(to.ID), kind, to.ID, kind.Reversed(), from.ID)
		}
	}
}

// An added relation writes a row on each of its work items, of the same
// columns (rows 20 and 21 of M4 design 3.13): the type as the work item
// reads it, each of the four, updated, from "" to the other's identifier,
// the other's id the new identifier, Plane's sentence with the type.
func TestRelationAdded(t *testing.T) {
	other := Ref{ID: uuid.NewV7(), Identifier: "OPS-3"}
	got, want := map[RelationType]Activity{}, map[RelationType]Activity{}
	for kind, comment := range map[RelationType]string{RelatesTo: "added relates_to relation", Duplicate: "added duplicate relation",
		BlockedBy: "added blocked_by relation", Blocking: "added blocking relation"} {
		got[kind] = RelationAdded(kind, other)
		want[kind] = Activity{Field: ptr(string(kind)), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr("OPS-3"),
			NewIdentifier: &other.ID, Comment: comment}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RelationAdded() = %+v, want %+v", got, want)
	}
}

// A deleted relation writes a row on each of its work items, of the same
// columns (rows 22 and 23 of M4 design 3.13): the type as the work item
// reads it, each of the four, deleted, from the other's identifier to "",
// the other's id the old identifier, Plane's sentence with the type.
func TestRelationRemoved(t *testing.T) {
	other := Ref{ID: uuid.NewV7(), Identifier: "OPS-3"}
	got, want := map[RelationType]Activity{}, map[RelationType]Activity{}
	for kind, comment := range map[RelationType]string{RelatesTo: "deleted relates_to relation", Duplicate: "deleted duplicate relation",
		BlockedBy: "deleted blocked_by relation", Blocking: "deleted blocking relation"} {
		got[kind] = RelationRemoved(kind, other)
		want[kind] = Activity{Field: ptr(string(kind)), Verb: VerbDeleted, OldValue: ptr("OPS-3"), NewValue: ptr(""),
			OldIdentifier: &other.ID, Comment: comment}
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("RelationRemoved() = %+v, want %+v", got, want)
	}
}
