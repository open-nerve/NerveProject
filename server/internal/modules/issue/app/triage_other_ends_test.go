package app_test

import (
	"reflect"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// inIntake is the work item i with the intake row facts, as a read of the
// GetIssue family carries it.
func inIntake(i domain.Issue, status domain.IntakeStatus) domain.Issue {
	i.Intake = &domain.IntakeFacts{Status: status}
	return i
}

// The other end of a write may not be a work item in its project's intake
// (M7 P2a spec 2.6, DP-8), refused with the 422 an end the write may not
// have answers, at its place: a parent of a creation, of an update and of
// a draft's publication (ParentNotAllowed), a sub-item (SubIssueNotAllowed),
// a relation's other work item (RelatedNotAllowed). Its row pending or
// declined, it is in the intake; accepted, it is out of it, and taken as
// any other.
func TestTheOtherEndOfAWriteIsNoWorkItemInTheIntake(t *testing.T) {
	for _, tt := range []struct {
		name string
		run  func(t *testing.T, status domain.IntakeStatus) error
		want error
	}{
		{"a creation's parent", func(_ *testing.T, status domain.IntakeStatus) error {
			uc, w := newCreate()
			k := withKin(w)
			w.issues.others[k.up.ID] = inIntake(k.up, status)
			_, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "Part", ParentID: &k.up.ID})
			return err
		}, shared.Invalid(domain.ParentNotAllowed())},
		{"an update's parent", func(_ *testing.T, status domain.IntakeStatus) error {
			uc, w, id := newUpdate()
			k := withKin(w)
			w.issues.others[k.up.ID] = inIntake(k.up, status)
			_, err := uc.Execute(as(alice), id, domain.IssuePatch{SetParentID: true, ParentID: &k.up.ID})
			return err
		}, shared.Invalid(domain.ParentNotAllowed())},
		{"a draft's publication's parent", func(_ *testing.T, status domain.IntakeStatus) error {
			w := newDraftWorld()
			w.p.id, w.p.found, w.p.locked, w.p.states = web, true, &app.LockedProject{WorkspaceID: acme, Identifier: "WEB"}, []app.StateRef{backlog}
			parent := inIntake(domain.Issue{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, SequenceID: 3, StateID: backlog.ID, CreatedByID: &bob}, status)
			issues := &fakeIssues{log: w.log, others: map[uuid.UUID]domain.Issue{parent.ID: parent}}
			clock := clockAt{at: clockNow, log: w.log}
			creation := app.NewIssueCreation(app.IssueCreationDeps{Attachments: noIssueAttachments{}, Events: discardEvents{}, Sanitizer: descriptionSanitizer{},
				Issues: issues, Sequences: w.p, States: w.p, Labels: w.p, Members: w.p, Profiles: fakeProfiles{log: w.log}, Clock: clock})
			uc := app.NewPublishDraft(app.PublishDraftDeps{Locks: w.locks(), Decoder: payloadOf{domain.NewIssue{Name: "Published", ParentID: &parent.ID}},
				Creation: creation, Drafts: w.s, Transfer: assetlessDraft{clock}})
			_, err := uc.Execute(as(alice), w.s.row.ID)
			return err
		}, shared.Invalid(domain.ParentNotAllowed())},
		{"a sub-item", func(_ *testing.T, status domain.IntakeStatus) error {
			_, w, id := newUpdate()
			k := withKin(w)
			w.issues.others[k.top.ID] = inIntake(k.top, status)
			_, err := addOver(w).Execute(as(alice), id, []uuid.UUID{k.up.ID, k.top.ID})
			return err
		}, shared.Invalid(domain.SubIssueNotAllowed(1))},
		{"a relation's other work item", func(_ *testing.T, status domain.IntakeStatus) error {
			uc, w, _, id, r := newRelate()
			w.issues.others[farID] = inIntake(r.far, status)
			_, err := uc.Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{nearID, farID})
			return err
		}, shared.Invalid(domain.RelatedNotAllowed(1))},
	} {
		for _, status := range []domain.IntakeStatus{domain.IntakePending, domain.IntakeDeclined} {
			t.Run(tt.name+"/"+string(status), func(t *testing.T) {
				wantRefused(t, tt.run(t, status), tt.want)
			})
		}
		t.Run(tt.name+"/accepted", func(t *testing.T) {
			if err := tt.run(t, domain.IntakeAccepted); err != nil {
				t.Errorf("Execute() = %v, want the write", err)
			}
		})
	}
}

// A search from a work item in its project's intake is refused as one
// from an archived work item is (M7 P2a spec 2.6, DP-5): the one value
// 404 issue.not_found, before any candidate is read. From an accepted one,
// the search runs, in its project.
func TestASearchFromAWorkItemInTheIntakeIsNotFound(t *testing.T) {
	source := uuid.NewV7()
	for _, tt := range []struct {
		name string
		set  func(i *domain.Issue)
		want error
	}{
		{"pending", func(i *domain.Issue) { *i = inIntake(*i, domain.IntakePending) }, domain.ErrIssueNotFound},
		{"snoozed", func(i *domain.Issue) { *i = inIntake(*i, domain.IntakeSnoozed) }, domain.ErrIssueNotFound},
		{"archived", func(i *domain.Issue) { i.ArchivedAt = &earlier }, domain.ErrIssueNotFound},
		{"accepted", func(i *domain.Issue) { *i = inIntake(*i, domain.IntakeAccepted) }, nil},
	} {
		t.Run(tt.name, func(t *testing.T) {
			u, w, _, _, rows := searchFixture()
			i := domain.Issue{ID: source, WorkspaceID: acme, ProjectID: web, SequenceID: 5, CreatedByID: &alice}
			tt.set(&i)
			w.issues.others[source] = i
			rows.rows = nil
			_, err := u.Execute(as(alice), "acme", map[string][]string{"for": {"relation"}, "issue_id": {source.String()}})
			web := []app.VisibleProject{{ID: web, OwnOnly: true}}
			switch {
			case tt.want == nil && (err != nil || len(rows.in) != 1 || !reflect.DeepEqual(rows.in[0].Projects, web)):
				t.Errorf("Execute() = %v after reading %+v; want the search run in %+v", err, rows.in, web)
			case tt.want != nil && (err != tt.want || len(rows.in) != 0):
				t.Errorf("Execute() = %#v after reading %+v; want the very value %#v, no candidate read", err, rows.in, tt.want)
			}
		})
	}
}
