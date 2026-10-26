package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// EventTypes declares the event types published by the issue module.
func EventTypes() []string {
	return []string{
		domain.EventIssueCreated, domain.EventIssueUpdated, domain.EventIssueDeleted,
		domain.EventIssueArchived, domain.EventIssueUnarchived,
		domain.EventCommentCreated, domain.EventCommentUpdated, domain.EventCommentDeleted,
		domain.EventRelationAdded, domain.EventRelationRemoved,
		domain.EventReactionAdded, domain.EventReactionRemoved,
		domain.EventLinkCreated, domain.EventLinkUpdated, domain.EventLinkDeleted,
	}
}

// Subscribers is empty during P6 because the module has no production consumer.
func Subscribers(_ riveradapter.Inserter) []shared.EventSubscriber { return nil }
