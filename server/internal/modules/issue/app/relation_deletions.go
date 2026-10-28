package app

import (
	"context"
	"time"
	"uuid"
)

// RelationDeletionRow identifies a locked relation and both endpoint projects.
// Endpoint issues can already be deleted by the preceding issue-row step.
type RelationDeletionRow struct{ ID, ProjectID, RelatedProjectID uuid.UUID }

// RelationDeletionRows locks candidates in relation ID order and updates only
// the supplied locked IDs, in the transaction carried by ctx.
type RelationDeletionRows interface {
	LockProjectRelationRows(context.Context, RowBatch) ([]RelationDeletionRow, error)
	LockIssueRelationRows(context.Context, IssueDeletion) ([]RelationDeletionRow, error)
	DeleteLockedRelationRows(context.Context, []uuid.UUID, uuid.UUID, time.Time) (int, error)
}

// BatchProgress separates scanning progress from deletion ownership: a full
// skipped batch still advances After and does not end a scan.
type BatchProgress struct {
	After               uuid.UUID
	Candidates, Written int
}

// RelationDeletions assigns a shared relation to the first deleted endpoint.
// It never locks the other project; the caller owns the transaction and order.
type RelationDeletions struct {
	rows     RelationDeletionRows
	projects ProjectGate
}

func NewRelationDeletions(rows RelationDeletionRows, projects ProjectGate) *RelationDeletions {
	return &RelationDeletions{rows: rows, projects: projects}
}

func (d *RelationDeletions) ForProject(ctx context.Context, b RowBatch) (BatchProgress, error) {
	rows, err := d.rows.LockProjectRelationRows(ctx, b)
	if err != nil {
		return BatchProgress{}, err
	}
	progress := BatchProgress{Candidates: len(rows)}
	for _, row := range rows {
		if row.ID.Compare(progress.After) > 0 {
			progress.After = row.ID
		}
	}
	progress.Written, err = d.delete(ctx, rows, b.ProjectID, b.By, b.At, false)
	return progress, err
}

func (d *RelationDeletions) ForIssues(ctx context.Context, projectID uuid.UUID, deletion IssueDeletion) error {
	rows, err := d.rows.LockIssueRelationRows(ctx, deletion)
	if err != nil {
		return err
	}
	_, err = d.delete(ctx, rows, projectID, deletion.By, deletion.At, true)
	return err
}

func (d *RelationDeletions) delete(ctx context.Context, rows []RelationDeletionRow, projectID, by uuid.UUID, at time.Time, live bool) (int, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	var otherProjects []uuid.UUID
	for _, row := range rows {
		other := otherProject(row, projectID)
		if other != projectID {
			otherProjects = append(otherProjects, other)
		}
	}
	deleted, err := d.projects.DeletedAt(ctx, ascending(otherProjects))
	if err != nil {
		return 0, err
	}
	var ids []uuid.UUID
	for _, row := range rows {
		other := otherProject(row, projectID)
		if other != projectID {
			if earlier, found := deleted[other]; found && (live || earlier.Before(at)) {
				continue
			}
		}
		ids = append(ids, row.ID)
	}
	if len(ids) == 0 {
		return 0, nil
	}
	return d.rows.DeleteLockedRelationRows(ctx, ids, by, at)
}

func otherProject(row RelationDeletionRow, projectID uuid.UUID) uuid.UUID {
	if row.ProjectID == projectID {
		return row.RelatedProjectID
	}
	return row.ProjectID
}
