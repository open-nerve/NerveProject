package postgresadapter_test

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

const searchPerfSeed = 20261027

type searchPerfRow struct {
	ID, Author             uuid.UUID
	Project, Sequence      int
	Name                   string
	Chinese, Rare, Deleted bool
	Archived, ExactTitle   bool
	Created, Updated       time.Time
}

type searchPerfFixture struct {
	Actor, Other uuid.UUID
	Projects     [2]project
	Rows         []searchPerfRow
}

// Reuse newStore's migrated pgtest clone and production pool. The owner seeds
// and ANALYZEs; a separate login in the deployed runtime group measures reads.
func searchPerfPools(t *testing.T, trace *searchPerfTrace) (*pgxpool.Pool, *pgxpool.Pool) {
	t.Helper()
	_, owner := newStore(t)
	u, err := url.Parse(owner.Config().ConnString())
	if err != nil {
		t.Fatal("parse fixture database URL")
	}
	var database string
	if err = owner.QueryRow(context.Background(), "SELECT current_database()").Scan(&database); err != nil {
		t.Fatal(err)
	}
	role := database + "_search_runtime"
	exec(t, owner, "DO $$ BEGIN CREATE ROLE nerve_runtime NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$")
	exec(t, owner, "CREATE ROLE "+pgx.Identifier{role}.Sanitize()+" LOGIN PASSWORD 'performance' IN ROLE nerve_runtime")
	_, file, _, _ := runtime.Caller(0)
	grants, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../../../../deploy/runtime-grants.sql"))
	if err != nil {
		t.Fatal(err)
	}
	exec(t, owner, string(grants))
	u.User = url.UserPassword(role, "performance")
	source, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: u.String(), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	cfg := source.Config()
	cfg.ConnConfig.Tracer = trace
	source.Close()
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return owner, pool
}

func searchPerfID(n int) uuid.UUID {
	id, err := uuid.Parse(fmt.Sprintf("20261027-0000-4000-8000-%012d", n))
	if err != nil {
		panic(err)
	}
	return id
}

func seedSearchPerformance(t *testing.T, owner *pgxpool.Pool) searchPerfFixture {
	t.Helper()
	f := searchPerfFixture{Actor: newAccount(t, owner, "perf-actor"), Other: newAccount(t, owner, "perf-other")}
	w := newWorkspace(t, owner, "search-performance")
	f.Projects = [2]project{newProject(t, owner, w, "PERFA"), newProject(t, owner, w, "PERFB")}
	terms := []string{"lo", "login", "登录", "修复登录"}
	base := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	f.Rows = make([]searchPerfRow, 100000)
	input := make([][]any, len(f.Rows))
	for i := range f.Rows {
		group, language := i/8, i/2%2
		family := language*2 + i/4%2
		r := searchPerfRow{ID: searchPerfID(i + 1), Author: f.Actor, Project: i % 2, Sequence: i/2 + 1,
			Name: terms[family], Chinese: language == 1, Rare: group%37 == 0, Deleted: group%103 == 0,
			Archived: group%101 == 0, ExactTitle: group%10 == 0}
		if group%17 == 0 {
			r.Author = f.Other
		}
		if !r.ExactTitle {
			// Four letters give varied titles without adding numeric matches.
			n := (group*7919 + searchPerfSeed) % (26 * 26 * 26 * 26)
			suffix := []byte("aaaa")
			for j := range suffix {
				suffix[j], n = byte('a'+n%26), n/26
			}
			r.Name += " worker " + string(suffix)
		}
		r.Created = base.Add(time.Duration(group) * time.Second)
		if r.ExactTitle {
			// Exact short titles lead exact longer titles even in a locale that
			// treats Chinese as non-word characters (both scores then zero).
			r.Created = r.Created.Add(365 * 24 * time.Hour)
			if family%2 == 0 {
				r.Created = r.Created.Add(24 * time.Hour)
			}
		}
		r.Updated = r.Created.Add(time.Duration(i%7) * time.Hour)
		var start, archived, deleted any
		if r.Rare {
			start = time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)
		}
		if r.Archived {
			archived = base
		}
		if r.Deleted {
			deleted = base
		}
		p := f.Projects[r.Project]
		input[i] = []any{r.ID, w, p.id, p.state, r.Sequence, r.Name, r.Author, r.Created, r.Updated, start, archived, deleted}
		f.Rows[i] = r
	}
	ctx := context.Background()
	n, err := owner.CopyFrom(ctx, pgx.Identifier{"issues"}, []string{"id", "workspace_id", "project_id", "state_id", "sequence_id", "name", "created_by_id", "created_at", "updated_at", "start_date", "archived_at", "deleted_at"}, pgx.CopyFromRows(input))
	if err != nil || n != 100000 {
		t.Fatalf("seed issues: copied %d: %v", n, err)
	}
	// Populate the final full projection: exact-title leaders have one active
	// and one soft-deleted link, with metadata on the active link.
	links := make([][]any, 0, 20000)
	for _, r := range f.Rows {
		if !r.ExactTitle {
			continue
		}
		for j := 0; j < 2; j++ {
			var deleted any
			if j == 1 {
				deleted = base
			}
			links = append(links, []any{uuid.NewV7(), w, f.Projects[r.Project].id, r.ID, fmt.Sprintf("https://example.com/%d", j), `{"title":"fixture"}`, deleted})
		}
	}
	if _, err = owner.CopyFrom(ctx, pgx.Identifier{"issue_links"}, []string{"id", "workspace_id", "project_id", "issue_id", "url", "metadata", "deleted_at"}, pgx.CopyFromRows(links)); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"issues", "issue_relations", "issue_assignees", "issue_labels", "issue_links"} {
		exec(t, owner, "ANALYZE "+pgx.Identifier{table}.Sanitize())
	}
	return f
}

func (f searchPerfFixture) distribution(t *testing.T, pool *pgxpool.Pool) []map[string]any {
	t.Helper()
	rows, err := pool.Query(context.Background(), `SELECT project_id,
		count(*),count(*) FILTER(WHERE name ~ '[登录修复]'),
		count(*) FILTER(WHERE created_by_id=$1),count(*) FILTER(WHERE archived_at IS NOT NULL),
		count(*) FILTER(WHERE deleted_at IS NOT NULL),count(*) FILTER(WHERE start_date IS NOT NULL)
		FROM issues GROUP BY project_id ORDER BY project_id`, f.Actor)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var result []map[string]any
	for rows.Next() {
		var id uuid.UUID
		var total, chinese, own, archived, deleted, rare int
		if err = rows.Scan(&id, &total, &chinese, &own, &archived, &deleted, &rare); err != nil {
			t.Fatal(err)
		}
		if total != 50000 || chinese != 25000 {
			t.Fatalf("physical project distribution: %s total=%d Chinese=%d", id, total, chinese)
		}
		result = append(result, map[string]any{"project": id, "total": total, "english": total - chinese, "chinese": chinese, "actor_owned": own, "archived": archived, "deleted": deleted, "rare_date": rare})
	}
	if rows.Err() != nil || len(result) != 2 {
		t.Fatal("physical distribution", result, rows.Err())
	}
	return result
}
