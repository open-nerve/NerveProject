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

// Subscribers composes transactional consumers independently of New: an
// attachment's change, a link's crawl, a project's or a workspace's
// deletion, and a cycle's or a module's deletion (M6 P3a spec R6).
func Subscribers(ports Provided, inserter riveradapter.Inserter) []shared.EventSubscriber {
	return []shared.EventSubscriber{app.NewAttachmentChanges(ports.attachmentChanges), app.NewCrawlOnLinkChange(riveradapter.NewLinkCrawls(inserter)),
		app.NewDeleteRowsOnDeletion(riveradapter.NewRowDeletions(inserter)), app.NewMembershipRemovals(ports.membershipRemovals)}
}
