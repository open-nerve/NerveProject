package app_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// A registration's invitation allows the address it was sent to, with its
// link's token, while it is pending: carol's to acme, erin's to beta. The
// token is checked first; the invitation is read without a transaction.
func TestSignupInvitationsAllowTheInvitedAddress(t *testing.T) {
	for _, tt := range []struct {
		email string
		id    uuid.UUID
	}{{"carol@corp.com", carolToAcme.ID}, {"erin@corp.com", erinToBeta.ID}} {
		f := newInvitations()
		allowed, err := app.NewSignupInvitations(f.invitations, f.mac).Allows(context.Background(), tt.email, tt.id, tokenOf(f.mac, tt.id))
		if err != nil || !allowed {
			t.Errorf("%s: Allows() = %v, %v; want true", tt.email, allowed, err)
		}
		if want := []string{"Verify " + tt.id.String(), "InvitationByID " + tt.id.String() + " outside tx"}; !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.email, f.log.calls, want)
		}
	}
}

// Every other case is false, without an error: a token not the
// invitation's reads nothing; an invitation not there or deleted (as an
// accepted one is), a declined one, another address, the address not
// normalized. A failed read is an error.
func TestSignupInvitationsRefuseEveryOtherCase(t *testing.T) {
	failure := errors.New("connection reset")
	nobodys := uuid.NewV7()
	for _, tt := range []struct {
		name, email string
		id          uuid.UUID
		token       string // "" for the invitation's
		set         func(f *invitationsFixture)
		read        bool
		err         error
	}{
		{"another invitation's token", "carol@corp.com", carolToAcme.ID, "other", nil, false, nil},
		{"no such invitation", "carol@corp.com", nobodys, "", nil, true, nil},
		{"deleted or accepted", "carol@corp.com", carolToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta} }, true, nil},
		{"declined", "dave@corp.com", daveToAcme.ID, "", nil, true, nil},
		{"another address", "erin@corp.com", carolToAcme.ID, "", nil, true, nil},
		{"the address not normalized", "Carol@corp.com", carolToAcme.ID, "", nil, true, nil},
		{"the read failed", "carol@corp.com", carolToAcme.ID, "", func(f *invitationsFixture) { f.invitations.readErr = failure }, true, failure},
	} {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		token := tokenOf(f.mac, tt.id)
		if tt.token == "other" {
			token = tokenOf(f.mac, daveToAcme.ID)
		}
		allowed, err := app.NewSignupInvitations(f.invitations, f.mac).Allows(context.Background(), tt.email, tt.id, token)
		if allowed || !errors.Is(err, tt.err) {
			t.Errorf("%s: Allows() = %v, %v; want false, %v", tt.name, allowed, err, tt.err)
		}
		want := []string{"Verify " + tt.id.String()}
		if tt.read {
			want = append(want, "InvitationByID "+tt.id.String()+" outside tx")
		}
		if !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: calls = %q, want %q", tt.name, f.log.calls, want)
		}
	}
}
