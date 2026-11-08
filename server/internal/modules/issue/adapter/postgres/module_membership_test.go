package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// A work item's module rows through the store (M6 P3a spec R4), as its
// labels': InsertModules puts it in Auth and Billing, a row each under it
// in its workspace and project, made and last changed by alice at now; it
// cannot be in a module twice, the partial unique key on (issue_id,
// module_id) refusing the second row (an error, not a refusal: the write's
// lock and checks never ask it). Its modules replaced by bob at later,
// RemoveModules soft-deletes its undeleted row of Auth, every other column
// kept, the row of Billing it keeps untouched, and InsertModules puts it in
// Core, both at later: GetIssue answers Billing and Core, in id order.
// Nothing else changes: another work item's row of Auth, a row deleted
// before, nor the work item's rows when RemoveModules names a module it is
// not in. A failed statement is its error.
func TestModuleRowsThroughTheStore(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	auth, billing, core := newModule(t, pool, web, "Auth"), newModule(t, pool, web, "Billing"), newModule(t, pool, web, "Core")
	id, other := uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	newIssue(t, pool, web, other, 2, alice)
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	exec(t, pool, `INSERT INTO module_issues (id, workspace_id, project_id, module_id, issue_id, updated_by_id, updated_at, deleted_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`, uuid.NewV7(), web.workspace, web.id, auth, id, alice, earlier)
	joined := linkRows(web, id, alice, now, auth, billing)
	for _, r := range []app.LinkRows{joined, linkRows(web, other, alice, now, auth)} {
		if err := s.InsertModules(ctx, r); err != nil {
			t.Fatal(err)
		}
	}
	made := textRows(t, pool, `SELECT concat_ws(' ', workspace_id, project_id, module_id, issue_id, created_by_id, updated_by_id,
		created_at = $2 AND updated_at = $2, deleted_at IS NULL) FROM module_issues WHERE id = ANY ($1) ORDER BY module_id`, joined.IDs, now)
	for i, module := range []uuid.UUID{auth, billing} {
		if want := web.workspace.String() + " " + web.id.String() + " " + module.String() + " " + id.String() + " " + alice.String() + " " +
			alice.String() + " t t"; i >= len(made) || made[i] != want {
			t.Errorf("the rows joined: %q, want %s at %d", made, want, i)
		}
	}
	if err := s.InsertModules(ctx, linkRows(web, id, alice, now, billing)); err == nil {
		t.Error("InsertModules() of a module the work item is in = nil; want the partial unique key's error")
	}
	before, stamps := rowsAsText(t, pool, "module_issues"), audits(t, pool, "module_issues")
	if err := s.RemoveModules(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{core}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	if after := audits(t, pool, "module_issues"); !maps.Equal(after, stamps) {
		t.Errorf("RemoveModules() of a module the work item is not in changed %v into %v", stamps, after)
	}
	if err := s.RemoveModules(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{auth}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	replaced := linkRows(web, id, bob, later, core)
	if err := s.InsertModules(ctx, replaced); err != nil {
		t.Fatal(err)
	}
	after, afterStamps := rowsAsText(t, pool, "module_issues"), audits(t, pool, "module_issues")
	delete(after, replaced.IDs[0])
	delete(afterStamps, replaced.IDs[0])
	stamps[joined.IDs[0]] = deletion(later, bob)
	if !maps.Equal(after, before) || !maps.Equal(afterStamps, stamps) {
		t.Errorf("module_issues after the replacement:\n%v\n%v\nwant\n%v\n%v", after, afterStamps, before, stamps)
	}
	if got, _, err := s.GetIssue(ctx, id); err != nil || !slices.Equal(got.ModuleIDs, []uuid.UUID{billing, core}) {
		t.Errorf("GetIssue() = %+v, %v; want it in Billing and Core", got, err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.InsertModules(cancelled, linkRows(web, other, alice, now, core)); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertModules() = %v; want context.Canceled", err)
	}
	if err := s.RemoveModules(cancelled, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{core}, By: bob, At: later}); !errors.Is(err, context.Canceled) {
		t.Errorf("RemoveModules() = %v; want context.Canceled", err)
	}
}
