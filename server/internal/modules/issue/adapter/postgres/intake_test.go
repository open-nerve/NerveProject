package postgresadapter_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
)

// newIntakeRow inserts an intake row of the work item issue of p, of the
// status, till when snoozed and target duplicated, nil for none, by by at
// now, and deleted at now when deleted; it returns its id. The rows of
// the submission and the triage are the store's (TestIntakeRowsAreWritten).
func newIntakeRow(t *testing.T, pool *pgxpool.Pool, p project, issue uuid.UUID, status string, till *time.Time, target *uuid.UUID,
	by uuid.UUID, deleted bool) uuid.UUID {
	t.Helper()
	id := uuid.NewV7()
	var deletedAt *time.Time
	if deleted {
		deletedAt = &now
	}
	exec(t, pool, `INSERT INTO intake_issues (id, workspace_id, project_id, issue_id, status, snoozed_till, duplicate_to_id, created_by_id,
		updated_by_id, created_at, updated_at, deleted_at) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, $9, $9, $10)`,
		id, p.workspace, p.id, issue, status, till, target, by, now, deletedAt)
	return id
}

// intakeTable is every row of intake_issues, in id order, as text: its
// id, its work item's, its status, time and target, "-" for none, its
// audit columns, whether it is deleted.
func intakeTable(t *testing.T, pool *pgxpool.Pool) []string {
	t.Helper()
	return textRows(t, pool, `SELECT format('%s %s/%s/%s %s %s %s by %s/%s at %s/%s deleted %s', id, workspace_id, project_id, issue_id,
		status, coalesce(snoozed_till::text, '-'), coalesce(duplicate_to_id::text, '-'), created_by_id, updated_by_id, created_at, updated_at,
		deleted_at IS NOT NULL)
		FROM intake_issues ORDER BY id`)
}

// stampOf is at as timestamptz reads as text.
func stampOf(t *testing.T, pool *pgxpool.Pool, at time.Time) string {
	t.Helper()
	var stamp string
	if err := pool.QueryRow(context.Background(), "SELECT $1::timestamptz::text", at).Scan(&stamp); err != nil {
		t.Fatal(err)
	}
	return stamp
}

// The intake rows a submission and a triage write (M7 P2a spec 2.4, 2.5):
// the submission's row pending, without a time or a target, created and
// last changed by the submitter at its moment; a triage changes the work
// item's undeleted row alone, its status, time and target, and who last
// changed it and when, not who created it: not the work item's deleted
// row before it, nor another work item's row. A triage of a work item
// without an undeleted row, or one that fails, is an error.
func TestIntakeRowsAreWritten(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice, bob := newAccount(t, pool, "alice@corp.com"), newAccount(t, pool, "bob@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	item, other, target := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	newIssue(t, pool, web, item, 1, alice)
	newIssue(t, pool, web, other, 2, alice)
	old := newIntakeRow(t, pool, web, item, "declined", nil, nil, alice, true)
	others := newIntakeRow(t, pool, web, other, "pending", nil, nil, alice, false)

	submitted, at := uuid.NewV7(), now.Add(time.Minute)
	if err := s.InsertIntake(ctx, app.IntakeRow{ID: submitted, WorkspaceID: web.workspace, ProjectID: web.id, IssueID: item, By: bob, At: at}); err != nil {
		t.Fatal(err)
	}
	place := func(issue uuid.UUID) string { return fmt.Sprintf("%s/%s/%s", web.workspace, web.id, issue) }
	before, then := stampOf(t, pool, now), stampOf(t, pool, at)
	oldLine := fmt.Sprintf("%s %s declined - - by %s/%s at %s/%s deleted t", old, place(item), alice, alice, before, before)
	othersLine := fmt.Sprintf("%s %s pending - - by %s/%s at %s/%s deleted f", others, place(other), alice, alice, before, before)
	want := []string{oldLine, othersLine, fmt.Sprintf("%s %s pending - - by %s/%s at %s/%s deleted f", submitted, place(item), bob, bob, then, then)}
	if got := intakeTable(t, pool); !slices.Equal(got, want) {
		t.Errorf("after the submission, intake_issues:\n%v\nwant\n%v", got, want)
	}

	till, later := now.Add(48*time.Hour), now.Add(time.Hour)
	for _, step := range []struct {
		triage app.IntakeTriage
		line   string
	}{
		{app.IntakeTriage{IssueID: item, Status: domain.IntakeSnoozed, SnoozedTill: &till, By: alice, At: later},
			fmt.Sprintf("snoozed %s - by %s/%s at %s/%s", stampOf(t, pool, till), bob, alice, then, stampOf(t, pool, later))},
		{app.IntakeTriage{IssueID: item, Status: domain.IntakeDuplicate, DuplicateToID: &target, By: bob, At: till},
			fmt.Sprintf("duplicate - %s by %s/%s at %s/%s", target, bob, bob, then, stampOf(t, pool, till))},
	} {
		if err := s.TriageIntake(ctx, step.triage); err != nil {
			t.Fatal(err)
		}
		want[2] = fmt.Sprintf("%s %s %s deleted f", submitted, place(item), step.line)
		if got := intakeTable(t, pool); !slices.Equal(got, want) {
			t.Errorf("after the triage to %s, intake_issues:\n%v\nwant\n%v", step.triage.Status, got, want)
		}
	}

	none := uuid.NewV7()
	newIssue(t, pool, web, none, 3, alice)
	newIntakeRow(t, pool, web, none, "pending", nil, nil, alice, true)
	if err := s.TriageIntake(ctx, app.IntakeTriage{IssueID: none, Status: domain.IntakeAccepted, By: alice, At: later}); err == nil {
		t.Error("TriageIntake() of a work item whose one intake row is deleted = nil; want an error")
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := s.InsertIntake(cancelled, app.IntakeRow{ID: uuid.NewV7(), WorkspaceID: web.workspace, ProjectID: web.id, IssueID: none, By: bob,
		At: at}); !errors.Is(err, context.Canceled) {
		t.Errorf("InsertIntake() = %v; want context.Canceled", err)
	}
	if err := s.TriageIntake(cancelled, app.IntakeTriage{IssueID: other, Status: domain.IntakeAccepted, By: alice, At: later}); !errors.Is(err, context.Canceled) {
		t.Errorf("TriageIntake() = %v; want context.Canceled", err)
	}
}

// intakeCase is a work item of the intake reads' tests, and the intake
// facts a read of the GetIssue family carries for it: nil for none.
type intakeCase struct {
	name string
	id   uuid.UUID
	want *domain.IntakeFacts
}

// newIntakeCases stores, under the work item parent of web, by alice, a
// work item of each kind an intake read tells apart: without an intake
// row; pending; snoozed till later; snoozed till earlier, run out, as
// stored; declined; a duplicate of target; accepted; and one whose only
// row, declined, is deleted, which reads as none. Each has a deleted row
// before its undeleted one but the first and the last.
func newIntakeCases(t *testing.T, pool *pgxpool.Pool, web project, parent, alice uuid.UUID) []intakeCase {
	t.Helper()
	earlier, later, target := now.Add(-time.Hour).Truncate(time.Microsecond), now.Add(time.Hour).Truncate(time.Microsecond), uuid.NewV7()
	cases := []intakeCase{
		{"none", uuid.NewV7(), nil},
		{"pending", uuid.NewV7(), &domain.IntakeFacts{Status: domain.IntakePending}},
		{"snoozed", uuid.NewV7(), &domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &later}},
		{"snoozed, run out", uuid.NewV7(), &domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &earlier}},
		{"declined", uuid.NewV7(), &domain.IntakeFacts{Status: domain.IntakeDeclined}},
		{"duplicate", uuid.NewV7(), &domain.IntakeFacts{Status: domain.IntakeDuplicate, DuplicateToID: &target}},
		{"accepted", uuid.NewV7(), &domain.IntakeFacts{Status: domain.IntakeAccepted}},
		{"deleted", uuid.NewV7(), nil},
	}
	for n, c := range cases {
		newIssue(t, pool, web, c.id, 100+n, alice)
		exec(t, pool, "UPDATE issues SET parent_id = $2 WHERE id = $1", c.id, parent)
		switch {
		case c.name == "deleted":
			newIntakeRow(t, pool, web, c.id, "declined", nil, nil, alice, true)
		case c.want != nil:
			newIntakeRow(t, pool, web, c.id, "accepted", nil, nil, alice, true)
			newIntakeRow(t, pool, web, c.id, string(c.want.Status), c.want.SnoozedTill, c.want.DuplicateToID, alice, false)
		}
	}
	return cases
}

// The three reads of the GetIssue family carry a work item's undeleted
// intake row (M7 P2a spec 2.3): its stored status, time and target, a
// snooze run out as stored; none for a work item without one, nor for
// one whose only row is deleted, whatever their deleted rows. A work item
// accepted is not in triage; every other with a row is.
func TestTheGetIssueReadsCarryTheIntakeRow(t *testing.T) {
	s, pool := newStore(t)
	ctx := context.Background()
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	parent := uuid.NewV7()
	newIssue(t, pool, web, parent, 1, alice)
	cases := newIntakeCases(t, pool, web, parent, alice)

	ids := make([]uuid.UUID, len(cases))
	for n, c := range cases {
		ids[n] = c.id
	}
	many, err := s.GetIssues(ctx, ids)
	if err != nil || len(many) != len(cases) {
		t.Fatalf("GetIssues() = %d work items, %v; want %d", len(many), err, len(cases))
	}
	under, err := s.SubIssues(ctx, parent, nil)
	if err != nil || len(under) != len(cases) {
		t.Fatalf("SubIssues() = %d work items, %v; want %d", len(under), err, len(cases))
	}
	for n, c := range cases {
		one, found, err := s.GetIssue(ctx, c.id)
		if err != nil || !found {
			t.Fatalf("%s: GetIssue() = %v, %v", c.name, found, err)
		}
		for read, got := range map[string]domain.Issue{"GetIssue": one, "GetIssues": many[n], "SubIssues": under[n]} {
			if got.ID != c.id || !reflect.DeepEqual(got.Intake, c.want) {
				t.Errorf("%s: %s() = %s with %+v; want %s with %+v", c.name, read, got.ID, got.Intake, c.id, c.want)
			}
			if inTriage := c.want != nil && c.name != "accepted"; got.InTriage() != inTriage {
				t.Errorf("%s: %s()'s InTriage() = %v, want %v", c.name, read, got.InTriage(), inTriage)
			}
		}
	}
}

// A search leaves out the work items in triage (M7 P2a spec 2.6): those
// with an undeleted intake row that is not accepted, pending, snoozed,
// one run out too, declined or a duplicate. It finds a work item accepted,
// one without a row and one whose only row is deleted; none of them
// carries an intake row, a search's rows never do (spec 2.3).
func TestSearchLeavesOutTheWorkItemsInTriage(t *testing.T) {
	s, pool := newStore(t)
	alice := newAccount(t, pool, "alice@corp.com")
	web := newProject(t, pool, newWorkspace(t, pool, "acme"), "WEB")
	parent := uuid.NewV7()
	newIssue(t, pool, web, parent, 1, alice)
	cases := newIntakeCases(t, pool, web, parent, alice)

	q, err := domain.ParseSearchQuery(map[string][]string{"q": {"Work"}})
	if err != nil {
		t.Fatal(err)
	}
	rows, err := s.Search(context.Background(), app.SearchRead{ActorID: alice, Projects: []app.VisibleProject{{ID: web.id}}, Query: q})
	if err != nil {
		t.Fatal(err)
	}
	found := map[uuid.UUID]bool{}
	for _, r := range rows {
		found[r.ID] = true
		if r.Intake != nil {
			t.Errorf("Search() reads %s with %+v; want no intake row on a search's rows", r.ID, r.Intake)
		}
	}
	if !found[parent] {
		t.Errorf("Search() = %d rows without the parent, which has no intake row", len(rows))
	}
	for _, c := range cases {
		if want := c.want == nil || c.name == "accepted"; found[c.id] != want {
			t.Errorf("%s: Search() finds it %v, want %v", c.name, found[c.id], want)
		}
	}
	if len(rows) != 4 {
		t.Errorf("Search() = %d rows, want 4: the parent, none, accepted and deleted", len(rows))
	}
}
