package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The notification module's errors (M7 P1c spec 2.2, 2.4). Its operations
// in a workspace answer the workspace module's code for the workspace,
// which they declare too, with that module's detail.
var (
	// ErrNotificationNotFound answers, all with this one value, a
	// notification that does not exist, another receiver's, a deleted one,
	// one in a workspace the caller is not an active member of, one of a
	// project he does not read now, and one of a work item the guest rule
	// hides from him (spec 2.7, R5): its answer never tells which.
	ErrNotificationNotFound = shared.NewError(shared.KindNotFound, "notification.not_found", "The notification does not exist, or you cannot see it.")
	// ErrWorkspaceNotFound answers a workspace that does not exist, is
	// deleted, or of which the caller is not an active member (M3 design
	// 5.1, 8.2).
	ErrWorkspaceNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
)
