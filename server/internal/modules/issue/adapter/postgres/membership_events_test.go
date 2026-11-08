package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// A deleted cycle's and a deleted module's memberships through the store (M6 P3a spec 2.5, R6): in the deleting
// transaction, the removal soft-deletes the target's undeleted rows, each deleted and last changed by bob at later,
// every other column kept, and answers their work items in id order, not in the rows' order (the work item made last
// joined first); a row of the target deleted before keeps its deletion, another target's row is untouched. A target
// no work item is in answers none. Outside a transaction it refuses and writes nothing (precedent
// RecordAttachmentChange). A cancelled context is its error.
func TestRemovingADeletedTargetsMemberships(t *testing.T) {
	for _, tc := range []struct {
		table, column string
		remove        func(*postgresadapter.Store) func(context.Context, uuid.UUID, time.Time, uuid.UUID) ([]uuid.UUID, error)
	}{
		{"cycle_issues", "cycle_id", func(s *postgresadapter.Store) func(context.Context, uuid.UUID, time.Time, uuid.UUID) ([]uuid.UUID, error) {
			return s.RemoveCycleMemberships
		}},
		{"module_issues", "module_id", func(s *postgresadapter.Store) func(context.Context, uuid.UUID, time.Time, uuid.UUID) ([]uuid.UUID, error) {
			return s.RemoveModuleMemberships
		}},
	} {
		t.Run(tc.table, func(t *testing.T) {
			s, pool := newStore(t)
			alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
			web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
			target, other, empty := newCycle(t, pool, web, "Sprint"), newCycle(t, pool, web, "Next"), newCycle(t, pool, web, "Empty")
			if tc.table == "module_issues" {
				target, other, empty = newModule(t, pool, web, "Auth"), newModule(t, pool, web, "Billing"), newModule(t, pool, web, "Empty")
			}
			first, second, third := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
			for i, id := range []uuid.UUID{first, second, third} {
				newIssue(t, pool, web, id, i+1, alice)
			}
			earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
			member := func(issue, of uuid.UUID, deleted *time.Time) uuid.UUID {
				id := uuid.NewV7()
				exec(t, pool, `INSERT INTO `+tc.table+` (id, workspace_id, project_id, `+tc.column+`, issue_id, created_by_id, updated_by_id,
					created_at, updated_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $6, $7::timestamptz, coalesce($8::timestamptz, $7), $8)`,
					id, web.workspace, web.id, of, issue, alice, now, deleted)
				return id
			}
			joinedLast, joinedFirst := member(third, target, nil), member(first, target, nil)
			member(second, target, &earlier)
			member(second, other, nil)
			before, stamps := rowsAsText(t, pool, tc.table), audits(t, pool, tc.table)
			ctx := context.Background()
			if _, err := tc.remove(s)(ctx, target, later, bob); err == nil {
				t.Error("a removal outside a transaction = nil; want its refusal")
			}
			if got := audits(t, pool, tc.table); !maps.Equal(got, stamps) {
				t.Errorf("the refused removal wrote %v, want %v", got, stamps)
			}
			var removed, none []uuid.UUID
			err := postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
				var err error
				if removed, err = tc.remove(s)(ctx, target, later, bob); err != nil {
					return err
				}
				none, err = tc.remove(s)(ctx, empty, later, bob)
				return err
			})
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(removed, []uuid.UUID{first, third}) || len(none) != 0 {
				t.Errorf("removed %v and %v; want %v in id order, then none", removed, none, []uuid.UUID{first, third})
			}
			stamps[joinedLast], stamps[joinedFirst] = deletion(later, bob), deletion(later, bob)
			if after, afterStamps := rowsAsText(t, pool, tc.table), audits(t, pool, tc.table); !maps.Equal(after, before) || !maps.Equal(afterStamps, stamps) {
				t.Errorf("%s after the removal:\n%v\n%v\nwant\n%v\n%v", tc.table, after, afterStamps, before, stamps)
			}
			err = postgres.NewTxManager(pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				_, err := tc.remove(s)(cancelled, other, later, bob)
				return err
			})
			if !errors.Is(err, context.Canceled) {
				t.Errorf("a removal with a cancelled context = %v; want context.Canceled", err)
			}
		})
	}
}
