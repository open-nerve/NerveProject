package app_test

import (
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// newSubmit is web's world (newCreate) with its intake turned on, and the
// submission's use case over it: a creation core over the world's ports,
// as newCreate's, and the intake rows' fake over its work items.
func newSubmit() (*app.SubmitIntakeIssue, *world, *fakeIntake) {
	_, w := newCreate()
	w.project.locked.IntakeView = true
	intake := &fakeIntake{fakeIssues: w.issues}
	creation := app.NewIssueCreation(app.IssueCreationDeps{Attachments: noIssueAttachments{}, Events: discardEvents{}, Sanitizer: descriptionSanitizer{},
		Issues: w.issues, Sequences: w.project, States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles,
		Clock: clockAt{at: clockNow, log: w.log}})
	uc := app.NewSubmitIntakeIssue(app.SubmitIntakeIssueDeps{Locks: w.locks, Creation: creation, Intake: intake, Issues: intake})
	return uc, w, intake
}

// submitted are the calls of a submission by alice in web: a creation's
// (created), in the triage state, decided on intake.submit; whether alice
// subscribes, read for the core's answer; then the intake row inserted,
// the work item read back and answered as she sees it (M7 P2a spec 2.8):
// its sub-items counted and whether she subscribes, read again.
func submitted(id uuid.UUID, asked, labelsAsked, assignees, labels, subscribers []uuid.UUID, rows int) []string {
	calls := created(id, triage.ID, asked, labelsAsked, assignees, labels, subscribers, rows)
	calls[1] = line("Authorize", alice, domain.ActionSubmitIntake, acme, web, "outside tx")
	calls[4] = line("Authorize", alice, domain.ActionSubmitIntake, acme, web)
	return append(calls, line("IsSubscribed", id, alice), line("InsertIntake", id), line("GetIssue", id), counted(id), line("IsSubscribed", id, alice))
}

// A submission (M7 P2a spec 2.4) is a creation by the one core, in the
// project's triage state rather than its default one: decided on
// intake.submit, the work item and its rows written as a creation writes
// them, sorted after the greatest of the triage state's work items, not
// completed; then, in the same transaction, its intake row, the
// submitter's, at the creation's one moment. The answer is the work item
// read back after it, as the caller sees it: whether she subscribes too,
// and its intake row, pending.
func TestSubmitIntakeIssuePutsTheWorkItemInTriage(t *testing.T) {
	uc, w, intake := newSubmit()
	w.issues.subscribers = []uuid.UUID{alice}
	w.issues.greatest[triage.ID] = 75535
	high, html := domain.PriorityHigh, "<p>It breaks</p>"
	got, err := uc.Execute(as(alice), web, domain.NewIntakeIssue{Name: "  Login breaks ", DescriptionHTML: &html, Priority: &high,
		AssigneeIDs: []uuid.UUID{carol, bob}, LabelIDs: []uuid.UUID{ui, bug}, TargetDate: october(19)})
	if err != nil || w.issues.issue == nil {
		t.Fatalf("Execute() = %+v, %v; want the work item", got, err)
	}
	id := w.issues.issue.ID
	if want := submitted(id, []uuid.UUID{carol, bob}, []uuid.UUID{ui, bug}, []uuid.UUID{bob, carol}, []uuid.UUID{bug, ui},
		[]uuid.UUID{bob, alice, carol}, 3); !slices.Equal(w.log.calls, want) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
	}
	text := "It breaks"
	row := app.IssueRow{DescriptionHTML: html, DescriptionStripped: &text, ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7,
		Name: "Login breaks", StateID: triage.ID, Priority: domain.PriorityHigh, TargetDate: october(19), SortOrder: 85535, By: alice, At: clockNow}
	if !reflect.DeepEqual(*w.issues.issue, row) {
		t.Errorf("the work item %+v, want %+v", *w.issues.issue, row)
	}
	wantActivity(t, w, id, domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "bob"}, {ID: carol, DisplayName: "Carol C."}}))
	if len(intake.rows) != 1 || intake.rows[0].ID == (uuid.UUID{}) ||
		intake.rows[0] != (app.IntakeRow{ID: intake.rows[0].ID, WorkspaceID: acme, ProjectID: web, IssueID: id, By: alice, At: clockNow}) {
		t.Errorf("the intake rows %+v, want one of the work item %s, alice's at %v", intake.rows, id, clockNow)
	}
	stored, _, _ := w.issues.GetIssue(context.Background(), id)
	stored.Subscribed, stored.Intake = true, &domain.IntakeFacts{Status: domain.IntakePending}
	if !reflect.DeepEqual(got, stored) {
		t.Errorf("Execute() = %+v, want the work item as stored, subscribed, pending, %+v", got, stored)
	}
}

// A submission's refusals, in the order of M7 P2a spec 2.4, each case
// breaking the later rules too, so that its own refusal answers first:
// the body (domain.CheckNewIntakeIssue), before any read; a project not
// there or not seen, 404 project.not_found, then a role intake.submit does
// not allow, 403, before any lock; under the lock, the decision again;
// then an archived project, 409 project.archived, before its intake turned
// off, 409 issue.intake_disabled, read under the lock alone; then the
// creation's checks of the dependent rows, every problem in one 422. A
// refusal writes nothing, takes no number and inserts no intake row.
func TestSubmitIntakeIssueRefusalOrder(t *testing.T) {
	precheck := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionSubmitIntake, acme, web, "outside tx")}
	locked := append(slices.Clone(precheck), "Begin", line("Lock", web))
	decided := append(slices.Clone(locked), line("Authorize", alice, domain.ActionSubmitIntake, acme, web))
	other := uuid.NewV7() // after bug: the labels {other, bug} are asked out of id order
	bad := domain.NewIntakeIssue{Name: "a", AssigneeIDs: []uuid.UUID{gus, bob, out}, LabelIDs: []uuid.UUID{other, bug}}
	shut := func(w *world) { w.project.locked.Archived, w.project.locked.IntakeView = true, false }
	off := func(w *world) { w.project.locked.IntakeView = false }
	notAllowed := func(field, message string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: message}
	}
	unassignable := "must be an active member of the project who is not its guest"
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		in    domain.NewIntakeIssue
		want  error
		calls []string
	}{
		{"a blank name, in a project not there", func(w *world) { w.project.found = false }, domain.NewIntakeIssue{Name: " "},
			shared.Invalid(shared.FieldError{Field: "name", Code: "too_short", Message: "must not be empty"}), nil},
		{"a project not there", func(w *world) { w.project.found = false }, bad, domain.ErrProjectNotFound, precheck[:1]},
		{"a project not seen", func(w *world) { w.auth.answers = []error{shared.ErrNotVisible} }, bad, domain.ErrProjectNotFound, precheck},
		{"a role not allowed, archived, its intake off", func(w *world) { w.auth.answers = []error{shared.Forbidden()}; shut(w) }, bad,
			shared.Forbidden(), precheck},
		{"deleted while the lock waited", func(w *world) { w.project.locked = nil }, bad, domain.ErrProjectNotFound, locked},
		{"no longer seen", func(w *world) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, bad, domain.ErrProjectNotFound, decided},
		{"no longer allowed, archived, its intake off", func(w *world) { w.auth.answers = []error{nil, shared.Forbidden()}; shut(w) }, bad,
			shared.Forbidden(), decided},
		{"archived, its intake off", shut, bad, domain.ErrProjectArchived, decided},
		{"its intake off", off, bad, domain.ErrIntakeDisabled, decided},
		{"a guest, no member, a label not the project's", func(*world) {}, bad,
			shared.Invalid(notAllowed("assignee_ids[0]", unassignable), notAllowed("assignee_ids[2]", unassignable),
				notAllowed("label_ids[0]", "must be a label of the project")),
			append(slices.Clone(decided), line("States", web), line("Roles", web, []uuid.UUID{gus, bob, out}), line("Labels", web, []uuid.UUID{other, bug}))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, intake := newSubmit()
			tt.set(w)
			_, err := uc.Execute(as(alice), web, tt.in)
			refused(t, w, err, tt.want, tt.calls)
			if len(intake.rows) != 0 {
				t.Errorf("after the refusal: the intake rows %+v, want none", intake.rows)
			}
		})
	}
}

// What cannot be is an error, not a refusal (M7 P2a spec 2.4): a project
// without a triage state, as one without a default state is for a
// creation, before its number is taken. A failing insert of the intake row
// ends the submission with its failure, as itself, and no call after it
// runs: the transaction rolls back the work item written before it.
func TestSubmitIntakeIssueWithoutATriageStateOrItsRow(t *testing.T) {
	uc, w, intake := newSubmit()
	w.project.states = []app.StateRef{backlog, done}
	var refusal *shared.Error
	if _, err := uc.Execute(as(alice), web, domain.NewIntakeIssue{Name: "a"}); err == nil || errors.As(err, &refusal) || len(intake.rows) != 0 ||
		slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, "Next") }) {
		t.Errorf("Execute() in a project without a triage state = %v after %q; want an error before the number", err, w.log.calls)
	}
	uc, w, _ = newSubmit()
	w.log.fail = "InsertIntake"
	_, err := uc.Execute(as(alice), web, domain.NewIntakeIssue{Name: "a"})
	wantStoppedAt(t, w, err, "InsertIntake")
}

// payloadOf decodes every stored payload as in.
type payloadOf struct{ in domain.NewIssue }

func (d payloadOf) DecodeCreatePayload([]byte) (domain.NewIssue, error) { return d.in, nil }

// The creation core's triage placement is a submission's alone (M7 P2a
// spec 2.4): a creation and a draft's publication that ask for no state
// are in the project's default state, its triage state listed first.
func TestCreationAndPublicationStayInTheDefaultState(t *testing.T) {
	states := []app.StateRef{triage, done, backlog}
	t.Run("a creation", func(t *testing.T) {
		uc, w := newCreate()
		w.project.states = states
		if got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a"}); err != nil || w.issues.issue == nil || w.issues.issue.StateID != backlog.ID {
			t.Errorf("Execute() = %+v, %v, the work item %+v; want it in Backlog", got, err, w.issues.issue)
		}
	})
	t.Run("a draft's publication", func(t *testing.T) {
		w := newDraftWorld()
		w.p.id, w.p.found, w.p.locked, w.p.states = web, true, &app.LockedProject{WorkspaceID: acme}, states
		issues := &fakeIssues{log: w.log}
		clock := clockAt{at: clockNow, log: w.log}
		creation := app.NewIssueCreation(app.IssueCreationDeps{Attachments: noIssueAttachments{}, Events: discardEvents{}, Sanitizer: descriptionSanitizer{},
			Issues: issues, Sequences: w.p, States: w.p, Labels: w.p, Members: w.p, Profiles: fakeProfiles{log: w.log}, Clock: clock})
		uc := app.NewPublishDraft(app.PublishDraftDeps{Locks: w.locks(), Decoder: payloadOf{domain.NewIssue{Name: "Published"}}, Creation: creation,
			Drafts: w.s, Transfer: assetlessDraft{clock}})
		if got, err := uc.Execute(as(alice), w.s.row.ID); err != nil || issues.issue == nil || issues.issue.StateID != backlog.ID {
			t.Errorf("Execute() = %+v, %v, the work item %+v; want it in Backlog", got, err, issues.issue)
		}
	})
}
