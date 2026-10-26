package app_test

import (
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// earlier is the moment web's work item 7 was made, an hour before the
// clock's; dave, drawn after every account of newCreate's world, is the
// member newUpdate adds to web; docs, drawn after its labels, is Docs, the
// label TestUpdateIssueWritesTheChangesAndTheirRows adds to web and to the
// work item.
var (
	earlier = clockNow.Add(-time.Hour)
	dave    = uuid.NewV7()
	docs    = uuid.NewV7()
)

// newUpdate is updateIssue over web's world (newCreate's), dave a member
// of web too, with web's work item 7 stored in it, made by carol at
// earlier: Fix, in Backlog, high, not completed, assigned to bob and to
// out, who is no member of web since, and labelled Bug.
func newUpdate() (*app.UpdateIssue, *world, uuid.UUID) {
	_, w := newCreate()
	id := uuid.NewV7()
	w.issues.issue = &app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID,
		Priority: domain.PriorityHigh, SortOrder: 65535, By: carol, At: earlier}
	w.issues.assignees, w.issues.labels = []uuid.UUID{bob, out}, []uuid.UUID{bug}
	w.profiles.names[out] = "out"
	w.project.roles[dave], w.profiles.names[dave] = shared.RoleMember, "dave"
	return updateOver(w), w, id
}

// updateOver is updateIssue over w's ports.
func updateOver(w *world) *app.UpdateIssue {
	return app.NewUpdateIssue(app.UpdateIssueDeps{Sanitizer: descriptionSanitizer{}, Locks: w.locks, Issues: w.issues, States: w.project, Labels: w.project, Members: w.project,
		Profiles: w.profiles, Clock: clockAt{at: clockNow, log: w.log}})
}

// name is a patch's name.
func name(s string) *string {
	return &s
}

// set is a patch's set of ids.
func set(ids ...uuid.UUID) *[]uuid.UUID {
	return &ids
}

// locked are the calls of an update of id by alice up to the decision under
// the lock (lockedAs).
func locked(id uuid.UUID) []string {
	return lockedAs(domain.ActionUpdate, id)
}

// lockedAs are the calls of a write of action on the work item id by alice
// up to the decision under the lock: the work item read and the decision,
// before any lock and outside the transaction; in it, web's lock, the work
// item read again and the decision again.
func lockedAs(action shared.Action, id uuid.UUID) []string {
	return []string{line("GetIssue", id, "outside tx"), line("Authorize", alice, action, acme, web, "outside tx"), "Begin", line("Lock", web),
		line("GetIssue", id), line("Authorize", alice, action, acme, web)}
}

// An update, in the order of M4 design 3.5 and 6.7: Locks' calls (locked);
// then the new state, the roles of the accounts added and the labels added
// and removed, checked or named; the clock; the work item's fields; the
// assignees removed and added, the labels the same, each in id order; the
// accounts added subscribed; the display names of the accounts added and
// removed; the activity: the name, the priority, the state by its names,
// each label added and removed, each account added and removed
// (domain.UpdateActivity), all of it alice's at the moment; the answer is
// the work item read back, its sub-items counted (Locks.seen). The name is
// stored without its surrounding space; in Done the work item is completed
// at the moment. out, who is no member of web, stays assigned: only the
// accounts added, carol and dave, are checked, and each of them, not only
// the first, is assigned, subscribed and in the activity. Docs, which the
// work item has and keeps, is neither read nor written again (M4 design
// 3.7): Bug's row alone is removed and UI's alone added.
func TestUpdateIssueWritesTheChangesAndTheirRows(t *testing.T) {
	uc, w, id := newUpdate()
	w.project.labels = append(w.project.labels, app.LabelRef{ID: docs, Name: "Docs"})
	w.issues.labels = append(w.issues.labels, docs)
	low := domain.PriorityLow
	got, err := uc.Execute(as(alice), id, domain.IssuePatch{Name: name(" Mend "), StateID: &done.ID, Priority: &low,
		AssigneeIDs: set(out, dave, carol), LabelIDs: set(docs, ui)})
	want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Mend", StateID: done.ID, Priority: low,
		SortOrder: 65535, CompletedAt: &clockNow, AssigneeIDs: []uuid.UUID{carol, out, dave}, LabelIDs: []uuid.UUID{ui, docs}, CreatedByID: &carol,
		UpdatedByID: &alice, CreatedAt: earlier, UpdatedAt: clockNow}
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	calls := append(locked(id), line("States", web), line("Roles", web, []uuid.UUID{carol, dave}), line("Labels", web, []uuid.UUID{bug, ui}), "Now",
		line("UpdateIssue", id), line("RemoveAssignees", []uuid.UUID{bob}), line("InsertAssignees", []uuid.UUID{carol, dave}),
		line("RemoveLabels", []uuid.UUID{bug}), line("InsertLabels", []uuid.UUID{ui}), line("InsertSubscribers", []uuid.UUID{carol, dave}),
		line("DisplayNames", []uuid.UUID{carol, dave, bob}), line("InsertActivities", 8), line("GetIssue", id), counted(id))
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if fields := (app.IssueFields{SortOrder: 65535, ID: id, Name: "Mend", StateID: done.ID, Priority: low, CompletedAt: &clockNow, By: alice,
		At: clockNow}); w.issues.changed == nil || !reflect.DeepEqual(*w.issues.changed, fields) {
		t.Errorf("the fields written %+v, want %+v", w.issues.changed, fields)
	}
	wantActivity(t, w, id, domain.UpdateActivity(domain.Update{
		Name: &domain.Changed[string]{Old: "Fix", New: "Mend"}, Priority: &domain.Changed[domain.Priority]{Old: domain.PriorityHigh, New: low},
		State:       &domain.Changed[domain.Named]{Old: domain.Named{ID: backlog.ID, Name: "Backlog"}, New: domain.Named{ID: done.ID, Name: "Done"}},
		LabelsAdded: []domain.Named{{ID: ui, Name: "UI"}}, LabelsRemoved: []domain.Named{{ID: bug, Name: "Bug"}},
		AssigneesAdded:   []domain.Assignee{{ID: carol, DisplayName: "Carol C."}, {ID: dave, DisplayName: "dave"}},
		AssigneesRemoved: []domain.Assignee{{ID: bob, DisplayName: "bob"}},
	}))
}

// A patch that changes nothing, given empty or with each field as it is:
// its sets in another order, its start and target dates given as the work
// item has them, none. The precheck, the lock and the decisions run as for
// any update; nothing is written, not even the clock read, and the answer
// is the work item as read under the lock, its last writer and moment as
// they were, its sub-items counted.
func TestUpdateIssueThatChangesNothingWritesNothing(t *testing.T) {
	high := domain.PriorityHigh
	for what, p := range map[string]domain.IssuePatch{
		"nothing given": {},
		"each field as it is": {Name: name(" Fix"), StateID: &backlog.ID, Priority: &high, AssigneeIDs: set(out, bob),
			LabelIDs: set(bug), SetStartDate: true, SetTargetDate: true},
	} {
		t.Run(what, func(t *testing.T) {
			uc, w, id := newUpdate()
			got, err := uc.Execute(as(alice), id, p)
			want := domain.Issue{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix", StateID: backlog.ID, Priority: high,
				SortOrder: 65535, AssigneeIDs: []uuid.UUID{bob, out}, LabelIDs: []uuid.UUID{bug}, CreatedByID: &carol, UpdatedByID: &carol,
				CreatedAt: earlier, UpdatedAt: earlier}
			if err != nil || !reflect.DeepEqual(got, want) {
				t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
			}
			if calls := append(locked(id), counted(id)); !slices.Equal(w.log.calls, calls) || w.issues.changed != nil {
				t.Errorf("calls:\n%q\nwant\n%q; the fields written %+v, want none", w.log.calls, calls, w.issues.changed)
			}
		})
	}
}

// A state of the completed group completes the work item at the moment,
// also from another state of that group; any other state leaves it not
// completed, a cancelled one too; a change that leaves its state leaves its
// completion as it is (M4 design 3.7, 6.7).
func TestUpdateIssueCompletesTheWorkItemByItsState(t *testing.T) {
	shipped := app.StateRef{ID: uuid.NewV7(), Name: "Shipped", Group: domain.GroupCompleted}
	dropped := app.StateRef{ID: uuid.NewV7(), Name: "Dropped", Group: "cancelled"}
	for _, tt := range []struct {
		name      string
		from      app.StateRef
		p         domain.IssuePatch
		completed *time.Time
	}{
		{"into Done", backlog, domain.IssuePatch{StateID: &done.ID}, &clockNow},
		{"out of Done", done, domain.IssuePatch{StateID: &backlog.ID}, nil},
		{"from Done into Shipped", done, domain.IssuePatch{StateID: &shipped.ID}, &clockNow},
		{"into Dropped, cancelled", backlog, domain.IssuePatch{StateID: &dropped.ID}, nil},
		{"from Done into Dropped", done, domain.IssuePatch{StateID: &dropped.ID}, nil},
		{"in Done, renamed", done, domain.IssuePatch{Name: name("Mend")}, &earlier},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.project.states = append(w.project.states, shipped, dropped)
			w.issues.issue.StateID = tt.from.ID
			if tt.from.Group == domain.GroupCompleted {
				w.issues.issue.CompletedAt = &earlier
			}
			got, err := uc.Execute(as(alice), id, tt.p)
			if err != nil || !reflect.DeepEqual(got.CompletedAt, tt.completed) {
				t.Errorf("Execute() = %+v, %v; want completed at %v", got, err, tt.completed)
			}
		})
	}
}

// unchanged checks an update's refusal (wantRefusal), which writes
// nothing: the work item, its assignees and its labels are as they were.
func unchanged(t *testing.T, w *world, err, want error, calls []string) {
	t.Helper()
	wantRefusal(t, w, err, want, calls)
	if w.issues.changed != nil || !slices.Equal(w.issues.assignees, []uuid.UUID{bob, out}) || !slices.Equal(w.issues.labels, []uuid.UUID{bug}) {
		t.Errorf("after the refusal: the fields written %+v, assignees %v, labels %v; want none, bob and out, Bug", w.issues.changed,
			w.issues.assignees, w.issues.labels)
	}
}

// Every refusal the decision makes is answered before any lock and outside
// any transaction (M4 design 3.5): a work item not there, or whose project
// the caller does not see, is issue.not_found, a role the rule does not
// allow is forbidden, and nothing is locked. Values that break the rules
// are refused before even the work item is read.
func TestUpdateIssueRefusesBeforeAnyLock(t *testing.T) {
	for _, tt := range []struct {
		name  string
		set   func(w *world, id *uuid.UUID)
		p     domain.IssuePatch
		want  error
		calls int // of locked's
	}{
		{"a blank name", func(*world, *uuid.UUID) {}, domain.IssuePatch{Name: name(" ")}, shared.Invalid(shared.FieldError{Field: "name",
			Code: "too_short", Message: "must not be empty"}), 0},
		{"no work item", func(_ *world, id *uuid.UUID) { *id = uuid.NewV7() }, domain.IssuePatch{}, domain.ErrIssueNotFound, 1},
		{"a project not seen", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.IssuePatch{},
			domain.ErrIssueNotFound, 2},
		{"a role not allowed", func(w *world, _ *uuid.UUID) { w.auth.answers = []error{shared.Forbidden()} }, domain.IssuePatch{},
			shared.Forbidden(), 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			tt.set(w, &id)
			_, err := uc.Execute(as(alice), id, tt.p)
			unchanged(t, w, err, tt.want, locked(id)[:tt.calls])
		})
	}
}

// Under the lock the decision is made again, and is final: a project
// deleted while the lock waited, a work item deleted meanwhile or no longer
// in it, or a project no longer seen, is issue.not_found; a role no longer
// allowed forbidden; an archived project refuses the update, one that
// changes nothing too (409 project.archived), its archived work items too;
// then an archived work item refuses it, one that changes nothing too,
// before the state it moves into is read, and before the order of the
// dates it gives is checked (409 issue.archived). Then the dependent rows
// the update adds: a state not the project's, or its triage state; an
// account added who is its guest; a label added that is not its: every
// problem in one 422, each at its place in the patch (the label second in
// label_ids, though first among the labels added). out, kept, is not
// checked, though no member of web.
func TestUpdateIssueRefusesUnderTheLock(t *testing.T) {
	other := uuid.NewV7() // after every id of the world
	notAllowed := func(field, message string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: message}
	}
	stateNotAllowed := notAllowed("state_id", "must be a state of the project that is not its triage state")
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		p     domain.IssuePatch
		want  error
		calls func(id uuid.UUID) []string
	}{
		{"the project deleted while the lock waited", func(w *world) { w.project.locked = nil }, domain.IssuePatch{}, domain.ErrIssueNotFound,
			func(id uuid.UUID) []string { return locked(id)[:4] }},
		{"the work item deleted meanwhile", func(w *world) { w.project.waited = func() { w.issues.issue.ID = uuid.NewV7() } },
			domain.IssuePatch{}, domain.ErrIssueNotFound, func(id uuid.UUID) []string { return locked(id)[:5] }},
		{"the work item in another project", func(w *world) { w.project.waited = func() { w.issues.issue.ProjectID = other } },
			domain.IssuePatch{}, domain.ErrIssueNotFound, func(id uuid.UUID) []string { return locked(id)[:5] }},
		{"no longer seen", func(w *world) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, domain.IssuePatch{}, domain.ErrIssueNotFound,
			locked},
		{"no longer allowed", func(w *world) { w.auth.answers = []error{nil, shared.Forbidden()} }, domain.IssuePatch{}, shared.Forbidden(), locked},
		{"archived", func(w *world) { w.project.locked.Archived = true }, domain.IssuePatch{}, domain.ErrProjectArchived, locked},
		{"archived, the work item archived", func(w *world) { w.project.locked.Archived, w.issues.archivedAt = true, &earlier },
			domain.IssuePatch{}, domain.ErrProjectArchived, locked},
		{"an archived work item", func(w *world) { w.issues.archivedAt = &earlier }, domain.IssuePatch{}, domain.ErrIssueArchived, locked},
		{"an archived work item moved into Done", func(w *world) { w.issues.archivedAt = &earlier }, domain.IssuePatch{StateID: &done.ID},
			domain.ErrIssueArchived, locked},
		{"an archived work item given dates out of order", func(w *world) { w.issues.archivedAt = &earlier },
			domain.IssuePatch{SetStartDate: true, StartDate: october(20), SetTargetDate: true, TargetDate: october(19)}, domain.ErrIssueArchived,
			locked},
		{"the triage state, a guest added, a label not the project's", func(*world) {},
			domain.IssuePatch{StateID: &triage.ID, AssigneeIDs: set(gus, bob, out, carol), LabelIDs: set(bug, other)},
			shared.Invalid(stateNotAllowed, notAllowed("assignee_ids[0]", "must be an active member of the project who is not its guest"),
				notAllowed("label_ids[1]", "must be a label of the project")),
			func(id uuid.UUID) []string {
				return append(locked(id), line("States", web), line("Roles", web, []uuid.UUID{carol, gus}), line("Labels", web, []uuid.UUID{other}))
			}},
		{"another project's state", func(*world) {}, domain.IssuePatch{StateID: &other}, shared.Invalid(stateNotAllowed),
			func(id uuid.UUID) []string { return append(locked(id), line("States", web)) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			tt.set(w)
			_, err := uc.Execute(as(alice), id, tt.p)
			unchanged(t, w, err, tt.want, tt.calls(id))
		})
	}
}

// A port that fails ends the update with its failure, as itself, and no
// call after it runs, the read and the decision under the lock too once
// the precheck's have passed, and the count of its answer's sub-items,
// the last: the transaction rolls back whatever was written. What cannot
// be is an error too, before any write: a work item whose state is not
// its project's; a label it is removed from that is not its project's; an
// account without a display name, before the activity; and a work item
// not there to read back after its write.
func TestUpdateIssueStopsAtAFailingPort(t *testing.T) {
	low := domain.PriorityLow
	p := domain.IssuePatch{Name: name("Mend"), StateID: &done.ID, Priority: &low, AssigneeIDs: set(out, carol), LabelIDs: set(ui)}
	for _, port := range []string{"GetIssue", "Authorize", "Lock", "States", "Roles", "Labels", "UpdateIssue", "RemoveAssignees",
		"InsertAssignees", "RemoveLabels", "InsertLabels", "InsertSubscribers", "DisplayNames", "InsertActivities", "SubIssueCounts"} {
		t.Run(port, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.log.fail = port
			_, err := uc.Execute(as(alice), id, p)
			wantStoppedAt(t, w, err, port)
		})
	}
	for _, tt := range []struct {
		name, port string
		calls      int // of locked's
	}{
		{"the work item read again under the lock", "GetIssue", 5},
		{"the decision under the lock", "Authorize", 6},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			w.project.waited = func() { w.log.fail = tt.port }
			_, err := uc.Execute(as(alice), id, p)
			if calls := locked(id)[:tt.calls]; !errors.Is(err, errDisk) || !slices.Equal(w.log.calls, calls) {
				t.Errorf("Execute() = %v after the calls\n%q\nwant %v after\n%q", err, w.log.calls, errDisk, calls)
			}
		})
	}
	for _, tt := range []struct {
		name   string
		set    func(w *world)
		before string // a call that does not run
	}{
		{"a state not the project's", func(w *world) { w.project.states = []app.StateRef{done, triage} }, "Now"},
		{"a label not the project's", func(w *world) { w.project.labels = []app.LabelRef{{ID: ui, Name: "UI"}} }, "Now"},
		{"no display name", func(w *world) { delete(w.profiles.names, carol) }, "InsertActivities"},
		{"lost after its write", func(w *world) { w.issues.lost = true }, ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w, id := newUpdate()
			tt.set(w)
			if got, err := uc.Execute(as(alice), id, p); err == nil ||
				tt.before != "" && slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, tt.before) }) {
				t.Errorf("Execute() = %+v, %v after %q; want an error before %s", got, err, w.log.calls, tt.before)
			}
		})
	}
}

func TestUpdateIssuePreservesLockedDescription(t *testing.T) {
	for _, text := range []*string{nil, name(""), name("locked text")} {
		uc, w, id := newUpdate()
		w.issues.issue.DescriptionHTML = "<p>precheck</p>"
		w.issues.issue.DescriptionStripped = name("precheck")
		w.project.waited = func() {
			w.issues.issue.DescriptionHTML = "<p>locked</p>"
			w.issues.issue.DescriptionStripped = text
		}
		got, err := uc.Execute(as(alice), id, domain.IssuePatch{Name: name("renamed")})
		if err != nil {
			t.Fatal(err)
		}
		if w.issues.changed == nil || w.issues.changed.DescriptionHTML != "<p>locked</p>" || !reflect.DeepEqual(w.issues.changed.DescriptionStripped, text) {
			t.Fatalf("write did not preserve lock-time description: %+v", w.issues.changed)
		}
		if got.DescriptionHTML != "<p>locked</p>" || !reflect.DeepEqual(got.DescriptionStripped, text) {
			t.Fatalf("read-back lost description: %+v", got)
		}
	}
}
