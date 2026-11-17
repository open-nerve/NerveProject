package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// linkEvent keeps link identity in Entity and business fields in Changes,
// followed by the work item it belongs to (withIssue).
func (s stamp) linkEvent(kind string, id uuid.UUID, changes []shared.Change) shared.Event {
	e := s.event(kind, s.withIssue(kind, changes))
	e.Entity = shared.EntityRef{Kind: "issue_link", ID: id}
	return e
}
