package domain

import "github.com/open-nerve/NerveProject/server/internal/shared"

// The workspace module's errors (M3 design 5.3). api/modules/workspace.yaml
// declares the codes of those the API answers with in x-problem-codes; the
// last two are `nerve workspaces create`'s only (M3 design 3.11).
var (
	// ErrNotFound answers a workspace that does not exist, is deleted, or of
	// which the caller is not an active member: the same 404 for all three
	// (M3 design 8.2).
	ErrNotFound = shared.NewError(shared.KindNotFound, "workspace.not_found", "The workspace does not exist, or you are not a member of it.")
	// ErrCreationDisabled answers a creation while workspace.creation_enabled
	// is false (M3 design 3.11).
	ErrCreationDisabled = shared.NewError(shared.KindForbidden, "workspace.creation_disabled", "Creating workspaces is disabled on this instance.")
	// ErrSlugTaken answers a creation with a slug an undeleted workspace has.
	ErrSlugTaken = shared.NewError(shared.KindConflict, "workspace.slug_taken", "A workspace with this slug exists.")
	// ErrAccountNotFound answers `nerve workspaces create` for an address no
	// account has.
	ErrAccountNotFound = shared.NewError(shared.KindNotFound, "workspace.account_not_found", "No account has this e-mail address.")
	// ErrAccountDeactivated answers `nerve workspaces create` for a
	// deactivated account, as read under the lock (M3 design 3.6 convention 6).
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "workspace.account_deactivated", "The account is deactivated.")
)
