package app_test

import (
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/content"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// part is bob's work item 3 of web, under none, the sub-item an addition
// of sub-items puts under the work item (sourceWrites).
var part = uuid.NewV7()

// intoIntake gives w's work item id the intake row facts: w's locks are
// rebuilt over the intake rows' fake, so that each read of the work item
// through them carries its row, as the GetIssue family's does, and a use
// case built over w after it reads through them. The work item's state is
// left as it is: the row alone says whether it is in the intake (M7 P2a
// spec R2).
func intoIntake(w *world, id uuid.UUID, facts domain.IntakeFacts) *fakeIntake {
	intake := &fakeIntake{fakeIssues: w.issues, facts: map[uuid.UUID]domain.IntakeFacts{id: facts}}
	w.locks = app.NewLocks(w.project, intake, w.auth, fakeGate{}, fakeTx{log: w.log}, noIssueAttachments{})
	return intake
}

// sourceWrite is one of the six writes that refuse their own work item
// while it is in its project's intake (M7 P2a spec 2.6, R4): its name; its
// world, in which it runs on the work item id and, the work item in no
// intake, writes; how it runs there, built over w's locks as they are when
// it runs; and its calls up to its decision under the lock.
type sourceWrite struct {
	name  string
	world func() (*world, uuid.UUID)
	run   func(w *world, id uuid.UUID) error
	calls func(id uuid.UUID) []string
}

// sourceWrites are the six: updateIssue, a rename; archiveIssue and
// unarchiveIssue of web's work item 7 in Done, not archived (newArchive);
// deleteIssue; addSubIssues of part under it; createIssueRelations from it
// to far, then near (newRelate).
func sourceWrites() []sourceWrite {
	as7 := func(action shared.Action) func(uuid.UUID) []string {
		return func(id uuid.UUID) []string { return lockedAs(action, id) }
	}
	plain := func() (*world, uuid.UUID) { _, w, id := newUpdate(); return w, id }
	inDone := func() (*world, uuid.UUID) { _, _, w, id := newArchive(); return w, id }
	return []sourceWrite{
		{"updateIssue", plain, func(w *world, id uuid.UUID) error {
			_, err := updateOver(w).Execute(as(alice), id, domain.IssuePatch{Name: name("Mend")})
			return err
		}, as7(domain.ActionUpdate)},
		{"archiveIssue", inDone, func(w *world, id uuid.UUID) error {
			archive, _ := archiveOver(w)
			_, err := archive.Execute(as(alice), id)
			return err
		}, as7(domain.ActionArchive)},
		{"unarchiveIssue", inDone, func(w *world, id uuid.UUID) error {
			_, unarchive := archiveOver(w)
			_, err := unarchive.Execute(as(alice), id)
			return err
		}, as7(domain.ActionArchive)},
		{"deleteIssue", plain, func(w *world, id uuid.UUID) error { return deleteOver(w).Execute(as(alice), id) }, as7(domain.ActionDelete)},
		{"addSubIssues", func() (*world, uuid.UUID) {
			w, id := plain()
			w.issues.others = map[uuid.UUID]domain.Issue{part: {ID: part, WorkspaceID: acme, ProjectID: web, SequenceID: 3, Name: "Part",
				StateID: backlog.ID, CreatedByID: &bob}}
			return w, id
		}, func(w *world, id uuid.UUID) error {
			_, err := addOver(w).Execute(as(alice), id, []uuid.UUID{part})
			return err
		}, as7(domain.ActionUpdate)},
		{"createIssueRelations", func() (*world, uuid.UUID) { _, w, _, id, _ := newRelate(); return w, id }, func(w *world, id uuid.UUID) error {
			_, err := relateOver(w, &fakeRelations{fakeIssues: w.issues}).Execute(as(alice), id, domain.RelatesTo, []uuid.UUID{farID, nearID})
			return err
		}, func(id uuid.UUID) []string {
			return reaching(id, relating{near: domain.Issue{ID: nearID}, far: domain.Issue{ID: farID}})[:10]
		}},
	}
}

// Each of the six writes refuses its own work item while it is in its
// project's intake, its row pending or declined, any status but accepted
// (M7 P2a spec 2.6, R4): 409 issue.in_triage, the guard each of them takes
// (write.check), before any lock, and again under it. It comes after the
// 404 of a work item the guest rule hides and the 403 of a role not
// allowed, and before every refusal under the lock: an archived project's,
// an archived work item's, and an archiving's or unarchiving's of a work
// item already as asked, which answers it as read when it is in no intake.
// A work item put in the intake while the lock waited is refused under the
// lock, its project archived too. Nothing is written.
func TestTheSixWritesRefuseAWorkItemInTheIntake(t *testing.T) {
	pending := domain.IntakeFacts{Status: domain.IntakePending}
	for _, s := range sourceWrites() {
		for _, tt := range []struct {
			name  string
			set   func(w *world, in *fakeIntake, id uuid.UUID)
			want  error
			under bool // refused under the lock, else before it
		}{
			{"pending", func(*world, *fakeIntake, uuid.UUID) {}, domain.ErrInTriage, false},
			{"declined", func(_ *world, in *fakeIntake, id uuid.UUID) {
				in.facts[id] = domain.IntakeFacts{Status: domain.IntakeDeclined}
			}, domain.ErrInTriage, false},
			{"its project archived", func(w *world, _ *fakeIntake, _ uuid.UUID) { w.project.locked.Archived = true }, domain.ErrInTriage, false},
			{"archived", func(w *world, _ *fakeIntake, _ uuid.UUID) { w.issues.archivedAt = &earlier }, domain.ErrInTriage, false},
			{"put in the intake while the lock waited, its project archived", func(w *world, in *fakeIntake, id uuid.UUID) {
				delete(in.facts, id)
				w.project.locked.Archived, w.project.waited = true, func() { in.facts[id] = pending }
			}, domain.ErrInTriage, true},
			{"hidden by the guest rule, a role not allowed", func(w *world, _ *fakeIntake, _ uuid.UUID) {
				w.auth.answers, w.auth.grants = []error{shared.Forbidden()}, []shared.Grant{ownOnly}
			}, domain.ErrIssueNotFound, false},
			{"a role not allowed", func(w *world, _ *fakeIntake, _ uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, shared.Forbidden(),
				false},
		} {
			t.Run(s.name+"/"+tt.name, func(t *testing.T) {
				w, id := s.world()
				in := intoIntake(w, id, pending)
				tt.set(w, in, id)
				err := s.run(w, id)
				calls := s.calls(id)
				if !tt.under {
					calls = calls[:2]
				}
				wantRefusal(t, w, err, tt.want, calls)
			})
		}
	}
}

// An accepted work item is out of the intake (M7 P2a spec 2.3): each of
// the six writes takes it as any other.
func TestTheSixWritesTakeAnAcceptedWorkItem(t *testing.T) {
	for _, s := range sourceWrites() {
		t.Run(s.name, func(t *testing.T) {
			w, id := s.world()
			intoIntake(w, id, domain.IntakeFacts{Status: domain.IntakeAccepted})
			if err := s.run(w, id); err != nil {
				t.Errorf("Execute() = %v, want the write", err)
			}
		})
	}
}

// A work item in its project's intake takes the writes of its
// collaboration as any other (M7 design 3.6, P2a spec 2.6, DP-6): a link
// created, changed and deleted, though issue.update decides them as it
// decides the six writes that refuse it; a comment; a reaction; the
// caller's subscription.
func TestAWorkItemInTheIntakeTakesItsCollaboration(t *testing.T) {
	links := func(w *world, id uuid.UUID) *writableLinks {
		return &writableLinks{fakeLinks: &fakeLinks{fakeRelations: &fakeRelations{fakeIssues: w.issues}}, found: true,
			current: domain.Link{ID: uuid.NewV7(), WorkspaceID: acme, ProjectID: web, IssueID: id, URL: "http://169.254.169.254/before",
				CreatedByID: &bob, CreatedAt: earlier, UpdatedAt: earlier}}
	}
	for _, tt := range []struct {
		name string
		run  func(t *testing.T, w *world, id uuid.UUID) error
	}{
		{"a link created", func(t *testing.T, w *world, id uuid.UUID) error {
			uc := app.NewCreateIssueLink(app.CreateIssueLinkDeps{Locks: w.locks, Links: links(w, id), Clock: clockAt{at: clockNow, log: w.log},
				Events: &recordedEvents{t: t}})
			_, err := uc.Execute(as(alice), id, domain.NewLink{URL: "169.254.169.254/after"})
			return err
		}},
		{"a link changed", func(t *testing.T, w *world, id uuid.UUID) error {
			l := links(w, id)
			uc := app.NewUpdateIssueLink(app.UpdateIssueLinkDeps{Locks: w.locks, Links: l, Clock: clockAt{at: clockNow, log: w.log},
				Events: &recordedEvents{t: t}})
			_, err := uc.Execute(as(alice), l.current.ID, domain.LinkPatch{URL: evptr("169.254.169.254/after")})
			return err
		}},
		{"a link deleted", func(t *testing.T, w *world, id uuid.UUID) error {
			l := links(w, id)
			uc := app.NewDeleteIssueLink(app.DeleteIssueLinkDeps{Locks: w.locks, Links: l, Clock: clockAt{at: clockNow, log: w.log},
				Events: &recordedEvents{t: t}})
			return uc.Execute(as(alice), l.current.ID)
		}},
		{"a comment", func(_ *testing.T, w *world, id uuid.UUID) error {
			uc := app.NewCreateIssueComment(app.CreateIssueCommentDeps{Events: discardEvents{}, Locks: w.locks,
				Comments: &fakeComments{fakeRelations: &fakeRelations{fakeIssues: w.issues}}, Members: w.project, Sanitizer: content.NewSanitizer(),
				Clock: clockAt{at: clockNow, log: w.log}})
			_, err := uc.Execute(as(alice), id, domain.NewComment{HTML: "<p>Looks right</p>"})
			return err
		}},
		{"a reaction", func(_ *testing.T, w *world, id uuid.UUID) error {
			uc := app.NewAddIssueReaction(app.AddIssueReactionDeps{Events: discardEvents{}, Locks: w.locks,
				Reactions: &fakeReactions{fakeRelations: &fakeRelations{fakeIssues: w.issues}, rows: map[app.ReactionKey]domain.IssueReaction{}},
				Clock:     clockAt{at: clockNow, log: w.log}})
			_, err := uc.Execute(as(alice), id, "10084-65039")
			return err
		}},
		{"a subscription", func(_ *testing.T, w *world, id uuid.UUID) error {
			uc := app.NewSubscribeIssue(app.SubscriptionDeps{Locks: w.locks, Subscriptions: &fakeSubscriptions{log: w.log, subscribed: map[uuid.UUID]bool{}},
				Clock: clockAt{at: clockNow, log: w.log}})
			return uc.Execute(as(alice), id)
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, w, id := newUpdate()
			intoIntake(w, id, domain.IntakeFacts{Status: domain.IntakePending})
			if err := tt.run(t, w, id); err != nil {
				t.Errorf("Execute() = %v, want the write", err)
			}
		})
	}
}
