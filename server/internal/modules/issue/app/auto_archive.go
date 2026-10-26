package app

import (
	"context"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AutoArchiveDeps supplies the system job's raw transaction manager, without
// request admission or conversion of PostgreSQL lock failures to HTTP errors.
type AutoArchiveDeps struct {
	Policies ArchivePolicies
	Projects ProjectGate
	States   StateCatalog
	Issues   AutoArchiveStore
	Tx       shared.TxManager
	Clock    Clock
	Events   shared.EventPublisher
}

// AutoArchive archives inactive terminal issues, one committed project at a time.
type AutoArchive struct{ d AutoArchiveDeps }

func NewAutoArchive(d AutoArchiveDeps) *AutoArchive { return &AutoArchive{d: d} }

// Execute returns only committed issues; a failure preserves earlier projects
// and its original error chain for the job runner to retry.
func (u *AutoArchive) Execute(ctx context.Context) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	projects, err := u.d.Policies.ArchiveProjects(ctx)
	if err != nil {
		return 0, err
	}
	total := 0
	for _, id := range projects {
		if err := ctx.Err(); err != nil {
			return total, err
		}
		count := 0
		err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
			p, found, err := u.d.Projects.Lock(ctx, id)
			if err != nil {
				return err
			}
			if !found || p.Archived || p.ArchiveIn <= 0 {
				return nil
			}
			states, err := u.d.States.States(ctx, id)
			if err != nil {
				return err
			}
			var terminal []uuid.UUID
			for _, s := range states {
				if domain.Archivable(s.Group) {
					terminal = append(terminal, s.ID)
				}
			}
			if len(terminal) == 0 {
				return nil
			}
			at := u.d.Clock.Now()
			ids, err := u.d.Issues.AutoArchive(ctx, AutoArchiveRows{ProjectID: id, StateIDs: terminal, Cutoff: at.Add(-time.Duration(p.ArchiveIn) * 30 * 24 * time.Hour), At: at})
			if err != nil {
				return err
			}
			slices.SortFunc(ids, uuid.UUID.Compare)
			rows := make([]ActivityRow, len(ids))
			for i, issue := range ids {
				rows[i] = ActivityRow{ID: uuid.NewV7(), WorkspaceID: p.WorkspaceID, ProjectID: id, IssueID: issue, Activity: domain.AutoArchiveActivity(), At: at}
			}
			if len(rows) == 0 {
				return nil
			}
			if err := u.d.Issues.InsertActivities(ctx, rows); err != nil {
				return err
			}
			for _, issue := range ids {
				e := shared.Event{ID: uuid.NewV7(), Type: domain.EventIssueArchived, OccurredAt: at, WorkspaceID: p.WorkspaceID, ProjectID: id, Entity: shared.EntityRef{Kind: "issue", ID: issue}, Changes: []shared.Change{{Field: "archived_at", New: eventPtr("archive")}}}
				if err := u.d.Events.Publish(ctx, e); err != nil {
					return err
				}
			}
			count = len(ids)
			return nil
		})
		if err != nil {
			return total, err
		}
		total += count
	}
	return total, nil
}
