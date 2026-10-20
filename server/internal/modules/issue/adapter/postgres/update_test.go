package postgresadapter_test

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// updateIssue's rows through the store (M4 design 3.7): UpdateIssue writes
// the work item's name, state, priority, dates and completion as given, its
// start date given nil NULL, last changed by bob at later, its number, its
// sort order and its creation as they were; RemoveAssignees and
// RemoveLabels soft-delete, by bob at later, the work item's undeleted rows
// of every account and label given, two of each in one call, every other
// column kept. Nothing else changes: not another work item, nor its rows of
// a given account and label, nor the work item's rows of an account or a
// label not given, nor its rows of a given account and label deleted an
// hour before, which keep their stamps. Moved back into a state of no
// completed group, its completion given nil, the work item is completed no
// more.
func TestUpdatingAWorkItemWritesItsRows(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob, carol, dave := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com"),
		newAccount(t, pool, "carol@corp.com"), newAccount(t, pool, "dave@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	done := uuid.NewV7()
	exec(t, pool, `INSERT INTO states (id, workspace_id, project_id, name, color, "group") VALUES ($1, $2, $3, 'Done', '#000', 'completed')`,
		done, web.workspace, web.id)
	bug, ui, docs := newLabel(t, pool, web, "Bug"), newLabel(t, pool, web, "UI"), newLabel(t, pool, web, "Docs")
	id, other := uuid.NewV7(), uuid.NewV7()
	createWebIssue(t, s, web, id, alice)
	newIssue(t, pool, web, other, 8, alice)
	assignees, labels := linkRows(web, id, alice, now, bob, carol, dave), linkRows(web, id, alice, now, bug, ui, docs)
	for _, step := range []struct {
		insert func(context.Context, app.LinkRows) error
		rows   app.LinkRows
	}{{s.InsertAssignees, assignees}, {s.InsertLabels, labels}, {s.InsertAssignees, linkRows(web, other, alice, now, bob)},
		{s.InsertLabels, linkRows(web, other, alice, now, ui)}} {
		if err := step.insert(ctx, step.rows); err != nil {
			t.Fatal(err)
		}
	}
	earlier, later := now.Add(-time.Hour), now.Add(time.Hour)
	for _, deleted := range []struct {
		table, column string
		target        uuid.UUID
	}{{"issue_assignees", "assignee_id", bob}, {"issue_labels", "label_id", ui}} {
		exec(t, pool, `INSERT INTO `+deleted.table+` (id, workspace_id, project_id, issue_id, `+deleted.column+`, updated_by_id, updated_at,
			deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`, uuid.NewV7(), web.workspace, web.id, id, deleted.target, alice, earlier)
	}
	tables := []string{"issues", "issue_assignees", "issue_labels"}
	before, stamps := map[string]map[uuid.UUID]string{}, map[string]map[uuid.UUID]string{}
	for _, table := range tables {
		before[table], stamps[table] = rowsAsText(t, pool, table), audits(t, pool, table)
	}

	target := day(26)
	if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Mend the login", StateID: done, Priority: domain.PriorityLow, TargetDate: &target,
		CompletedAt: &later, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveAssignees(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{bob, dave}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}
	if err := s.RemoveLabels(ctx, app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{ui, docs}, By: bob, At: later}); err != nil {
		t.Fatal(err)
	}

	got, found, err := s.GetIssue(ctx, id)
	want := domain.Issue{ID: id, WorkspaceID: web.workspace, ProjectID: web.id, SequenceID: 7, Name: "Mend the login", StateID: done,
		Priority: domain.PriorityLow, TargetDate: &target, SortOrder: 75535, CompletedAt: &later, CreatedByID: &alice, UpdatedByID: &bob,
		CreatedAt: now, UpdatedAt: later, AssigneeIDs: []uuid.UUID{carol}, LabelIDs: []uuid.UUID{bug}}
	if err != nil || !found || !reflect.DeepEqual(got, want) {
		t.Errorf("GetIssue() = %+v, %v, %v; want %+v", got, found, err, want)
	}
	removed := map[string][]uuid.UUID{"issue_assignees": {assignees.IDs[0], assignees.IDs[2]}, "issue_labels": {labels.IDs[1], labels.IDs[2]}}
	for _, table := range tables {
		after, afterStamps := rowsAsText(t, pool, table), audits(t, pool, table)
		wantStamps := maps.Clone(stamps[table])
		if table == "issues" {
			delete(after, id)
			delete(before[table], id)
			delete(afterStamps, id)
			delete(wantStamps, id)
		}
		for _, row := range removed[table] {
			wantStamps[row] = deletion(later, bob)
		}
		if !maps.Equal(after, before[table]) {
			t.Errorf("%s changed:\n%v\nwas\n%v", table, after, before[table])
		}
		if !maps.Equal(afterStamps, wantStamps) {
			t.Errorf("%s's stamps:\n%v\nwant\n%v", table, afterStamps, wantStamps)
		}
	}

	if err := s.UpdateIssue(ctx, app.IssueFields{ID: id, Name: "Mend the login", StateID: web.state, Priority: domain.PriorityLow, By: bob,
		At: later}); err != nil {
		t.Fatal(err)
	}
	if got, _, err := s.GetIssue(ctx, id); err != nil || got.StateID != web.state || got.CompletedAt != nil {
		t.Errorf("GetIssue() moved back = %+v, %v; want it in its first state, completed_at NULL", got, err)
	}
}

// archiveIssue's and unarchiveIssue's row through the store (M4 design
// 3.8): SetArchivedAt archives the work item at later, changed last by bob
// then, and unarchives it, changed last by alice at latest; each time every
// other column of it is as it was, and so are the work items older and
// other, drawn before and after it, so that a statement that took its work
// item by the ids' order, not by equality, would reach one of them.
func TestArchivingAWorkItemWritesItsRow(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	older, id, other := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	createWebIssue(t, s, web, id, alice)
	newIssue(t, pool, web, other, 8, alice)
	newIssue(t, pool, web, older, 9, alice)
	// row is the work item's row as text, but the columns SetArchivedAt writes.
	row := `SELECT (to_jsonb(i) - 'archived_at' - 'updated_at' - 'updated_by_id')::text FROM issues i WHERE id = $1`
	before, stamps, columns := rowsAsText(t, pool, "issues"), audits(t, pool, "issues"), textRows(t, pool, row, id)
	created, _, err := s.GetIssue(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	later, latest := now.Add(time.Hour), now.Add(2*time.Hour)
	for _, tt := range []struct {
		archivedAt *time.Time
		by         uuid.UUID
		at         time.Time
	}{{&later, bob, later}, {nil, alice, latest}} {
		if err := s.SetArchivedAt(ctx, app.IssueArchival{ID: id, ArchivedAt: tt.archivedAt, By: tt.by, At: tt.at}); err != nil {
			t.Fatal(err)
		}
		got, found, err := s.GetIssue(ctx, id)
		want := created
		want.ArchivedAt, want.UpdatedByID, want.UpdatedAt = tt.archivedAt, &tt.by, tt.at
		if err != nil || !found || !reflect.DeepEqual(got, want) {
			t.Errorf("GetIssue() archived at %v = %+v, %v, %v; want %+v", tt.archivedAt, got, found, err, want)
		}
		if kept := textRows(t, pool, row, id); !slices.Equal(kept, columns) {
			t.Errorf("archived at %v, the work item's other columns:\n%s\nwere\n%s", tt.archivedAt, kept, columns)
		}
		after, afterStamps := rowsAsText(t, pool, "issues"), audits(t, pool, "issues")
		for _, unwritten := range []uuid.UUID{older, other} {
			if after[unwritten] != before[unwritten] || afterStamps[unwritten] != stamps[unwritten] {
				t.Errorf("the work item %s:\n%s %s\nwas\n%s %s", unwritten, after[unwritten], afterStamps[unwritten], before[unwritten],
					stamps[unwritten])
			}
		}
	}
}

// A write of an update that fails answers its error as itself (M4 design
// 3.5).
func TestAFailedUpdateStatementIsAnError(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	id := uuid.NewV7()
	newIssue(t, pool, web, id, 1, alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if err := s.UpdateIssue(cancelled, app.IssueFields{ID: id, Name: "a", StateID: web.state, Priority: domain.PriorityNone, By: alice,
		At: now}); !errors.Is(err, context.Canceled) {
		t.Errorf("UpdateIssue() = %v; want context.Canceled", err)
	}
	if err := s.SetArchivedAt(cancelled, app.IssueArchival{ID: id, ArchivedAt: &now, By: alice, At: now}); !errors.Is(err, context.Canceled) {
		t.Errorf("SetArchivedAt() = %v; want context.Canceled", err)
	}
	unlink := app.UnlinkRows{IssueID: id, Targets: []uuid.UUID{alice}, By: alice, At: now}
	for name, remove := range map[string]func(context.Context, app.UnlinkRows) error{"RemoveAssignees": s.RemoveAssignees,
		"RemoveLabels": s.RemoveLabels} {
		if err := remove(cancelled, unlink); !errors.Is(err, context.Canceled) {
			t.Errorf("%s() = %v; want context.Canceled", name, err)
		}
	}
}
