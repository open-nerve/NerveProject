package postgresadapter_test

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// Each exclusion has a row satisfying every other predicate. Exact cutoff
// catches < instead of <=; distinct old/at stamps catch missing audit writes.
func TestAutoArchiveRows(t *testing.T) {
	for _, stateSet := range []string{"terminal states", "nil states", "empty states"} {
		empty := stateSet != "terminal states"
		name := stateSet
		t.Run(name, func(t *testing.T) {
			s, pool := newStore(t)
			var _ app.AutoArchiveStore = s
			alice := newAccount(t, pool, "alice@corp.com")
			acme := newWorkspace(t, pool, "acme")
			web, other := newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OTHER")
			done, cancelled := uuid.NewV7(), uuid.NewV7()
			for i, id := range []uuid.UUID{done, cancelled} {
				group := "completed"
				if i == 1 {
					group = "cancelled"
				}
				exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group") VALUES ($1,$2,$3,$4,'#000',$4)`, id, acme, web.id, group)
			}
			cutoff, at := now.Add(-30*24*time.Hour), now.Add(time.Hour)
			old := cutoff.Add(-time.Microsecond)
			type row struct {
				name              string
				p                 project
				state             uuid.UUID
				updated           time.Time
				archived, deleted bool
				match             bool
			}
			cases := []row{
				{"old completed", web, done, old, false, false, true},
				{"exact cutoff", web, done, cutoff, false, false, true},
				{"old cancelled", web, cancelled, old, false, false, true},
				{"new completed", web, done, cutoff.Add(time.Microsecond), false, false, false},
				{"old backlog", web, web.state, old, false, false, false},
				{"deleted", web, done, old, false, true, false},
				{"archived", web, done, old, true, false, false},
				{"other project", other, done, old, false, false, false},
			}
			for _, group := range []string{"unstarted", "started", "triage"} {
				id := uuid.NewV7()
				exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group") VALUES ($1,$2,$3,$4,'#000',$4)`, id, acme, web.id, group)
				cases = append(cases, row{"old " + group, web, id, old, false, false, false})
			}
			var want []uuid.UUID
			ids := make([]uuid.UUID, len(cases))
			for i, c := range cases {
				id := uuid.NewV7()
				ids[i] = id
				newIssue(t, pool, c.p, id, i+1, alice)
				exec(t, pool, "UPDATE issues SET state_id=$2, updated_at=$3 WHERE id=$1", id, c.state, c.updated)
				if c.archived {
					exec(t, pool, "UPDATE issues SET archived_at=$2 WHERE id=$1", id, old)
				}
				if c.deleted {
					exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=$1", id, old)
				}
				if c.match && !empty {
					want = append(want, id)
				}
			}
			before := rowsAsText(t, pool, "issues")
			beforeAudits := audits(t, pool, "issues")
			states := []uuid.UUID{done, cancelled}
			if stateSet == "nil states" {
				states = nil
			}
			if stateSet == "empty states" {
				states = []uuid.UUID{}
			}
			args := app.AutoArchiveRows{ProjectID: web.id, StateIDs: states, Cutoff: cutoff, At: at}
			got, err := s.AutoArchive(context.Background(), args)
			slices.SortFunc(got, func(a, b uuid.UUID) int { return slices.Compare(a[:], b[:]) })
			if err != nil || !slices.Equal(got, want) {
				t.Errorf("AutoArchive() = %v, %v; want %v", got, err, want)
			}
			after := rowsAsText(t, pool, "issues")
			afterAudits := audits(t, pool, "issues")
			for i, c := range cases {
				if !c.match || empty {
					if after[ids[i]] != before[ids[i]] || afterAudits[ids[i]] != beforeAudits[ids[i]] {
						t.Errorf("%s changed unexpectedly", c.name)
					}
					continue
				}
				// Every column except the three specified archival stamps remains intact.
				var beforeFields, afterFields map[string]any
				if err := json.Unmarshal([]byte(before[ids[i]]), &beforeFields); err != nil {
					t.Fatal(err)
				}
				if err := json.Unmarshal([]byte(after[ids[i]]), &afterFields); err != nil {
					t.Fatal(err)
				}
				delete(beforeFields, "archived_at")
				delete(afterFields, "archived_at")
				if !reflect.DeepEqual(beforeFields, afterFields) {
					t.Errorf("%s changed non-archival fields", c.name)
				}
				var archived *time.Time
				var updated time.Time
				var by *uuid.UUID
				if err := pool.QueryRow(context.Background(), "SELECT archived_at, updated_at, updated_by_id FROM issues WHERE id=$1", ids[i]).Scan(&archived, &updated, &by); err != nil {
					t.Fatal(err)
				}
				if archived == nil || !archived.Equal(at) || !updated.Equal(at) || by != nil {
					t.Errorf("%s audit = %v, %v, %v; want %v, %v, nil", c.name, archived, updated, by, at, at)
				}
			}
			if got, err := s.AutoArchive(context.Background(), args); err != nil || len(got) != 0 {
				t.Errorf("repeat = %v, %v; want no actual IDs", got, err)
			}
			ctx, cancel := context.WithCancel(context.Background())
			cancel()
			if got, err := s.AutoArchive(ctx, args); !errors.Is(err, context.Canceled) || got != nil {
				t.Errorf("cancelled update = %v, %v", got, err)
			}
		})
	}
}
