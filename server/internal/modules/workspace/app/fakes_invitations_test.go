package app_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"fmt"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// fakeMAC tags a message with the first 16 bytes of HMAC-SHA256 under a key
// of its own, as identity's does, and logs each Verify with the invitation
// id its message names. Two fakeMACs of other keys tag differently.
type fakeMAC struct {
	log *callLog
	key string
}

func (m fakeMAC) Tag(message []byte) [16]byte {
	h := hmac.New(sha256.New, []byte(m.key))
	h.Write(message)
	return [16]byte(h.Sum(nil)[:16])
}

func (m fakeMAC) Verify(message []byte, tag [16]byte) bool {
	id := uuid.UUID(message[len(message)-16:])
	m.log.calls = append(m.log.calls, "Verify "+id.String())
	want := m.Tag(message)
	return hmac.Equal(want[:], tag[:])
}

// tokenOf is the token of the invitation id under mac.
func tokenOf(mac fakeMAC, id uuid.UUID) string {
	return domain.FormatToken(mac.Tag(domain.InvitationMessage(id)))
}

// fakeInvitations is the invitations' repositories over fakeWorkspaces,
// which answers the workspaces and their locks: it logs every call with its
// arguments, answers from the undeleted invitations it holds, and fails a
// call with the error set for it, wrapped as the store wraps it.
type fakeInvitations struct {
	*fakeWorkspaces
	invitations []domain.Invitation
	listErr     error // for ListInvitations
}

func (f *fakeInvitations) ListInvitations(ctx context.Context, workspaceID uuid.UUID) ([]domain.Invitation, error) {
	f.log.add(ctx, "ListInvitations %s", workspaceID)
	if f.listErr != nil {
		return nil, fmt.Errorf("list workspace invitations: %w", f.listErr)
	}
	var out []domain.Invitation
	for _, inv := range f.invitations {
		if inv.WorkspaceID == workspaceID {
			out = append(out, inv)
		}
	}
	return out, nil
}
