package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ProjectGrant is a project whose work items a reader may read, only those
// he created when OwnOnly: the access module's answer for it (M7 P1c spec
// 2.5), which bootstrap converts.
type ProjectGrant struct {
	ProjectID uuid.UUID
	OwnOnly   bool
}

// IssueRef is a work item as another names it (M7 P1c spec 2.6): its id,
// its project, its number and its name.
type IssueRef struct {
	ID         uuid.UUID
	ProjectID  uuid.UUID
	SequenceID int
	Name       string
}

// IssueRefs names, for the notification module, the work items a
// notification's change names, its parent and its relations' other ends,
// as a reader sees them now (M7 P1c spec 2.6, R4).
type IssueRefs struct {
	rows IssueRefReader
}

// NewIssueRefs returns the IssueRefs that reads through rows.
func NewIssueRefs(rows IssueRefReader) *IssueRefs {
	return &IssueRefs{rows: rows}
}

// IssueRefs answers, by id, those of the work items ids that are undeleted,
// archived or not, in a project among grants, archived or not, and that
// the guest rule, domain.Issue.HiddenFrom, does not hide from reader by his
// grant there; another workspace's work item is in no project of grants.
// It reads once, in the transaction ctx carries or on the pool, and not at
// all for no ids.
func (r *IssueRefs) IssueRefs(ctx context.Context, reader uuid.UUID, grants []ProjectGrant, ids []uuid.UUID) (map[uuid.UUID]IssueRef, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.rows.IssueRefRows(ctx, ids)
	if err != nil {
		return nil, err
	}
	ownOnly := make(map[uuid.UUID]bool, len(grants))
	for _, g := range grants {
		ownOnly[g.ProjectID] = g.OwnOnly
	}
	refs := make(map[uuid.UUID]IssueRef, len(rows))
	for _, row := range rows {
		own, readable := ownOnly[row.ProjectID]
		if readable && !(domain.Issue{CreatedByID: row.CreatedByID}).HiddenFrom(reader, own) {
			refs[row.ID] = row.IssueRef
		}
	}
	return refs, nil
}
