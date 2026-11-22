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
// another transaction holds, and answers how many it deleted.
type RecentVisitWriter interface {
	RecordRecentVisit(ctx context.Context, v domain.RecentVisit, by uuid.UUID) error
	TrimRecentVisits(ctx context.Context, user, workspace uuid.UUID, keep int) (int, error)
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
