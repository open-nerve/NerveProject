package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// ListIssueRelations serves GET /api/v0/issues/{issue_id}/relations.
func (h handler) ListIssueRelations(ctx context.Context, req gen.ListIssueRelationsRequestObject) (gen.ListIssueRelationsResponseObject,
	error) {
	relations, err := h.uc.ListIssueRelations.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	return gen.ListIssueRelations200JSONResponse(issueRelationCollection(relations)), nil
}

// CreateIssueRelations serves POST /api/v0/issues/{issue_id}/relations:
// the type and the work items go to the use case as they came, which
// checks them.
func (h handler) CreateIssueRelations(ctx context.Context, req gen.CreateIssueRelationsRequestObject) (gen.CreateIssueRelationsResponseObject,
	error) {
	relations, err := h.uc.CreateIssueRelations.Execute(ctx, req.IssueID, domain.RelationType(req.Body.RelationType), req.Body.IssueIds)
	if err != nil {
		return nil, err
	}
	return gen.CreateIssueRelations201JSONResponse(issueRelationCollection(relations)), nil
}

// DeleteIssueRelation serves DELETE /api/v0/issue-relations/{relation_id}.
func (h handler) DeleteIssueRelation(ctx context.Context, req gen.DeleteIssueRelationRequestObject) (gen.DeleteIssueRelationResponseObject,
	error) {
	if err := h.uc.DeleteIssueRelation.Execute(ctx, req.RelationID); err != nil {
		return nil, err
	}
	return gen.DeleteIssueRelation204Response{}, nil
}

// issueRelationCollection is relations as the API lists them, in their
// order, [] for none: each with its type as its work item reads it, and
// the other work item as a list shows it (issueCollection).
func issueRelationCollection(relations []app.IssueRelation) gen.IssueRelationCollection {
	data := make([]gen.IssueRelation, len(relations))
	for n, r := range relations {
		data[n] = gen.IssueRelation{ID: r.ID, RelationType: gen.IssueRelationType(r.Type), Issue: issueRow(r.Issue), CreatedAt: r.CreatedAt}
	}
	return gen.IssueRelationCollection{Data: data}
}
