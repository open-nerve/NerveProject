package app

import (
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/navigation/domain"
)

// RecentVisitReader reads one's recent visits without a lock, in the
// transaction ctx carries or on the pool (P3d spec 2.5). RecentVisits lists
// user's undeleted visits in workspace, of kind alone when kind is not
// nil, by visit, the newest first, then by id descending; every one of
// them, which the cap keeps few, whether he sees their targets now or not.
type RecentVisitReader interface {
	RecentVisits(ctx context.Context, user, workspace uuid.UUID, kind *domain.RecentVisitKind) ([]domain.RecentVisit, error)
}

// RecentVisitLocker takes, in the transaction ctx carries, a recording's
// last lock, after its workspace's and its target's project's FOR SHARE
// (spec 2.8): LockRecentVisits takes the transaction-level advisory lock of
// user's visits in workspace, held until the transaction ends, so his
// recordings there run one after another and the cap holds. Another
// account's, and his in another workspace, are other locks.
type RecentVisitLocker interface {
	LockRecentVisits(ctx context.Context, user, workspace uuid.UUID) error
}

// RecentVisitWriter writes one's recent visits in the transaction ctx
// carries, which holds a recording's locks, the advisory one last (spec
// 2.5, 2.8); neither waits for a row. RecordRecentVisit stores v, made by
// by at its VisitedAt, unless its owner has an undeleted visit of its
// target in its workspace: then that one is revisited, its visit and its
// last change stamped by by at v's VisitedAt, its id and creation kept.
// TrimRecentVisits deletes, for good, user's undeleted visits in workspace
// past the keep newest by visit (then by id descending), passing over one
// another transaction holds.
type RecentVisitWriter interface {
	RecordRecentVisit(ctx context.Context, v domain.RecentVisit, by uuid.UUID) error
	TrimRecentVisits(ctx context.Context, user, workspace uuid.UUID, keep int) error
}

// RecentVisitCascades soft-deletes, in the transaction ctx carries, which
// holds the deleting work item's project FOR NO KEY UPDATE (spec 2.6, 2.8),
// every account's undeleted visits of the work item issue in workspace,
// each locked FOR NO KEY UPDATE in id order, deleted and last changed at
// at by by, nil for none; their visits are kept, and one deleted before
// keeps its deletion.
type RecentVisitCascades interface {
	DeleteIssueRecentVisits(ctx context.Context, workspace, issue uuid.UUID, at time.Time, by *uuid.UUID) error
}

// IssuePlace is where a work item is (P3d spec 2.7, R3): its workspace,
// which a recording decides on, and its project, which it locks FOR SHARE.
// Its fields are issue's NavigationIssuePlace's, which bootstrap converts
// into it.
type IssuePlace struct {
	WorkspaceID, ProjectID uuid.UUID
}

// IssueSummary is a work item as a recent visit is answered with it now
// (spec 2.4, 2.5, R3): its id, its project, its number, its name, its
// state, its priority as written, the accounts of its undeleted assignees'
// rows in id order, and whether it is in its project's intake, which keeps
// it out of the list until it is accepted (R6). Its fields are issue's
// NavigationIssue's, which bootstrap converts into it.
type IssueSummary struct {
	ID, ProjectID uuid.UUID
	SequenceID    int
	Name          string
	StateID       uuid.UUID
	Priority      string
	AssigneeIDs   []uuid.UUID
	InTriage      bool
}

// Issues are the work items recent visits name (spec 2.5, 2.7, R3): issue's
// NavigationIssues, which bootstrap converts. Each reads once, without a
// lock, in the transaction ctx carries or on the pool, and not at all for
// no ids. IssuePlaces answers, by id, where each of the undeleted work
// items among ids is, archived or not, of any workspace; it decides
// nothing. IssueSummaries answers, by id, those of the work items ids that
// reader sees now by grants, the projects he reads in a workspace
// (Readable): undeleted, archived or not, in a project among grants, and
// not hidden from him by the guest rule, the issue module's one
// HiddenFrom, by his grant there; another workspace's work item is in no
// project of grants. A deleted work item and an id of none are among
// neither.
type Issues interface {
	IssuePlaces(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]IssuePlace, error)
	IssueSummaries(ctx context.Context, reader uuid.UUID, grants []ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]IssueSummary, error)
}
