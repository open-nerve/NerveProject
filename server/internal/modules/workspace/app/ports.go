// Package app holds the workspace module's use cases (M3 design 6.2) and the
// ports they need: small repository interfaces per use case, the clock, and
// the other modules' adapters as bootstrap converts them.
package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ErrNotFound is a repository's answer for a row that is not there.
var ErrNotFound = errors.New("workspace: not found")

// Clock gives the time the use cases write into the audit columns (M2
// design 3.13).
type Clock interface {
	Now() time.Time
}

// AccountState is an account's state as Accounts reads it under its lock:
// identity's value, converted in bootstrap/ports.go (M3 design 6.5).
type AccountState struct {
	ID     uuid.UUID
	Email  string
	Active bool
}

// Accounts locks an account row FOR SHARE and returns the account's state;
// found is false when there is no such account (M3 design 6.5). identity
// implements it (identity.Provide). Call it as the transaction's first lock
// and never later (M3 design 3.6 conventions 1 and 6): the lock keeps the
// account from being deactivated until the transaction ends, and the state is
// the one committed before it. An address is matched as given.
type Accounts interface {
	ShareAccount(ctx context.Context, id uuid.UUID) (state AccountState, found bool, err error)
	ShareAccountByEmail(ctx context.Context, email string) (state AccountState, found bool, err error)
}

// WorkspaceRow is a workspace to insert: checked values, its id, its
// creator and the time of the use case's clock.
type WorkspaceRow struct {
	ID               uuid.UUID
	Name             string
	Slug             string
	OrganizationSize *string
	Timezone         string
	CreatedBy        uuid.UUID
	Now              time.Time
}

// MemberRow is a workspace membership to insert, active.
type MemberRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	MemberID    uuid.UUID
	Role        shared.Role
	CreatedBy   uuid.UUID
	Now         time.Time
}

// WorkspaceCreator inserts a workspace and its first member.
type WorkspaceCreator interface {
	// CreateWorkspace inserts w and returns it as stored, without a role and
	// with no member counted; domain.ErrSlugTaken when an undeleted
	// workspace has its slug.
	CreateWorkspace(ctx context.Context, w WorkspaceRow) (domain.Workspace, error)
	// CreateMember inserts m.
	CreateMember(ctx context.Context, m MemberRow) error
}

// WorkspaceLister lists a user's workspaces.
type WorkspaceLister interface {
	// ListWorkspaces returns the undeleted workspaces of which userID is an
	// active member, with his role and the number of active members, by
	// name, then id (M3 design 3.12).
	ListWorkspaces(ctx context.Context, userID uuid.UUID) ([]domain.Workspace, error)
}

// WorkspaceFinder reads a workspace by its slug.
type WorkspaceFinder interface {
	// WorkspaceBySlug returns the undeleted workspace with slug and its
	// number of active members, without a role; ErrNotFound when there is
	// none.
	WorkspaceBySlug(ctx context.Context, slug string) (domain.Workspace, error)
}

// SlugChecker tells whether a slug is taken.
type SlugChecker interface {
	// SlugTaken reports whether an undeleted workspace has slug.
	SlugTaken(ctx context.Context, slug string) (bool, error)
}
