package domain

import (
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// A relation between two work items (M4 design 3.10): one row a pair,
// whatever its type and direction. relates_to and duplicate read the same
// from both ends; blocked_by reads as blocking from the work item that
// blocks, so a relation asked for as blocking is stored the other way
// round, as the other work item's blocked_by (Plane the same). The work
// items may be of two projects of a workspace.

// RelationType is a relation's type as one of its work items reads it.
type RelationType string

// The types a caller names, from his work item (3.10): the page's four.
const (
	RelatesTo RelationType = "relates_to"
	Duplicate RelationType = "duplicate"
	BlockedBy RelationType = "blocked_by"
	Blocking  RelationType = "blocking"
)

// relationTypes are the types a caller may name.
var relationTypes = []RelationType{RelatesTo, Duplicate, BlockedBy, Blocking}

// Reversed is t as the relation's other end reads it: blocked_by and
// blocking each the other; relates_to and duplicate themselves.
func (t RelationType) Reversed() RelationType {
	switch t {
	case BlockedBy:
		return Blocking
	case Blocking:
		return BlockedBy
	}
	return t
}

// MaxRelatedIssues is how many work items an addition of relations names
// at most (M4 design 3.10, 5.1).
const MaxRelatedIssues = 20

// CheckNewRelations checks an addition of relations of the type t to the
// work items ids, before any read (M4 design 3.10): t one of the four, at
// least one work item, at most MaxRelatedIssues, each named once; every
// problem in one 422 validation_failed. Whether each may be related is the
// use case's, under the projects' locks (RelatedNotAllowed,
// RelatedAlready).
func CheckNewRelations(t RelationType, ids []uuid.UUID) error {
	var found []*shared.FieldError
	if !slices.Contains(relationTypes, t) {
		found = append(found, &shared.FieldError{Field: "relation_type", Code: shared.FieldInvalidFormat,
			Message: "must be one of relates_to, duplicate, blocked_by, blocking"})
	}
	if len(ids) == 0 {
		found = append(found, &shared.FieldError{Field: "issue_ids", Code: shared.FieldTooShort, Message: "must name a work item"})
	}
	return invalid(append(found, checkIDs("issue_ids", ids, MaxRelatedIssues)...)...)
}

// RelatedNotAllowed is the problem of the work item at index i of those an
// addition of relations names that the work item may not be related to
// (M4 design 3.10).
func RelatedNotAllowed(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("issue_ids[%d]", i), Code: shared.FieldNotAllowed,
		Message: "must be another work item of the workspace that you see, not archived, in a project that is not archived"}
}

// RelatedAlready is the problem of the work item at index i of those an
// addition of relations names that relates to the work item already, in
// either direction, of any type (M4 design 3.10).
func RelatedAlready(i int) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("issue_ids[%d]", i), Code: shared.FieldDuplicate, Message: "relates to this work item already"}
}

// Relation is a relation as stored (M4 design 4.6): its id; its work item
// at the issue_id end, of the project ProjectID; its work item at the
// related end; its type as stored, never blocking; who made it, and when.
type Relation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	IssueID     uuid.UUID
	RelatedID   uuid.UUID
	Type        RelationType
	CreatedByID *uuid.UUID
	CreatedAt   time.Time
}

// NewRelation is the relation id that the account by makes at at, of the
// type t from the work item from to the work item to, as stored: from's
// row, blocking stored the other way round, as to's blocked_by, to's row.
func NewRelation(id uuid.UUID, from, to Issue, t RelationType, by uuid.UUID, at time.Time) Relation {
	r := Relation{ID: id, WorkspaceID: from.WorkspaceID, ProjectID: from.ProjectID, IssueID: from.ID, RelatedID: to.ID, Type: t,
		CreatedByID: &by, CreatedAt: at}
	if t == Blocking {
		r.ProjectID, r.IssueID, r.RelatedID, r.Type = to.ProjectID, to.ID, from.ID, BlockedBy
	}
	return r
}

// TypeFrom is r's type as its work item id reads it: as stored from its
// issue_id end, reversed from its related end.
func (r Relation) TypeFrom(id uuid.UUID) RelationType {
	if id == r.RelatedID {
		return r.Type.Reversed()
	}
	return r.Type
}

// Other is r's end that is not the work item id.
func (r Relation) Other(id uuid.UUID) uuid.UUID {
	if id == r.RelatedID {
		return r.IssueID
	}
	return r.RelatedID
}

// RelationAdded is the row of an added relation on one of its work items
// (rows 20 and 21 of M4 design 3.13, of the same columns): its type as the
// work item reads it, updated, from "" to the other work item's
// identifier, the other's id the new identifier.
func RelationAdded(t RelationType, other Ref) Activity {
	return Activity{Field: ptr(string(t)), Verb: VerbUpdated, OldValue: ptr(""), NewValue: ptr(other.Identifier), NewIdentifier: &other.ID,
		Comment: "added " + string(t) + " relation"}
}
