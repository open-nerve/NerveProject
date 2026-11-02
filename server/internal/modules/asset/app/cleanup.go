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
	Version       int       `json:"version"`
	Stage         string    `json:"stage"`
	After         uuid.UUID `json:"after"`
	Token         string    `json:"token"`
	Cutoff        time.Time `json:"cutoff"`
	RuntimeCutoff time.Time `json:"runtime_cutoff"`
}
type SweepResult struct {
	Cursor   SweepCursor `json:"cursor"`
	Scanned  int         `json:"scanned"`
	Removed  int         `json:"removed"`
	Complete bool        `json:"complete"`
}
type CleanupDeps struct {
	Rows                     RecoveryReader
	Locks                    AssetLocker
	Writer                   MetadataWriter
	Parents                  ParentDeletionReader
	Objects                  ObjectDeleter
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
	if u.d.Batch < 1 || u.d.Batch > 1000 || u.d.Budget <= 0 || u.d.OperationTimeout <= 0 {
		return result, errors.New("invalid asset cleanup configuration")
	}
	if cursor.Version == 0 {
		if cursor != (SweepCursor{}) {
			return result, errors.New("uninitialized asset sweep has state")
		}
		cursor = SweepCursor{Version: 1, Stage: "rows", Cutoff: u.d.Clock.Now().UTC(), RuntimeCutoff: u.d.RuntimeNow().Add(-24 * time.Hour).UTC()}
	}
	if cursor.Version != 1 || cursor.Stage != "rows" || cursor.Token != "" || cursor.Cutoff.IsZero() || cursor.RuntimeCutoff.IsZero() {
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
		rows, err := u.d.Rows.ListRecovery(work, cursor.After, u.d.Batch)
		if err != nil {
			return sweepError(ctx, work, result, err)
		}
		if len(rows) == 0 {
			result.Complete = true
			return result, nil
		}
		hosts := make([]Host, 0, len(rows))
		last := cursor.After
		for _, a := range rows {
			if a.ID.Compare(last) <= 0 {
				return result, errors.New("asset recovery page is not ascending")
			}
			last = a.ID
			hosts = append(hosts, hostOf(a.Placement))
		}
		facts, err := u.d.Parents.Deletions(work, hosts)
		if err != nil {
			return sweepError(ctx, work, result, err)
		}
		for _, a := range rows {
			removed, attempted, err := u.cleanRow(work, a, facts, cursor.Cutoff, until)
			result.Removed += removed
			if attempted {
				cursor.After = a.ID
				result.Cursor = cursor
				result.Scanned++
			}
			if err != nil {
				return sweepError(ctx, work, result, err)
			}
		}
	}
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
