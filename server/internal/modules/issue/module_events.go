package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/modules/issue/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// EventTypes declares the events published in P6. Links and cross-module
// deletions join the directory when their publishing use cases arrive.
func EventTypes() []string {
	return []string{
		domain.EventIssueCreated, domain.EventIssueUpdated, domain.EventIssueDeleted,
		domain.EventIssueArchived, domain.EventIssueUnarchived,
		domain.EventCommentCreated, domain.EventCommentUpdated, domain.EventCommentDeleted,
		domain.EventRelationAdded, domain.EventRelationRemoved,
		domain.EventReactionAdded, domain.EventReactionRemoved,
	}
}

// Subscribers is deliberately empty during P6 (M4 design 3.19). P7's link
// subscriber will consume the transaction-bound insertion port here.
func Subscribers(_ riveradapter.Inserter) []shared.EventSubscriber { return nil }
