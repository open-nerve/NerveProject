package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/project/domain"
)

// The project store's states, as the state operations read and write them
// (M3 design 3.17). The triage state is never one of them.

// StateCreator is createState's repository. Each method runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateCreator interface {
	// GreatestSequence is the greatest sequence of projectID's undeleted
	// states but its triage state, nil when it has none.
	GreatestSequence(ctx context.Context, projectID uuid.UUID) (*float64, error)
	// CreateState inserts r and answers it as stored. A name another
	// undeleted state of the project has is domain.ErrStateNameTaken.
	CreateState(ctx context.Context, r StateRow) (domain.State, error)
}

// StateFinder reads a state a write names by its id: first without a lock,
// for its project and the project's workspace, then again under their
// locks (lockRowAndDecide).
type StateFinder interface {
	// StateByID is the undeleted state id, unless it is the triage state;
	// found is false when there is none.
	StateByID(ctx context.Context, id uuid.UUID) (s domain.State, found bool, err error)
}

// GroupCounter counts the states of a group of a project: a write that
// takes a state from its group refuses to leave the group without one
// (domain.CheckGroupKept). It runs in the transaction ctx carries, under
// the project's FOR NO KEY UPDATE, which every write of its states takes,
// so the count stays as read.
type GroupCounter interface {
	// CountGroupStates is the number of projectID's undeleted states in
	// group.
	CountGroupStates(ctx context.Context, projectID uuid.UUID, group domain.StateGroup) (int, error)
}

// StateUpdater is updateState's repository. Its writes run in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateUpdater interface {
	StateFinder
	GroupCounter
	// UpdateState changes the fields p gives of the undeleted state id, by
	// the account by at now, and answers it as stored. A name another
	// undeleted state of the project has is domain.ErrStateNameTaken.
	UpdateState(ctx context.Context, id uuid.UUID, p domain.StatePatch, by uuid.UUID, now time.Time) (domain.State, error)
}

// StateDeleter is deleteState's repository. Its writes run in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type StateDeleter interface {
	StateFinder
	GroupCounter
	// DeleteState deletes the undeleted state id, unless it is its project's
	// default or its triage state, by the account by at now: a guarded
	// write (M3 design 3.17). deleted is false when it did not.
	DeleteState(ctx context.Context, id, by uuid.UUID, now time.Time) (deleted bool, err error)
}

// DefaultMarker is markDefaultState's repository. It runs in the
// transaction ctx carries, under the project's FOR NO KEY UPDATE.
type DefaultMarker interface {
	StateFinder
	// MarkDefaultState makes the undeleted state id of projectID its
	// default, by the account by at now, in two statements: the project's
	// default state the default no longer, then this one the default (M3
	// design 3.17). marked is false when the second wrote no row, and the
	// caller rolls the first back.
	MarkDefaultState(ctx context.Context, projectID, id, by uuid.UUID, now time.Time) (marked bool, err error)
}
