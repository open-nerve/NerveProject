package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

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

// issueRelationCollection is relations as the API lists them, in their
// order, [] for none: each with its type as its work item reads it, and
// the other work item as a list shows it (issueCollection).
func issueRelationCollection(relations []app.IssueRelation) gen.IssueRelationCollection {
	data := make([]gen.IssueRelation, len(relations))
	for n, r := range relations {
		data[n] = gen.IssueRelation{ID: r.ID, RelationType: gen.IssueRelationType(r.Type), Issue: gen.Issue(issueDetail(r.Issue)), CreatedAt: r.CreatedAt}
	}
	return gen.IssueRelationCollection{Data: data}
}
