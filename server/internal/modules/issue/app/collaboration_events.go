package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// commentEvent keeps HTML snapshots separate from the mutable response value.
func (s stamp) commentEvent(kind string, id uuid.UUID, old, new *string) shared.Event {
	e := s.event(kind, s.withIssue(kind, []shared.Change{{Field: "comment_html", Old: old, New: new}}))
	e.Entity = shared.EntityRef{Kind: "issue_comment", ID: id}
	return e
}

// withIssue appends the work item a comment or a link belongs to after every
// other change (M7 P1a spec 2.2): on the Old side for a deletion, whose
// changes are all Old, like a removal's container (R1); on the New side
// otherwise. The Entity stays the comment or the link.
func (s stamp) withIssue(kind string, changes []shared.Change) []shared.Change {
	c := shared.Change{Field: "issue", NewID: eventPtr(s.issue)}
	if kind == domain.EventCommentDeleted || kind == domain.EventLinkDeleted {
		c.OldID, c.NewID = c.NewID, nil
	}
	return append(changes, c)
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
