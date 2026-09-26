package app

import (
	"context"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ListAPITokens lists the caller's tokens a page at a time:
// GET /api/v0/me/api-tokens.
type ListAPITokens struct {
	tokens APITokenLister
}

// NewListAPITokens returns the use case.
func NewListAPITokens(tokens APITokenLister) *ListAPITokens {
	return &ListAPITokens{tokens: tokens}
}

// APITokenPage is one page of tokens and the cursor of the next, "" after
// the last page.
type APITokenPage struct {
	Tokens     []domain.APIToken
	NextCursor string
}

// Execute returns the page that limit and cursor ask for, from the start
// when cursor is nil. A cursor this list did not issue is 400 bad_request
// and a limit outside 1–100 is 422 validation_failed; the cursor, being
// the request's structure, is judged first. One row more than the page is
// read to tell whether another page follows.
func (l *ListAPITokens) Execute(ctx context.Context, limit *int, cursor *string) (APITokenPage, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return APITokenPage{}, err
	}
	var after *domain.APITokenCursor
	if cursor != nil {
		after = new(domain.APITokenCursor)
		if err := shared.DecodeCursor(*cursor, after); err != nil {
			return APITokenPage{}, err
		}
	}
	size, err := domain.PageSize(limit)
	if err != nil {
		return APITokenPage{}, err
	}
	tokens, err := l.tokens.ListAPITokens(ctx, actor.UserID, after, size+1)
	if err != nil || len(tokens) <= size {
		return APITokenPage{Tokens: tokens}, err
	}
	last := tokens[size-1]
	next, err := shared.EncodeCursor(domain.APITokenCursor{CreatedAt: last.CreatedAt, ID: last.ID})
	if err != nil {
		return APITokenPage{}, err
	}
	return APITokenPage{Tokens: tokens[:size], NextCursor: next}, nil
}
