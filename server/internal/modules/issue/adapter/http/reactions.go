package httpadapter

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// AddIssueReaction serves POST /api/v0/issues/{issue_id}/reactions.
func (h handler) AddIssueReaction(ctx context.Context, req gen.AddIssueReactionRequestObject) (gen.AddIssueReactionResponseObject, error) {
	r, err := h.uc.AddIssueReaction.Execute(ctx, req.IssueID, req.Body.Reaction)
	if err != nil {
		return nil, err
	}
	return gen.AddIssueReaction201JSONResponse(reactionDetail(r)), nil
}

// RemoveIssueReaction serves DELETE /api/v0/issues/{issue_id}/reactions/{reaction}.
func (h handler) RemoveIssueReaction(ctx context.Context, req gen.RemoveIssueReactionRequestObject) (gen.RemoveIssueReactionResponseObject, error) {
	if err := h.uc.RemoveIssueReaction.Execute(ctx, req.IssueID, req.Reaction); err != nil {
		return nil, err
	}
	return gen.RemoveIssueReaction204Response{}, nil
}

// ListIssueReactions serves GET /api/v0/issues/{issue_id}/reactions.
func (h handler) ListIssueReactions(ctx context.Context, req gen.ListIssueReactionsRequestObject) (gen.ListIssueReactionsResponseObject, error) {
	rows, err := h.uc.ListIssueReactions.Execute(ctx, req.IssueID)
	if err != nil {
		return nil, err
	}
	data := make([]gen.IssueReaction, len(rows))
	for n, r := range rows {
		data[n] = reactionDetail(r)
	}
	return gen.ListIssueReactions200JSONResponse{Data: data}, nil
}
func reactionDetail(r domain.IssueReaction) gen.IssueReaction {
	return gen.IssueReaction{ID: r.ID, IssueID: r.IssueID, ActorID: r.ActorID, Reaction: r.Reaction, CreatedAt: r.CreatedAt}
}
