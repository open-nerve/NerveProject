package domain

import (
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Invitation is an undeleted row of workspace_member_invites (M3 design
// 4.4, 5.2): an invitation of an address to a workspace, with a role,
// pending or declined. An accepted invitation is deleted as it is accepted,
// so the stores never answer one.
type Invitation struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Email       string // normalized (M3 design 3.13)
	Role        shared.Role
	Accepted    bool
	RespondedAt *time.Time
	CreatedAt   time.Time
	CreatedByID *uuid.UUID
}

// InvitationWithToken is an invitation and the token of its link, which
// only who may manage the workspace's invitations is given (M3 design 5.2).
type InvitationWithToken struct {
	Invitation
	Token string
}

// NewInvitation is an invitation as a request asks for it: an address and a
// role.
type NewInvitation struct {
	Email string
	Role  shared.Role
}

// MaxInvitations is how many invitations one request makes at most (M3
// design 5.1).
const MaxInvitations = 100

// CheckInvitations normalizes each address as registration does (M3 design
// 3.13) and checks what the request alone decides, before anything is read
// (M3 design 3.6 convention 2): 1 to MaxInvitations invitations; each
// address valid, and not listed twice; each role one of the three. The
// batch is refused as a whole, one 422 validation_failed with every problem,
// each on invitations[i] of the request as sent (M3 design 3.8). It returns
// the batch in the request's order, the addresses normalized.
func CheckInvitations(batch []NewInvitation) ([]NewInvitation, error) {
	switch {
	case len(batch) == 0:
		return nil, shared.Invalid(shared.FieldError{Field: "invitations", Code: shared.FieldTooShort, Message: "must list an invitation"})
	case len(batch) > MaxInvitations:
		return nil, shared.Invalid(shared.FieldError{Field: "invitations", Code: shared.FieldTooLong,
			Message: fmt.Sprintf("must list at most %d invitations", MaxInvitations)})
	}
	var fields []shared.FieldError
	listed := map[string]bool{}
	out := make([]NewInvitation, len(batch))
	for i, inv := range batch {
		email := shared.NormalizeEmail(inv.Email)
		switch {
		case !shared.ValidEmail(email):
			fields = append(fields, invitationField(i, "email", shared.FieldInvalidFormat, "is not a valid e-mail address"))
		case listed[email]:
			fields = append(fields, invitationField(i, "email", shared.FieldDuplicate, "is listed twice"))
		}
		listed[email] = true
		if !slices.Contains(roles, inv.Role) {
			fields = append(fields, invitationField(i, "role", shared.FieldInvalidFormat, "is not 5, 15 or 20"))
		}
		out[i] = NewInvitation{Email: email, Role: inv.Role}
	}
	if len(fields) > 0 {
		return nil, shared.Invalid(fields...)
	}
	return out, nil
}

// MemberAddress is the problem of the address of invitation i of a request
// that an active member of the workspace has: an invitation never changes an
// active membership (M3 design 3.8).
func MemberAddress(i int) shared.FieldError {
	return invitationField(i, "email", shared.FieldNotAllowed, "is an active member's address")
}

// InvitedAddress is the problem of the address of invitation i of a request
// that an undeleted invitation of the workspace has, pending or declined
// (M3 design 3.8), however the use case learned it: before it inserted, or
// from the unique key of a concurrent request's insert.
func InvitedAddress(i int) shared.FieldError {
	return invitationField(i, "email", shared.FieldDuplicate, "has an invitation to the workspace already")
}

// invitationField is a problem with field of invitation i of a request.
func invitationField(i int, field, code, message string) shared.FieldError {
	return shared.FieldError{Field: fmt.Sprintf("invitations[%d].%s", i, field), Code: code, Message: message}
}
