package app_test

import (
	"context"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// fakeRelations is the relations' store over the work items' fake
// (app.RelationCreator, app.RelationLister, app.RelationDeleter), each
// call logged: related are the work items that relate to the work item
// asked about already, whichever it is; relations the relations a read
// answers, and the last insert's; marked the last work items marked
// changed; deleted the last relation deleted.
type fakeRelations struct {
	*fakeIssues
	related   []uuid.UUID
	relations []domain.Relation
	marked    *app.ChangedRows
	deleted   *app.RelationDeletion
}

// GetRelation logs the read and answers the fake's relation id, as it is;
// found is false when it has none.
func (f *fakeRelations) GetRelation(ctx context.Context, id uuid.UUID) (domain.Relation, bool, error) {
	if err := f.log.add(ctx, "GetRelation", id); err != nil {
		return domain.Relation{}, false, err
	}
	at := slices.IndexFunc(f.relations, func(r domain.Relation) bool { return r.ID == id })
	if at < 0 {
		return domain.Relation{}, false, nil
	}
	return f.relations[at], true, nil
}

// DeleteRelation logs the deletion, of which relation, and keeps it.
func (f *fakeRelations) DeleteRelation(ctx context.Context, d app.RelationDeletion) error {
	if err := f.log.add(ctx, "DeleteRelation", d.ID); err != nil {
		return err
	}
	f.deleted = &d
	return nil
}

// Relations logs the read and answers the fake's relations, as they are,
// whichever work item is asked about.
func (f *fakeRelations) Relations(ctx context.Context, id uuid.UUID) ([]domain.Relation, error) {
	if err := f.log.add(ctx, "Relations", id); err != nil {
		return nil, err
	}
	return f.relations, nil
}

// Related logs the read and answers those of ids that the fake's related
// names, in ids' order.
func (f *fakeRelations) Related(ctx context.Context, id uuid.UUID, ids []uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Related", id, ids); err != nil {
		return nil, err
	}
	return slices.DeleteFunc(slices.Clone(ids), func(i uuid.UUID) bool { return !slices.Contains(f.related, i) }), nil
}

// InsertRelations logs the insert, of how many, and keeps the relations.
func (f *fakeRelations) InsertRelations(ctx context.Context, rs []domain.Relation) error {
	if err := f.log.add(ctx, "InsertRelations", len(rs)); err != nil {
		return err
	}
	f.relations = rs
	return nil
}

// MarkChanged logs the write, keeps its rows and marks each of the fake's
// others it names last changed by its writer at its moment.
func (f *fakeRelations) MarkChanged(ctx context.Context, r app.ChangedRows) error {
	if err := f.log.add(ctx, "MarkChanged", r.IDs); err != nil {
		return err
	}
	f.marked = &r
	for _, id := range r.IDs {
		if o, ok := f.others[id]; ok {
			o.UpdatedByID, o.UpdatedAt = &r.By, r.At
			f.others[id] = o
		}
	}
	return nil
}
