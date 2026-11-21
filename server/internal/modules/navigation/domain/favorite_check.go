package domain

import (
	"math"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// NewFavorite is a favorite as createFavorite's body gives it (P3c spec
// 2.4): its kind as given, and each other field nil when it is not given.
// It has no project: the use case finds it from the target (R8).
type NewFavorite struct {
	Kind     FavoriteKind
	EntityID *uuid.UUID
	Name     *string
	ParentID *uuid.UUID
}

// CheckNewFavorite is n with a folder's name without its surrounding white
// space; or every problem of n that needs no read, in the body's order
// (spec 2.4, 2.5, 2.7): a kind none of the five, which is its one problem,
// the rules of its other fields being a kind's; a favorite of a target
// without its entity_id or with a name; a folder with an entity_id,
// without a name or with one cleanName refuses, or with a parent_id, no
// folder holding a folder (one level, D42). The use case answers them in
// one 422 before it reads anything. Whether the target and the parent may
// be is its own to say, under its locks.
func CheckNewFavorite(n NewFavorite) (NewFavorite, []shared.FieldError) {
	if !n.Kind.valid() {
		return NewFavorite{}, []shared.FieldError{{Field: "entity_type", Code: shared.FieldInvalidFormat,
			Message: "must be one of project, cycle, module, view, folder"}}
	}
	var found []*shared.FieldError
	folder := n.Kind == FavoriteFolder
	switch {
	case !folder && n.EntityID == nil:
		found = append(found, &shared.FieldError{Field: "entity_id", Code: shared.FieldRequired, Message: "is required for a favorite of a target"})
	case folder && n.EntityID != nil:
		found = append(found, &shared.FieldError{Field: "entity_id", Code: shared.FieldNotAllowed, Message: "is not given for a folder"})
	}
	found = append(found, n.checkName())
	if folder && n.ParentID != nil {
		found = append(found, noParent())
	}
	if problems := problemsOf(found); problems != nil {
		return NewFavorite{}, problems
	}
	return n, nil
}

// checkName is the problem of n's name, by its kind, or nil; a folder's
// name is put in its place without its surrounding white space.
func (n *NewFavorite) checkName() *shared.FieldError {
	switch {
	case n.Kind != FavoriteFolder && n.Name != nil:
		return folderOnly("name")
	case n.Kind != FavoriteFolder:
		return nil
	case n.Name == nil:
		return &shared.FieldError{Field: "name", Code: shared.FieldRequired, Message: "is required for a folder"}
	}
	name, problem := cleanName(*n.Name)
	n.Name = &name
	return problem
}

// FavoritePatch is a change of a favorite (updateFavorite's body, spec
// 2.4): SetParentID tells whether parent_id is given, ParentID its value,
// nil for null, out of its folder; Sequence and Name are nil when they are
// not given. A field given replaces the favorite's.
type FavoritePatch struct {
	SetParentID bool
	ParentID    *uuid.UUID
	Sequence    *float64
	Name        *string
}

// CheckFavoritePatch is p with its name without its surrounding white
// space; or every problem of p as a change of a favorite of kind k, in the
// body's order (spec 2.4, 2.7): a folder given a parent, null being the
// none it has; a sequence that is no finite number; a name given to a
// favorite of a target, or a folder's that cleanName refuses. The use case
// answers them in one 422 once it has read the favorite, whose kind it
// checks by, and before its locks.
func CheckFavoritePatch(p FavoritePatch, k FavoriteKind) (FavoritePatch, []shared.FieldError) {
	var found []*shared.FieldError
	if k == FavoriteFolder && p.ParentID != nil {
		found = append(found, noParent())
	}
	if p.Sequence != nil && (math.IsNaN(*p.Sequence) || math.IsInf(*p.Sequence, 0)) {
		found = append(found, &shared.FieldError{Field: "sequence", Code: shared.FieldInvalidFormat, Message: "must be a finite number"})
	}
	switch {
	case p.Name != nil && k != FavoriteFolder:
		found = append(found, folderOnly("name"))
	case p.Name != nil:
		name, problem := cleanName(*p.Name)
		p.Name = &name
		found = append(found, problem)
	}
	if problems := problemsOf(found); problems != nil {
		return FavoritePatch{}, problems
	}
	return p, nil
}

// Apply is f with each field p gives in its place, and whether that changes
// it: a field given as it is changes nothing, nor does {} (spec 2.4); a
// name differing in case alone does. The use case applies a checked patch
// to the favorite it reads under its lock, and writes only a change.
func (p FavoritePatch) Apply(f Favorite) (Favorite, bool) {
	next := f
	if p.SetParentID {
		next.ParentID = p.ParentID
	}
	if p.Sequence != nil {
		next.Sequence = *p.Sequence
	}
	if p.Name != nil {
		next.Name = *p.Name
	}
	return next, next.Sequence != f.Sequence || next.Name != f.Name || !sameID(next.ParentID, f.ParentID)
}

// folderOnly is the problem of field given to a favorite of a target: only
// a folder has it.
func folderOnly(field string) *shared.FieldError {
	return &shared.FieldError{Field: field, Code: shared.FieldNotAllowed, Message: "is only given for a folder"}
}

// noParent is the problem of a parent given to a folder: no folder holds
// one (D42).
func noParent() *shared.FieldError {
	return &shared.FieldError{Field: "parent_id", Code: shared.FieldNotAllowed, Message: "is not given for a folder"}
}

// sameID tells whether a and b are both nil or both the same id.
func sameID(a, b *uuid.UUID) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
