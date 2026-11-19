package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
)

// The pending counts (M7 P2b spec 2.6): for each project given, once, in
// id order, a project with none too, its undeleted intake rows of
// undeleted work items pending at the request's moment, a snooze whose
// time is that moment too, a snooze till after it, a triaged row, a
// deleted row and a deleted work item's not; where the caller sees only
// his own, those he created alone, no unknown creator's. Which projects is
// the use case's: the store counts those it is given.
func TestThePendingCountsCountTheirRows(t *testing.T) {
	s, pool := newStore(t)
	guest, alice := newAccount(t, pool, "guest@corp.com"), newAccount(t, pool, "alice@corp.com")
	ws := newWorkspace(t, pool, "acme")
	web, ops, quiet := newProject(t, pool, ws, "WEB"), newProject(t, pool, ws, "OPS"), newProject(t, pool, ws, "QUI")
	out, in := listNow, listNow.Add(time.Microsecond)
	seq := 0
	row := func(p project, status string, till *time.Time, by *uuid.UUID, deleted string) {
		t.Helper()
		seq++
		id := uuid.NewV7()
		newIssue(t, pool, p, id, seq, alice)
		exec(t, pool, "UPDATE issues SET created_by_id=$2 WHERE id=$1", id, by)
		var target *uuid.UUID
		if status == "duplicate" {
			target = &id
		}
		newIntakeRow(t, pool, p, id, status, till, target, alice, deleted == "row")
		if deleted == "issue" {
			exec(t, pool, "UPDATE issues SET deleted_at=$2 WHERE id=$1", id, now)
		}
	}
	row(web, "pending", nil, &guest, "")
	row(web, "pending", nil, &alice, "")
	row(web, "pending", nil, nil, "")
	row(web, "snoozed", &out, &alice, "")
	row(web, "snoozed", &in, &guest, "")
	for _, status := range []string{"accepted", "declined", "duplicate"} {
		row(web, status, nil, &guest, "")
	}
	row(web, "pending", nil, &guest, "row")
	row(web, "pending", nil, &guest, "issue")
	row(ops, "pending", nil, &guest, "")
	row(ops, "snoozed", &out, &alice, "")
	count := func(projects ...app.VisibleProject) []app.IntakeCount {
		t.Helper()
		got, err := s.CountPendingIntake(context.Background(), guest, projects, listNow)
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	sorted := func(counts ...app.IntakeCount) []app.IntakeCount {
		return slices.SortedFunc(slices.Values(counts), func(a, b app.IntakeCount) int { return a.ProjectID.Compare(b.ProjectID) })
	}
	for _, tc := range []struct {
		name     string
		projects []app.VisibleProject
		want     []app.IntakeCount
	}{
		{"all seen, none quiet", []app.VisibleProject{{ID: quiet.id}, {ID: web.id}, {ID: ops.id}},
			sorted(app.IntakeCount{ProjectID: web.id, Pending: 4}, app.IntakeCount{ProjectID: ops.id, Pending: 2}, app.IntakeCount{ProjectID: quiet.id})},
		{"own-only in Web", []app.VisibleProject{{ID: web.id, OwnOnly: true}, {ID: ops.id}},
			sorted(app.IntakeCount{ProjectID: web.id, Pending: 1}, app.IntakeCount{ProjectID: ops.id, Pending: 2})},
		{"own-only in Ops", []app.VisibleProject{{ID: ops.id, OwnOnly: true}},
			[]app.IntakeCount{{ProjectID: ops.id, Pending: 1}}},
		{"none", []app.VisibleProject{}, []app.IntakeCount{}},
	} {
		if got := count(tc.projects...); !slices.Equal(got, tc.want) || got == nil {
			t.Errorf("%s: counts %+v, want %+v", tc.name, got, tc.want)
		}
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.CountPendingIntake(cancelled, guest, []app.VisibleProject{{ID: web.id}}, listNow); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("CountPendingIntake() = %+v, %v; want context.Canceled", got, err)
	}
}

// Deleting an accepted work item's intake (M7 P2b spec 2.5, DP-10)
// soft-deletes its undeleted intake row alone, at the moment and by the
// account given, its creator, status and the rest as they were: not its
// deleted row before it, nor another work item's, nor the work item. A
// work item without an undeleted row, the row deleted already too, is an
// error.
func TestAnIntakeRowIsSoftDeletedAlone(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	item, other, never := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, item, 1, alice)
	newIssue(t, pool, web, other, 2, alice)
	newIssue(t, pool, web, never, 3, alice)
	newIntakeRow(t, pool, web, item, "declined", nil, nil, alice, true)
	current := newIntakeRow(t, pool, web, item, "accepted", nil, nil, alice, false)
	newIntakeRow(t, pool, web, other, "accepted", nil, nil, alice, false)
	issueRows := func() []string {
		return textRows(t, pool, "SELECT format('%s %s %s', id, updated_at, deleted_at IS NULL) FROM issues ORDER BY id")
	}
	issuesBefore, before := issueRows(), intakeTable(t, pool)

	at := now.Add(time.Hour)
	if err := s.DeleteIntake(ctx, app.IntakeDeletion{IssueID: item, By: bob, At: at}); err != nil {
		t.Fatal(err)
	}
	stamp, then := stampOf(t, pool, now), stampOf(t, pool, at)
	place := fmt.Sprintf("%s/%s/%s", web.workspace, web.id, item)
	want := slices.Clone(before)
	for n, line := range before {
		if line[:36] == current.String() {
			want[n] = fmt.Sprintf("%s %s accepted - - by %s/%s at %s/%s deleted t", current, place, alice, bob, stamp, then)
		}
	}
	if got := intakeTable(t, pool); !slices.Equal(got, want) {
		t.Errorf("after the deletion, intake_issues:\n%v\nwant\n%v", got, want)
	}
	if got, deletedAt := issueRows(), textRows(t, pool, "SELECT deleted_at::text FROM intake_issues WHERE id=$1", current); !slices.Equal(got, issuesBefore) ||
		!slices.Equal(deletedAt, []string{then}) {
		t.Errorf("issues %v (want %v unchanged); the row deleted at %v, want %s", got, issuesBefore, deletedAt, then)
	}
	for _, id := range []uuid.UUID{item, never} {
		if err := s.DeleteIntake(ctx, app.IntakeDeletion{IssueID: id, By: bob, At: at}); err == nil {
			t.Errorf("DeleteIntake() of %s, without an undeleted intake row = nil; want an error", id)
		}
	}
	if got := intakeTable(t, pool); !slices.Equal(got, want) {
		t.Errorf("after the refused deletions, intake_issues:\n%v\nwant\n%v", got, want)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.DeleteIntake(cancelled, app.IntakeDeletion{IssueID: other, By: bob, At: at}); !errors.Is(err, context.Canceled) {
		t.Errorf("DeleteIntake() = %v; want context.Canceled", err)
	}
}
