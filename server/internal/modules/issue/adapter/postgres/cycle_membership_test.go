package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// A work item's cycle rows through the store (M6 P3a spec R4): InsertCycle
// puts it in a cycle, a row under it in its workspace and project, made and
// last changed by alice at now; it cannot be in two cycles at once, the
// partial unique key on issue_id refusing the second row (an error, not a
// refusal: the write's lock and checks never ask it). Moved by bob at
// later, RemoveCycle soft-deletes its undeleted row of the cycle it leaves,
// every other column kept, and InsertCycle puts it in the other, both at
// later: GetIssue answers the new cycle. Nothing else changes: another work
// item's row of the same cycle, a row deleted before, nor the work item's
// row when RemoveCycle names a cycle it is not in. A failed statement is
// its error.
func TestCycleRowsThroughTheStore(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	sprint, next := newCycle(t, pool, web, "Sprint"), newCycle(t, pool, web, "Next")
	id, other := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	newIssue(t, pool, web, other, 2, alice)
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	exec(t, pool, `INSERT INTO cycle_issues (id, workspace_id, project_id, cycle_id, issue_id, updated_by_id, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`, uuid.NewV7(), web.workspace, web.id, sprint, id, alice, earlier)
	joined := linkRows(web, id, alice, now, sprint)
	for _, r := range []app.LinkRows{joined, linkRows(web, other, alice, now, sprint)} {
		if err := s.InsertCycle(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	var made string
	if err := pool.QueryRow(ctx, `SELECT concat_ws(' ', workspace_id, project_id, cycle_id, issue_id, created_by_id, updated_by_id,
		created_at = $2 AND updated_at = $2, deleted_at IS NULL) FROM cycle_issues WHERE id = $1`, joined.IDs[0], now).Scan(&made); err != nil {
		t.Fatal(err)
	}
	if want := web.workspace.String() + " " + web.id.String() + " " + sprint.String() + " " + id.String() + " " + alice.String() + " " +
		alice.String() + " t t"; made != want {
		t.Errorf("the row joined: %s, want %s", made, want)
	}
	if err := s.InsertCycle(ctx, linkRows(web, id, alice, now, next)); err == nil {
		t.Error("InsertCycle() of a second cycle = nil; want the partial unique key's error")
	}
	before, stamps := rowsAsText(t, pool, "cycle_issues"), audits(t, pool, "cycle_issues")
	if err := s.RemoveCycle(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{next}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	if after := audits(t, pool, "cycle_issues"); !maps.Equal(after, stamps) {
		t.Errorf("RemoveCycle() of a cycle the work item is not in changed %v into %v", stamps, after)
	}
	if err := s.RemoveCycle(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{sprint}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	moved := linkRows(web, id, bob, later, next)
	if err := s.InsertCycle(ctx, moved); err != nil {
		t.Fatal(err)
	}
	after, afterStamps := rowsAsText(t, pool, "cycle_issues"), audits(t, pool, "cycle_issues")
	delete(after, moved.IDs[0])
	delete(afterStamps, moved.IDs[0])
	stamps[joined.IDs[0]] = deletion(later, bob)
	if !maps.Equal(after, before) || !maps.Equal(afterStamps, stamps) {
		t.Errorf("cycle_issues after the move:\n%v\n%v\nwant\n%v\n%v", after, afterStamps, before, stamps)
	}
	if got, _, err := s.GetIssue(ctx, id); err != nil || got.CycleID == nil || *got.CycleID != next {
		t.Errorf("GetIssue() = %+v, %v; want it in Next", got, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.InsertCycle(cancelled, linkRows(web, other, alice, now, next)); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertCycle() = %v; want context.Canceled", err)
	}
	if err := s.RemoveCycle(cancelled, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{next}, By: bob, At: later}); !errors.Is(err, context.Canceled) {
		t.Errorf("RemoveCycle() = %v; want context.Canceled", err)
	}
}
