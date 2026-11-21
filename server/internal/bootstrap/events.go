package bootstrap

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/open-nerve/NerveProject/server/internal/modules/asset"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue"
	"github.com/open-nerve/NerveProject/server/internal/modules/navigation"
	"github.com/open-nerve/NerveProject/server/internal/modules/notification"
	"github.com/open-nerve/NerveProject/server/internal/modules/planning"
	"github.com/open-nerve/NerveProject/server/internal/modules/project"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// dispatcher holds an immutable registration snapshot. Delivery is synchronous,
// in the caller's context, so errors participate in the caller's transaction.
// A publication's every event type is checked before any delivery; then the
// subscribers are walked in registration order (M7 P1b spec R12): a run of
// adjacent Handle subscribers takes the events one at a time, each event going
// to each subscriber of the run that takes its type, as publishing each event
// in turn does; a HandleAll subscriber is called once, in its place, with its
// events in their order, and not at all without any. Each handler owns its
// copies; the first error stops the delivery and is returned as itself. With
// Handle subscribers only, PublishAll(events) is Publish of each event in
// turn; to subscribers independent of each other it is that in any case.
// Publish(e) is PublishAll([e]).
type dispatcher struct {
	types       map[string]struct{}
	subscribers []shared.EventSubscriber
}

var (
	_ shared.EventPublisher      = (*dispatcher)(nil)
	_ shared.EventBatchPublisher = (*dispatcher)(nil)
)

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
		if (subscriber.Handle == nil) == (subscriber.HandleAll == nil) {
			return nil, fmt.Errorf("event subscriber %q must have exactly one of Handle and HandleAll", subscriber.Name)
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
	return d.PublishAll(ctx, []shared.Event{event})
}

func (d *dispatcher) PublishAll(ctx context.Context, events []shared.Event) error {
	for _, event := range events {
		if _, exists := d.types[event.Type]; !exists {
			return fmt.Errorf("cannot publish undeclared event type %q", event.Type)
		}
	}
	for rest := d.subscribers; len(rest) > 0; {
		if rest[0].HandleAll != nil {
			if err := deliverAll(ctx, rest[0], events); err != nil {
				return err
			}
			rest = rest[1:]
			continue
		}
		run := 1
		for run < len(rest) && rest[run].Handle != nil {
			run++
		}
		if err := deliverEach(ctx, rest[:run], events); err != nil {
			return err
		}
		rest = rest[run:]
	}
	return nil
}

// deliverEach hands the events one at a time to the run of Handle
// subscribers, each event to each of them that takes its type.
func deliverEach(ctx context.Context, run []shared.EventSubscriber, events []shared.Event) error {
	for _, event := range events {
		for _, subscriber := range run {
			if slices.Contains(subscriber.Types, event.Type) {
				if err := subscriber.Handle(ctx, cloneEvent(event)); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// deliverAll hands the HandleAll subscriber the events of its types at once,
// in their order, if there are any.
func deliverAll(ctx context.Context, subscriber shared.EventSubscriber, events []shared.Event) error {
	var taken []shared.Event
	for _, event := range events {
		if slices.Contains(subscriber.Types, event.Type) {
			taken = append(taken, cloneEvent(event))
		}
	}
	if len(taken) == 0 {
		return nil
	}
	return subscriber.HandleAll(ctx, taken)
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
	return slices.Concat(asset.EventTypes(), issue.EventTypes(), planning.EventTypes(), project.EventTypes(), workspace.EventTypes())
}

// eventSubscribers combines the modules' subscribers for the full
// transactional dispatcher, in their registration order, which is the
// order of their delivery (dispatcher): issue, planning, asset,
// notification, navigation.
func eventSubscribers(issuePorts issue.Provided, assetPorts asset.Provided, notificationPorts notification.Provided, inserter jobInserter) []shared.EventSubscriber {
	return slices.Concat(issue.Subscribers(issuePorts, inserter), planning.Subscribers(inserter), asset.Subscribers(assetPorts, inserter),
		notification.Subscribers(notificationPorts, inserter), navigation.Subscribers(inserter))
}
