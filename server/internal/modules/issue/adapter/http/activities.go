package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// ListIssueActivities serves a visible activity page without internal audit/detail fields.
func (h handler) ListIssueActivities(ctx context.Context, req gen.ListIssueActivitiesRequestObject) (gen.ListIssueActivitiesResponseObject, error) {
	page, err := h.uc.ListIssueActivities.Execute(ctx, req.IssueID, req.Params.Limit, req.Params.Cursor)
	if err != nil {
		return nil, err
	}
	data := make([]gen.IssueActivity, len(page.Data))
	for n, r := range page.Data {
		a := r.Activity
		data[n] = gen.IssueActivity{ID: r.ID, IssueID: r.IssueID, ProjectID: r.ProjectID, ActorID: orNull(r.ActorID), Verb: string(a.Verb), Field: orNull(a.Field), OldValue: orNull(a.OldValue), NewValue: orNull(a.NewValue), OldIdentifier: orNull(a.OldIdentifier), NewIdentifier: orNull(a.NewIdentifier), CreatedAt: r.CreatedAt}
	}
	var cursor *string
	if page.NextCursor != "" {
		cursor = &page.NextCursor
	}
	return gen.ListIssueActivities200JSONResponse{Data: data, NextCursor: orNull(cursor)}, nil
}
