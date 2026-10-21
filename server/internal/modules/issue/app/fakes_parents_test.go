package app_test

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Ancestry logs the read and answers the work item id's ancestry as the
// fake is told it, or the work item alone (fakeIssues' ancestry).
func (f *fakeIssues) Ancestry(ctx context.Context, id uuid.UUID) ([]uuid.UUID, error) {
	if err := f.log.add(ctx, "Ancestry", id); err != nil {
		return nil, err
	}
	if a, ok := f.ancestry[id]; ok {
		return a, nil
	}
	return []uuid.UUID{id}, nil
}

// SubIssueCounts logs the count and answers the numbers of sub-items the
// fake is told of the work items ids, and none of the others (fakeIssues'
// counts).
func (f *fakeIssues) SubIssueCounts(ctx context.Context, actor uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]app.SubIssueCount, error) {
	if err := f.log.add(ctx, "SubIssueCounts", actor, ids); err != nil {
		return nil, err
	}
	counts := map[uuid.UUID]app.SubIssueCount{}
	for _, id := range ids {
		if c, ok := f.counts[id]; ok {
			counts[id] = c
		}
	}
	return counts, nil
}

// GetIssues logs the read and answers the work items among ids that the
// fake has (read), in id order.
func (f *fakeIssues) GetIssues(ctx context.Context, ids []uuid.UUID) ([]domain.Issue, error) {
	if err := f.log.add(ctx, "GetIssues", ids); err != nil {
		return nil, err
	}
	var out []domain.Issue
	for _, id := range slices.SortedFunc(slices.Values(ids), uuid.UUID.Compare) {
		if i, found := f.read(id); found {
			out = append(out, i)
		}
	}
	return out, nil
}

// SubIssues logs the read, the parent's and whose sub-items it asks for,
// anyone's for none, and answers the fake's subItems, whatever is asked.
func (f *fakeIssues) SubIssues(ctx context.Context, parent uuid.UUID, by *uuid.UUID) ([]domain.Issue, error) {
	whose := "anyone's"
	if by != nil {
		whose = by.String() + "'s"
	}
	if err := f.log.add(ctx, "SubIssues", parent, whose); err != nil {
		return nil, err
	}
	return f.subItems, nil
}

// SetParents logs the write, keeps its rows (parented) and puts each of
// the fake's others it names under its parent, last changed by its writer
// at its moment; a work item it names that the fake has not is an error.
func (f *fakeIssues) SetParents(ctx context.Context, r app.ParentRows) error {
	if err := f.log.add(ctx, "SetParents", r.IDs, r.ParentID); err != nil {
		return err
	}
	f.parented = &r
	for _, id := range r.IDs {
		o, ok := f.others[id]
		if !ok {
			return fmt.Errorf("SetParents: no work item %s", id)
		}
		o.ParentID, o.UpdatedByID, o.UpdatedAt = &r.ParentID, &r.By, r.At
		f.others[id] = o
	}
	return nil
}

// lose loses the work items the activity rows name, so that they are not
// there to read back (fakeIssues' lost): each of the fake's others named,
// and the work item it keeps when it is named.
func (f *fakeIssues) lose(rows []app.ActivityRow) {
	for _, r := range rows {
		delete(f.others, r.IssueID)
		if f.issue != nil && f.issue.ID == r.IssueID {
			f.issue = nil
		}
	}
}
