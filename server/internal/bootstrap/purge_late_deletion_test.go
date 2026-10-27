package bootstrap

import (
	"context"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

// This is an isolated storage state: only the parent tombstone is written.
// Current project/workspace APIs still cascade synchronously. No late River
// delete_rows worker, retry or deadlock property is claimed by this fixture.
func TestPurgeParentCascadesUnprocessedIssueRows(t *testing.T) {
	for _, parent := range []string{"projects", "workspaces"} {
		t.Run(parent, func(t *testing.T) {
			w := newRetentionWorld(t)
			registerAccount(t, w.contract, w.base, "member@example.com")
			var parents []uuid.UUID
			for _, slug := range []string{"old", "recent", "kept"} {
				workspace := seedWorkspace(t, w.contract, w.base, w.pool, w.tokens["alice"], slug)
				id := workspace
				if parent == "projects" {
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM projects WHERE workspace_id=$1 AND name='Web'", workspace).Scan(&id); err != nil {
						t.Fatal(err)
					}
				}
				parents = append(parents, id)
			}
			keys := keysTo(t, w.pool, parent)
			// Every FK endpoint has fixture rows, including comment_reactions and
			// drafts. Record IDs now so an absent parent cannot hide surviving children.
			recorded := map[foreignKey][]uuid.UUID{}
			kept := map[foreignKey]string{}
			recent := map[foreignKey]string{}
			for _, k := range keys {
				ids := k.undeleted(t, w.pool, parents[0])
				if len(ids) == 0 {
					t.Fatalf("%s has no live child to cascade", k)
				}
				recorded[k] = append(ids, k.deleted(t, w.pool, parents[0])...)
				kept[k] = k.rows(t, w.pool, parents[2])
				if kept[k] == "" {
					t.Fatalf("%s has no live neighbor", k)
				}
			}
			// A project cascade also leaves its live sibling and unassigned
			// drafts in the same workspace untouched.
			var sibling uuid.UUID
			var looseDrafts string
			siblingRows := map[foreignKey]string{}
			if parent == "projects" {
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT id FROM projects WHERE workspace_id=(SELECT workspace_id FROM projects WHERE id=$1) AND id<>$1", parents[0]).Scan(&sibling); err != nil {
					t.Fatal(err)
				}
				for _, k := range keys {
					siblingRows[k] = k.rows(t, w.pool, sibling)
				}
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT jsonb_agg(to_jsonb(d) ORDER BY id)::text FROM draft_issues d WHERE project_id IS NULL AND workspace_id=(SELECT workspace_id FROM projects WHERE id=$1)", sibling).Scan(&looseDrafts); err != nil {
					t.Fatal(err)
				}
			}
			// Exercise issue self-CASCADE as well as the already seeded label tree.
			issueKey := foreignKey{parent, "issues", "workspace_id"}
			if parent == "projects" {
				issueKey.column = "project_id"
			}
			items := recorded[issueKey]
			if len(items) < 2 {
				t.Fatal("fixture needs issue descendants")
			}
			underWorkItem(t, w.pool, items[0], items[1])
			// Also record every issue-level FK, including the second-level comment
			// reaction chain, independently of the direct parent-key guard.
			descendants := map[foreignKey][]uuid.UUID{}
			for _, k := range keysTo(t, w.pool, "issues") {
				for _, id := range items {
					descendants[k] = append(descendants[k], k.undeleted(t, w.pool, id)...)
					descendants[k] = append(descendants[k], k.deleted(t, w.pool, id)...)
				}
				if len(descendants[k]) == 0 {
					t.Fatalf("%s missing issue descendant", k)
				}
			}
			accounts := tableRows(t, w.pool, func(name string) bool { return name != "public.users" && name != "public.profiles" })
			at := time.Now().UTC().Truncate(time.Microsecond)
			// Both tombstones commit together. The recent parent chain has no old
			// ancestor, so its 59-day children cannot disappear through another purge.
			tx, err := w.pool.Begin(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			defer requestRelease(t, tx)
			for i, age := range []int{61, 59} {
				if _, err := tx.Exec(pgtest.Soon(t), "UPDATE "+pgx.Identifier{parent}.Sanitize()+" SET deleted_at=$2,updated_at=$2,updated_by_id=$3 WHERE id=$1", parents[i], at.Add(-time.Duration(age)*24*time.Hour), w.ids["alice"]); err != nil {
					t.Fatal(err)
				}
			}
			// Snapshot the recent parent while its updated row is still uncommitted.
			// Child rows remain exactly as seeded, including their earlier tombstones.
			for _, k := range keys {
				var body string
				if err := tx.QueryRow(pgtest.Soon(t), "SELECT coalesce(string_agg(r::text,E'\\n' ORDER BY r::text),'') FROM "+pgx.Identifier{k.table}.Sanitize()+" r WHERE "+k.scope(), parents[1]).Scan(&body); err != nil {
					t.Fatal(err)
				}
				recent[k] = body
			}
			if err := tx.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := retentionPurge(w, at).Execute(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			for k, ids := range recorded {
				if rows := k.asText(t, w.pool, ids); rows != "" {
					t.Errorf("parent CASCADE left %s rows: %s", k, rows)
				}
				if rows := k.rows(t, w.pool, parents[2]); rows != kept[k] {
					t.Errorf("live neighbor changed at %s", k)
				}
				if rows := k.rows(t, w.pool, parents[1]); rows != recent[k] {
					t.Errorf("59-day parent/child audit changed at %s", k)
				}
			}
			if parent == "projects" {
				for k, before := range siblingRows {
					if k.rows(t, w.pool, sibling) != before {
						t.Errorf("same-workspace sibling changed at %s", k)
					}
				}
				var after string
				if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT jsonb_agg(to_jsonb(d) ORDER BY id)::text FROM draft_issues d WHERE project_id IS NULL AND workspace_id=(SELECT workspace_id FROM projects WHERE id=$1)", sibling).Scan(&after); err != nil || after != looseDrafts {
					t.Fatalf("unassigned drafts changed: %v", err)
				}
			}
			for k, ids := range descendants {
				if rows := k.asText(t, w.pool, ids); rows != "" {
					t.Errorf("issue FK chain survived at %s: %s", k, rows)
				}
			}
			if after := tableRows(t, w.pool, func(name string) bool { return name != "public.users" && name != "public.profiles" }); !reflect.DeepEqual(accounts, after) {
				t.Fatal("parent purge changed users/profiles")
			}
			before := registrationRows(t, w.pool)
			if _, err := retentionPurge(w, at).Execute(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			if registrationRows(t, w.pool) != before {
				t.Fatal("repeat parent purge changed survivors")
			}
			t.Logf("%s CASCADE: %d parent FK endpoints and %d issue FK endpoints empty; 59d and live neighbors/account rows unchanged", parent, len(recorded), len(descendants))
		})
	}
}
