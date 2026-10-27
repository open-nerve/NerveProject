package bootstrap

import (
	"context"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	issuepg "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
)

func retentionPurge(w issueWorld, at time.Time) *purgeDeleted {
	return newPurgeDeleted([]Purger{issue.Provide(w.pool).Purger, project.Provide(w.pool).Purger, workspace.Provide(w.pool).Purger, identity.Provide(w.pool).Purger}, clockAt(at), 60*24*time.Hour, 2)
}

// Each scenario publishes its complete fixture atomically. The live parent
// chain makes the 59-day tombstones ineligible for any parent cascade. Real
// periodic jobs remain enabled; both requested orders converge to the same IDs.
func TestTrimAndPurgeUseDisjointVersionSets(t *testing.T) {
	for _, archival := range []string{"live", "manual", "auto"} {
		for _, first := range []string{"trim", "purge"} {
			t.Run(archival+"/"+first+"_first", func(t *testing.T) {
				w := newRetentionWorld(t)
				at := time.Now().UTC().Truncate(time.Microsecond)
				item := w.newWorkItemIn(t, w.web, `{"name":"Disjoint versions"}`)
				switch archival {
				case "auto":
					if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2,updated_at=$3 WHERE id=$1", item, w.done, at.Add(-40*24*time.Hour)); err != nil {
						t.Fatal(err)
					}
					w.send(t, "alice", "PATCH", "/api/v0/projects/"+w.web.String(), `{"archive_in":1}`, 200)
					events := newArchiveRaceEvents(t, w)
					if n, err := archiveRaceJob(w, clockAt(at), events, nil, nil).Execute(pgtest.Soon(t)); err != nil || n < 0 || n > 1 {
						t.Fatalf("auto archive=%d %v", n, err)
					}
				case "manual":
					if _, err := w.pool.Exec(pgtest.Soon(t), "UPDATE issues SET state_id=$2 WHERE id=$1", item, w.done); err != nil {
						t.Fatal(err)
					}
					w.send(t, "alice", "POST", "/api/v0/issues/"+item.String()+"/archive", "", 200)
					w.send(t, "alice", "POST", "/api/v0/projects/"+w.web.String()+"/archive", "", 200)
				}
				stable := collaborationRows(t, w, item)
				delete(stable, "issue_description_versions")
				tx, err := w.pool.Begin(pgtest.Soon(t))
				if err != nil {
					t.Fatal(err)
				}
				defer requestRelease(t, tx)
				if _, err := tx.Exec(pgtest.Soon(t), "DELETE FROM issue_description_versions WHERE issue_id=$1", item); err != nil {
					t.Fatal(err)
				}
				live := retentionSeed(t, tx, item, 23, at.Add(-time.Hour), nil)
				oldAt, recentAt := at.Add(-61*24*time.Hour), at.Add(-59*24*time.Hour)
				old := retentionSeed(t, tx, item, 2, oldAt, &oldAt)
				recent := retentionSeed(t, tx, item, 2, recentAt, &recentAt)
				// Read all survivor bytes before publishing to background workers.
				rows, err := tx.Query(pgtest.Soon(t), "SELECT id,to_jsonb(v)::text FROM issue_description_versions v WHERE issue_id=$1", item)
				if err != nil {
					t.Fatal(err)
				}
				before := map[uuid.UUID]string{}
				for rows.Next() {
					var id uuid.UUID
					var body string
					if err := rows.Scan(&id, &body); err != nil {
						t.Fatal(err)
					}
					before[id] = body
				}
				rows.Close()
				if err := rows.Err(); err != nil {
					t.Fatal(err)
				}
				if err := tx.Commit(context.Background()); err != nil {
					t.Fatal(err)
				}
				if archival == "auto" {
					var count int
					if err := w.pool.QueryRow(pgtest.Soon(t), "SELECT count(*) FROM issue_activities WHERE issue_id=$1 AND field='archived_at' AND new_value='archive' AND actor_id IS NULL", item).Scan(&count); err != nil || count != 1 {
						t.Fatalf("automatic archival activity=%d %v", count, err)
					}
				}
				trim := func() {
					if _, err := retentionJob(w, retentionTx(w), issuepg.New(w.pool)).Execute(pgtest.Soon(t)); err != nil {
						t.Fatal(err)
					}
				}
				purge := func() {
					if _, err := retentionPurge(w, at).Execute(pgtest.Soon(t)); err != nil {
						t.Fatal(err)
					}
				}
				if first == "trim" {
					trim()
					purge()
				} else {
					purge()
					trim()
				}
				want := append(slices.Clone(live[3:]), recent...)
				retentionCheck(t, w, item, want, before)
				if got := retentionSnapshot(t, w.pool, old); len(got) != 0 {
					t.Fatal("61-day tombstones survived", got)
				}
				after := collaborationRows(t, w, item)
				delete(after, "issue_description_versions")
				if !reflect.DeepEqual(stable, after) {
					t.Fatal("trim/purge changed issue, activities or other child audit")
				}
				var parentsLive, archived bool
				if err := w.pool.QueryRow(pgtest.Soon(t), `SELECT p.deleted_at IS NULL AND w.deleted_at IS NULL,i.archived_at IS NOT NULL FROM issues i JOIN projects p ON p.id=i.project_id JOIN workspaces w ON w.id=i.workspace_id WHERE i.id=$1`, item).Scan(&parentsLive, &archived); err != nil || !parentsLive || archived != (archival != "live") {
					t.Fatalf("parent/archival boundary live=%t archived=%t: %v", parentsLive, archived, err)
				}
				// A second sweep is idempotent, including all old deletion audit fields.
				beforeAgain := registrationRows(t, w.pool)
				trim()
				purge()
				if registrationRows(t, w.pool) != beforeAgain {
					t.Fatal("second retention/cleanup rewrote stable business rows")
				}
			})
		}
	}
}
