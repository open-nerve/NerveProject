package bootstrap

import (
	"context"
	"net/http"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	issueapp "github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Keep all real periodic workers active at the task's two-second cadence.
// Fixtures are atomic and expectations allow a background pass to win; no
// startup drain, delayed schedule or elapsed-time exclusion protects the test.
func newRetentionWorld(t *testing.T) issueWorld {
	t.Helper()
	return newIssueWorldWith(t, func(cfg *config.Config) {
		cfg.Cleanup.PurgeInterval = 2 * time.Second
		cfg.Issue.TrimVersionsInterval = 2 * time.Second
		cfg.Issue.AutoArchiveInterval = 2 * time.Second
	})
}

// Pause before the real DELETE, after the production project gate. This seam
// neither acquires a lock nor substitutes candidate discovery or retention SQL.
type retentionRows struct {
	issueapp.DescriptionVersionTrimmer
	barrier *gate
	calls   int
}

func (r *retentionRows) TrimDescriptionVersions(ctx context.Context, id uuid.UUID) (int, error) {
	r.calls++
	if r.barrier != nil {
		if err := r.barrier.wait(ctx); err != nil {
			return 0, err
		}
	}
	return r.DescriptionVersionTrimmer.TrimDescriptionVersions(ctx, id)
}
func retentionJob(w issueWorld, tx shared.TxManager, rows issueapp.DescriptionVersionTrimmer) *issueapp.TrimVersions {
	p := issueProjects{projects: project.Provide(w.pool).IssueProjects}
	return issueapp.NewTrimVersions(issueapp.TrimVersionsDeps{Projects: p, Gate: p, Rows: rows, Tx: tx})
}
func retentionTx(w issueWorld) *requestProbe {
	return &requestProbe{TxManager: newIssueJobTx(w.pool, config.Config{Database: config.DatabaseConfig{CommitTimeout: 2 * time.Second}, Issue: testIssueConfig})}
}

// All seeded live rows tie on last_saved_at. Insert in reverse UUID order so
// the physical order cannot accidentally stand in for the required id DESC.
func retentionSeed(t *testing.T, tx pgx.Tx, item uuid.UUID, n int, at time.Time, deleted *time.Time) []uuid.UUID {
	t.Helper()
	ids := make([]uuid.UUID, n)
	for i := range ids {
		ids[i] = uuid.NewV7()
	}
	for i := len(ids) - 1; i >= 0; i-- {
		_, err := tx.Exec(pgtest.Soon(t), `INSERT INTO issue_description_versions(id,workspace_id,project_id,issue_id,owned_by_id,last_saved_at,created_at,updated_at,created_by_id,updated_by_id,deleted_at,description_html,description_stripped)
 SELECT $1,workspace_id,project_id,id,created_by_id,$3,$3,$3,created_by_id,created_by_id,$4,'<p>Retained</p>','Retained' FROM issues WHERE id=$2`, ids[i], item, at, deleted)
		if err != nil {
			t.Fatal(err)
		}
	}
	return ids
}
func retentionFixture(t *testing.T, w issueWorld, item uuid.UUID, n int, at time.Time) []uuid.UUID {
	t.Helper()
	tx, err := w.pool.Begin(pgtest.Soon(t))
	if err != nil {
		t.Fatal(err)
	}
	defer requestRelease(t, tx)
	if _, err := tx.Exec(pgtest.Soon(t), "DELETE FROM issue_description_versions WHERE issue_id=$1", item); err != nil {
		t.Fatal(err)
	}
	ids := retentionSeed(t, tx, item, n, at, nil)
	if err := tx.Commit(context.Background()); err != nil {
		t.Fatal(err)
	}
	return ids
}
func retentionSnapshot(t *testing.T, pool *pgxpool.Pool, ids []uuid.UUID) map[uuid.UUID]string {
	t.Helper()
	rows, err := pool.Query(pgtest.Soon(t), "SELECT id,to_jsonb(v)::text FROM issue_description_versions v WHERE id=ANY($1)", ids)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]string{}
	for rows.Next() {
		var id uuid.UUID
		var body string
		if err := rows.Scan(&id, &body); err != nil {
			t.Fatal(err)
		}
		out[id] = body
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
func retentionCheck(t *testing.T, w issueWorld, item uuid.UUID, want []uuid.UUID, before map[uuid.UUID]string) {
	t.Helper()
	want = slices.Clone(want)
	slices.SortFunc(want, uuid.UUID.Compare)
	if got := trimVersionIDs(t, w.pool, item); !slices.Equal(got, want) {
		t.Fatalf("retained IDs=%v want=%v", got, want)
	}
	after := retentionSnapshot(t, w.pool, want)
	for id, body := range before {
		if slices.Contains(want, id) && after[id] != body {
			t.Errorf("retention rewrote surviving version %s", id)
		}
	}
}

func TestTrimVersionsSerializesWithDescriptionWrites(t *testing.T) {
	for _, first := range []string{"request", "trim"} {
		t.Run(first+"_first", func(t *testing.T) {
			w := newRetentionWorld(t)
			item := w.newWorkItemIn(t, w.web, `{"name":"Retention"}`)
			at := time.Now().UTC().Truncate(time.Microsecond)
			ids := retentionFixture(t, w, item, 20, at)
			before := retentionSnapshot(t, w.pool, ids)
			oldActivities := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1", item)
			activityKey := foreignKey{table: "issue_activities"}
			oldActivityRows := activityKey.asText(t, w.pool, oldActivities)
			g := newGate()
			rp, jp := requestTx(w.pool), retentionTx(w)
			var tx shared.TxManager = rp
			rows := &retentionRows{DescriptionVersionTrimmer: issuepg.New(w.pool)}
			if first == "request" {
				tx = heldTx{rp, g}
			} else {
				rows.barrier = g
			}
			route := newIssueRouteWithClock(t, w.pool, tx, clockAt(at))
			var reply exchange
			request := w.sending(&reply, route, "bob", http.MethodPatch, "/api/v0/issues/"+item.String(), `{"description_html":"<p>Concurrent</p>"}`)
			trim := func() error { _, err := retentionJob(w, jp, rows).Execute(context.Background()); return err }
			retentionInterleave(t, w, g, first, request, trim, rp, jp)
			w.answered(t, reply, 200)
			// A following real pass converges both valid commit orders. Periodic trim
			// may already have done this; its exact count is deliberately not inferred.
			if _, err := retentionJob(w, retentionTx(w), issuepg.New(w.pool)).Execute(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			added := queryIDs(t, w.pool, "SELECT id FROM issue_description_versions WHERE issue_id=$1 AND NOT(id=ANY($2))", item, ids)
			if len(added) != 1 {
				t.Fatalf("description write created %d versions, want 1", len(added))
			}
			retentionCheck(t, w, item, append(slices.Clone(ids[1:]), added...), before)
			var savedAt time.Time
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT last_saved_at FROM issue_description_versions WHERE id=$1", added[0]).Scan(&savedAt); err != nil || !savedAt.Equal(at) {
				t.Fatalf("timestamp tie lost: %v %v", savedAt, err)
			}
			var html string
			if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT description_html FROM issues WHERE id=$1", item).Scan(&html); err != nil || html != "<p>Concurrent</p>" {
				t.Fatal(html, err)
			}
			if activityKey.asText(t, w.pool, oldActivities) != oldActivityRows {
				t.Fatal("retention/request rewrote old activity audit")
			}
			activities := queryIDs(t, w.pool, "SELECT id FROM issue_activities WHERE issue_id=$1", item)
			if len(activities) != len(oldActivities)+1 {
				t.Fatalf("description/trim activities=%d want=%d", len(activities), len(oldActivities)+1)
			}
		})
	}
}

func TestTrimVersionsSerializesWithProjectDeletion(t *testing.T) {
	for _, first := range []string{"request", "trim"} {
		t.Run(first+"_first", func(t *testing.T) {
			w := newRetentionWorld(t)
			item := w.newWorkItemIn(t, w.web, `{"name":"Deleted retention"}`)
			ids := retentionFixture(t, w, item, 20, time.Now().UTC().Add(-time.Hour))
			oldAt := time.Now().UTC().Add(-59 * 24 * time.Hour)
			seed, err := w.pool.Begin(pgtest.Soon(t))
			if err != nil {
				t.Fatal(err)
			}
			defer requestRelease(t, seed)
			old := retentionSeed(t, seed, item, 2, oldAt, &oldAt)
			if err := seed.Commit(context.Background()); err != nil {
				t.Fatal(err)
			}
			oldRows := retentionSnapshot(t, w.pool, old)
			g := newGate()
			rp, jp := requestTx(w.pool), retentionTx(w)
			var tx shared.TxManager = rp
			rows := &retentionRows{DescriptionVersionTrimmer: issuepg.New(w.pool)}
			if first == "request" {
				tx = heldTx{rp, g}
			} else {
				rows.barrier = g
			}
			var reply exchange
			request := deletesWeb(t, w, tx, &reply)
			count := 0
			trim := func() error {
				var err error
				count, err = retentionJob(w, jp, rows).Execute(context.Background())
				return err
			}
			retentionInterleave(t, w, g, first, request, trim, rp, jp)
			w.answered(t, reply, 204)
			if count != 0 || (first == "request" && rows.calls != 0) {
				t.Fatalf("trim after deletion count=%d SQL calls=%d", count, rows.calls)
			}
			waitProjectDeletion(t, w.pool, w.web)
			retentionCheck(t, w, item, append(slices.Clone(ids), old...), oldRows)
			var bad int
			if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT count(*) FROM issue_description_versions v JOIN projects p ON p.id=v.project_id WHERE v.issue_id=$1 AND v.id=ANY($3) AND (v.deleted_at IS DISTINCT FROM p.deleted_at OR v.updated_at IS DISTINCT FROM p.deleted_at OR v.updated_by_id IS DISTINCT FROM $2)`, item, w.ids["alice"], ids).Scan(&bad); err != nil || bad != 0 {
				t.Fatalf("deletion audit mismatch=%d %v", bad, err)
			}
			before := registrationRows(t, w.pool)
			if _, err := retentionJob(w, retentionTx(w), issuepg.New(w.pool)).Execute(pgtest.Soon(t)); err != nil {
				t.Fatal(err)
			}
			if after := registrationRows(t, w.pool); before != after {
				t.Fatal("trim changed deleted rows/audit")
			}
		})
	}
}

func retentionInterleave(t *testing.T, w issueWorld, g *gate, first string, request, trim func() error, rp, jp *requestProbe) {
	t.Helper()
	a, b, owner, waiter := request, trim, rp, jp
	if first == "trim" {
		a, b, owner, waiter = trim, request, jp, rp
	}
	done := run(a)
	collected := false
	t.Cleanup(func() {
		requestOpen(g)
		if !collected {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := result(t, ctx, done, "retention holder cleanup"); err != nil {
				t.Error(err)
			}
		}
	})
	held(t, pgtest.Soon(t), g, done, "retention first operation")
	pid := requestPID(t, owner)
	identity := deactivationIdentityOf(t, w.pool, pid)
	if identity.xid == "" {
		t.Fatal("holder has no real transaction ID")
	}
	// Strongest row mode is attributed only to the target's transaction ID.
	if mode := deactivationRowLock(t, w.pool, w.pool, map[string]string{identity.xid: "retention holder"}, "projects WHERE id=$1", w.web); mode != "FOR NO KEY UPDATE" {
		t.Fatalf("holder project mode=%s want N", mode)
	}
	join := requestBackground(t, b, func() { requestOpen(g) })
	requestBlocked(t, w.pool, waiter, pid, "projects")
	if after := deactivationIdentityOf(t, w.pool, pid); !reflect.DeepEqual(identity, after) {
		t.Fatalf("holder identity changed: %+v -> %+v", identity, after)
	}
	requestOpen(g)
	if err := result(t, pgtest.Soon(t), done, "retention holder"); err != nil {
		t.Fatal(err)
	}
	collected = true
	join()
	t.Logf("%s first: holder PID=%d XID=%s; both operations committed", first, pid, identity.xid)
}
