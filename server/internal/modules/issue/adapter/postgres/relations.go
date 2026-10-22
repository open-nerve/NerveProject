package postgresadapter

import (
	"context"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// The relations between work items (relations.sql; M4 design 3.10, 4.6).

// Related reads which of the work items ids relate to the work item id: an
// undeleted relation between the two, of any type, either way, in id
// order.
func (s *Store) Related(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	related, err := s.queries(ctx).RelatedIssues(ctx, gen.RelatedIssuesParams{ID: id, Ids: ids})
	if err != nil {
		return nil, fmt.Errorf("read the relations of work item %s to %d work items: %w", id, len(ids), err)
	}
	return related, nil
}

// InsertRelations inserts the relations rs in their order, one statement;
// they are of one workspace, made by one account at one moment, as one
// write makes them (M4 design 3.10).
func (s *Store) InsertRelations(ctx context.Context, rs []domain.Relation) error {
	if len(rs) == 0 {
		return nil
	}
	p := gen.InsertRelationsParams{WorkspaceID: rs[0].WorkspaceID, By: rs[0].CreatedByID, At: rs[0].CreatedAt}
	for _, r := range rs {
		if r.WorkspaceID != p.WorkspaceID || r.CreatedByID == nil || p.By == nil || *r.CreatedByID != *p.By || !r.CreatedAt.Equal(p.At) {
			return fmt.Errorf("insert %d relations: not of one workspace, account and moment", len(rs))
		}
		p.Ids, p.ProjectIds, p.IssueIds = append(p.Ids, r.ID), append(p.ProjectIds, r.ProjectID), append(p.IssueIds, r.IssueID)
		p.RelatedIssueIds, p.RelationTypes = append(p.RelatedIssueIds, r.RelatedID), append(p.RelationTypes, string(r.Type))
	}
	if err := s.queries(ctx).InsertRelations(ctx, p); err != nil {
		return fmt.Errorf("insert %d relations: %w", len(p.Ids), err)
	}
	return nil
}

// MarkChanged writes the work items r names last changed at its moment and
// by its account (M4 design 3.7). The use case read each of them undeleted
// under its project's lock, so one not written is an error.
func (s *Store) MarkChanged(ctx context.Context, r app.ChangedRows) error {
	n, err := s.queries(ctx).MarkChanged(ctx, gen.MarkChangedParams{By: r.By, At: r.At, Ids: r.IDs})
	switch {
	case err != nil:
		return fmt.Errorf("mark %d work items changed: %w", len(r.IDs), err)
	case n != int64(len(r.IDs)):
		return fmt.Errorf("marked %d of %d work items changed", n, len(r.IDs))
	}
	return nil
}
