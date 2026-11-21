package domain

import (
	"time"
	"uuid"
)

// FavoriteKind is what a favorite is of (P3c spec 2.3, R4): its target, a
// project, a cycle, a module or a view, a project's or a workspace's
// alike; or nothing, a folder, which holds favorites of targets.
type FavoriteKind string

// The five kinds of favorites.
const (
	FavoriteProject FavoriteKind = "project"
	FavoriteCycle   FavoriteKind = "cycle"
	FavoriteModule  FavoriteKind = "module"
	FavoriteView    FavoriteKind = "view"
	FavoriteFolder  FavoriteKind = "folder"
)

// valid tells whether k is one of the five, as they are written.
func (k FavoriteKind) valid() bool {
	switch k {
	case FavoriteProject, FavoriteCycle, FavoriteModule, FavoriteView, FavoriteFolder:
		return true
	}
	return false
}

// Favorite is one of an account's favorites in a workspace as the module
// stores it (spec 2.3, 2.4). Only he reads and writes it (spec 2.2).
type Favorite struct {
	ID, WorkspaceID uuid.UUID
	// ProjectID is its target's project, the project itself for a
	// project's favorite; nil for a workspace's view's and for a folder.
	ProjectID *uuid.UUID
	UserID    uuid.UUID
	Kind      FavoriteKind
	// EntityID is its target's id; nil for a folder.
	EntityID *uuid.UUID
	// Name is a folder's name; "" for a favorite of a target, which is
	// answered with its target's.
	Name string
	// ParentID is the folder that holds it; nil for one at the top, and
	// for a folder, which no folder holds (one level, D42). A parent is
	// one's own undeleted folder in the same workspace (spec 2.5): the
	// use case finds it so under its lock.
	ParentID *uuid.UUID
	// Sequence places it among his favorites in the workspace, the
	// greatest first (spec 2.4, 2.5): any finite number.
	Sequence             float64
	CreatedAt, UpdatedAt time.Time
}

// sequenceStep is the gap between the greatest sequence of an account's
// favorites in a workspace and his new one's, and firstSequence his first
// one's there: Plane's (D33, PE-7).
const (
	sequenceStep  = 10000
	firstSequence = 65535
)

// SequenceAfter is the sequence of an account's new favorite in a workspace
// whose undeleted favorites there have greatest as their greatest sequence,
// nil when he has none (spec 2.5, D33): one step past it, so the new one is
// listed first, or firstSequence. Two created at once may share it, which
// the list's order by id tells apart; it takes no lock. A finite greatest
// gives a finite sequence.
func SequenceAfter(greatest *float64) float64 {
	if greatest == nil {
		return firstSequence
	}
	return *greatest + sequenceStep
}
