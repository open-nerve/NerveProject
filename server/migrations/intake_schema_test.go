package migrations_test

import (
	"context"
	"io/fs"
	"slices"
	"testing"
	"testing/fstest"

	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/migrations"
)

// The ids the intake table's tests write besides newMembershipDatabase's,
// as SQL literals: a third work item of web, numbered 3, and the intake
// row of web's work item.
const (
	webThird   = "'0199a2b4-0000-7000-8000-0000000000f1'"
	intakeItem = "'0199a2b4-0000-7000-8000-0000000000f2'"
)

// intakeOf is the insertion of an undeleted pending intake row of the work
// item issue of project, submitted by bob.
func intakeOf(project, issue string) string {
	return "INSERT INTO intake_issues (id, workspace_id, project_id, issue_id, status, created_by_id, updated_by_id) VALUES (gen_random_uuid(), " +
		issueAcme + ", " + project + ", " + issue + ", 'pending', " + issueBob + ", " + issueBob + ")"
}

// The intake_issues table (M7 P2a spec 2.2): its columns, indexes, foreign
// keys and CHECKs; the five statuses and no other, a snoozed row with its
// date and only it with one, a duplicate with its target and only it with
// one, whatever the date (an elapsed snooze is stored as it was, spec 2.3)
// or the target (the table does not judge it); the target no foreign key,
// so its work item's purge leaves the row as it was (§15 D2).
func TestIntakeIssuesSchema(t *testing.T) {
	pool := newMembershipDatabase(t)
	if got, want := columnsOf(t, pool, "intake_issues"), []string{
		"id uuid NO -", "workspace_id uuid NO -", "project_id uuid NO -", "issue_id uuid NO -", "status character varying NO -",
		"snoozed_till timestamp with time zone YES -", "duplicate_to_id uuid YES -",
		"created_by_id uuid YES -", "updated_by_id uuid YES -", "created_at timestamp with time zone NO now()",
		"updated_at timestamp with time zone NO now()", "deleted_at timestamp with time zone YES -",
	}; !slices.Equal(got, want) {
		t.Fatalf("intake_issues columns\n%q\nwant\n%q", got, want)
	}
	if got, want := indexesOf(t, pool, "intake_issues"), []string{
		"CREATE INDEX intake_issues_deleted_at_idx ON public.intake_issues USING btree (deleted_at) WHERE (deleted_at IS NOT NULL)",
		"CREATE INDEX intake_issues_issue_id_idx ON public.intake_issues USING btree (issue_id)",
		"CREATE UNIQUE INDEX intake_issues_issue_id_key ON public.intake_issues USING btree (issue_id) WHERE (deleted_at IS NULL)",
		"CREATE UNIQUE INDEX intake_issues_pkey ON public.intake_issues USING btree (id)",
		"CREATE INDEX intake_issues_project_id_idx ON public.intake_issues USING btree (project_id)",
		"CREATE INDEX intake_issues_project_id_status_idx ON public.intake_issues USING btree (project_id, status) WHERE (deleted_at IS NULL)",
		"CREATE INDEX intake_issues_workspace_id_idx ON public.intake_issues USING btree (workspace_id)",
	}; !slices.Equal(got, want) {
		t.Errorf("intake_issues indexes\n%q\nwant\n%q", got, want)
	}
	if got, want := foreignKeysOf(t, pool, "intake_issues"), []string{
		"intake_issues_created_by_id_fkey users n", "intake_issues_issue_id_fkey issues c", "intake_issues_project_id_fkey projects c",
		"intake_issues_updated_by_id_fkey users n", "intake_issues_workspace_id_fkey workspaces c",
	}; !slices.Equal(got, want) {
		t.Errorf("intake_issues foreign keys\n%q\nwant\n%q", got, want)
	}

	accepted(t, pool, "INSERT INTO intake_issues (id, workspace_id, project_id, issue_id, status) VALUES ("+intakeItem+", "+issueAcme+", "+
		issueWeb+", "+webIssue+", 'pending')")
	update := func(set string) string { return "UPDATE intake_issues SET " + set + " WHERE id=" + intakeItem }
	for _, set := range []string{
		"status='snoozed', snoozed_till=now() + interval '1 day'", "snoozed_till=now() - interval '1 day'",
		"status='pending', snoozed_till=NULL", "status='duplicate', duplicate_to_id=" + opsIssue, "duplicate_to_id=gen_random_uuid()",
		"status='accepted', duplicate_to_id=NULL", "status='declined'",
	} {
		accepted(t, pool, update(set))
	}
	const snooze, duplicate, status = "intake_issues_snoozed_till_check", "intake_issues_duplicate_to_id_check", "intake_issues_status_check"
	for _, tt := range []struct{ set, constraint string }{
		{"status='snoozed'", snooze},
		{"status='duplicate'", duplicate},
		{"status='pending', snoozed_till=now()", snooze},
		{"status='accepted', snoozed_till=now()", snooze},
		{"snoozed_till=now()", snooze},
		{"status='duplicate', duplicate_to_id=" + webOther + ", snoozed_till=now()", snooze},
		{"status='pending', duplicate_to_id=" + webOther, duplicate},
		{"status='accepted', duplicate_to_id=" + webOther, duplicate},
		{"duplicate_to_id=" + webOther, duplicate},
		{"status='snoozed', snoozed_till=now(), duplicate_to_id=" + webOther, duplicate},
		{"status='triage'", status},
		{"status='rejected'", status},
		{"status='Pending'", status},
		{"status='-2'", status},
		{"status=''", status},
	} {
		refusedBy(t, pool, update(tt.set), "23514", tt.constraint)
	}
	for _, column := range []string{"workspace_id", "project_id", "issue_id", "status", "created_at", "updated_at"} {
		nullRefused(t, pool, update(column+"=NULL"), column)
	}

	accepted(t, pool, update("status='duplicate', duplicate_to_id="+webOther), "DELETE FROM issues WHERE id="+webOther)
	if got := names(t, pool, "SELECT status||' '||duplicate_to_id::text FROM intake_issues WHERE id="+intakeItem); !slices.Equal(got,
		[]string{"duplicate " + webOther[1:len(webOther)-1]}) {
		t.Fatalf("the duplicate after its target's purge = %q, want it as it was", got)
	}
}

// A work item has at most one intake row among the undeleted rows (spec
// 2.2): a second is refused, before and after the first is triaged; a
// soft-deleted row holds no place, however many share the work item.
// Other work items hold their own.
func TestIntakeIssueKeyHoldsAmongUndeletedRowsOnly(t *testing.T) {
	pool := newMembershipDatabase(t)
	accepted(t, pool, intakeOf(issueWeb, webIssue), intakeOf(issueWeb, webOther), intakeOf(issueOps, opsIssue))
	refusedBy(t, pool, intakeOf(issueWeb, webIssue), "23505", "intake_issues_issue_id_key")
	accepted(t, pool, "UPDATE intake_issues SET status='accepted' WHERE issue_id="+webIssue)
	refusedBy(t, pool, intakeOf(issueWeb, webIssue), "23505", "intake_issues_issue_id_key")
	accepted(t, pool, "UPDATE intake_issues SET deleted_at=now() WHERE issue_id="+webIssue, intakeOf(issueWeb, webIssue),
		"UPDATE intake_issues SET deleted_at=now() WHERE issue_id="+webIssue+" AND deleted_at IS NULL", intakeOf(issueWeb, webIssue))
	refusedBy(t, pool, intakeOf(issueWeb, webIssue), "23505", "intake_issues_issue_id_key")
}

// Where intake rows go (spec 2.2), each step taking its own rows only
// (membershipsGo): purging their author's account keeps them without him;
// purging a work item takes its row, and not the row that names it as its
// duplicate; purging web takes its rows, and the workspace the rest.
func TestIntakeRowsGoWithTheRowsAboveThem(t *testing.T) {
	pool := newMembershipDatabase(t)
	accepted(t, pool,
		"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES ("+webThird+", "+issueAcme+", "+issueWeb+", "+
			webState+", 'Third', 3)",
		intakeOf(issueWeb, webIssue), intakeOf(issueWeb, webOther), intakeOf(issueWeb, webThird), intakeOf(issueOps, opsIssue),
		"UPDATE intake_issues SET status='duplicate', duplicate_to_id="+webIssue+" WHERE issue_id="+opsIssue)
	membershipsGo(t, pool, "intake_issues", "DELETE FROM issues WHERE id="+webIssue, "issue_id="+webIssue)
}

// Triage's activity (spec 2.2, 00051): before it, the activity's verbs are
// the three; with it, a row of the field intake takes the five statuses as
// its verb too, and only such a row: neither one without a field (IS NOT
// DISTINCT FROM, pre-flight re-85 I1) nor one of another field; Down
// deletes the intake rows, keeps the others as they were and restores the
// CHECK as 00019 wrote it (re-85 m5); Up works once more.
func TestIntakeVerbsMigration(t *testing.T) {
	ctx := context.Background()
	pool := newPool(t, pgtest.NewEmptyDatabase(t))
	before, through := fstest.MapFS{}, fstest.MapFS{}
	entries, err := fs.ReadDir(migrations.FS(), ".")
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() > "00051_z" {
			continue
		}
		data, err := fs.ReadFile(migrations.FS(), e.Name())
		if err != nil {
			t.Fatal(err)
		}
		through[e.Name()] = &fstest.MapFile{Data: data}
		if e.Name() <= "00050_z" {
			before[e.Name()] = through[e.Name()]
		}
	}
	m, err := postgres.NewMigrator(pool, before)
	if err != nil {
		t.Fatal(err)
	}
	if up, err := m.Up(ctx); err != nil || len(up) != 50 {
		t.Fatalf("old Up: %v, %v", up, err)
	}
	if err := m.Close(); err != nil {
		t.Fatal(err)
	}
	activity := func(field, verb string) string {
		return "INSERT INTO issue_activities (id, workspace_id, project_id, issue_id, verb, field) VALUES (gen_random_uuid(), " + issueAcme + ", " +
			issueWeb + ", " + webIssue + ", '" + verb + "', " + field + ")"
	}
	accepted(t, pool,
		"INSERT INTO users (id, email, password, display_name) VALUES ("+issueBob+", 'bob@corp.com', 'x', 'bob')",
		"INSERT INTO workspaces (id, name, slug) VALUES ("+issueAcme+", 'Acme', 'acme')",
		"INSERT INTO projects (id, workspace_id, name, identifier) VALUES ("+issueWeb+", "+issueAcme+", 'Web', 'WEB')",
		"INSERT INTO states (id, workspace_id, project_id, name, color) VALUES ("+webState+", "+issueAcme+", "+issueWeb+", 'Backlog', '#60646C')",
		"INSERT INTO issues (id, workspace_id, project_id, state_id, name, sequence_id) VALUES ("+webIssue+", "+issueAcme+", "+issueWeb+", "+
			webState+", 'Work', 1)",
		activity("NULL", "created"), activity("'state'", "updated"), activity("NULL", "deleted"))
	const verbCheck = "issue_activities_verb_check"
	definition := func() string {
		t.Helper()
		got := names(t, pool, "SELECT pg_get_constraintdef(oid) FROM pg_constraint WHERE conrelid='public.issue_activities'::regclass AND conname='"+verbCheck+"'")
		if len(got) != 1 {
			t.Fatalf("%s = %q, want one", verbCheck, got)
		}
		return got[0]
	}
	others := func() []string {
		t.Helper()
		return names(t, pool, "SELECT to_jsonb(a)::text FROM issue_activities a WHERE field IS DISTINCT FROM 'intake' ORDER BY id")
	}
	old, kept := definition(), others()
	refusedBy(t, pool, activity("'intake'", "accepted"), "23514", verbCheck)

	m, err = postgres.NewMigrator(pool, through)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := m.Close(); err != nil {
			t.Error(err)
		}
	})
	for pass := range 2 {
		up, err := m.Up(ctx)
		if err != nil || len(up) != 1 || up[0].Version != 51 || up[0].Source != "00051_issue_issue_activities_intake_verbs.sql" {
			t.Fatalf("Up %d = %v, %v; want only 51", pass, up, err)
		}
		for _, verb := range []string{"pending", "snoozed", "accepted", "declined", "duplicate"} {
			accepted(t, pool, activity("'intake'", verb))
		}
		for _, row := range []struct{ field, verb string }{
			{"NULL", "accepted"}, {"'state'", "accepted"}, {"'intake'", "commented"}, {"'intake'", "Accepted"}, {"''", "pending"},
		} {
			refusedBy(t, pool, activity(row.field, row.verb), "23514", verbCheck)
		}
		if pass == 1 {
			break
		}
		down, err := m.Down(ctx)
		if err != nil || down == nil || down.Version != 51 {
			t.Fatalf("Down = %v, %v", down, err)
		}
		if got := definition(); got != old {
			t.Fatalf("after Down, %s = %s, want 00019's %s", verbCheck, got, old)
		}
		if got := others(); !slices.Equal(got, kept) {
			t.Fatalf("after Down, the activity =\n%q\nwant the rows other than the intake ones as they were\n%q", got, kept)
		}
		if got := names(t, pool, "SELECT verb FROM issue_activities WHERE field = 'intake' ORDER BY verb"); len(got) != 0 {
			t.Fatalf("after Down, the intake activity = %q, want none", got)
		}
		refusedBy(t, pool, activity("'intake'", "accepted"), "23514", verbCheck)
	}
}
