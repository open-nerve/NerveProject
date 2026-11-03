package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type AccessIntent uint8

const (
	ReadAsset AccessIntent = iota + 1
	PrepareAsset
	RemoveAsset
)

type HostRequest struct {
	Scope  domain.Scope
	Intent AccessIntent
}

// AssetAccessRequest binds each locked row to its semantic owner request.
// Duplicate source and target authorization must never depend on UUID order.
type AssetAccessRequest struct {
	ID   uuid.UUID
	Host HostRequest
}

type Host struct {
	Scope                                               domain.Scope
	WorkspaceID, ProjectID, IssueID, CommentID, DraftID uuid.UUID
}

type HostAccess interface {
	Inspect(context.Context, shared.Actor, HostRequest) (Host, error)
	Check(context.Context, shared.Actor, HostRequest, Host) error
}

type WorkspaceLocks interface {
	ShareMany(context.Context, []uuid.UUID) ([]uuid.UUID, error)
}
type ProjectLocks interface {
	LockMany(context.Context, []uuid.UUID) ([]uuid.UUID, error)
}
type DraftLocks interface {
	LockMany(context.Context, []uuid.UUID) ([]uuid.UUID, error)
}
type CallerLock interface {
	LockCaller(context.Context, shared.Actor, time.Time) error
}
type WriteGate interface {
	Enter(context.Context, ...uuid.UUID) (func(), error)
}
type AssetLocker interface {
	LockAssets(context.Context, []uuid.UUID) ([]domain.Asset, error)
}
type Clock interface{ Now() time.Time }

type ParentKey struct {
	Kind domain.ScopeKind
	ID   uuid.UUID
}
type DeletionStamp struct {
	At time.Time
	By *uuid.UUID
}

// Only explicitly returned deletion facts are deletions; a missing parent is
// an error in the owner adapter, never an invented timestamp.
type ParentDeletionReader interface {
	Deletions(context.Context, []Host) (map[ParentKey]DeletionStamp, error)
}
