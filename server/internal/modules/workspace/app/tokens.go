package app

import (
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// invitationTokens computes and checks the invitations' tokens with the
// invitation MAC (M3 design 3.8). The server stores no token: it computes
// one again from the invitation's id.
type invitationTokens struct {
	mac InvitationMAC
}

// of is the token of the invitation id.
func (t invitationTokens) of(id uuid.UUID) string {
	return domain.FormatToken(t.mac.Tag(domain.InvitationMessage(id)))
}

// valid reports whether token is the invitation id's: the MAC checks the
// tag in constant time (M3 design 8.1). A token not of the format has no
// tag to check.
func (t invitationTokens) valid(id uuid.UUID, token string) bool {
	tag, ok := domain.ParseToken(token)
	return ok && t.mac.Verify(domain.InvitationMessage(id), tag)
}

// withTokens is invitations, each with its token.
func (t invitationTokens) withTokens(invitations []domain.Invitation) []domain.InvitationWithToken {
	out := make([]domain.InvitationWithToken, len(invitations))
	for i, inv := range invitations {
		out[i] = domain.InvitationWithToken{Invitation: inv, Token: t.of(inv.ID)}
	}
	return out
}
