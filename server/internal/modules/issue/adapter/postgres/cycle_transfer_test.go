package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// transferWorld is the memberships a transfer reads and writes (M6 P4 spec 2.6, R8): Web's cycles Sprint, which
// the work items leave, Next, which they join, and Other, and Ops' Far. In Sprint: moving, six work items, in turn
// in Web's default state and in Doing, which move; c, in Done; shelved, archived, and gone, deleted with its
// membership left, both in the default state; left, in the default state, whose membership of Sprint was deleted
// before. In Next: x, there before. In Other: g. In Far: h, of Ops. The memberships are written in the reverse of
// their work items' id order, their own ids descending with the work items' (ids made in one millisecond need not
// ascend: both are sorted), so a statement that does not order the work items answers them out of order: by the
// rows' order, by their ids, or by a hash of them (six of them, so a hash's order is not theirs by chance).
type transferWorld struct {
	s                               *postgresadapter.Store
	pool                            *pgxpool.Pool
	web, ops                        project
	doing                           uuid.UUID
	alice, bob                      uuid.UUID
	sprint, next, other, far        uuid.UUID
	moving                          []uuid.UUID
	c, shelved, gone, left, x, g, h uuid.UUID
}

func newTransferWorld(t *testing.T) transferWorld {
	t.Helper()
	s, pool := newStore(t)
	w := transferWorld{s: s, pool: pool}
	w.alice, w.bob = newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	acme := newWorkspace(t, pool, "acme")
	w.web, w.ops = newProject(t, pool, acme, "WEB"), newProject(t, pool, acme, "OPS")
	w.doing = uuid.NewV7()
	done := uuid.NewV7()
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group") VALUES ($1, $3, $4, 'Doing', '#F59E0B', 'started'),
		($2, $3, $4, 'Done', '#46A758', 'completed')`, w.doing, done, w.web.workspace, w.web.id)
	w.sprint, w.next, w.other = newCycle(t, pool, w.web, "Sprint"), newCycle(t, pool, w.web, "Next"), newCycle(t, pool, w.web, "Other")
	w.far = newCycle(t, pool, w.ops, "Far")
	ids, memberships := make([]uuid.UUID, 13), make([]uuid.UUID, 13)
	for n := range ids {
		ids[n], memberships[n] = uuid.NewV7(), uuid.NewV7()
	}
	slices.SortFunc(ids, uuid.UUID.Compare)
	slices.SortFunc(memberships, uuid.UUID.Compare)
	w.moving = ids[:6]
	w.c, w.shelved, w.gone, w.left, w.x, w.g, w.h = ids[6], ids[7], ids[8], ids[9], ids[10], ids[11], ids[12]
	for n := len(ids) - 1; n >= 0; n-- {
		p := w.web
		if ids[n] == w.h {
			p = w.ops
		}
		newIssue(t, pool, p, ids[n], n+1, w.alice)
	}
	for n := 1; n < len(w.moving); n += 2 {
		exec(t, pool, "UPDATE issues SET state_id = $2 WHERE id = $1", w.moving[n], w.doing)
	}
	exec(t, pool, "UPDATE issues SET state_id = $2 WHERE id = $1", w.c, done)
	exec(t, pool, "UPDATE issues SET archived_at = $2 WHERE id = $1", w.shelved, now)
	exec(t, pool, "UPDATE issues SET deleted_at = $2 WHERE id = $1", w.gone, now)
	for n := len(ids) - 1; n >= 0; n-- {
		p, cycle, deleted := w.web, w.sprint, (*time.Time)(nil)
		switch ids[n] {
		case w.x:
			cycle = w.next
		case w.g:
			cycle = w.other
		case w.h:
			p, cycle = w.ops, w.far
		case w.left:
			deleted = &now
		}
		exec(t, pool, `INSERT INTO cycle_issues (id, workspace_id, project_id, cycle_id, issue_id, created_by_id, updated_by_id, created_at,
			updated_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $6, $7, $7, $8)`, memberships[len(ids)-1-n], p.workspace, p.id, cycle, ids[n], w.alice,
			now, deleted)
	}
	return w
}

// transfer is bob's transfer of Sprint's work items in Web's default state and in Doing to Next, at later.
func (w transferWorld) transfer(later time.Time) app.CycleTransfer {
	return app.CycleTransfer{ProjectID: w.web.id, WorkspaceID: w.web.workspace, From: domain.Named{ID: w.sprint, Name: "Sprint"},
		To: domain.Named{ID: w.next, Name: "Next"}, StateIDs: []uuid.UUID{w.web.state, w.doing}, By: w.bob, At: later}
}

// A transfer's memberships through the store (M6 P4 spec 2.6, R8; P3a spec R4), in its transaction: the removal
// soft-deletes, of the cycle left, the undeleted memberships whose work items are the project's, undeleted, not
// archived and in one of the states asked for, each deleted and last changed by bob at later, every other column kept
// (the cycle too: no row changes cycle in place), and answers their work items in id order, not in the rows' order.
// The insertion then writes one new undeleted row of the cycle joined for each, with the ids given, in the transfer's
// workspace and project, created and last changed by bob at later. A work item in another state, archived or deleted,
// a membership deleted before, the cycle joined's own and other cycles' and projects' rows are untouched; each work
// item keeps one undeleted membership at most. Asked in another project, or of no state, the removal answers none.
// Outside a transaction each refuses and writes nothing; a cancelled context is its error.
func TestTransferringACyclesMemberships(t *testing.T) {
	w := newTransferWorld(t)
	later := now.Add(time.Hour)
	tr := w.transfer(later)
	before, stamps := rowsAsText(t, w.pool, "cycle_issues"), audits(t, w.pool, "cycle_issues")
	ctx := context.Background()
	if _, err := w.s.RemoveTransferredCycleIssues(ctx, tr); err == nil {
		t.Error("a removal outside a transaction = nil; want its refusal")
	}
	joins := app.CycleJoins{IssueIDs: w.moving, CycleID: w.next, WorkspaceID: w.web.workspace, ProjectID: w.web.id, By: w.bob, At: later}
	for range w.moving {
		joins.IDs = append(joins.IDs, uuid.NewV7())
	}
	if err := w.s.InsertTransferredCycleIssues(ctx, joins); err == nil {
		t.Error("an insertion outside a transaction = nil; want its refusal")
	}
	if got, gotStamps := rowsAsText(t, w.pool, "cycle_issues"), audits(t, w.pool, "cycle_issues"); !maps.Equal(got, before) || !maps.Equal(gotStamps, stamps) {
		t.Fatalf("the refused writes wrote\n%v\n%v", got, gotStamps)
	}

	var moved, elsewhere, stateless []uuid.UUID
	err := postgres.NewTxManager(w.pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
		var err error
		if elsewhere, err = w.s.RemoveTransferredCycleIssues(ctx, app.CycleTransfer{ProjectID: w.ops.id, WorkspaceID: w.ops.workspace,
			From: tr.From, To: tr.To, StateIDs: []uuid.UUID{w.web.state, w.doing, w.ops.state}, By: w.bob, At: later}); err != nil {
			return err
		}
		if stateless, err = w.s.RemoveTransferredCycleIssues(ctx, app.CycleTransfer{ProjectID: w.web.id, WorkspaceID: w.web.workspace,
			From: tr.From, To: tr.To, By: w.bob, At: later}); err != nil {
			return err
		}
		if moved, err = w.s.RemoveTransferredCycleIssues(ctx, tr); err != nil {
			return err
		}
		return w.s.InsertTransferredCycleIssues(ctx, joins)
	})
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(moved, w.moving) || len(elsewhere) != 0 || len(stateless) != 0 {
		t.Fatalf("moved %v, asked in Ops %v, of no state %v; want %v in id order, then none, none", moved, elsewhere, stateless, w.moving)
	}

	after, afterStamps := rowsAsText(t, w.pool, "cycle_issues"), audits(t, w.pool, "cycle_issues")
	for id, row := range before {
		if after[id] != row {
			t.Errorf("membership %s is %s after the transfer, want %s kept", id, after[id], row)
		}
		if member := membershipOf(t, w.pool, id); slices.Contains(w.moving, member.issue) && member.cycle == w.sprint {
			stamps[id] = deletion(later, w.bob)
		}
	}
	for id, stamp := range stamps {
		if afterStamps[id] != stamp {
			t.Errorf("membership %s: %s after the transfer, want %s", id, afterStamps[id], stamp)
		}
	}
	if len(after) != len(before)+len(joins.IDs) {
		t.Errorf("%d memberships after the transfer, want %d: the old ones and one new of each moved", len(after), len(before)+len(joins.IDs))
	}
	for n, id := range joins.IDs {
		want := cycleMembership{workspace: w.web.workspace, project: w.web.id, cycle: w.next, issue: joins.IssueIDs[n], created: w.bob, updated: w.bob,
			createdAt: later, updatedAt: later}
		if got := membershipOf(t, w.pool, id); !got.equal(want) {
			t.Errorf("new membership %s = %+v, want %+v", id, got, want)
		}
	}
	want := map[uuid.UUID]uuid.UUID{w.c: w.sprint, w.shelved: w.sprint, w.gone: w.sprint, w.x: w.next, w.g: w.other, w.h: w.far}
	for _, id := range w.moving {
		want[id] = w.next
	}
	if got := undeletedCycles(t, w.pool); !maps.Equal(got, want) {
		t.Errorf("undeleted memberships %v, want %v: one at most for each work item", got, want)
	}

	err = postgres.NewTxManager(w.pool, 5*time.Second).WithinTx(ctx, func(ctx context.Context) error {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		_, removeErr := w.s.RemoveTransferredCycleIssues(cancelled, tr)
		insertErr := w.s.InsertTransferredCycleIssues(cancelled, app.CycleJoins{IDs: []uuid.UUID{uuid.NewV7()}, IssueIDs: []uuid.UUID{w.c},
			CycleID: w.next, WorkspaceID: w.web.workspace, ProjectID: w.web.id, By: w.bob, At: later})
		if !errors.Is(removeErr, context.Canceled) || !errors.Is(insertErr, context.Canceled) {
			t.Errorf("with a cancelled context: removal %v, insertion %v; want context.Canceled", removeErr, insertErr)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// cycleMembership is a cycle_issues row's columns but its id.
type cycleMembership struct {
	workspace, project, cycle, issue uuid.UUID
	created, updated                 uuid.UUID
	createdAt, updatedAt             time.Time
	deletedAt                        *time.Time
}

func (m cycleMembership) equal(o cycleMembership) bool {
	return m.workspace == o.workspace && m.project == o.project && m.cycle == o.cycle && m.issue == o.issue && m.created == o.created &&
		m.updated == o.updated && m.createdAt.Equal(o.createdAt) && m.updatedAt.Equal(o.updatedAt) && (m.deletedAt == nil) == (o.deletedAt == nil) &&
		(m.deletedAt == nil || m.deletedAt.Equal(*o.deletedAt))
}

// membershipOf is the membership id as stored.
func membershipOf(t *testing.T, pool *pgxpool.Pool, id uuid.UUID) cycleMembership {
	t.Helper()
	var m cycleMembership
	err := pool.QueryRow(context.Background(), `SELECT workspace_id, project_id, cycle_id, issue_id, created_by_id, updated_by_id, created_at,
		updated_at, deleted_at FROM cycle_issues WHERE id = $1`, id).Scan(&m.workspace, &m.project, &m.cycle, &m.issue, &m.created, &m.updated,
		&m.createdAt, &m.updatedAt, &m.deletedAt)
	if err != nil {
		t.Fatalf("membership %s: %v", id, err)
	}
	return m
}

// undeletedCycles are the cycle of each work item's undeleted memberships, by work item; a second one of a work item
// fails the test.
func undeletedCycles(t *testing.T, pool *pgxpool.Pool) map[uuid.UUID]uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT issue_id, cycle_id FROM cycle_issues WHERE deleted_at IS NULL")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	out := map[uuid.UUID]uuid.UUID{}
	for rows.Next() {
		var issue, cycle uuid.UUID
		if err := rows.Scan(&issue, &cycle); err != nil {
			t.Fatal(err)
		}
		if _, twice := out[issue]; twice {
			t.Errorf("work item %s has two undeleted memberships", issue)
		}
		out[issue] = cycle
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	return out
}
