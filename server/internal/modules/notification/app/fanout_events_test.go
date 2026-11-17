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

type enqueueFanouts func(context.Context, []shared.Event) error

func (f enqueueFanouts) Enqueue(ctx context.Context, events []shared.Event) error {
	return f(ctx, events)
}

// html is a description's HTML, which the job's arguments never keep.
const html = `<p>hi <mention-component entity_identifier="x"></mention-component></p>`

// recordFanouts is a subscriber whose enqueueings are recorded, each as the events it enqueued at once, in the
// publisher's context.
func recordFanouts(t *testing.T) (shared.EventSubscriber, *[][]shared.Event) {
	t.Helper()
	var calls [][]shared.Event
	sub := app.NewNotifyOnIssueEvents(enqueueFanouts(func(ctx context.Context, events []shared.Event) error {
		if ctx != t.Context() {
			t.Error("context replaced")
		}
		calls = append(calls, events)
		return nil
	}))
	if sub.Name != "notify_on_issue_events" || !reflect.DeepEqual(sub.Types, domain.EventTypes()) || sub.Handle != nil || sub.HandleAll == nil {
		t.Fatalf("subscriber %s %v; want notify_on_issue_events of the module's events, taking them at once", sub.Name, sub.Types)
	}
	return sub, &calls
}

// asSent is the event as the subscriber enqueues it: every change at its position, the HTML of a description and a
// comment blanked (R4).
func asSent(e shared.Event) shared.Event {
	e.Changes = domain.WithoutHTML(e.Changes)
	return e
}

// An update with a description's HTML, a comment with its HTML and nerve's own write mentioning alice: each notifies
// anyone but its actor of something (nerve's mention of anyone does).
func notifyingEvents() map[string]shared.Event {
	described := renamed()
	described.Changes[1] = shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}
	comment := commented()
	comment.Changes[0] = shared.Change{Field: "comment_html", New: str("<p>look</p>")}
	byNerve := fanoutEvent("issue.updated", shared.EntityRef{Kind: "issue", ID: workItem}, nil,
		shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}, shared.Change{Field: "mention", NewID: idOf(alice)})
	return map[string]shared.Event{"an update": described, "a comment": comment, "nerve mentioning alice": byNerve}
}

// Events that notify no one but their actor of anything: a description's change alone, which notifies of nothing;
// a change of a link's title alone; an update whose only notifiable item is its actor's mention of himself; an event
// of a type the module does not notify of.
func silentEvents() map[string]shared.Event {
	issueRef := shared.EntityRef{Kind: "issue", ID: workItem}
	return map[string]shared.Event{
		"a description alone": fanoutEvent("issue.updated", issueRef, &alice, shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}),
		"a link's title alone": fanoutEvent("issue.link_updated", shared.EntityRef{Kind: "issue_link", ID: uuid.NewV7()}, &alice,
			shared.Change{Field: "title", Old: str("a"), New: str("b")}, shared.Change{Field: "issue", NewID: idOf(workItem)}),
		"alice mentioning herself": fanoutEvent("issue.updated", issueRef, &alice,
			shared.Change{Field: "description", Old: str("<p>a</p>"), New: str(html)}, shared.Change{Field: "mention", NewID: idOf(alice)}),
		"a deletion": fanoutEvent("issue.deleted", issueRef, &alice),
	}
}

// The subscriber reads the events the module notifies of. An event that notifies anyone but its actor of anything
// enqueues one fan-out, in the publisher's context, its transaction (M7 P1b spec 2.9): the event itself, every change
// at its position, the HTML of a description and a comment blanked (R4). nerve's own write's mention of an account
// notifies him.
func TestNotifyOnIssueEventsEnqueuesTheEventWithoutItsHTML(t *testing.T) {
	for name, e := range notifyingEvents() {
		sub, calls := recordFanouts(t)
		before := shared.Event{ID: e.ID, Type: e.Type, OccurredAt: e.OccurredAt, WorkspaceID: e.WorkspaceID, ProjectID: e.ProjectID,
			ActorID: e.ActorID, Entity: e.Entity, Changes: append([]shared.Change(nil), e.Changes...)}
		if err := sub.HandleAll(t.Context(), []shared.Event{e}); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !reflect.DeepEqual(*calls, [][]shared.Event{{asSent(e)}}) {
			t.Errorf("%s: enqueued %+v; want once %+v", name, *calls, asSent(e))
		}
		if !reflect.DeepEqual(e, before) {
			t.Errorf("%s: the published event was changed: %+v", name, e)
		}
		for n, c := range (*calls)[0][0].Changes {
			if (c.Field == "description" || c.Field == "comment_html") && (c.Old != nil || c.New != nil) {
				t.Errorf("%s: change %d keeps its HTML", name, n)
			}
		}
	}
}

// A publication's events are enqueued at once (M7 P1b spec R12): exactly those, in their order and as they are sent,
// that each alone would have enqueued, an event between them that notifies no one skipped; nothing is enqueued for a
// publication none of whose events notifies anyone.
func TestNotifyOnIssueEventsEnqueuesAPublicationAtOnce(t *testing.T) {
	notifying, silent := notifyingEvents(), silentEvents()
	batch := []shared.Event{silent["a description alone"], notifying["an update"], silent["alice mentioning herself"], notifying["a comment"],
		silent["a deletion"], notifying["nerve mentioning alice"], silent["a link's title alone"]}
	var each []shared.Event
	for _, e := range batch {
		sub, calls := recordFanouts(t)
		if err := sub.HandleAll(t.Context(), []shared.Event{e}); err != nil {
			t.Fatal(err)
		}
		for _, call := range *calls {
			each = append(each, call...)
		}
	}
	want := []shared.Event{asSent(notifying["an update"]), asSent(notifying["a comment"]), asSent(notifying["nerve mentioning alice"])}
	if !reflect.DeepEqual(each, want) {
		t.Fatalf("each alone enqueued %+v; want %+v", each, want)
	}
	sub, calls := recordFanouts(t)
	if err := sub.HandleAll(t.Context(), batch); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(*calls, [][]shared.Event{each}) {
		t.Errorf("enqueued %+v; want once %+v", *calls, each)
	}
}

// An event that notifies no one but its actor of anything enqueues nothing, alone or with others like it.
func TestNotifyOnIssueEventsEnqueuesNothingForNoOne(t *testing.T) {
	var all []shared.Event
	for name, e := range silentEvents() {
		all = append(all, e)
		sub, calls := recordFanouts(t)
		if err := sub.HandleAll(t.Context(), []shared.Event{e}); err != nil || len(*calls) != 0 {
			t.Errorf("%s: %v, enqueued %+v", name, err, *calls)
		}
	}
	sub, calls := recordFanouts(t)
	if err := sub.HandleAll(t.Context(), all); err != nil || len(*calls) != 0 {
		t.Errorf("all at once: %v, enqueued %+v", err, *calls)
	}
}

// An event to enqueue that the job would refuse, one that does not name its work item, its id, its moment or a real
// actor, fails the publisher's write before anything is enqueued, the events before it and after it too; so does the
// enqueueing's failure.
func TestNotifyOnIssueEventsRefusesAnIncompleteEventAndFailsWithTheEnqueueing(t *testing.T) {
	for name, change := range map[string]func(e *shared.Event){
		"no id":        func(e *shared.Event) { e.ID = uuid.UUID{} },
		"no moment":    func(e *shared.Event) { e.OccurredAt = time.Time{} },
		"zero actor":   func(e *shared.Event) { e.ActorID = &uuid.UUID{} },
		"no work item": func(e *shared.Event) { e.Changes = e.Changes[:2] },
	} {
		e := commented()
		change(&e)
		sub, calls := recordFanouts(t)
		if err := sub.HandleAll(t.Context(), []shared.Event{commented(), e, renamed()}); err == nil || len(*calls) != 0 {
			t.Errorf("%s: %v, enqueued %+v; want a refusal before the enqueueing", name, err, *calls)
		}
	}
	failure := errors.New("insert failed")
	sub := app.NewNotifyOnIssueEvents(enqueueFanouts(func(context.Context, []shared.Event) error { return failure }))
	if err := sub.HandleAll(t.Context(), []shared.Event{commented()}); !errors.Is(err, failure) {
		t.Fatalf("HandleAll = %v; want the enqueueing's failure", err)
	}
}
