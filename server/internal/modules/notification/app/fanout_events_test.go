package app_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/notification/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type enqueueFanout func(context.Context, shared.Event) error

func (f enqueueFanout) Enqueue(ctx context.Context, e shared.Event) error { return f(ctx, e) }

// html is a description's HTML, which the job's arguments never keep.
const html = `<p>hi <mention-component entity_identifier="x"></mention-component></p>`

// The subscriber reads the events the module notifies of. An event that notifies anyone but its actor of anything
// enqueues one fan-out, in the publisher's context, its transaction (M7 P1b spec 2.9): the event itself, every change
// at its position, the HTML of a description and a comment blanked (R4). nerve's own write's mention of an account
// notifies him.
func TestNotifyOnIssueEventsEnqueuesTheEventWithoutItsHTML(t *testing.T) {
	described := renamed()
	described.Changes[1] = shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}
	comment := commented()
	comment.Changes[0] = shared.Change{Field: "comment_html", New: str("<p>look</p>")}
	byNerve := fanoutEvent("issue.updated", shared.EntityRef{Kind: "issue", ID: workItem}, nil,
		shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}, shared.Change{Field: "mention", NewID: idOf(alice)})
	for name, e := range map[string]shared.Event{"an update": described, "a comment": comment, "nerve mentioning alice": byNerve} {
		var got []shared.Event
		sub := app.NewNotifyOnIssueEvents(enqueueFanout(func(ctx context.Context, e shared.Event) error {
			if ctx != t.Context() {
				t.Errorf("%s: context replaced", name)
			}
			got = append(got, e)
			return nil
		}))
		if sub.Name != "notify_on_issue_events" || !reflect.DeepEqual(sub.Types, domain.EventTypes()) {
			t.Fatalf("subscriber %s %v; want notify_on_issue_events of the module's events", sub.Name, sub.Types)
		}
		before := shared.Event{ID: e.ID, Type: e.Type, OccurredAt: e.OccurredAt, WorkspaceID: e.WorkspaceID, ProjectID: e.ProjectID,
			ActorID: e.ActorID, Entity: e.Entity, Changes: append([]shared.Change(nil), e.Changes...)}
		if err := sub.Handle(t.Context(), e); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		want := e
		want.Changes = domain.WithoutHTML(e.Changes)
		if !reflect.DeepEqual(got, []shared.Event{want}) {
			t.Errorf("%s: enqueued %+v; want once %+v", name, got, want)
		}
		if !reflect.DeepEqual(e, before) {
			t.Errorf("%s: the published event was changed: %+v", name, e)
		}
		for n, c := range got[0].Changes {
			if (c.Field == "description" || c.Field == "comment_html") && (c.Old != nil || c.New != nil) {
				t.Errorf("%s: change %d keeps its HTML", name, n)
			}
		}
	}
}

// An event that notifies no one but its actor of anything enqueues nothing: a description's change alone, which
// notifies of nothing; a change of a link's title alone; an update whose only notifiable item is its actor's mention
// of himself; an event of a type the module does not notify of.
func TestNotifyOnIssueEventsEnqueuesNothingForNoOne(t *testing.T) {
	issueRef := shared.EntityRef{Kind: "issue", ID: workItem}
	for name, e := range map[string]shared.Event{
		"a description alone": fanoutEvent("issue.updated", issueRef, &alice, shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}),
		"a link's title alone": fanoutEvent("issue.link_updated", shared.EntityRef{Kind: "issue_link", ID: uuid.NewV7()}, &alice,
			shared.Change{Field: "title", Old: str("a"), New: str("b")}, shared.Change{Field: "issue", NewID: idOf(workItem)}),
		"alice mentioning herself": fanoutEvent("issue.updated", issueRef, &alice,
			shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}, shared.Change{Field: "mention", NewID: idOf(alice)}),
		"a deletion": fanoutEvent("issue.deleted", issueRef, &alice),
	} {
		sub := app.NewNotifyOnIssueEvents(enqueueFanout(func(context.Context, shared.Event) error {
			t.Errorf("%s enqueued a fan-out", name)
			return nil
		}))
		if err := sub.Handle(t.Context(), e); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// An event to enqueue that the job would refuse, one that does not name its work item, its id, its moment or a real
// actor, fails the publisher's write, as does the enqueueing's failure.
func TestNotifyOnIssueEventsRefusesAnIncompleteEventAndFailsWithTheEnqueueing(t *testing.T) {
	for name, change := range map[string]func(e *shared.Event){
		"no id":        func(e *shared.Event) { e.ID = uuid.UUID{} },
		"no moment":    func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"zero actor":   func(e *shared.Event) { e.ActorID = &uuid.UUID{} },
		"no work item": func(e *shared.Event) { e.Changes = e.Changes[:2] },
	} {
		e := commented()
		change(&e)
		called := false
		sub := app.NewNotifyOnIssueEvents(enqueueFanout(func(context.Context, shared.Event) error {
			called = true
			return nil
		}))
		if err := sub.Handle(t.Context(), e); err == nil || called {
			t.Errorf("%s: %v, enqueued %t; want a refusal before the enqueueing", name, err, called)
		}
	}
	failure := errors.New("insert failed")
	sub := app.NewNotifyOnIssueEvents(enqueueFanout(func(context.Context, shared.Event) error { return failure }))
	if err := sub.Handle(t.Context(), commented()); !errors.Is(err, failure) {
		t.Fatalf("Handle = %v; want the enqueueing's failure", err)
	}
}
