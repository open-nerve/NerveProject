package app_test

import (
	"bytes"
	"context"
	"errors"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// commentedCalls are the calls of commented's fan-out, in their order: the work item's row FOR SHARE first; then only
// reads that take no lock, of the work item, its project's names and the recheck, and the one insert (M7 P1b spec
// 2.6, 2.9). No call after the lock takes a project's or a workspace's lock: the ports have none to take.
func commentedCalls() []string {
	return []string{callLine("ShareIssue", workItem), callLine("NotificationIssue", workItem), callLine("SubscriberIDs", workItem),
		callLine("State", rowProject, doing), callLine("Identifier", rowWorkspace, rowProject),
		callLine("DecideMany", shared.Action("issue.read"), rowWorkspace, rowProject, []uuid.UUID{alice, carol}),
		callLine("CommentText", comment), callLine("InsertNotifications", 2)}
}

// The fan-out locks the work item's row first, and makes every call in its one transaction, in commentedCalls' order;
// the comment's notifications keep its excerpt, the creator's is created's and the mention's mentioned's.
func TestFanoutLocksTheWorkItemFirstThenOnlyReads(t *testing.T) {
	f := newFanoutPorts()
	f.columns.CreatedByID = &alice
	f.subscribers = []uuid.UUID{alice, bob}
	f.decisions = map[uuid.UUID]domain.Decision{alice: {Allowed: true}, carol: {Allowed: true}}
	f.text = str("Look at this")
	if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), commented()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(f.calls, commentedCalls()) {
		t.Fatalf("calls\n%s\nwant\n%s", strings.Join(f.calls, "\n"), strings.Join(commentedCalls(), "\n"))
	}
	rows, bad := sentRows(f.inserted[0])
	excerpt := domain.ChangeSnapshot{Field: "comment", Verb: domain.VerbCreated, CommentID: &comment, Excerpt: str("Look at this")}
	mention := domain.ChangeSnapshot{Field: "comment", Verb: domain.VerbMentioned, CommentID: &comment, Excerpt: str("Look at this")}
	if want := []sent{{carol, domain.SenderMentioned, 1, mention}, {alice, domain.SenderCreated, 0, excerpt}}; len(bad) > 0 || !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows %+v (bad %v); want %+v", rows, bad, want)
	}
}

// A work item deleted before its row is locked, or while the lock waits, is notified of to no one: the fan-out stops
// at the lock, writes nothing and completes (spec 2.9).
func TestFanoutOfADeletedWorkItemWritesNothing(t *testing.T) {
	f := newFanoutPorts()
	f.exists = false
	if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), commented()); err != nil || !slices.Equal(f.calls, commentedCalls()[:1]) || len(f.inserted) != 0 {
		t.Fatalf("Execute = %v, calls %v, inserted %v; want the lock alone and nothing written", err, f.calls, f.inserted)
	}
}

// The recipients are the domain's (spec 2.4): each mention its account's, the actor's not; each subscriber, but the
// actor and the mentioned, of each change, as the creator, an assignee or a subscriber. The recheck asks issue.read on
// the locked row's workspace and project for each once, in id order, and keeps those it allows from whom the issue
// module's guest rule, the port, does not hide the work item, asked with the creator its columns name and each one's
// OwnOnly: one refused and one it does not answer are dropped, and so are the two the rule hides; the creator who sees
// only his own keeps his, the rule not hiding it. Each row is the event's, by its actor, at its moment, of the work
// item and its creator.
func TestFanoutSendsToTheRecipientsTheRecheckAllows(t *testing.T) {
	f := newFanoutPorts()
	f.subscribers = []uuid.UUID{alice, bob, carol, dave, eve, frank, gina, hank}
	f.decisions = map[uuid.UUID]domain.Decision{bob: {Allowed: true}, carol: {Allowed: true, OwnOnly: true}, dave: {Allowed: true},
		eve: {Allowed: true}, frank: {}, gina: {Allowed: true, OwnOnly: true}, gus: {Allowed: true, OwnOnly: true}}
	f.hidden = []uuid.UUID{gina, gus}
	e := renamed()
	if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), e); err != nil {
		t.Fatal(err)
	}
	var questions []string
	for account, d := range f.decisions {
		if d.Allowed {
			questions = append(questions, callLine("HiddenFrom", carol, account, d.OwnOnly))
		}
	}
	slices.Sort(questions)
	if asked := slices.Compact(slices.Sorted(slices.Values(f.asked))); !slices.Equal(asked, questions) {
		t.Fatalf("the guest rule asked\n%s\nwant\n%s", strings.Join(asked, "\n"), strings.Join(questions, "\n"))
	}
	asked := callLine("DecideMany", shared.Action("issue.read"), rowWorkspace, rowProject, []uuid.UUID{bob, carol, dave, eve, frank, gina, gus, hank})
	if !slices.Contains(f.calls, asked) || slices.Contains(f.calls, callLine("CommentText", comment)) || len(f.inserted) != 1 {
		t.Fatalf("calls %v; want the recheck of every recipient but the actor, no comment read and one insert", f.calls)
	}
	n := f.inserted[0]
	if n.WorkspaceID != rowWorkspace || n.ProjectID != rowProject || n.IssueID != workItem || n.EventID != e.ID || n.TriggeredByID == nil ||
		*n.TriggeredByID != alice || n.IssueCreatedByID == nil || *n.IssueCreatedByID != carol || !n.At.Equal(moment) {
		t.Fatalf("notifications %+v; want the event's, by alice at its moment, of the work item carol created", n)
	}
	name := domain.ChangeSnapshot{Field: "name", Verb: domain.VerbUpdated, OldValue: str("Fix it"), NewValue: str("Fix it now")}
	state := domain.ChangeSnapshot{Field: "state", Verb: domain.VerbUpdated, OldValue: str("Todo"), NewValue: str("Doing"), OldID: &todo, NewID: &doing}
	want := []sent{{eve, domain.SenderMentioned, 3, domain.ChangeSnapshot{Field: "description", Verb: domain.VerbMentioned}},
		{bob, domain.SenderSubscribed, 0, name}, {bob, domain.SenderSubscribed, 2, state},
		{carol, domain.SenderCreated, 0, name}, {carol, domain.SenderCreated, 2, state},
		{dave, domain.SenderAssigned, 0, name}, {dave, domain.SenderAssigned, 2, state}}
	if rows, bad := sentRows(n); len(bad) > 0 || !reflect.DeepEqual(rows, want) {
		t.Fatalf("rows %+v (bad %v); want %+v", rows, bad, want)
	}
	nerve := renamed()
	nerve.ActorID = nil
	f.decisions[alice] = domain.Decision{Allowed: true}
	f.calls, f.inserted = nil, nil
	if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), nerve); err != nil || f.inserted[0].TriggeredByID != nil || len(f.inserted[0].Rows) != 9 {
		t.Fatalf("nerve's own write: %v, %+v; want no sender and alice notified too", err, f.inserted)
	}
}

// The rows' workspace and project are the locked row's, which the reads and the recheck take too; an event that names
// others is logged as a warning (spec 2.9). One that names the row's is not.
func TestFanoutTakesTheWorkspaceAndProjectOfTheLockedRow(t *testing.T) {
	f := newFanoutPorts()
	f.subscribers = []uuid.UUID{bob}
	f.decisions = map[uuid.UUID]domain.Decision{bob: {Allowed: true}}
	var logs bytes.Buffer
	if err := fanout(f, &logs).Execute(t.Context(), renamed()); err != nil || logs.Len() != 0 {
		t.Fatalf("Execute = %v, logs %q; want no warning", err, logs.String())
	}
	e := renamed()
	e.WorkspaceID, e.ProjectID = uuid.NewV7(), uuid.NewV7()
	f.calls, f.inserted = nil, nil
	if err := fanout(f, &logs).Execute(t.Context(), e); err != nil {
		t.Fatal(err)
	}
	n := f.inserted[0]
	asked := callLine("DecideMany", shared.Action("issue.read"), rowWorkspace, rowProject, []uuid.UUID{bob, eve, gus})
	if n.WorkspaceID != rowWorkspace || n.ProjectID != rowProject || !slices.Contains(f.calls, asked) ||
		!slices.Contains(f.calls, callLine("Identifier", rowWorkspace, rowProject)) || !slices.Contains(f.calls, callLine("State", rowProject, doing)) {
		t.Fatalf("notifications of %s/%s, calls %v; want the locked row's", n.WorkspaceID, n.ProjectID, f.calls)
	}
	for _, s := range []string{"level=WARN", e.ID.String(), e.WorkspaceID.String(), e.ProjectID.String(), rowWorkspace.String(), rowProject.String()} {
		if !strings.Contains(logs.String(), s) {
			t.Errorf("logs %q; want %s", logs.String(), s)
		}
	}
}

// blocking is alice's relation event of kind, of a relation of the workspace across two projects, stored as the
// other end's blocked_by and so of the other end's project, which the event names, asked on the work item as
// blocking (R11): its type (0), its stored ends (1, 2) and the end it was requested on (3), on the New side for an
// addition, the Old for a removal.
func blocking(kind string, other, otherProject uuid.UUID) shared.Event {
	changes := []shared.Change{{Field: "relation_type", New: str("blocked_by")}, {Field: "issue", NewID: idOf(other)},
		{Field: "related_issue", NewID: idOf(workItem)}, {Field: "requested_on", New: str("blocking"), NewID: idOf(workItem)}}
	if kind == "issue.relation_removed" {
		for n, c := range changes {
			changes[n] = shared.Change{Field: c.Field, Old: c.New, OldID: c.NewID}
		}
	}
	e := fanoutEvent(kind, shared.EntityRef{Kind: "issue_relation", ID: uuid.NewV7()}, &alice, changes...)
	e.ProjectID = otherProject
	return e
}

// A relation's event names its stored row's project, which is not the work item's when it was asked on the other end
// of a relation across projects (R11): one that names another project than the locked row's is not logged, its rows
// are the locked row's workspace's and project's; one that names another workspace is logged. Any other event that
// names another project alone is logged.
func TestFanoutComparesARelationsWorkspaceAlone(t *testing.T) {
	other, otherProject := uuid.NewV7(), uuid.NewV7()
	for kind, verb := range map[string]domain.Verb{"issue.relation_added": domain.VerbCreated, "issue.relation_removed": domain.VerbDeleted} {
		f := newFanoutPorts()
		f.subscribers = []uuid.UUID{bob}
		f.decisions = map[uuid.UUID]domain.Decision{bob: {Allowed: true}}
		var logs bytes.Buffer
		e := blocking(kind, other, otherProject)
		if err := fanout(f, &logs).Execute(t.Context(), e); err != nil || logs.Len() != 0 {
			t.Fatalf("%s of another project: %v, logs %q; want no warning", kind, err, logs.String())
		}
		change := domain.ChangeSnapshot{Field: "blocking", Verb: verb}
		if verb == domain.VerbDeleted {
			change.OldID = &other
		} else {
			change.NewID = &other
		}
		n := f.inserted[0]
		if rows, bad := sentRows(n); n.WorkspaceID != rowWorkspace || n.ProjectID != rowProject || len(bad) > 0 ||
			!reflect.DeepEqual(rows, []sent{{bob, domain.SenderSubscribed, 3, change}}) {
			t.Fatalf("%s: notifications of %s/%s, rows %+v (bad %v); want bob's of the locked row's", kind, n.WorkspaceID, n.ProjectID, rows, bad)
		}
		e.WorkspaceID = uuid.NewV7()
		if err := fanout(f, &logs).Execute(t.Context(), e); err != nil || !strings.Contains(logs.String(), "level=WARN") ||
			!strings.Contains(logs.String(), e.WorkspaceID.String()) {
			t.Fatalf("%s of another workspace: %v, logs %q; want a warning", kind, err, logs.String())
		}
	}
	f := newFanoutPorts()
	f.subscribers = []uuid.UUID{bob}
	f.decisions = map[uuid.UUID]domain.Decision{bob: {Allowed: true}}
	var logs bytes.Buffer
	e := renamed()
	e.ProjectID = otherProject
	if err := fanout(f, &logs).Execute(t.Context(), e); err != nil || !strings.Contains(logs.String(), "level=WARN") ||
		!strings.Contains(logs.String(), otherProject.String()) || f.inserted[0].ProjectID != rowProject {
		t.Fatalf("an update of another project: %v, logs %q; want a warning and the locked row's project", err, logs.String())
	}
}

// A change's position is notifications.change_seq, a smallint: the last position it holds is sent; one past it fails
// the fan-out, which writes nothing.
func TestFanoutNumbersTheChangesAsSmallints(t *testing.T) {
	for _, tt := range []struct {
		before int
		fails  bool
	}{{32767, false}, {32768, true}} {
		f := newFanoutPorts()
		f.subscribers = []uuid.UUID{bob}
		f.decisions = map[uuid.UUID]domain.Decision{bob: {Allowed: true}}
		e := fanoutEvent("issue.updated", shared.EntityRef{Kind: "issue", ID: workItem}, &alice)
		e.Changes = append(make([]shared.Change, tt.before), shared.Change{Field: "name", Old: str("Fix it"), New: str("Fix it now")})
		for i := range tt.before {
			e.Changes[i] = shared.Change{Field: "sort_order"}
		}
		err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), e)
		if tt.fails {
			if err == nil || !strings.Contains(err.Error(), "32768") || len(f.inserted) != 0 {
				t.Fatalf("position %d: %v, %v; want a failure naming it and nothing written", tt.before, err, f.inserted)
			}
			continue
		}
		if err != nil || len(f.inserted) != 1 || f.inserted[0].Rows[0].ChangeSeq != 32767 {
			t.Fatalf("position %d: %v, %+v; want it sent", tt.before, err, f.inserted)
		}
	}
}

// A project deleted since the event, whose state or identifier is no longer read, is notified of to no one: its work
// items are read by no one (spec 2.7), and the deletion of its notifications waits for them (2.8). Nor is a write
// whose only recipient is its actor, mentioning himself, which needs no recheck; nor one the recheck leaves to no one.
func TestFanoutWritesNothingNoOneIsSentOrReads(t *testing.T) {
	read := commentedCalls()
	unmentioned := commented()
	unmentioned.Changes[1] = shared.Change{Field: "mention", NewID: idOf(bob)}
	for _, tt := range []struct {
		name  string
		e     shared.Event
		setup func(f *fanoutPorts)
		calls []string
	}{
		{"state deleted", commented(), func(f *fanoutPorts) { f.state = nil }, read[:4]},
		{"project deleted", commented(), func(f *fanoutPorts) { f.identifier = nil }, read[:5]},
		{"the actor alone", unmentioned, func(f *fanoutPorts) { f.subscribers = []uuid.UUID{bob} }, read[:3]},
		{"no one allowed", commented(), func(f *fanoutPorts) { f.decisions = nil }, read[:6]},
	} {
		t.Run(tt.name, func(t *testing.T) {
			f := newFanoutPorts()
			f.subscribers = []uuid.UUID{alice, bob}
			f.decisions = map[uuid.UUID]domain.Decision{alice: {Allowed: true}, carol: {Allowed: true}}
			tt.setup(f)
			if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), tt.e); err != nil || !slices.Equal(f.calls, tt.calls) || len(f.inserted) != 0 {
				t.Fatalf("Execute = %v, calls %v, inserted %v; want %v and nothing written", err, f.calls, f.inserted, tt.calls)
			}
		})
	}
}

// A comment deleted since the event leaves its notifications without an excerpt.
func TestFanoutOfACommentDeletedSinceKeepsNoExcerpt(t *testing.T) {
	f := newFanoutPorts()
	f.subscribers = []uuid.UUID{alice, bob}
	f.decisions = map[uuid.UUID]domain.Decision{alice: {Allowed: true}, carol: {Allowed: true}}
	if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), commented()); err != nil || len(f.inserted) != 1 {
		t.Fatalf("Execute = %v, %v", err, f.inserted)
	}
	for _, r := range f.inserted[0].Rows {
		if r.Data.Change.CommentID == nil || *r.Data.Change.CommentID != comment || r.Data.Change.Excerpt != nil {
			t.Fatalf("row %+v; want the comment without an excerpt", r.Data.Change)
		}
	}
}

// Each port's failure fails the fan-out at it, with nothing written after it; a work item the lock answers but its read
// does not is a failure too.
func TestFanoutFailsAtAPortsFailure(t *testing.T) {
	read := commentedCalls()
	for n, line := range read {
		f := newFanoutPorts()
		f.subscribers = []uuid.UUID{alice, bob}
		f.decisions = map[uuid.UUID]domain.Decision{alice: {Allowed: true}, carol: {Allowed: true}}
		f.text = str("Look at this")
		f.fail = strings.Fields(line)[0]
		if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), commented()); !errors.Is(err, f.failure) || !slices.Equal(f.calls, read[:n+1]) || len(f.inserted) != 0 {
			t.Errorf("%s failing: %v, calls %v; want its failure and no call after it", f.fail, err, f.calls)
		}
	}
	f := newFanoutPorts()
	ports := &unreadIssue{f}
	err := app.NewFanout(app.FanoutDeps{Lock: f, Issues: ports, Projects: f, Readers: f, GuestRule: f, Rows: f, Tx: fanoutTx{}}).Execute(t.Context(), commented())
	if err == nil || !strings.Contains(err.Error(), workItem.String()) || len(f.inserted) != 0 {
		t.Fatalf("a locked work item not read: %v; want a failure naming it", err)
	}
}

// unreadIssue are fanoutPorts' reads of a work item that its read does not find.
type unreadIssue struct{ *fanoutPorts }

func (u *unreadIssue) NotificationIssue(ctx context.Context, id uuid.UUID) (app.IssueColumns, bool, error) {
	return app.IssueColumns{}, false, u.call(ctx, "NotificationIssue", id)
}

// An event that does not name its work item, its id, its moment or a real actor is refused before anything is read.
func TestFanoutRefusesAnIncompleteEvent(t *testing.T) {
	for name, change := range map[string]func(e *shared.Event){
		"no id":        func(e *shared.Event) { e.ID = uuid.UUID{} },
		"no moment":    func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"zero actor":   func(e *shared.Event) { e.ActorID = &uuid.UUID{} },
		"no work item": func(e *shared.Event) { e.Changes = e.Changes[:2] },
		"another type": func(e *shared.Event) { e.Type = "issue.deleted" },
	} {
		f := newFanoutPorts()
		e := commented()
		change(&e)
		if err := fanout(f, &bytes.Buffer{}).Execute(t.Context(), e); err == nil || len(f.calls) != 0 {
			t.Errorf("%s: %v, calls %v; want a refusal before any call", name, err, f.calls)
		}
	}
}
