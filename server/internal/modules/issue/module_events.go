package issue

import (
	riveradapter "github.com/open-nerve/NerveProject/server/internal/modules/issue/adapter/river"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// EventTypes declares the events published in P6. Links and cross-module
// deletions join the directory when their publishing use cases arrive.
func EventTypes() []string {
	return []string{
		"issue.created", "issue.updated", "issue.deleted",
		"issue.archived", "issue.unarchived",
		"issue.comment_created", "issue.comment_updated", "issue.comment_deleted",
		"issue.relation_added", "issue.relation_removed",
		"issue.reaction_added", "issue.reaction_removed",
	}
}

// Subscribers is deliberately empty during P6 (M4 design 3.19). P7's link
// subscriber will consume the transaction-bound insertion port here.
func Subscribers(_ riveradapter.Inserter) []shared.EventSubscriber { return nil }
