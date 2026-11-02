package app

import (
	"context"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset/domain"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type SweepCursor struct {
	Version       int              `json:"version"`
	Stage         string           `json:"stage"`
	After         uuid.UUID        `json:"after"`
	Token         string           `json:"token"`
	ObjectAfter   domain.ObjectKey `json:"object_after"`
	Cutoff        time.Time        `json:"cutoff"`
	RuntimeCutoff time.Time        `json:"runtime_cutoff"`
}
type SweepResult struct {
	Cursor   SweepCursor `json:"cursor"`
	Scanned  int         `json:"scanned"`
	Removed  int         `json:"removed"`
	Failed   int         `json:"failed"`
	Complete bool        `json:"complete"`
}
type CleanupDeps struct {
	Rows                     RecoveryReader
	Locks                    AssetLocker
	Writer                   MetadataWriter
	Parents                  ParentDeletionReader
	Objects                  ObjectDeleter
	Inventory                ObjectLister
	Temporary                TemporaryCleaner
	Tx                       shared.TxManager
	Clock                    Clock
	RuntimeNow               func() time.Time
	Batch                    int
	Budget, OperationTimeout time.Duration
}
type Cleanup struct{ d CleanupDeps }

func NewCleanup(d CleanupDeps) *Cleanup { return &Cleanup{d: d} }
func (u *Cleanup) Execute(ctx context.Context, cursor SweepCursor) (SweepResult, error) {
	result := SweepResult{Cursor: cursor}
	if u.d.Batch < 1 || u.d.Batch > 1000 || u.d.Budget <= 0 || u.d.OperationTimeout <= 0 || u.d.Inventory == nil || u.d.Temporary == nil {
		return result, errors.New("invalid asset cleanup configuration")
	}
	if cursor.Version == 0 {
		if cursor != (SweepCursor{}) {
			return result, errors.New("uninitialized asset sweep has state")
		}
		cursor = SweepCursor{Version: 1, Stage: "rows", Cutoff: u.d.Clock.Now().UTC(), RuntimeCutoff: u.d.RuntimeNow().Add(-24 * time.Hour).UTC()}
	}
	if !cursor.ValidCleanup() {
		return result, errors.New("invalid asset sweep cursor")
	}
	result.Cursor = cursor
	until := time.Now().Add(u.d.Budget)
	work, cancel := context.WithDeadline(ctx, until)
	defer cancel()
	for {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if work.Err() != nil {
			return result, nil
		}
		var err error
		switch result.Cursor.Stage {
		case "rows":
			err = u.cleanRowsPage(work, &result, until)
		case "objects":
			err = u.cleanInventoryPage(work, &result)
		case "tmp":
			err = u.cleanTemporaryPage(work, &result)
		case "done":
			result.Complete = true
			return result, nil
		}
		if err != nil {
			if work.Err() == nil {
				result.Failed++
			}
			return sweepError(ctx, work, result, err)
		}
	}
}

func (u *Cleanup) cleanRowsPage(ctx context.Context, result *SweepResult, until time.Time) error {
	rows, err := u.d.Rows.ListRecovery(ctx, result.Cursor.After, u.d.Batch)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		result.Cursor.Stage, result.Cursor.After = "objects", uuid.Nil()
		return nil
	}
	if len(rows) > u.d.Batch {
		return errors.New("asset recovery page exceeds batch")
	}
	hosts := make([]Host, 0, len(rows))
	last := result.Cursor.After
	for _, a := range rows {
		if a.ID.Compare(last) <= 0 {
			return errors.New("asset recovery page is not ascending")
		}
		last = a.ID
		hosts = append(hosts, hostOf(a.Placement))
	}
	facts, err := u.d.Parents.Deletions(ctx, hosts)
	if err != nil {
		return err
	}
	for _, a := range rows {
		removed, attempted, err := u.cleanRow(ctx, a, facts, result.Cursor.Cutoff, until)
		result.Removed += removed
		if attempted {
			result.Cursor.After = a.ID
			result.Scanned++
		}
		if err != nil {
			return err
		}
	}
	return nil
}
func sweepError(parent, work context.Context, result SweepResult, err error) (SweepResult, error) {
	if parent.Err() != nil {
		return result, parent.Err()
	}
	if work.Err() == context.DeadlineExceeded && errors.Is(err, context.DeadlineExceeded) {
		return result, nil
	}
	return result, err
}
func hostOf(p domain.Placement) Host {
	return Host{Scope: p.Scope, WorkspaceID: p.WorkspaceID, ProjectID: p.ProjectID, IssueID: p.IssueID, CommentID: p.CommentID, DraftID: p.DraftID}
}
