package shared

import (
	"context"
	"time"
	"uuid"
)

// EntityRef identifies the entity whose semantic change an event describes.
type EntityRef struct {
	Kind string
	ID   uuid.UUID
}

// Change holds a field's text and identifier values before and after a write.
type Change struct {
	Field        string
	Old, New     *string
	OldID, NewID *uuid.UUID
}

// Event describes one completed business write in the caller's transaction.
// A nil ActorID denotes a system action.
type Event struct {
	ID          uuid.UUID
	Type        string
	OccurredAt  time.Time
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	ActorID     *uuid.UUID
	Entity      EntityRef
	Changes     []Change
}

// EventPublisher delivers synchronously in the caller's context.
type EventPublisher interface {
	Publish(context.Context, Event) error
}

// EventSubscriber is a named handler for declared event types.
type EventSubscriber struct {
	Name   string
	Types  []string
	Handle func(context.Context, Event) error
}
