package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// batchCalls records each delivery of a dispatcher as "subscriber:event" for
// a Handle subscriber and "subscriber:[events]" for a HandleAll subscriber, the
// events by their Entity.Kind, which the tests name them by.
type batchCalls struct {
	ctx   context.Context
	calls []string
	fail  string
	err   error
}

func (c *batchCalls) handle(t *testing.T, name string) func(context.Context, shared.Event) error {
	return func(ctx context.Context, e shared.Event) error {
		if ctx != c.ctx {
			t.Errorf("%s lost the caller's context", name)
		}
		call := name + ":" + e.Entity.Kind
		c.calls = append(c.calls, call)
		if call == c.fail {
			return c.err
		}
		return nil
	}
}

func (c *batchCalls) handleAll(t *testing.T, name string) func(context.Context, []shared.Event) error {
	return func(ctx context.Context, events []shared.Event) error {
		if ctx != c.ctx {
			t.Errorf("%s lost the caller's context", name)
		}
		kinds := make([]string, len(events))
		for n, e := range events {
			kinds[n] = e.Entity.Kind
		}
		call := fmt.Sprintf("%s:%v", name, kinds)
		c.calls = append(c.calls, call)
		if call == c.fail {
			return c.err
		}
		return nil
	}
}

// batchEvent is an event of the type kind, named name.
func batchEvent(kind, name string) shared.Event {
	return shared.Event{ID: uuid.NewV7(), Type: kind, Entity: shared.EntityRef{Kind: name}}
}

// mixedDispatcher registers [Handle A, Handle B, HandleAll N, Handle C] and
// two subscribers no event below is of a type of, a Handle D and a HandleAll M.
func mixedDispatcher(t *testing.T, c *batchCalls) *dispatcher {
	t.Helper()
	d, err := newDispatcher([]string{"x", "y", "z", "w"}, []shared.EventSubscriber{
		{Name: "A", Types: []string{"x", "y"}, Handle: c.handle(t, "A")},
		{Name: "B", Types: []string{"x"}, Handle: c.handle(t, "B")},
		{Name: "M", Types: []string{"z"}, HandleAll: c.handleAll(t, "M")},
		{Name: "N", Types: []string{"y", "x"}, HandleAll: c.handleAll(t, "N")},
		{Name: "D", Types: []string{"z"}, Handle: c.handle(t, "D")},
		{Name: "C", Types: []string{"y"}, Handle: c.handle(t, "C")},
	})
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func mixedEvents() []shared.Event {
	return []shared.Event{batchEvent("x", "x1"), batchEvent("y", "y1"), batchEvent("w", "w1"), batchEvent("x", "x2"), batchEvent("y", "y2")}
}

// PublishAll walks the subscribers in registration order (M7 P1b spec R12): a
// run of adjacent Handle subscribers takes the events one at a time, each going
// to each subscriber of the run that takes its type; a HandleAll subscriber is
// called once, in its place, with its events in their order; a subscriber none
// of whose types is published is not called.
func TestDispatcherPublishAllOrder(t *testing.T) {
	type key struct{}
	c := &batchCalls{ctx: context.WithValue(t.Context(), key{}, "transaction")}
	if err := mixedDispatcher(t, c).PublishAll(c.ctx, mixedEvents()); err != nil {
		t.Fatal(err)
	}
	want := []string{"A:x1", "B:x1", "A:y1", "A:x2", "B:x2", "A:y2", "N:[x1 y1 x2 y2]", "C:y1", "C:y2"}
	if !slices.Equal(c.calls, want) {
		t.Errorf("calls = %v\nwant    %v", c.calls, want)
	}
}

// The first error stops the delivery and is returned as itself: inside a run
// of Handle subscribers, at the event it fails on; at a HandleAll subscriber,
// before the subscribers after it.
func TestDispatcherPublishAllStopsAtFirstError(t *testing.T) {
	cause := errors.New("database failure")
	for _, tt := range []struct {
		fail string
		want []string
	}{
		{"B:x2", []string{"A:x1", "B:x1", "A:y1", "A:x2", "B:x2"}},
		{"A:y1", []string{"A:x1", "B:x1", "A:y1"}},
		{"N:[x1 y1 x2 y2]", []string{"A:x1", "B:x1", "A:y1", "A:x2", "B:x2", "A:y2", "N:[x1 y1 x2 y2]"}},
		{"C:y1", []string{"A:x1", "B:x1", "A:y1", "A:x2", "B:x2", "A:y2", "N:[x1 y1 x2 y2]", "C:y1"}},
	} {
		t.Run(tt.fail, func(t *testing.T) {
			failure := fmt.Errorf("subscriber: %w", cause)
			c := &batchCalls{ctx: t.Context(), fail: tt.fail, err: failure}
			if err := mixedDispatcher(t, c).PublishAll(c.ctx, mixedEvents()); err != failure || !errors.Is(err, cause) {
				t.Errorf("error = %v; want the subscriber's own", err)
			}
			if !slices.Equal(c.calls, tt.want) {
				t.Errorf("calls = %v\nwant    %v", c.calls, tt.want)
			}
		})
	}
}

// Every event's type is checked before any delivery: one undeclared type
// refuses the whole publication and no subscriber is called.
func TestDispatcherPublishAllChecksEveryTypeFirst(t *testing.T) {
	for _, kind := range []string{"unknown", ""} {
		c := &batchCalls{ctx: t.Context()}
		events := append(mixedEvents(), batchEvent(kind, "bad"))
		if err := mixedDispatcher(t, c).PublishAll(c.ctx, events); err == nil {
			t.Errorf("accepted undeclared event %q", kind)
		}
		if len(c.calls) != 0 {
			t.Errorf("%q: delivered %v before refusing", kind, c.calls)
		}
	}
}

// With Handle subscribers only, PublishAll(events) is exactly Publish of each
// event in turn: the same calls in the same order, and the same first error.
func TestDispatcherPublishAllOfHandleSubscribersIsEachPublishInTurn(t *testing.T) {
	failure := errors.New("subscriber failure")
	register := func(c *batchCalls) *dispatcher {
		d, err := newDispatcher([]string{"x", "y", "w"}, []shared.EventSubscriber{
			{Name: "A", Types: []string{"x", "y"}, Handle: c.handle(t, "A")},
			{Name: "B", Types: []string{"x"}, Handle: c.handle(t, "B")},
			{Name: "C", Types: []string{"y", "x"}, Handle: c.handle(t, "C")},
		})
		if err != nil {
			t.Fatal(err)
		}
		return d
	}
	events := []shared.Event{batchEvent("x", "x1"), batchEvent("y", "y1"), batchEvent("w", "w1"), batchEvent("x", "x2")}
	for _, fail := range []string{"", "A:x1", "B:x1", "C:x1", "A:y1", "C:y1", "A:x2", "B:x2", "C:x2"} {
		each := &batchCalls{ctx: t.Context(), fail: fail, err: failure}
		var eachErr error
		for _, e := range events {
			if eachErr = register(each).Publish(each.ctx, e); eachErr != nil {
				break
			}
		}
		all := &batchCalls{ctx: t.Context(), fail: fail, err: failure}
		allErr := register(all).PublishAll(all.ctx, events)
		if allErr != eachErr || !slices.Equal(all.calls, each.calls) {
			t.Errorf("failing at %q: PublishAll %v, %v; each Publish %v, %v", fail, all.calls, allErr, each.calls, eachErr)
		}
	}
}

// Publish(e) is PublishAll([e]): it reaches a HandleAll subscriber with a
// one-event slice, in its place, and skips one that does not take its type.
func TestDispatcherPublishReachesAHandleAllSubscriber(t *testing.T) {
	c := &batchCalls{ctx: t.Context()}
	d := mixedDispatcher(t, c)
	if err := d.Publish(c.ctx, batchEvent("y", "y1")); err != nil {
		t.Fatal(err)
	}
	if want := []string{"A:y1", "N:[y1]", "C:y1"}; !slices.Equal(c.calls, want) {
		t.Errorf("calls = %v; want %v", c.calls, want)
	}
	c.calls = nil
	if err := d.Publish(c.ctx, batchEvent("w", "w1")); err != nil || len(c.calls) != 0 {
		t.Errorf("an event no one takes: %v, calls %v", err, c.calls)
	}
}

// Each handler owns its events, the slice a HandleAll subscriber gets as well:
// what one handler changes, no later handler and not the caller sees.
func TestDispatcherPublishAllCopiesEachEvent(t *testing.T) {
	ids, actor, oldID, at := []uuid.UUID{uuid.NewV7(), uuid.NewV7()}, uuid.NewV7(), uuid.NewV7(), time.Now().UTC()
	// Each call owns independent pointers, so that aliasing cannot mask a mutation.
	event := func(n int) shared.Event {
		a, o, v := actor, oldID, "old"
		return shared.Event{ID: ids[n], Type: "known", OccurredAt: at, ActorID: &a, Entity: shared.EntityRef{Kind: fmt.Sprint(n)},
			Changes: []shared.Change{{Field: "state", Old: &v, OldID: &o}}}
	}
	events, want := []shared.Event{event(0), event(1)}, []shared.Event{event(0), event(1)}
	pollute := func(e shared.Event) {
		*e.ActorID = uuid.NewV7()
		*e.Changes[0].Old, *e.Changes[0].OldID = "polluted", uuid.NewV7()
		e.Changes[0].Field = "polluted"
	}
	var seen int
	observe := func(e shared.Event) {
		seen++
		if i := slices.IndexFunc(want, func(w shared.Event) bool { return w.ID == e.ID }); i < 0 || !reflect.DeepEqual(e, want[i]) {
			t.Errorf("handler received polluted event: %+v", e)
		}
	}
	d, err := newDispatcher([]string{"known"}, []shared.EventSubscriber{
		{Name: "all mutator", Types: []string{"known"}, HandleAll: func(_ context.Context, es []shared.Event) error {
			for _, e := range es {
				observe(e)
				pollute(e)
			}
			es[0] = shared.Event{Type: "replaced"}
			return nil
		}},
		{Name: "each mutator", Types: []string{"known"}, Handle: func(_ context.Context, e shared.Event) error { observe(e); pollute(e); return nil }},
		{Name: "all observer", Types: []string{"known"}, HandleAll: func(_ context.Context, es []shared.Event) error {
			for _, e := range es {
				observe(e)
			}
			return nil
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.PublishAll(t.Context(), events); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(events, want) {
		t.Error("PublishAll changed the caller's events")
	}
	if seen != 6 {
		t.Errorf("handlers saw %d events; want 6", seen)
	}
}
