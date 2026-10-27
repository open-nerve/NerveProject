package domain

// Event types are the semantic writes exposed by the issue module.
const (
	EventIssueCreated    = "issue.created"
	EventIssueUpdated    = "issue.updated"
	EventIssueDeleted    = "issue.deleted"
	EventIssueArchived   = "issue.archived"
	EventIssueUnarchived = "issue.unarchived"
	EventCommentCreated  = "issue.comment_created"
	EventCommentUpdated  = "issue.comment_updated"
	EventCommentDeleted  = "issue.comment_deleted"
	EventRelationAdded   = "issue.relation_added"
	EventRelationRemoved = "issue.relation_removed"
	EventReactionAdded   = "issue.reaction_added"
	EventReactionRemoved = "issue.reaction_removed"
	EventLinkCreated     = "issue.link_created"
	EventLinkUpdated     = "issue.link_updated"
	EventLinkDeleted     = "issue.link_deleted"
)
