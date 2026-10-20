package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The issue module's errors (M4 design 5.3). Its operations on a project
// answer the project module's codes for the project, and its operations in
// a workspace the workspace module's code for the workspace, which they
// declare too (M4 design 5.1), with those modules' details.
var (
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
	// ErrIssueNotFound answers a work item that does not exist, is deleted,
	// or whose project the caller does not see (M4 design 5.3).
	ErrIssueNotFound = shared.NewError(shared.KindNotFound, "issue.not_found", "The work item does not exist, or you cannot see it.")
	// ErrNotArchivable answers the archiving of a work item in a state of
	// neither the completed nor the cancelled group (M4 design 3.8).
	ErrNotArchivable = shared.NewError(shared.KindConflict, "issue.not_archivable",
		"Only a completed or cancelled work item can be archived; move it to such a state first.")
)
