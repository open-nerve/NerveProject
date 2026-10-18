package app_test

import (
	"context"
	"errors"
	"fmt"
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

// The accounts, drawn in this order, ascending: bob, web's member, then
// alice, the caller, so that the caller is not the least of them; alice
// and carol are web's admins, gus its guest; out is not its member. acme
// is web's workspace. web's states are Backlog, the default,
// Done, completed, and the triage state; its labels Bug and UI.
var (
	bob, alice, carol, gus, out = uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	acme, web                   = uuid.NewV7(), uuid.NewV7()
	backlog                     = app.StateRef{ID: uuid.NewV7(), Group: "backlog", Default: true}
	done                        = app.StateRef{ID: uuid.NewV7(), Group: domain.GroupCompleted}
	triage                      = app.StateRef{ID: uuid.NewV7(), Group: domain.GroupTriage}
	bug, ui                     = uuid.NewV7(), uuid.NewV7()
	clockNow                    = time.Date(2026, 10, 17, 10, 0, 0, 123456789, time.UTC)
)

// world is web as createIssue's ports see it: its states, the greatest
// sort order of Done's work items 85535, its labels, its last number 6,
// its members' roles and display names; an Authorizer that allows each
// call it is not told to refuse.
type world struct {
	log      *callLog
	project  *fakeProject
	issues   *fakeIssues
	auth     *fakeAuthorizer
	profiles fakeProfiles
}

func newCreate() (*app.CreateIssue, *world) {
	log := &callLog{}
	w := &world{log: log}
	w.project = &fakeProject{log: log, id: web, workspace: acme, found: true, locked: &app.LockedProject{WorkspaceID: acme}, last: 6,
		states: []app.StateRef{backlog, done, triage}, labels: []app.LabelRef{{ID: bug, Name: "Bug"}, {ID: ui, Name: "UI"}},
		roles: map[uuid.UUID]shared.Role{alice: shared.RoleAdmin, bob: shared.RoleMember, carol: shared.RoleAdmin, gus: shared.RoleGuest}}
	w.issues = &fakeIssues{log: log, greatest: map[uuid.UUID]float64{done.ID: 85535}}
	w.auth = &fakeAuthorizer{log: log}
	w.profiles = fakeProfiles{log: log, names: map[uuid.UUID]string{alice: "Alice", bob: "bob", carol: "Carol C.", gus: "gus"}}
	uc := app.NewCreateIssue(app.CreateIssueDeps{Locks: app.NewLocks(w.project, w.auth), Issues: w.issues, Sequences: w.project,
		States: w.project, Labels: w.project, Members: w.project, Profiles: w.profiles, Tx: fakeTx{log: log}, Clock: clockAt{at: clockNow, log: log}})
	return uc, w
}

// as is a context carrying the caller id.
func as(id uuid.UUID) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: id})
}

// line is a logged call, as callLog writes it.
func line(name string, args ...any) string {
	for _, a := range args {
		name += fmt.Sprint(" ", a)
	}
	return name
}

// created are the calls of a creation by alice in web: id is the work
// item's, state its state's; asked the accounts whose roles are read and
// labelsAsked the labels asked for, none when empty; assignees, labels and
// subscribers the accounts and labels it is created with, in id order;
// rows its activity's rows.
func created(id, state uuid.UUID, asked, labelsAsked, assignees, labels, subscribers []uuid.UUID, rows int) []string {
	calls := []string{
		line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin", line("Lock", web),
		line("Authorize", alice, domain.ActionCreate, acme, web), line("States", web),
	}
	if len(asked) > 0 {
		calls = append(calls, line("Roles", web, asked))
	}
	if len(labelsAsked) > 0 {
		calls = append(calls, line("Labels", web, labelsAsked))
	}
	calls = append(calls, line("Next", web), "Now", line("GreatestSortOrder", web, state), line("InsertIssue", id))
	if len(assignees) > 0 {
		calls = append(calls, line("InsertAssignees", assignees))
	}
	if len(labels) > 0 {
		calls = append(calls, line("InsertLabels", labels))
	}
	calls = append(calls, line("InsertSubscribers", subscribers))
	if len(assignees) > 0 {
		calls = append(calls, line("DisplayNames", assignees))
	}
	return append(calls, line("InsertActivities", rows), line("GetIssue", id))
}

// A creation, in the order of M4 design 3.5: before any lock and outside
// the transaction, the project read and the decision; in it, web's lock,
// the decision again, the state, the assignees' roles and the labels
// checked, the next number, then the clock, the sort order after the
// greatest of the state's, the work item, its assignees and labels in id
// order, its subscribers, the caller and the assignees, in id order, then
// the assignees' display names and its activity: the creation and each
// assignee in id order; the answer is the work item read back. The name is
// stored without its surrounding space; the work item in the completed
// state is completed at the moment, which is also every row's, the
// caller's.
func TestCreateIssueWritesTheWorkItemAndItsRows(t *testing.T) {
	uc, w := newCreate()
	high := domain.PriorityHigh
	got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "  Fix the login ", StateID: &done.ID, Priority: &high,
		AssigneeIDs: []uuid.UUID{carol, bob}, LabelIDs: []uuid.UUID{ui, bug}})
	if err != nil || w.issues.issue == nil {
		t.Fatalf("Execute() = %v, %v; want the work item", got, err)
	}
	id := w.issues.issue.ID
	want := created(id, done.ID, []uuid.UUID{carol, bob}, []uuid.UUID{ui, bug}, []uuid.UUID{bob, carol}, []uuid.UUID{bug, ui},
		[]uuid.UUID{bob, alice, carol}, 3)
	if !slices.Equal(w.log.calls, want) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
	}
	row := app.IssueRow{ID: id, WorkspaceID: acme, ProjectID: web, SequenceID: 7, Name: "Fix the login", StateID: done.ID,
		Priority: domain.PriorityHigh, SortOrder: 95535, CompletedAt: &clockNow, By: alice, At: clockNow}
	if !reflect.DeepEqual(*w.issues.issue, row) {
		t.Errorf("the work item %+v, want %+v", *w.issues.issue, row)
	}
	activity := domain.CreationActivity([]domain.Assignee{{ID: bob, DisplayName: "bob"}, {ID: carol, DisplayName: "Carol C."}})
	if len(w.issues.activities) != len(activity) {
		t.Fatalf("the activity %+v, want %d rows", w.issues.activities, len(activity))
	}
	for i, r := range w.issues.activities {
		wantRow := app.ActivityRow{ID: r.ID, WorkspaceID: acme, ProjectID: web, IssueID: id, Activity: activity[i], ActorID: alice, At: clockNow}
		if !reflect.DeepEqual(r, wantRow) || i > 0 && r.ID.Compare(w.issues.activities[i-1].ID) <= 0 {
			t.Errorf("activity row %d: %+v, want %+v, its id after the row before", i, r, wantRow)
		}
	}
	if answer, _, _ := w.issues.GetIssue(context.Background(), id); !reflect.DeepEqual(got, answer) {
		t.Errorf("Execute() = %+v, want the work item as stored, %+v", got, answer)
	}
}

// What the caller leaves out (M4 design 3.7): the project's default state,
// the priority none, the first sort order of a state without work items,
// no completion; and the assignees: the default assignee when he is an
// active admin or member, assigned, subscribed and in the activity like
// one asked for; no one when he is a guest, has left, or there is none,
// without a problem; the ones asked for, not him, when some are. The
// caller among them is subscribed once.
func TestCreateIssueDefaults(t *testing.T) {
	for _, tt := range []struct {
		name      string
		fallback  *uuid.UUID
		asked     []uuid.UUID
		read      []uuid.UUID // the roles read
		assignees []uuid.UUID
	}{
		{"no default assignee", nil, nil, nil, nil},
		{"the default assignee, a member", &bob, nil, []uuid.UUID{bob}, []uuid.UUID{bob}},
		{"the default assignee, an admin", &carol, nil, []uuid.UUID{carol}, []uuid.UUID{carol}},
		{"the default assignee, a guest", &gus, nil, []uuid.UUID{gus}, nil},
		{"the default assignee left", &out, nil, []uuid.UUID{out}, nil},
		{"assignees asked for", &bob, []uuid.UUID{carol}, []uuid.UUID{carol}, []uuid.UUID{carol}},
		{"the caller among the assignees asked for", nil, []uuid.UUID{alice, bob}, []uuid.UUID{alice, bob}, []uuid.UUID{bob, alice}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			w.project.locked.DefaultAssigneeID = tt.fallback
			got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "Fix", AssigneeIDs: tt.asked})
			if err != nil || w.issues.issue == nil {
				t.Fatalf("Execute() = %v, %v; want the work item", got, err)
			}
			subscribers := slices.Compact(slices.SortedFunc(slices.Values(append([]uuid.UUID{alice}, tt.assignees...)), uuid.UUID.Compare))
			if want := created(w.issues.issue.ID, backlog.ID, tt.read, nil, tt.assignees, nil, subscribers, 1+len(tt.assignees)); !slices.Equal(
				w.log.calls, want) {
				t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, want)
			}
			r := w.issues.issue
			if r.StateID != backlog.ID || r.Priority != domain.PriorityNone || r.SortOrder != 65535 || r.CompletedAt != nil ||
				!slices.Equal(got.AssigneeIDs, tt.assignees) {
				t.Errorf("the work item %+v, answered %+v; want in Backlog, none, at 65535, not completed, assigned to %v", r, got, tt.assignees)
			}
		})
	}
}

// refused checks a refusal: err is want, the calls are those of a creation
// up to the last one given, and nothing is written, no number taken.
func refused(t *testing.T, w *world, err, want error, calls []string) {
	t.Helper()
	var se, wantSE *shared.Error
	if !errors.As(err, &se) || !errors.As(want, &wantSE) || se.Code != wantSE.Code || !reflect.DeepEqual(se.Fields, wantSE.Fields) {
		t.Errorf("Execute() = %v, want %v", err, want)
	}
	if !slices.Equal(w.log.calls, calls) {
		t.Errorf("calls:\n%q\nwant\n%q", w.log.calls, calls)
	}
	if w.issues.issue != nil || w.project.last != 6 {
		t.Errorf("after the refusal: the work item %+v, the last number %d; want none and 6", w.issues.issue, w.project.last)
	}
}

// Every refusal the decision makes is answered before any lock and outside
// any transaction (M4 design 3.5): a project not there, or that the
// caller does not see, is project.not_found, a role the rule does not
// allow is forbidden, and nothing is locked. Values that break the rules
// are refused before even the project is read.
func TestCreateIssueRefusesBeforeAnyLock(t *testing.T) {
	precheck := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx")}
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		in    domain.NewIssue
		want  error
		calls []string
	}{
		{"a blank name", func(*world) {}, domain.NewIssue{Name: " "}, shared.Invalid(shared.FieldError{Field: "name", Code: "too_short",
			Message: "must not be empty"}), nil},
		{"no project", func(w *world) { w.project.found = false }, domain.NewIssue{Name: "a"}, domain.ErrProjectNotFound, precheck[:1]},
		{"a project not seen", func(w *world) { w.auth.answers = []error{shared.ErrNotVisible} }, domain.NewIssue{Name: "a"},
			domain.ErrProjectNotFound, precheck},
		{"a role not allowed", func(w *world) { w.auth.answers = []error{shared.Forbidden()} }, domain.NewIssue{Name: "a"}, shared.Forbidden(),
			precheck},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			tt.set(w)
			_, err := uc.Execute(as(alice), web, tt.in)
			refused(t, w, err, tt.want, tt.calls)
		})
	}
}

// Under the lock the decision is made again, and is final: a project
// deleted while the lock waited, or no longer seen, is project.not_found,
// a role no longer allowed forbidden; an archived project refuses the
// creation (409 project.archived). Then the dependent rows: a state not
// the project's, or its triage state; an assignee who is its guest or not
// its member; a label not its: every problem in one 422, each at its
// place, and no number taken.
func TestCreateIssueRefusesUnderTheLock(t *testing.T) {
	locked := []string{line("Find", web, "outside tx"), line("Authorize", alice, domain.ActionCreate, acme, web, "outside tx"), "Begin",
		line("Lock", web)}
	decided := append(slices.Clone(locked), line("Authorize", alice, domain.ActionCreate, acme, web))
	other := uuid.NewV7() // after bug: the labels {other, bug} are asked out of id order
	notAllowed := func(field, message string) shared.FieldError {
		return shared.FieldError{Field: field, Code: "not_allowed", Message: message}
	}
	unassignable := "must be an active member of the project who is not its guest"
	for _, tt := range []struct {
		name  string
		set   func(w *world)
		in    domain.NewIssue
		want  error
		calls []string
	}{
		{"deleted while the lock waited", func(w *world) { w.project.locked = nil }, domain.NewIssue{Name: "a"}, domain.ErrProjectNotFound, locked},
		{"no longer seen", func(w *world) { w.auth.answers = []error{nil, shared.ErrNotVisible} }, domain.NewIssue{Name: "a"},
			domain.ErrProjectNotFound, decided},
		{"no longer allowed", func(w *world) { w.auth.answers = []error{nil, shared.Forbidden()} }, domain.NewIssue{Name: "a"}, shared.Forbidden(),
			decided},
		{"archived", func(w *world) { w.project.locked.Archived = true }, domain.NewIssue{Name: "a"}, domain.ErrProjectArchived, decided},
		{"the triage state, a guest, no member, a label not the project's", func(*world) {},
			domain.NewIssue{Name: "a", StateID: &triage.ID, AssigneeIDs: []uuid.UUID{gus, bob, out}, LabelIDs: []uuid.UUID{other, bug}},
			shared.Invalid(notAllowed("state_id", "must be a state of the project that is not its triage state"),
				notAllowed("assignee_ids[0]", unassignable), notAllowed("assignee_ids[2]", unassignable),
				notAllowed("label_ids[0]", "must be a label of the project")),
			append(slices.Clone(decided), line("States", web), line("Roles", web, []uuid.UUID{gus, bob, out}),
				line("Labels", web, []uuid.UUID{other, bug}))},
		{"another project's state", func(*world) {}, domain.NewIssue{Name: "a", StateID: &other},
			shared.Invalid(notAllowed("state_id", "must be a state of the project that is not its triage state")),
			append(slices.Clone(decided), line("States", web))},
	} {
		t.Run(tt.name, func(t *testing.T) {
			uc, w := newCreate()
			tt.set(w)
			_, err := uc.Execute(as(alice), web, tt.in)
			refused(t, w, err, tt.want, tt.calls)
		})
	}
}

// A port that fails ends the creation with its failure, as itself, and no
// call after it runs: the transaction rolls back whatever was written. What
// cannot be is an error too: an assignee without a display name, before the
// activity; a project without a default state, before its number is taken;
// a work item not there to read back after its write.
func TestCreateIssueStopsAtAFailingPort(t *testing.T) {
	for _, port := range []string{"Find", "Authorize", "Lock", "States", "Roles", "Labels", "Next", "GreatestSortOrder", "InsertIssue",
		"InsertAssignees", "InsertLabels", "InsertSubscribers", "DisplayNames", "InsertActivities", "GetIssue"} {
		t.Run(port, func(t *testing.T) {
			uc, w := newCreate()
			w.log.fail = port
			_, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", AssigneeIDs: []uuid.UUID{bob}, LabelIDs: []uuid.UUID{bug}})
			last := slices.IndexFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, port+" ") })
			if !errors.Is(err, errDisk) || last != len(w.log.calls)-1 {
				t.Errorf("Execute() = %v after the calls\n%q\nwant %v, %s the last call", err, w.log.calls, errDisk, port)
			}
		})
	}
	uc, w := newCreate()
	delete(w.profiles.names, bob)
	if _, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a", AssigneeIDs: []uuid.UUID{bob}}); err == nil ||
		slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, "InsertActivities") }) {
		t.Errorf("Execute() without bob's display name = %v after %q; want an error before the activity", err, w.log.calls)
	}
	uc, w = newCreate()
	w.project.states = []app.StateRef{done, triage}
	if _, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a"}); err == nil ||
		slices.ContainsFunc(w.log.calls, func(c string) bool { return strings.HasPrefix(c, "Next") }) {
		t.Errorf("Execute() in a project without a default state = %v after %q; want an error before the number", err, w.log.calls)
	}
	uc, w = newCreate()
	w.issues.lost = true
	if got, err := uc.Execute(as(alice), web, domain.NewIssue{Name: "a"}); err == nil {
		t.Errorf("Execute() whose work item is not there after its write = %+v; want an error", got)
	}
}
