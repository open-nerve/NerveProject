package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's errors (M4 design 5.3). Its operations on a project
// answer the project module's codes for the project, and its operations in
// a workspace the workspace module's code for the workspace, which they
// declare too (M4 design 5.1), with those modules' details.
var (
	// ErrLinkExists answers an exact duplicate live URL on this issue.
	ErrLinkExists = shared.NewError(shared.KindConflict, "issue.link_exists", "This work item already has this link.")
	// ErrMemberNotFound answers an absent or deleted target membership.
	ErrMemberNotFound = shared.NewError(shared.KindNotFound, "workspace.member_not_found", "The member does not exist, or you cannot see the workspace.")
	// ErrDescriptionVersionNotFound conceals missing versions and hidden parent issues.
	ErrDescriptionVersionNotFound = shared.NewError(shared.KindNotFound, "issue.description_version_not_found", "The description version does not exist, or you cannot see it.")
	// ErrReactionExists answers a duplicate live reaction of the caller.
	ErrReactionExists = shared.NewError(shared.KindConflict, "issue.reaction_exists", "You already have this reaction.")
	// ErrReactionNotFound answers a missing live reaction of the caller.
	ErrReactionNotFound = shared.NewError(shared.KindNotFound, "issue.reaction_not_found", "You do not have this reaction.")
	// ErrCommentNotFound conceals missing comments and hidden parent issues alike.
	ErrCommentNotFound = shared.NewError(shared.KindNotFound, "issue.comment_not_found", "The comment does not exist, or you cannot see it.")
	// ErrNotAuthor answers relative comment permissions after visibility succeeds.
	ErrNotAuthor = shared.NewError(shared.KindForbidden, "issue.not_author", "Only the author can edit a comment; only the author or a project administrator can delete it.")
	// ErrWorkspaceNotFound answers a workspace that does not exist, is
	// deleted, or of which the caller is not an active member (M3 design
	// 5.1, 8.2).
	ErrWorkspaceNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
	// ErrProjectNotFound answers a project that does not exist, is deleted,
	// or that the caller does not see (M3 design 3.4, 8.2).
	ErrProjectNotFound = shared.NewError(shared.KindNotFound, "project.not_found", "The project does not exist, or you cannot see it.")
	// ErrProjectArchived answers a write on the work items of an archived
	// project (M4 design 3.8).
	ErrProjectArchived = shared.NewError(shared.KindConflict, "project.archived", "The project is archived; unarchive it to change it.")
	// ErrIssueNotFound answers a work item that does not exist or is
	// deleted, one whose project the caller does not see, one the guest rule
	// hides from him, and every miss of a work item's identifier after the
	// workspace, all with this one value (M4 design 3.4, 3.7, 5.3).
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
	// ErrIssueArchived answers a change of an archived work item, even one
	// that changes nothing (M4 design 3.8).
	ErrIssueArchived = shared.NewError(shared.KindConflict, "issue.archived", "The work item is archived; unarchive it to change it.")
	// ErrNotArchivable answers the archiving of a work item in a state of
	// neither the completed nor the cancelled group (M4 design 3.8).
	ErrNotArchivable = shared.NewError(shared.KindConflict, "issue.not_archivable",
		"Only a completed or cancelled work item can be archived; move it to such a state first.")
	// ErrRelationNotFound answers a relation that does not exist or is
	// deleted, and one neither of whose work items the caller sees, all
	// with this one value (M4 design 3.10, 5.3).
	ErrRelationNotFound = shared.NewError(shared.KindNotFound, "issue.relation_not_found",
		"The relation does not exist, or you cannot see either of its work items.")
)
