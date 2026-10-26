package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestDispatcherRegistrationOrder(t *testing.T) {
	type transactionKey struct{}
	ctx := context.WithValue(t.Context(), transactionKey{}, "transaction")
	event := shared.Event{ID: uuid.NewV7(), Type: "issue.updated"}
	var got []string
	handle := func(name string) func(context.Context, shared.Event) error {
		return func(actual context.Context, e shared.Event) error {
			if actual != ctx || e.ID != event.ID {
				t.Error("lost caller context or event identity")
			}
			got = append(got, name)
			return nil
		}
	}
	d, err := newDispatcher([]string{"issue.created", "issue.updated", "issue.deleted"}, []shared.EventSubscriber{
		{Name: "zebra", Types: []string{"issue.created", "issue.updated"}, Handle: handle("zebra")},
		{Name: "middle", Types: []string{"issue.updated"}, Handle: handle("middle")},
		{Name: "alpha", Types: []string{"issue.updated", "issue.deleted"}, Handle: handle("alpha")},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		kind string
		want []string
	}{
		{"issue.updated", []string{"zebra", "middle", "alpha"}},
		{"issue.created", []string{"zebra"}},
		{"issue.deleted", []string{"alpha"}},
	} {
		got = nil
		event.Type = tt.kind
		if err := d.Publish(ctx, event); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("%s calls = %v, want %v", tt.kind, got, tt.want)
		}
	}
}

func TestDispatcherStopsAtFirstError(t *testing.T) {
	cause := errors.New("database failure")
	failure := fmt.Errorf("subscriber: %w", cause)
	var got []string
	d, err := newDispatcher([]string{"known"}, []shared.EventSubscriber{
		{Name: "first", Types: []string{"known"}, Handle: func(context.Context, shared.Event) error { got = append(got, "first"); return nil }},
		{Name: "second", Types: []string{"known"}, Handle: func(context.Context, shared.Event) error { got = append(got, "second"); return failure }},
		{Name: "third", Types: []string{"known"}, Handle: func(context.Context, shared.Event) error { got = append(got, "third"); return nil }},
	})
	if err != nil {
		t.Fatal(err)
	}
	err = d.Publish(t.Context(), shared.Event{Type: "known"})
	if err != failure || !errors.Is(err, cause) {
		t.Errorf("error = %v, want original error with cause", err)
	}
	if !reflect.DeepEqual(got, []string{"first", "second"}) {
		t.Errorf("calls = %v", got)
	}
}

func TestDispatcherRejectsInvalidRegistration(t *testing.T) {
	handle := func(context.Context, shared.Event) error { return nil }
	valid := shared.EventSubscriber{Name: "valid", Types: []string{"known"}, Handle: handle}
	for _, tt := range []struct {
		name        string
		types       []string
		subscribers []shared.EventSubscriber
	}{
		{"empty declaration", []string{""}, nil},
		{"duplicate declaration", []string{"known", "known"}, nil},
		{"unknown subscription", []string{"known"}, []shared.EventSubscriber{{Name: "unknown", Types: []string{"unknown"}, Handle: handle}}},
		{"duplicate name", []string{"known"}, []shared.EventSubscriber{valid, valid}},
		{"empty name", []string{"known"}, []shared.EventSubscriber{{Types: []string{"known"}, Handle: handle}}},
		{"blank name", []string{"known"}, []shared.EventSubscriber{{Name: " \t", Types: []string{"known"}, Handle: handle}}},
		{"nil handle", []string{"known"}, []shared.EventSubscriber{{Name: "nil", Types: []string{"known"}}}},
		{"duplicate subscription", []string{"known"}, []shared.EventSubscriber{{Name: "duplicate", Types: []string{"known", "known"}, Handle: handle}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := newDispatcher(tt.types, tt.subscribers); err == nil {
				t.Fatal("accepted invalid registration")
			}
		})
	}
}

func TestDispatcherEventDirectory(t *testing.T) {
	d, err := newDispatcher([]string{"known"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Publish(t.Context(), shared.Event{Type: "known"}); err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"unknown", ""} {
		if err := d.Publish(t.Context(), shared.Event{Type: kind}); err == nil {
			t.Errorf("accepted undeclared event %q", kind)
		}
	}
	empty, err := newDispatcher(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := empty.Publish(t.Context(), shared.Event{Type: "known"}); err == nil {
		t.Error("empty directory accepted event")
	}
}

func TestDispatcherCopiesRegistration(t *testing.T) {
	var calls []string
	types := []string{"known", "unused"}
	subscriptions := []string{"known"}
	subscribers := []shared.EventSubscriber{{Name: "first", Types: subscriptions, Handle: func(context.Context, shared.Event) error { calls = append(calls, "original"); return nil }}}
	d, err := newDispatcher(types, subscribers)
	if err != nil {
		t.Fatal(err)
	}
	types[0] = "replacement"
	subscriptions[0] = "unused"
	subscribers[0] = shared.EventSubscriber{Name: "replacement", Handle: func(context.Context, shared.Event) error { calls = append(calls, "replacement"); return nil }}
	if err := d.Publish(t.Context(), shared.Event{Type: "known"}); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"original"}) {
		t.Errorf("calls = %v", calls)
	}
	if err := d.Publish(t.Context(), shared.Event{Type: "replacement"}); err == nil {
		t.Error("external type mutation changed directory")
	}
}

func TestDispatcherCopiesEachEvent(t *testing.T) {
	actor, oldID, newID := uuid.NewV7(), uuid.NewV7(), uuid.NewV7()
	old, next := "old", "new"
	event := shared.Event{ID: uuid.NewV7(), Type: "known", OccurredAt: time.Now().UTC(), WorkspaceID: uuid.NewV7(), ProjectID: uuid.NewV7(), ActorID: &actor, Entity: shared.EntityRef{Kind: "issue", ID: uuid.NewV7()}, Changes: []shared.Change{{Field: "state", Old: &old, New: &next, OldID: &oldID, NewID: &newID}}}
	// The expected value owns independent pointers so aliasing cannot mask a mutation.
	wantActor, wantOldID, wantNewID := actor, oldID, newID
	wantOld, wantNew := "old", "new"
	want := event
	want.ActorID = &wantActor
	want.Changes = []shared.Change{{Field: "state", Old: &wantOld, New: &wantNew, OldID: &wantOldID, NewID: &wantNewID}}
	mutate := func(_ context.Context, e shared.Event) error {
		*e.ActorID = uuid.NewV7()
		*e.Changes[0].Old, *e.Changes[0].New = "polluted", "polluted"
		*e.Changes[0].OldID, *e.Changes[0].NewID = uuid.NewV7(), uuid.NewV7()
		e.Changes[0].Field = "polluted"
		e.Changes = append(e.Changes, shared.Change{Field: "extra"})
		return nil
	}
	var received int
	observe := func(_ context.Context, e shared.Event) error {
		received++
		if !reflect.DeepEqual(e, want) {
			t.Errorf("subscriber received polluted event: %+v", e)
		}
		return mutate(t.Context(), e)
	}
	d, err := newDispatcher([]string{"known"}, []shared.EventSubscriber{
		{Name: "mutator", Types: []string{"known"}, Handle: mutate},
		{Name: "observer", Types: []string{"known"}, Handle: observe},
		{Name: "last", Types: []string{"known"}, Handle: observe},
	})
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := d.Publish(t.Context(), event); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(event, want) {
			t.Error("publish mutated caller event")
		}
	}
	if received != 4 {
		t.Errorf("observers called %d times, want 4", received)
	}
}

func TestIssueEventDirectory(t *testing.T) {
	want := []string{"issue.created", "issue.updated", "issue.deleted", "issue.archived", "issue.unarchived", "issue.comment_created", "issue.comment_updated", "issue.comment_deleted", "issue.relation_added", "issue.relation_removed", "issue.reaction_added", "issue.reaction_removed"}
	types := issue.EventTypes()
	if !reflect.DeepEqual(types, want) {
		t.Fatalf("issue event directory = %v", types)
	}
	d, err := newDispatcher(types, issue.Subscribers(issueInserter{}))
	if err != nil {
		t.Fatal(err)
	}
	types[0] = "external mutation"
	if !reflect.DeepEqual(issue.EventTypes(), want) {
		t.Error("caller mutation changed module directory")
	}
	for _, kind := range want {
		if err := d.Publish(t.Context(), shared.Event{Type: kind}); err != nil {
			t.Fatalf("%s: %v", kind, err)
		}
	}
	for _, kind := range []string{"issue.link_created", "project.deleted", "workspace.deleted"} {
		if err := d.Publish(t.Context(), shared.Event{Type: kind}); err == nil {
			t.Errorf("accepted future event %s", kind)
		}
	}
}
