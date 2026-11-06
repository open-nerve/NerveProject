package bootstrap

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Independent observation directory: all thirteen issue tables, followed by
// the project tables with rows in this fixture. It does not read Module.Tables.
var purgeScaleTables = []string{
	"comment_reactions", "issue_reactions", "issue_mentions", "issue_description_versions",
	"issue_activities", "issue_comments", "issue_links", "issue_relations", "issue_assignees",
	"issue_labels", "issue_subscribers", "draft_issues", "issues",
	"project_user_properties", "project_members", "labels", "states", "projects",
}

type purgeScaleScope struct {
	Project uuid.UUID
	PAT     string
}

type purgeScaleFixture struct {
	Scopes                     []purgeScaleScope // 61d, 59d, live; in that order.
	Anchor                     time.Time
	Seed, Analyze, AgedAnalyze time.Duration
}

func preparePurgeScale(t *testing.T, roles splitRoles, dir string) purgeScaleFixture {
	t.Helper()
	var f purgeScaleFixture
	started := time.Now()
	if !t.Run("prepare", func(t *testing.T) {
		w := newIssueWorldOn(t, roles.serverURL, roles.owner, func(c *config.Config) { c.Auth.AccessTokenTTL = time.Hour })
		waitForCompletedKinds(t, roles.owner, map[string]int{
			"identity.cleanup_expired_sessions": 1, "issue.auto_archive": 1,
			"cleanup.purge_deleted": 1, "issue.trim_description_versions": 1,
		}, &lockedBuffer{})
		w.makeMember(t, "walt", w.web, shared.RoleMember)
		w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+w.web.String()+"/labels", `{"name":"Feature"}`, 201)
		f.Seed, f.Analyze = w.seedAtScale(t, []uuid.UUID{w.web})
		checkPurgeScaleSeed(t, w, dir)
		for i, name := range []string{"old", "recent", "live"} {
			project := w.web
			var ids []uuid.UUID
			if i == 0 {
				ids = queryIDs(t, roles.owner, "SELECT id FROM issues WHERE project_id=$1 ORDER BY sequence_id LIMIT 2", project)
			} else {
				project = createdProject(t, w.contract, w.base, w.tokens["alice"], "acme", name, fmt.Sprintf("PUR%d", i))
				w.makeMember(t, "bob", project, shared.RoleMember)
				var label struct{ ID uuid.UUID }
				decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/labels", `{"name":"Keep"}`, 201), &label)
				body := fmt.Sprintf(`{"name":"Keep","label_ids":[%q],"assignee_ids":[%q]}`, label.ID, w.ids["bob"])
				ids = []uuid.UUID{w.newWorkItemIn(t, project, body), w.newWorkItemIn(t, project, body)}
			}
			seedPurgeSparseRows(t, w, project, ids)
			pat := createPAT(t, w.contract, w.base, w.tokens["alice"])
			f.Scopes = append(f.Scopes, purgeScaleScope{Project: project, PAT: pat.ID})
			if i < 2 {
				w.send(t, "alice", http.MethodDelete, "/api/v0/api-tokens/"+pat.ID, "", 204)
				w.send(t, "alice", http.MethodDelete, "/api/v0/projects/"+project.String(), "", 204)
				waitDeletionJobWithin(t, roles.owner, projectDeletionJob(t, roles.owner, project), "completed", 90*time.Second)
			}
		}
	}) {
		t.Fatal("purge scale preparation failed")
	}
	// Subtest cleanups have stopped and joined the preparation app and workers.
	var pending int
	if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM river_job WHERE state NOT IN ('completed','cancelled','discarded')").Scan(&pending); err != nil || pending != 0 {
		t.Fatalf("preparation left pending work: jobs=%d err=%v", pending, err)
	}
	f.Anchor = time.Now().UTC().Truncate(time.Microsecond)
	tx, err := roles.owner.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	for i, age := range []int{61, 59} {
		at := f.Anchor.Add(-time.Duration(age) * 24 * time.Hour)
		for _, table := range purgeScaleTables {
			tag, err := tx.Exec(pgtest.Soon(t), "UPDATE "+pgx.Identifier{table}.Sanitize()+" SET deleted_at=$2 WHERE "+purgeScopeKey(table)+"=$1 AND deleted_at IS NOT NULL", f.Scopes[i].Project, at)
			if err != nil || tag.RowsAffected() == 0 {
				t.Fatalf("age %s/%dd: rows=%d err=%v", table, age, tag.RowsAffected(), err)
			}
		}
		if tag, err := tx.Exec(pgtest.Soon(t), "UPDATE api_tokens SET deleted_at=$2 WHERE id=$1 AND deleted_at IS NOT NULL", f.Scopes[i].PAT, at); err != nil || tag.RowsAffected() != 1 {
			t.Fatalf("age PAT/%dd: rows=%d err=%v", age, tag.RowsAffected(), err)
		}
	}
	if err := tx.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	// The seed's statistics predate the soft delete and the aging: they say no
	// row is tombstoned, so the planner would read every tombstone for each
	// page until autovacuum next analyzes. A deployment's autovacuum analyzed
	// these tombstones long before they aged 60 days.
	analyzing := time.Now()
	ctx, cancel := context.WithTimeout(t.Context(), time.Minute)
	defer cancel()
	if _, err := roles.owner.Exec(ctx, "ANALYZE "+strings.Join(append(append([]string{}, purgeScaleTables...), "api_tokens"), ", ")); err != nil {
		t.Fatal(err)
	}
	f.AgedAnalyze = time.Since(analyzing)
	listEvidence(t, dir, "fixture", map[string]any{
		"scopes": f.Scopes, "aging_anchor": f.Anchor, "ages_days": []int{61, 59, 0},
		"seed_seconds": f.Seed.Seconds(), "analyze_seconds": f.Analyze.Seconds(), "analyze_after_aging_seconds": f.AgedAnalyze.Seconds(),
		"preparation_including_api_soft_delete_seconds": time.Since(started).Seconds(), "pending_jobs_after_stop": pending,
	})
	return f
}

func seedPurgeSparseRows(t *testing.T, w issueWorld, project uuid.UUID, ids []uuid.UUID) {
	t.Helper()
	if len(ids) != 2 {
		t.Fatal("sparse fixture needs two issues", ids)
	}
	w.send(t, "alice", http.MethodPatch, "/api/v0/me/projects/"+project.String()+"/preferences", `{"sort_order":3}`, 200)
	patchDescription(t, w, "alice", ids[0], mentionHTML(w.ids["bob"]), false)
	w.send(t, "alice", http.MethodPost, "/api/v0/issues/"+ids[0].String()+"/relations", fmt.Sprintf(`{"relation_type":"relates_to","issue_ids":[%q]}`, ids[1]), 201)
	link := createLink(t, w, "alice", ids[0], "http://169.254.169.254/purge-scale")
	waitUniqueLinkCrawl(t, w, link)
	w.send(t, "bob", http.MethodPost, reactionsOf(ids[0]), `{"reaction":"128077"}`, 201)
	comment := w.newComment(t, "alice", ids[0])
	addCommentReaction(t, w, "bob", comment.ID, "128077")
	createDraftHTTP(t, w, "alice", fmt.Sprintf(`{"project_id":%q,"payload":{"name":"Keep draft"}}`, project))
}

func checkPurgeScaleSeed(t *testing.T, w issueWorld, dir string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(t.Context(), 3*time.Minute)
	defer cancel()
	var counts []scaleRows
	total := 0
	for _, table := range []struct {
		name          string
		count, fanout int
	}{
		{"issues", 20000, 1}, {"issue_activities", 400000, 20}, {"issue_assignees", 40000, 2},
		{"issue_labels", 40000, 2}, {"issue_subscribers", 60000, 3},
		{"issue_comments", 40000, 2}, {"issue_description_versions", 60000, 3},
	} {
		r := scaleRows{Table: table.name, Project: w.web}
		key := "issue_id"
		if table.name == "issues" {
			key = "id"
		}
		if err := w.pool.QueryRow(ctx, "SELECT count(*),count(*) FILTER(WHERE deleted_at IS NULL) FROM "+table.name+" WHERE project_id=$1", w.web).Scan(&r.Total, &r.Live); err != nil {
			t.Fatal(err)
		}
		if err := w.pool.QueryRow(ctx, "SELECT min(n),max(n) FROM (SELECT count(*) n FROM "+table.name+" WHERE project_id=$1 GROUP BY "+key+") s", w.web).Scan(&r.MinFanout, &r.MaxFanout); err != nil {
			t.Fatal(err)
		}
		if r.Total != table.count || r.Live != table.count || r.MinFanout != table.fanout || r.MaxFanout != table.fanout {
			t.Fatalf("purge scale seed %+v; want count=%d fanout=%d", r, table.count, table.fanout)
		}
		total += r.Total
		counts = append(counts, r)
	}
	r := scalePayload{Project: w.web}
	if err := w.pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE field='description'),
 min(octet_length(new_value)) FILTER(WHERE field='description'),max(octet_length(new_value)) FILTER(WHERE field='description'),
 count(DISTINCT new_value) FILTER(WHERE field='description') FROM issue_activities WHERE project_id=$1`, w.web).Scan(&r.Activities, &r.Descriptions, &r.MinBytes, &r.MaxBytes, &r.DistinctPayloads); err != nil {
		t.Fatal(err)
	}
	listEvidence(t, dir, "seed-distribution", map[string]any{"counts": counts, "payload": r, "total": total})
	if total != 660000 || r.Activities != 400000 || r.Descriptions != 80000 || r.MinBytes != 1024 || r.MaxBytes != 1024 || r.DistinctPayloads != 80000 {
		t.Fatalf("purge scale payload=%+v total=%d", r, total)
	}
}

func purgeScopeKey(table string) string {
	if table == "projects" {
		return "id"
	}
	return "project_id"
}

func purgeScopeCounts(t *testing.T, pool *pgxpool.Pool, scope purgeScaleScope, deleted bool) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for _, table := range append(append([]string{}, purgeScaleTables...), "api_tokens") {
		key, id := purgeScopeKey(table), any(scope.Project)
		if table == "api_tokens" {
			key, id = "id", scope.PAT
		}
		var total, dead int64
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*),count(*) FILTER(WHERE deleted_at IS NOT NULL) FROM "+table+" WHERE "+key+"=$1", id).Scan(&total, &dead); err != nil {
			t.Fatal(err)
		}
		if deleted && total != dead || !deleted && dead != 0 {
			t.Fatalf("purge scope %s tombstones=%d/%d want deleted=%t", table, dead, total, deleted)
		}
		out[table] = total
	}
	return out
}

// purgeableRows counts, across every scope, each table's rows tombstoned
// before cutoff.
func purgeableRows(t *testing.T, pool *pgxpool.Pool, cutoff time.Time, tables map[string]int64) map[string]int64 {
	t.Helper()
	out := map[string]int64{}
	for table := range tables {
		var n int64
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()+" WHERE deleted_at < $1", cutoff).Scan(&n); err != nil {
			t.Fatal(err)
		}
		out[table] = n
	}
	return out
}

// Retention scopes have only two issues each. Preserve complete rows, not
// only counts, so a touch/rewrite of a surviving row also fails.
func purgeRetentionSnapshot(t *testing.T, pool *pgxpool.Pool, scopes []purgeScaleScope) map[string]string {
	t.Helper()
	out := map[string]string{}
	for i, scope := range scopes {
		for _, table := range append(append([]string{}, purgeScaleTables...), "api_tokens") {
			key, id := purgeScopeKey(table), any(scope.Project)
			if table == "api_tokens" {
				key, id = "id", scope.PAT
			}
			var rows string
			if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]')::text FROM "+table+" r WHERE "+key+"=$1", id).Scan(&rows); err != nil {
				t.Fatal(err)
			}
			out[fmt.Sprintf("%d/%s", i, table)] = rows
		}
	}
	for _, table := range []string{"workspace_user_properties", "workspace_member_invites", "workspace_members", "workspaces"} {
		var rows string
		if err := pool.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]')::text FROM "+table+" r").Scan(&rows); err != nil {
			t.Fatal(err)
		}
		out[table] = rows
	}
	return out
}
