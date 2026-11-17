package httpadapter

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/http/gen"
)

// SubscriptionUseCase is app.SubscribeIssue, and app.UnsubscribeIssue.
type SubscriptionUseCase interface {
	Execute(ctx context.Context, id uuid.UUID) error
}

// SubscribeIssue serves POST /api/v0/issues/{issue_id}/subscribe.
func (h handler) SubscribeIssue(ctx context.Context, req gen.SubscribeIssueRequestObject) (gen.SubscribeIssueResponseObject, error) {
	if err := h.uc.SubscribeIssue.Execute(ctx, req.IssueID); err != nil {
		return nil, err
	}
	return gen.SubscribeIssue204Response{}, nil
}

// UnsubscribeIssue serves POST /api/v0/issues/{issue_id}/unsubscribe.
func (h handler) UnsubscribeIssue(ctx context.Context, req gen.UnsubscribeIssueRequestObject) (gen.UnsubscribeIssueResponseObject, error) {
	if err := h.uc.UnsubscribeIssue.Execute(ctx, req.IssueID); err != nil {
		return nil, err
	}
	return gen.UnsubscribeIssue204Response{}, nil
}
