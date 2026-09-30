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
// and never later (M3 design 3.6 conventions 1 and 6), and the transaction
// takes no stronger lock of this row afterwards: the lock keeps the account
// from being deactivated until the transaction ends, and the state is the
// one committed before it. An address is matched as given.
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

// PublicProfile is an account's public profile as MemberProfiles reads it:
// identity's value, converted in bootstrap/ports.go (M3 design 6.5).
type PublicProfile struct {
	ID          uuid.UUID
	Email       string
	FirstName   string
	LastName    string
	DisplayName string
}

// MemberProfiles reads the public profile of each account of ids that
// exists, deactivated ones too, without a lock (M3 design 6.5): the way a
// use case reads another account, also inside a transaction that holds a
// workspace's lock (M3 design 3.6 convention 1). identity implements it
// (identity.Provide).
type MemberProfiles interface {
	PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]PublicProfile, error)
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

// The parent locks of the writes on a workspace (M3 design 3.6 convention
// 2), in the transaction ctx carries: each locks the undeleted workspace
// with slug until the transaction ends and returns its id; ErrNotFound when
// there is none, also when it was deleted while the lock waited.

// MemberLister reads a workspace and lists its memberships.
type MemberLister interface {
	WorkspaceFinder
	// ListMembers returns the undeleted memberships of the workspace, active
	// or not, by created_at, then id (M3 design 3.12).
	ListMembers(ctx context.Context, workspaceID uuid.UUID) ([]domain.Membership, error)
}

// WorkspaceLocker locks FOR NO KEY UPDATE: for a write of the workspace row
// itself or of a membership.
type WorkspaceLocker interface {
	LockWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error)
}

// WorkspaceSharer locks FOR SHARE: for a write that adds or changes a row
// under the workspace.
type WorkspaceSharer interface {
	ShareWorkspaceBySlug(ctx context.Context, slug string) (uuid.UUID, error)
}

// WorkspaceUpdater changes a workspace row under its lock.
type WorkspaceUpdater interface {
	WorkspaceLocker
	// UpdateWorkspace applies p to the workspace id, by the account by at
	// now, and returns it as stored with its number of active members,
	// without a role.
	UpdateWorkspace(ctx context.Context, id uuid.UUID, p domain.WorkspacePatch, by uuid.UUID, now time.Time) (domain.Workspace, error)
}

// WorkspaceDeleter soft-deletes a workspace and the rows under it, under its
// lock. Each step sets deleted_at and updated_at to now and updated_by_id to
// by, on the undeleted rows only.
type WorkspaceDeleter interface {
	WorkspaceLocker
	// DeleteWorkspace soft-deletes the workspace row.
	DeleteWorkspace(ctx context.Context, id, by uuid.UUID, now time.Time) error
	// DeleteWorkspaceMembers soft-deletes its memberships, active or not.
	DeleteWorkspaceMembers(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
	// DeleteWorkspacePreferences soft-deletes its members' display
	// settings.
	DeleteWorkspacePreferences(ctx context.Context, workspaceID, by uuid.UUID, now time.Time) error
}

// PreferencesRow is a change of an account's display settings in a
// workspace, and the id of the row if the change inserts one.
type PreferencesRow struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Patch       domain.PreferencesPatch
	Now         time.Time
}

// PreferencesReader reads an account's display settings in a workspace.
type PreferencesReader interface {
	WorkspaceFinder
	// Preferences returns userID's settings in workspaceID; found is false
	// while there is no undeleted row.
	Preferences(ctx context.Context, workspaceID, userID uuid.UUID) (p domain.Preferences, found bool, err error)
}

// PreferencesWriter writes an account's display settings in a workspace
// under the workspace's lock.
type PreferencesWriter interface {
	WorkspaceSharer
	// UpsertPreferences applies r.Patch to the account's undeleted row, or
	// inserts one with domain.DefaultPreferences and the patch applied, and
	// returns the settings as stored.
	UpsertPreferences(ctx context.Context, r PreferencesRow) (domain.Preferences, error)
}

// SlugChecker tells whether a slug is taken.
type SlugChecker interface {
	// SlugTaken reports whether an undeleted workspace has slug.
	SlugTaken(ctx context.Context, slug string) (bool, error)
}
