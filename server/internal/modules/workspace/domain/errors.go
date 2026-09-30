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
	// ErrMemberNotFound answers a membership that does not exist, is deleted
	// or has ended, or whose workspace the caller cannot see: the same 404
	// for all (M3 design 5.3, 8.2).
	ErrMemberNotFound = shared.NewError(shared.KindNotFound, "workspace.member_not_found", "The member does not exist, or you cannot see the workspace.")
	// ErrOwnMembership answers a change of the caller's own membership (M3
	// design 3.4, 5.3): nobody changes his own role.
	ErrOwnMembership = shared.NewError(shared.KindConflict, "workspace.own_membership", "You cannot change your own membership.")
	// ErrInvitationNotFound answers an invitation that does not exist or is
	// deleted, as an accepted one is, or whose workspace the caller cannot
	// see; for the invitee, also a token that is not the invitation's: the
	// same 404 for all (M3 design 5.3, 8.2).
	ErrInvitationNotFound = shared.NewError(shared.KindNotFound, "workspace.invitation_not_found",
		"The invitation does not exist, or its link is not valid.")
	// ErrInvitationEmailMismatch answers a response to an invitation by an
	// account whose address, read under its lock, is not the invitation's;
	// it does not say which address the invitation is for (M3 design 3.8).
	ErrInvitationEmailMismatch = shared.NewError(shared.KindForbidden, "workspace.invitation_email_mismatch",
		"The invitation was sent to another e-mail address.")
	// ErrInvitationResponded answers a response to, or a change of, an
	// invitation that has been declined (M3 design 3.8).
	ErrInvitationResponded = shared.NewError(shared.KindConflict, "workspace.invitation_responded", "The invitation has been answered already.")
	// ErrSlugTaken answers a creation with a slug an undeleted workspace has.
	ErrSlugTaken = shared.NewError(shared.KindConflict, "workspace.slug_taken", "A workspace with this slug exists.")
	// ErrAccountNotFound answers `nerve workspaces create` for an address no
	// account has.
	ErrAccountNotFound = shared.NewError(shared.KindNotFound, "workspace.account_not_found", "No account has this e-mail address.")
	// ErrAccountDeactivated answers `nerve workspaces create` for a
	// deactivated account, as read under the lock (M3 design 3.6 convention 6).
	ErrAccountDeactivated = shared.NewError(shared.KindForbidden, "workspace.account_deactivated", "The account is deactivated.")
)
