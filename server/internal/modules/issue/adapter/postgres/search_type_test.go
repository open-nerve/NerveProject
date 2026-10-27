package postgresadapter_test

import (
	"context"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres/gen"
)

// Missing 00029 from sqlc's schema makes this interface{} and fails compilation.
var _ float32 = gen.SearchIssuesRow{}.Similarity

func TestSearchSimilarityType(t *testing.T) {
	_, pool := newStore(t)
	account := newAccount(t, pool, "search@example.com")
	workspace := newWorkspace(t, pool, "search")
	p := newProject(t, pool, workspace, "SRCH")
	id := uuid.NewV7()
	newIssue(t, pool, p, id, 1, account)
	queries := gen.New(pool)
	for _, tt := range []struct {
		query string
		want  float32
	}{{"Work", 1}, {"Wor", 0.5}, {"xyz", 0}} {
		number := int32(1)
		rows, err := queries.SearchIssues(context.Background(), gen.SearchIssuesParams{Query: tt.query, ProjectIds: []uuid.UUID{p.id}, ProjectOwn: []bool{false}, ExactProjectID: &p.id, IdentifierNumber: &number, PageSize: 1})
		if err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].ID != id || rows[0].Similarity != tt.want {
			t.Fatalf("SearchIssues(%q) = %+v, want issue %s similarity %v", tt.query, rows, id, tt.want)
		}
		t.Logf("SearchIssues(%q).Similarity = %v (%T)", tt.query, rows[0].Similarity, rows[0].Similarity)
	}
}
