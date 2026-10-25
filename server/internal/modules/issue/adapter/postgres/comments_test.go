package postgresadapter_test

import (
	"context"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Reverse insertion order and identical timestamps distinguish the id tie break.
// Foreign-issue and already-deleted rows must never enter a page.
func TestComments(t *testing.T) {
	s, pool := newStore(t)
	actor := newAccount(t, pool, "actor@corp.com")
	w := newWorkspace(t, pool, "acme")
	p := newProject(t, pool, w, "WEB")
	one, two := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, p, one, 1, actor)
	newIssue(t, pool, p, two, 2, actor)
	ids := []uuid.UUID{uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()}
	for _, n := range []int{3, 2, 1, 0} {
		issue := one
		if n == 2 {
			issue = two
		}
		c := domain.Comment{ID: ids[n], WorkspaceID: w, ProjectID: p.id, IssueID: issue, ActorID: actor, HTML: "<p>Hello</p>", Text: "Hello"}
		if err := s.InsertComment(context.Background(), app.CommentWrite{Comment: c, By: actor, At: now}); err != nil {
			t.Fatal(err)
		}
	}
	exec(t, pool, "UPDATE issue_comments SET deleted_at=$2 WHERE id=$1", ids[3], now)
	for _, tt := range []struct {
		after *domain.CommentCursor
		limit int
		want  []uuid.UUID
	}{{nil, 1, ids[:1]}, {nil, 50, ids[:2]}, {&domain.CommentCursor{CreatedAt: now, ID: ids[0]}, 100, ids[1:2]}} {
		got, err := s.Comments(context.Background(), one, tt.after, tt.limit)
		if err != nil {
			t.Fatal(err)
		}
		actual := make([]uuid.UUID, len(got))
		for n, c := range got {
			actual[n] = c.ID
			if c.EditedAt != nil || c.HTML != "<p>Hello</p>" || c.Text != "Hello" || !c.CreatedAt.Equal(now) || !c.UpdatedAt.Equal(now) || c.ActorID != actor || c.Reactions == nil {
				t.Fatal(c)
			}
		}
		if !slices.Equal(actual, tt.want) {
			t.Fatalf("comments: %v, want %v", actual, tt.want)
		}
	}
	for n, id := range ids {
		got, found, err := s.GetComment(context.Background(), id)
		if err != nil || found != (n != 3) || found && got.ID != id {
			t.Fatal(got, found, err)
		}
	}
}
