package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// dispatcher holds an immutable registration snapshot. Delivery is synchronous
// in registration order so errors participate in the caller's transaction.
type dispatcher struct {
	types       map[string]struct{}
	subscribers []shared.EventSubscriber
}

var _ shared.EventPublisher = (*dispatcher)(nil)

func newDispatcher(types []string, subscribers []shared.EventSubscriber) (*dispatcher, error) {
	d := &dispatcher{types: make(map[string]struct{}, len(types))}
	for _, kind := range types {
		if strings.TrimSpace(kind) == "" {
			return nil, fmt.Errorf("event type must not be empty")
		}
		if _, exists := d.types[kind]; exists {
			return nil, fmt.Errorf("duplicate event type %q", kind)
		}
		d.types[kind] = struct{}{}
	}
	names := make(map[string]struct{}, len(subscribers))
	for _, subscriber := range subscribers {
		if strings.TrimSpace(subscriber.Name) == "" {
			return nil, fmt.Errorf("event subscriber name must not be empty")
		}
		if _, exists := names[subscriber.Name]; exists {
			return nil, fmt.Errorf("duplicate event subscriber %q", subscriber.Name)
		}
		if subscriber.Handle == nil {
			return nil, fmt.Errorf("event subscriber %q has no handler", subscriber.Name)
		}
		subscribed := make(map[string]struct{}, len(subscriber.Types))
		for _, kind := range subscriber.Types {
			if _, exists := d.types[kind]; !exists {
				return nil, fmt.Errorf("event subscriber %q subscribes to undeclared type %q", subscriber.Name, kind)
			}
			if _, exists := subscribed[kind]; exists {
				return nil, fmt.Errorf("event subscriber %q repeats type %q", subscriber.Name, kind)
			}
			subscribed[kind] = struct{}{}
		}
		names[subscriber.Name] = struct{}{}
		subscriber.Types = slices.Clone(subscriber.Types)
		d.subscribers = append(d.subscribers, subscriber)
	}
	return d, nil
}

func (d *dispatcher) Publish(ctx context.Context, event shared.Event) error {
	if _, exists := d.types[event.Type]; !exists {
		return fmt.Errorf("cannot publish undeclared event type %q", event.Type)
	}
	for _, subscriber := range d.subscribers {
		if slices.Contains(subscriber.Types, event.Type) {
			if err := subscriber.Handle(ctx, cloneEvent(event)); err != nil {
				return err
			}
		}
	}
	return nil
}

// Each handler owns its payload, including every optional change value.
func cloneEvent(event shared.Event) shared.Event {
	event.ActorID = cloneEventValue(event.ActorID)
	event.Changes = slices.Clone(event.Changes)
	for i := range event.Changes {
		change := &event.Changes[i]
		change.Old, change.New = cloneEventValue(change.Old), cloneEventValue(change.New)
		change.OldID, change.NewID = cloneEventValue(change.OldID), cloneEventValue(change.NewID)
	}
	return event
}

func cloneEventValue[T any](value *T) *T {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

// eventTypes combines producer declarations for the full transactional dispatcher.
func eventTypes() []string {
	return slices.Concat(issue.EventTypes(), project.EventTypes(), workspace.EventTypes())
}
