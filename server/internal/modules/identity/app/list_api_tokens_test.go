package app_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fiveTokens are rows in the list's order, one second apart.
func fiveTokens() []domain.APIToken {
	var rows []domain.APIToken
	for i := range 5 {
		rows = append(rows, domain.APIToken{ID: uuid.NewV7(), Label: "t", CreatedAt: now.Add(-time.Duration(i) * time.Second)})
	}
	return rows
}

func listPage(t *testing.T, tokens *fakeAPITokens, limit *int, cursor *string) (app.APITokenPage, error) {
	t.Helper()
	ctx := shared.WithActor(context.Background(), sessionActor)
	return app.NewListAPITokens(tokens).Execute(ctx, limit, cursor)
}

func ptr[T any](v T) *T { return &v }

// Without a limit the page holds 50; one row more is read to tell whether
// another page follows.
func TestListAPITokensFirstPage(t *testing.T) {
	tokens := &fakeAPITokens{rows: fiveTokens()}

	page, err := listPage(t, tokens, nil, nil)

	if err != nil || len(page.Tokens) != 5 || page.NextCursor != "" {
		t.Errorf("Execute() = %d tokens, next %q, %v; want all 5 and no next page", len(page.Tokens), page.NextCursor, err)
	}
	if len(tokens.listed) != 1 || tokens.listed[0] != (listCall{userID, nil, 51}) {
		t.Errorf("store asked for %+v, want the caller's first 51 rows", tokens.listed)
	}
}

// A full page with a row after it names the page's last row in its
// cursor; the next call passes that row on to the store.
func TestListAPITokensNextPage(t *testing.T) {
	rows := fiveTokens()
	tokens := &fakeAPITokens{rows: rows}

	page, err := listPage(t, tokens, ptr(2), nil)
	if err != nil || len(page.Tokens) != 2 || page.Tokens[1].ID != rows[1].ID || page.NextCursor == "" {
		t.Fatalf("Execute() = %+v, %v; want the first 2 rows and a cursor", page, err)
	}
	var c domain.APITokenCursor
	if err := shared.DecodeCursor(page.NextCursor, &c); err != nil || c.ID != rows[1].ID || !c.CreatedAt.Equal(rows[1].CreatedAt) {
		t.Errorf("cursor = %+v, %v; want the second row", c, err)
	}

	if _, err := listPage(t, tokens, ptr(2), &page.NextCursor); err != nil {
		t.Fatal(err)
	}
	// The store compares rows by (created_at, id): it gets both.
	next := tokens.listed[1]
	if next.limit != 3 || next.after == nil || next.after.ID != rows[1].ID || !next.after.CreatedAt.Equal(rows[1].CreatedAt) {
		t.Errorf("second call asked for %+v (after %+v), want 3 rows after the second row", next, next.after)
	}
}

// A page that ends the list has no cursor, even when it is full.
func TestListAPITokensLastFullPage(t *testing.T) {
	page, err := listPage(t, &fakeAPITokens{rows: fiveTokens()[:2]}, ptr(2), nil)

	if err != nil || len(page.Tokens) != 2 || page.NextCursor != "" {
		t.Errorf("Execute() = %d tokens, next %q, %v; want 2 and no next page", len(page.Tokens), page.NextCursor, err)
	}
}

// A bad cursor is 400 and a bad limit 422; with both, the cursor's 400.
// The store is not asked.
func TestListAPITokensRejects(t *testing.T) {
	bad := "not a cursor"
	tests := []struct {
		name   string
		limit  *int
		cursor *string
		status int
		field  string
	}{
		{"a cursor the list did not issue", nil, &bad, 400, "cursor"},
		{"limit 0", ptr(0), nil, 422, "limit"},
		{"limit 101", ptr(101), nil, 422, "limit"},
		{"both", ptr(0), &bad, 400, "cursor"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := &fakeAPITokens{rows: fiveTokens()}

			_, err := listPage(t, tokens, tt.limit, tt.cursor)

			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != tt.status || len(se.Fields) != 1 || se.Fields[0].Field != tt.field {
				t.Errorf("Execute() = %v, want %d on %s", err, tt.status, tt.field)
			}
			if len(tokens.listed) != 0 {
				t.Errorf("store asked for %+v, want nothing", tokens.listed)
			}
		})
	}
}
