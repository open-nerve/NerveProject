package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/app"
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

// Subscribers composes transactional consumers independently of New.
func Subscribers(inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewCrawlOnLinkChange(riveradapter.NewLinkCrawls(inserter))}
}
