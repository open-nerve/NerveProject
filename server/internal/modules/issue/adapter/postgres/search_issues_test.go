package postgresadapter_test

import (
	"context"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// Missing OR branches, wildcard escaping, stable ties, or scope before LIMIT
// each changes a fixed expected answer from the real PostgreSQL query.
func TestSearchLiteralTermsAndOrder(t *testing.T) {
	s, pool := newStore(t)
	a, b := newAccount(t, pool, "search-a"), newAccount(t, pool, "search-b")
	w := newWorkspace(t, pool, "search")
	p1, p2 := newProject(t, pool, w, "WEB"), newProject(t, pool, w, "OTHER")
	names := []string{"different", "12 report", "WEB-12 notes", "100% done", "100X done", "under_score", "underXscore", `a\b`, "ab", "修复登录", "登录问题", "same tie", "same tie", "same tie", "description only"}
	ids := make([]uuid.UUID, len(names))
	for n, name := range names {
		ids[n], _ = uuid.Parse(fmt.Sprintf("00000000-0000-4000-8000-%012d", n+1))
		p, seq := p1, n+11
		if n == 0 {
			seq = 12
		}
		if n == 1 {
			p = p2
			seq = 12
		}
		newIssue(t, pool, p, ids[n], seq, a)
		exec(t, pool, "UPDATE issues SET name=$2,created_at='2026-01-01',updated_at='2026-01-01',description_stripped='needle' WHERE id=$1", ids[n], name)
	}
	// The first tie is newest, the remaining two tie down to their UUID.
	exec(t, pool, "UPDATE issues SET created_at='2026-01-02' WHERE id=$1", ids[11])
	exec(t, pool, "UPDATE issues SET updated_at='2026-01-03' WHERE id=$1", ids[0])
	exec(t, pool, "UPDATE issues SET updated_at='2026-01-02' WHERE id=ANY($1::uuid[])", ids[1:3])
	run := func(t *testing.T, raw map[string][]string, own bool, scope []app.VisibleProject, want ...uuid.UUID) {
		t.Helper()
		q, err := domain.ParseSearchQuery(raw)
		if err != nil {
			t.Fatal(err)
		}
		if own {
			scope = []app.VisibleProject{{ID: p1.id, OwnOnly: true}}
		}
		in := app.SearchRead{ActorID: a, Projects: scope, Query: q}
		if q.Identifier != "" {
			in.ExactProjectID = &p1.id
		}
		rows, err := s.Search(context.Background(), in)
		if err != nil {
			t.Fatal(err)
		}
		searchIDs(t, rows, want...)
	}
	scope := []app.VisibleProject{{ID: p1.id}, {ID: p2.id}}
	for _, tc := range []struct {
		term string
		want []uuid.UUID
	}{
		{"WEB-12", []uuid.UUID{ids[0], ids[2]}},
		{"12", []uuid.UUID{ids[1], ids[0], ids[2]}},
		{"00012", []uuid.UUID{ids[1], ids[0]}},
		{"100%", ids[3:4]}, {"under_", ids[5:6]}, {`a\b`, ids[7:8]},
		{"登录", []uuid.UUID{ids[10], ids[9]}}, {"needle", nil},
		{"same tie", []uuid.UUID{ids[11], ids[13], ids[12]}},
	} {
		t.Run(tc.term, func(t *testing.T) { run(t, map[string][]string{"q": {tc.term}}, false, scope, tc.want...) })
	}
	t.Run("empty updated then id", func(t *testing.T) { run(t, map[string][]string{"limit": {"3"}}, false, scope, ids[0], ids[2], ids[1]) })
	t.Run("limit one", func(t *testing.T) { run(t, map[string][]string{"q": {"WEB-12"}, "limit": {"1"}}, false, scope, ids[0]) })
	// Rejected rows lead the order: LIMIT before OwnOnly loses admitted ties.
	exec(t, pool, "UPDATE issues SET created_by_id=$2 WHERE id=$1", ids[11], b)
	t.Run("own before limit", func(t *testing.T) {
		run(t, map[string][]string{"q": {"same tie"}, "limit": {"2"}}, true, scope, ids[13], ids[12])
	})
	t.Run("scope before limit", func(t *testing.T) { run(t, map[string][]string{"q": {"12"}, "limit": {"1"}}, false, scope[:1], ids[0]) })
	t.Run("empty scope", func(t *testing.T) { run(t, nil, false, nil) })

	// Exact and numeric OR branches remain subject to every ordinary filter.
	exec(t, pool, "UPDATE issues SET start_date='2026-01-02' WHERE id=$1", ids[0])
	t.Run("exact date exclusion", func(t *testing.T) {
		run(t, map[string][]string{"q": {"WEB-12"}, "start_date": {"none"}}, false, scope, ids[2])
	})
	t.Run("exact date admission", func(t *testing.T) {
		run(t, map[string][]string{"q": {"WEB-12"}, "start_date": {"2026-01-02"}}, false, scope, ids[0])
	})
	exec(t, pool, "UPDATE issues SET target_date='2026-02-02' WHERE id=$1", ids[1])
	t.Run("sequence date exclusion", func(t *testing.T) {
		run(t, map[string][]string{"q": {"12"}, "target_date": {"none"}}, false, scope, ids[0], ids[2])
	})
	exec(t, pool, "UPDATE issues SET created_by_id=$2 WHERE id=$1", ids[0], b)
	t.Run("exact own exclusion", func(t *testing.T) { run(t, map[string][]string{"q": {"WEB-12"}}, true, scope, ids[2]) })
	t.Run("sequence own exclusion", func(t *testing.T) { run(t, map[string][]string{"q": {"12"}}, true, scope, ids[2]) })
}

func searchIDs(t *testing.T, rows []domain.Issue, want ...uuid.UUID) {
	t.Helper()
	got := make([]uuid.UUID, len(rows))
	for n, row := range rows {
		got[n] = row.ID
	}
	if !slices.Equal(got, want) {
		t.Fatalf("IDs %v; want %v", got, want)
	}
}
