package app

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// GetWorkspaceInvitation shows an invitation to whoever holds its link: the
// public GET /api/v0/workspace-invitations/{invitation_id}?token=….
type GetWorkspaceInvitation struct {
	invitations InvitationPreviewer
	tokens      invitationTokens
}

// NewGetWorkspaceInvitation returns the use case.
func NewGetWorkspaceInvitation(invitations InvitationPreviewer, mac InvitationMAC) *GetWorkspaceInvitation {
	return &GetWorkspaceInvitation{invitations: invitations, tokens: invitationTokens{mac: mac}}
}

// Execute checks the token, then reads. A token that is not the
// invitation's, and an invitation that does not exist, is deleted or
// accepted, or whose workspace is deleted, are the same
// workspace.invitation_not_found (M3 design 3.8, 8.2). A wrong token reads
// nothing: the token is the invitation id's MAC, which needs no row, so the
// answer to it cannot depend on whether the invitation exists. Nothing
// depends on the caller: the operation is public.
func (u *GetWorkspaceInvitation) Execute(ctx context.Context, id uuid.UUID, token string) (domain.InvitationPreview, error) {
	if !u.tokens.valid(id, token) {
		return domain.InvitationPreview{}, domain.ErrInvitationNotFound
	}
	p, err := u.invitations.InvitationPreview(ctx, id)
	if err != nil {
		return domain.InvitationPreview{}, invitationNotFound(err)
	}
	return p, nil
}
