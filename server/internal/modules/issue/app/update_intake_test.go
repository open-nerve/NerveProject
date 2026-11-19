package app_test

import (
	"reflect"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// webAdmin is the grant of web's admin, webMember of its member.
var (
	webAdmin  = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleAdmin, ProjectAdmin: true}
	webMember = shared.Grant{WorkspaceRole: shared.RoleMember, ProjectRole: shared.RoleMember}
)

// intakeEditWorld is the triage's world (newTriage): web's intake turned
// on, its work item 7, carol's, in it, pending, in the triage state, each
// read of it through the intake rows' fake carrying its intake row. The
// content edit runs over it with an update core of its own over the same
// ports, whose events w.events keeps. alice decides as web's admin
// (webAdmin), before the lock and under it, unless a case says otherwise.
type intakeEditWorld struct {
	*triageWorld
	edit *app.UpdateIntakeIssue
}

func newIntakeEdit(t *testing.T) *intakeEditWorld {
	w := newTriage(t)
	w.auth.grants = []shared.Grant{webAdmin, webAdmin}
	update := app.NewIssueUpdate(app.IssueUpdateDeps{Events: w.events, Sanitizer: descriptionSanitizer{}, Locks: w.locks, Issues: w.intake,
		States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles})
	edit := app.NewUpdateIntakeIssue(app.UpdateIntakeIssueDeps{Locks: w.locks, Update: update, Clock: clockAt{at: clockNow, log: w.log}})
	return &intakeEditWorld{triageWorld: w, edit: edit}
}

// edited are the calls of a content edit of the intake work item id by
// alice up to the decision under the lock (lockedAs, on intake.update).
func edited(id uuid.UUID) []string {
	return lockedAs(domain.ActionUpdateIntake, id)
}

// untouched checks that a content edit refused wrote nothing: no field of
// the work item, no activity, no event.
func untouched(t *testing.T, w *intakeEditWorld) {
	t.Helper()
	if w.issues.changed != nil || w.issues.descriptionWrites != 0 || len(w.intake.written) != 0 || len(w.events.rows) != 0 {
		t.Errorf("after the refusal: the fields %+v, the description writes %d, the activity %+v, the events %+v; want none",
			w.issues.changed, w.issues.descriptionWrites, w.intake.written, w.events.rows)
	}
}

// A content edit of a work item of the intake (M7 P2b spec 2.4), by web's
// admin, of the work item carol submitted: Locks' calls on intake.update
// (edited); then the update core with the patch of the edit's fields: the
// account added, and the labels added and removed, checked and named; the
// clock, read once, after the lock; the work item's fields, its state left
// in Triage; its assignees and its labels removed and added; the account
// added subscribed; the display names; its activity, all of it alice's at
// that moment; issue.updated; the answer, the work item read back as alice
// sees it, its intake row with it.
func TestUpdateIntakeIssueChangesTheWorkItemThroughTheCore(t *testing.T) {
	w := newIntakeEdit(t)
	id, low := w.id, domain.PriorityLow
	got, err := w.edit.Execute(as(alice), id, domain.IntakeIssueUpdate{Name: name(" Mend "), Priority: &low, AssigneeIDs: set(out, dave),
		LabelIDs: set(ui), SetTargetDate: true, TargetDate: october(19)})
	if err != nil {
		t.Fatalf("Execute() = %v", err)
	}
	activity := domain.UpdateActivity(domain.Update{Name: &domain.Changed[string]{Old: "Fix", New: "Mend"},
		Priority: &domain.Changed[domain.Priority]{Old: domain.PriorityHigh, New: low}, TargetDate: &domain.Changed[*shared.Date]{New: october(19)},
		LabelsAdded: []domain.Named{{ID: ui, Name: "UI"}}, LabelsRemoved: []domain.Named{{ID: bug, Name: "Bug"}},
		AssigneesAdded: []domain.Assignee{{ID: dave, DisplayName: "dave"}}, AssigneesRemoved: []domain.Assignee{{ID: bob, DisplayName: "bob"}}})
	calls := append(edited(id), line("Roles", web, []uuid.UUID{dave}), line("Labels", web, []uuid.UUID{bug, ui}), "Now", line("UpdateIssue", id),
		line("RemoveAssignees", []uuid.UUID{bob}), line("InsertAssignees", []uuid.UUID{dave}), line("RemoveLabels", []uuid.UUID{bug}),
		line("InsertLabels", []uuid.UUID{ui}), line("InsertSubscribers", []uuid.UUID{dave}), line("DisplayNames", []uuid.UUID{dave, bob}),
		line("InsertActivities", len(activity)), line("GetIssue", id), counted(id))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if f := (app.IssueFields{SortOrder: 65535, ID: id, Name: "Mend", StateID: triage.ID, Priority: low, TargetDate: october(19), By: alice,
		At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, f) {
		t.Errorf("the fields written %+v, want %+v", w.issues.changed, f)
	}
	wantWritten(t, w.triageWorld, activity...)
	wantIssueEvent(t, w.events, domain.EventIssueUpdated, id, []shared.Change{
		{Field: "name", Old: evptr("Fix"), New: evptr("Mend")}, {Field: "priority", Old: evptr("high"), New: evptr("low")},
		{Field: "target_date", Old: evptr(""), New: evptr("2026-10-19")},
		{Field: "labels", Old: evptr(""), New: evptr("UI"), NewID: &ui}, {Field: "labels", Old: evptr("Bug"), New: evptr(""), OldID: &bug},
		{Field: "assignees", Old: evptr(""), New: evptr("dave"), NewID: &dave}, {Field: "assignees", Old: evptr("bob"), New: evptr(""), OldID: &bob},
	})
	wantAnswer(t, w.triageWorld, got, domain.IntakeFacts{Status: domain.IntakePending})
}

// The description goes through the update core as updateIssue's does: a
// change writes its activity, its version and issue.updated; skip_activity
// with its content as it is writes the description alone, by the same
// rule (TestDescriptionMigrationWritesWithoutAudit).
func TestUpdateIntakeIssueWritesTheDescriptionAsUpdateIssueDoes(t *testing.T) {
	t.Run("a change", func(t *testing.T) {
		w := newIntakeEdit(t)
		w.issues.issue.DescriptionHTML = "<p>Before</p>"
		if _, err := w.edit.Execute(as(alice), w.id, domain.IntakeIssueUpdate{DescriptionHTML: name("<p>After</p>")}); err != nil {
			t.Fatalf("Execute() = %v", err)
		}
		wantWritten(t, w.triageWorld, domain.UpdateActivity(domain.Update{Description: &domain.Changed[string]{Old: "<p>Before</p>", New: "<p>After</p>"}})...)
		wantIssueEvent(t, w.events, domain.EventIssueUpdated, w.id, []shared.Change{{Field: "description", Old: evptr("<p>Before</p>"), New: evptr("<p>After</p>")}})
		if len(w.issues.versions) != 1 || w.issues.versions[0].HTML != "<p>After</p>" {
			t.Errorf("the versions %+v, want the new description's", w.issues.versions)
		}
	})
	t.Run("skip_activity, its content as it is", func(t *testing.T) {
		w := newIntakeEdit(t)
		w.issues.issue.DescriptionHTML = "<p>Same</p>"
		skip := true
		got, err := w.edit.Execute(as(alice), w.id, domain.IntakeIssueUpdate{DescriptionHTML: name("<p><strong>Same</strong></p>"), SkipActivity: &skip})
		if err != nil || got.DescriptionHTML != "<p><strong>Same</strong></p>" {
			t.Fatalf("Execute() = %+v, %v; want the description written", got, err)
		}
		if w.issues.changed != nil || w.issues.descriptionWrites != 1 || len(w.intake.written) != 0 || len(w.issues.versions) != 0 || len(w.events.rows) != 0 {
			t.Errorf("the fields %+v, the description writes %d, the activity %+v, the versions %+v, the events %+v; want the description alone",
				w.issues.changed, w.issues.descriptionWrites, w.intake.written, w.issues.versions, w.events.rows)
		}
	})
}

// The relative rules of intake.update (M7 P2b spec 2.4, R4, R11, DP-5,
// DP-6), as the precheck's decision answers the caller's grant: the
// project's admin (Grant.ProjectAdmin) edits any work item of its intake,
// every field, one of no known creator too, and so does the workspace's
// admin who is only its guest, her own; any other caller only the work
// item he submitted, its creator (Issue.CreatedByID), so that none edits
// one of no known creator; a guest of the project, only its name, its
// description and skip_activity: a field but those given, even as it is
// or cleared, is refused. Each refusal is 403 forbidden, before any lock,
// and writes nothing.
func TestUpdateIntakeIssueRelativeRules(t *testing.T) {
	low, skip := domain.PriorityLow, true
	content := domain.IntakeIssueUpdate{Name: name("Mend"), DescriptionHTML: name("<p>Mine</p>")}
	priority := domain.IntakeIssueUpdate{Name: name("Mend"), Priority: &low}
	for _, tt := range []struct {
		name    string
		grant   shared.Grant
		creator uuid.UUID // none for the zero id
		edit    domain.IntakeIssueUpdate
		want    error // nil for an edit let through
	}{
		{"an admin, carol's", webAdmin, carol, priority, nil},
		{"an admin, of no known creator", webAdmin, uuid.UUID{}, priority, nil},
		{"an admin who is only a guest, her own, its priority", adminGuest, alice, priority, nil},
		{"a member who submitted it", webMember, alice, priority, nil},
		{"a member who did not", webMember, carol, content, shared.Forbidden()},
		{"a member, of no known creator", webMember, uuid.UUID{}, content, shared.Forbidden()},
		{"a guest who submitted it, its name and description", everyItem, alice, content, nil},
		{"a guest who submitted it, its description with skip_activity", everyItem, alice,
			domain.IntakeIssueUpdate{DescriptionHTML: name("<p>Mine</p>"), SkipActivity: &skip}, nil},
		{"an own-only guest who submitted it, its name and description", ownOnly, alice, content, nil},
		{"a guest who submitted it, its priority", everyItem, alice, priority, shared.Forbidden()},
		{"an own-only guest who submitted it, its priority", ownOnly, alice, priority, shared.Forbidden()},
		{"a guest who submitted it, its target date cleared", everyItem, alice, domain.IntakeIssueUpdate{SetTargetDate: true}, shared.Forbidden()},
		{"a guest who submitted it, its assignees as they are", everyItem, alice, domain.IntakeIssueUpdate{AssigneeIDs: set(bob, out)},
			shared.Forbidden()},
		{"a guest who submitted it, no label", everyItem, alice, domain.IntakeIssueUpdate{LabelIDs: set()}, shared.Forbidden()},
		{"a guest who did not", everyItem, carol, content, shared.Forbidden()},
		{"a guest, of no known creator", everyItem, uuid.UUID{}, content, shared.Forbidden()},
	} {
		t.Run(tt.name, func(t *testing.T) {
			w := newIntakeEdit(t)
			w.issues.issue.By, w.auth.grants = tt.creator, []shared.Grant{tt.grant, tt.grant}
			_, err := w.edit.Execute(as(alice), w.id, tt.edit)
			if tt.want != nil {
				wantRefusal(t, w.world, err, tt.want, edited(w.id)[:2])
				untouched(t, w)
				return
			}
			if err != nil || len(w.log.calls) < 6 || !slices.Equal(w.log.calls[:6], edited(w.id)) || w.issues.changed == nil {
				t.Errorf("Execute() = %v, calls %q, the fields written %+v; want the edit written", err, w.log.calls, w.issues.changed)
			}
		})
	}
}

// A content edit is for a work item still to triage (M7 P2b spec 2.4,
// DP-7): one accepted, declined or a duplicate, accepted and archived
// since too, is 409 issue.intake_closed, by its stored status under the
// lock, the clock not read; a snoozed one is open, its snooze run out or
// not, and so is a pending one.
func TestUpdateIntakeIssueOnlyWhileOpen(t *testing.T) {
	for what, tt := range map[string]struct {
		stored   domain.IntakeFacts
		archived bool
		want     error
	}{
		"accepted":                {domain.IntakeFacts{Status: domain.IntakeAccepted}, false, domain.ErrIntakeClosed},
		"accepted, then archived": {domain.IntakeFacts{Status: domain.IntakeAccepted}, true, domain.ErrIntakeClosed},
		"declined":                {domain.IntakeFacts{Status: domain.IntakeDeclined}, false, domain.ErrIntakeClosed},
		"a duplicate":             {domain.IntakeFacts{Status: domain.IntakeDuplicate, DuplicateToID: &dup}, false, domain.ErrIntakeClosed},
		"pending":                 {domain.IntakeFacts{Status: domain.IntakePending}, false, nil},
		"snoozed":                 {domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &soon}, false, nil},
		"a snooze run out":        {domain.IntakeFacts{Status: domain.IntakeSnoozed, SnoozedTill: &gone}, false, nil},
	} {
		t.Run(what, func(t *testing.T) {
			w := newIntakeEdit(t)
			w.intake.facts[w.id] = tt.stored
			if tt.archived {
				w.issues.archivedAt = &earlier
			}
			_, err := w.edit.Execute(as(alice), w.id, domain.IntakeIssueUpdate{Name: name("Mend")})
			if tt.want != nil {
				wantRefusal(t, w.world, err, tt.want, edited(w.id))
				untouched(t, w)
				return
			}
			if err != nil || w.issues.changed == nil || w.issues.changed.Name != "Mend" {
				t.Errorf("Execute() = %v, the fields written %+v; want Mend", err, w.issues.changed)
			}
		})
	}
}
