package bootstrap

import (
	"context"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver/apitest"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// Every business fixture is created and soft deleted through the runtime app.
// Only the isolated owner connection ages tombstones and observes results.
func TestPurgeDeletedRunsWithRuntimeRole(t *testing.T) {
	roles := newSplitRoles(t)
	var runtimeUser string
	var isOwner bool
	if err := roles.server.QueryRow(pgtest.Soon(t), `SELECT current_user, current_user=pg_get_userbyid(relowner) FROM pg_class WHERE oid='issues'::regclass`).Scan(&runtimeUser, &isOwner); err != nil || runtimeUser != roles.serverName || isOwner {
		t.Fatalf("runtime connection=%s owner=%t err=%v", runtimeUser, isOwner, err)
	}
	// This observer runs with invoker privileges, including the actual runtime
	// login in its row. It cannot make a cascade look like correct module order.
	for _, statement := range []string{
		"CREATE TABLE purge_execution (ordinal bigint GENERATED ALWAYS AS IDENTITY, table_name text, module_name text, role_name text)",
		"GRANT INSERT ON purge_execution TO nerve_runtime",
		"GRANT USAGE ON SEQUENCE purge_execution_ordinal_seq TO nerve_runtime",
		`CREATE FUNCTION observe_purge() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN INSERT INTO purge_execution(table_name,module_name,role_name) VALUES (TG_TABLE_NAME,TG_ARGV[0],current_user); RETURN OLD; END $$`,
	} {
		if _, err := roles.owner.Exec(pgtest.Soon(t), statement); err != nil {
			t.Fatal(err)
		}
	}
	for _, group := range []struct {
		name   string
		tables []string
	}{
		{"issue", []string{"issue_labels", "issue_subscribers", "issue_comments", "issue_activities", "issues"}},
		{"project", []string{"project_members", "labels", "states", "projects"}},
		{"workspace", []string{"workspace_members", "workspaces"}},
		{"identity", []string{"api_tokens"}},
	} {
		for _, table := range group.tables {
			if _, err := roles.owner.Exec(pgtest.Soon(t), "CREATE TRIGGER observe_purge BEFORE DELETE ON "+pgx.Identifier{table}.Sanitize()+" FOR EACH ROW EXECUTE FUNCTION observe_purge('"+group.name+"')"); err != nil {
				t.Fatal(err)
			}
		}
	}
	type fixture struct {
		workspace uuid.UUID
		pat       string
	}
	var fixtures []fixture
	if !t.Run("prepare runtime fixtures", func(t *testing.T) {
		cfg := testConfig(t, roles.serverURL, false)
		cfg.Cleanup.PurgeInterval = time.Second
		cfg.Cleanup.PurgeBatch = 1
		base := startApp(t, cfg, migrations.FS())
		contract := apitest.Load(t)
		token := registerAccount(t, contract, base, "purge-runtime@example.com").AccessToken
		w := issueWorld{contract: contract, base: base, pool: roles.owner, tokens: map[string]string{"alice": token}}
		for _, slug := range []string{"old", "recent"} {
			var created struct {
				ID uuid.UUID `json:"id"`
			}
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/workspaces", `{"name":"Purge","slug":"`+slug+`"}`, 201), &created)
			workspace := created.ID
			project := createdProject(t, contract, base, token, slug, "Purge", "PUR")
			decodeAnswer(t, w.send(t, "alice", http.MethodPost, "/api/v0/projects/"+project.String()+"/labels", `{"name":"Purge"}`, 201), &created)
			item := w.newWorkItemIn(t, project, `{"name":"Purge","label_ids":["`+created.ID.String()+`"]}`)
			w.send(t, "alice", http.MethodPost, commentsOf(item), `{"comment_html":"<p>Keep the child visible to cleanup</p>"}`, 201)
			credential := createPAT(t, contract, base, token)
			w.send(t, "alice", http.MethodDelete, "/api/v0/api-tokens/"+credential.ID, "", 204)
			w.send(t, "alice", http.MethodDelete, "/api/v0/workspaces/"+slug, "", 204)
			waitWorkspaceDeletion(t, roles.owner, workspace)
			fixtures = append(fixtures, fixture{workspace, credential.ID})
		}
	}) {
		return
	}
	// The preparation app and all its workers have stopped before aging.
	// No purge can cross this commit between its independent module batches.
	tx, err := roles.owner.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	tables := []string{"issue_labels", "issue_subscribers", "issue_comments", "issue_activities", "issues", "labels", "states", "project_members", "projects", "workspace_members", "workspaces"}
	for i, f := range fixtures {
		age := 61
		if i == 1 {
			age = 59
		}
		for _, table := range tables {
			key := "workspace_id"
			if table == "workspaces" {
				key = "id"
			}
			tag, err := tx.Exec(pgtest.Soon(t), "UPDATE "+pgx.Identifier{table}.Sanitize()+" SET deleted_at=now()-make_interval(days=>$2) WHERE "+key+"=$1 AND deleted_at IS NOT NULL", f.workspace, age)
			if err != nil || tag.RowsAffected() == 0 {
				t.Fatalf("age %s: rows=%d err=%v", table, tag.RowsAffected(), err)
			}
		}
		if _, err := tx.Exec(pgtest.Soon(t), "UPDATE api_tokens SET deleted_at=now()-make_interval(days=>$2) WHERE id=$1", f.pat, age); err != nil {
			t.Fatal(err)
		}
	}
	if err := tx.Commit(pgtest.Soon(t)); err != nil {
		t.Fatal(err)
	}
	snapshots := make(map[string]string)
	for _, table := range tables {
		key := "workspace_id"
		if table == "workspaces" {
			key = "id"
		}
		var before string
		if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+pgx.Identifier{table}.Sanitize()+" r WHERE "+key+"=$1", fixtures[1].workspace).Scan(&before); err != nil {
			t.Fatal(err)
		}
		snapshots[table] = before
	}
	var patBefore string
	if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT to_jsonb(r)::text FROM api_tokens r WHERE id=$1", fixtures[1].pat).Scan(&patBefore); err != nil {
		t.Fatal(err)
	}
	// Record only completed work before starting the observation app. Its
	// worker must complete a further job after aging and retention snapshots.
	completedBefore := completedKind(t, roles.owner, "cleanup.purge_deleted")
	var observedBefore int
	if err := roles.owner.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM purge_execution").Scan(&observedBefore); err != nil || observedBefore != 0 {
		t.Fatalf("DELETE observer before observation app: rows=%d err=%v", observedBefore, err)
	}
	cfg := testConfig(t, roles.serverURL, false)
	cfg.Cleanup.PurgeInterval = time.Second
	cfg.Cleanup.PurgeBatch = 1
	var logs lockedBuffer
	startAppLogging(t, cfg, migrations.FS(), slog.New(slog.NewTextHandler(&logs, nil)))
	ctx, cancel := context.WithTimeout(t.Context(), 15*time.Second)
	defer cancel()
	awaitPurge(t, ctx, &logs, "runtime deletes 61d workspace and PAT", func() bool {
		var remaining int
		if err := roles.owner.QueryRow(ctx, `SELECT (SELECT count(*) FROM workspaces WHERE id=$1)+(SELECT count(*) FROM api_tokens WHERE id=$2)`, fixtures[0].workspace, fixtures[0].pat).Scan(&remaining); err != nil {
			t.Fatal(err)
		}
		return remaining == 0
	})
	for _, table := range tables {
		key := "workspace_id"
		if table == "workspaces" {
			key = "id"
		}
		var old int
		var after string
		if err := roles.owner.QueryRow(ctx, "SELECT count(*) FROM "+pgx.Identifier{table}.Sanitize()+" WHERE "+key+"=$1", fixtures[0].workspace).Scan(&old); err != nil {
			t.Fatal(err)
		}
		if err := roles.owner.QueryRow(ctx, "SELECT coalesce(jsonb_agg(to_jsonb(r) ORDER BY id),'[]'::jsonb)::text FROM "+pgx.Identifier{table}.Sanitize()+" r WHERE "+key+"=$1", fixtures[1].workspace).Scan(&after); err != nil {
			t.Fatal(err)
		}
		if old != 0 || after != snapshots[table] {
			t.Errorf("runtime %s: old=%d recent unchanged=%t", table, old, after == snapshots[table])
		}
	}
	var patAfter string
	if err := roles.owner.QueryRow(ctx, "SELECT to_jsonb(r)::text FROM api_tokens r WHERE id=$1", fixtures[1].pat).Scan(&patAfter); err != nil || patAfter != patBefore {
		t.Errorf("59d PAT changed: %v", err)
	}
	if strings.Contains(logs.String(), "permission denied") {
		t.Errorf("runtime permission failure:\n%s", logs.String())
	}
	waitForCompletedKinds(t, roles.owner, map[string]int{"cleanup.purge_deleted": completedBefore + 1}, &logs)
	var observed, moduleOrder []string
	var wrongRole int
	if err := roles.owner.QueryRow(ctx, "SELECT array_agg(table_name ORDER BY ordinal),array_agg(module_name ORDER BY ordinal),count(*) FILTER (WHERE role_name<>$1) FROM purge_execution", roles.serverName).Scan(&observed, &moduleOrder, &wrongRole); err != nil {
		t.Fatal(err)
	}
	moduleOrder = slices.Compact(moduleOrder)
	if !slices.Equal(moduleOrder, []string{"issue", "project", "workspace", "identity"}) {
		t.Fatalf("runtime DELETE module order=%v tables=%v; want [issue project workspace identity]", moduleOrder, observed)
	}
	if wrongRole != 0 {
		t.Fatalf("DELETEs outside runtime role %s: %d; tables=%v", roles.serverName, wrongRole, observed)
	}
	t.Logf("observation app completed a purge after preparation app stopped; completed before=%d", completedBefore)
	t.Logf("runtime DELETE module order=%v tables=%v", moduleOrder, observed)
	t.Logf("purge executed as %s (not owner): 61d hierarchy/PAT removed; 59d rows unchanged", runtimeUser)
}
