package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// commentEvent keeps HTML snapshots separate from the mutable response value.
func (s stamp) commentEvent(kind string, id uuid.UUID, old, new *string) shared.Event {
	e := s.event(kind, []shared.Change{{Field: "comment_html", Old: old, New: new}})
	e.Entity = shared.EntityRef{Kind: "issue_comment", ID: id}
	return e
}

// reactionEvent carries the stored row identity; the container has ID slots only.
func (s stamp) reactionEvent(kind, entity string, id uuid.UUID, reaction, field string, container uuid.UUID) shared.Event {
	changes := []shared.Change{{Field: "reaction", New: eventPtr(reaction)}, {Field: field, NewID: &container}}
	if kind == domain.EventReactionRemoved {
		for n := range changes {
			changes[n].Old, changes[n].OldID = changes[n].New, changes[n].NewID
			changes[n].New, changes[n].NewID = nil, nil
		}
	}
	e := s.event(kind, changes)
	e.Entity = shared.EntityRef{Kind: entity, ID: id}
	return e
}
