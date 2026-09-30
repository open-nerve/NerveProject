package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
)

// fakeSignupInvitations answers allowed and err, and records each question.
type fakeSignupInvitations struct {
	allowed bool
	err     error
	asked   []string
}

func (f *fakeSignupInvitations) Allows(_ context.Context, email string, id uuid.UUID, token string) (bool, error) {
	f.asked = append(f.asked, fmt.Sprintf("%s %s %s", email, id, token))
	return f.allowed, f.err
}

// While sign-up is on, every registration goes on and no invitation is
// asked about, a bad one neither; while it is off, none without an
// invitation, and one with an invitation as workspace answers, its failure
// the failure.
func TestSignupPolicy(t *testing.T) {
	inv := &identity.SignupInvitation{ID: uuid.NewV7(), Token: "nrv_inv_AAAAAAAAAAAAAAAAAAAAAA"}
	asked := []string{"carol@corp.com " + inv.ID.String() + " " + inv.Token}
	failure := errors.New("connection reset")
	for _, tt := range []struct {
		name       string
		enabled    bool
		invitation *identity.SignupInvitation
		workspace  fakeSignupInvitations
		want       bool
		err        error
		asked      []string
	}{
		{"on, without an invitation", true, nil, fakeSignupInvitations{}, true, nil, nil},
		{"on, with one workspace refuses", true, inv, fakeSignupInvitations{}, true, nil, nil},
		{"off, without an invitation", false, nil, fakeSignupInvitations{allowed: true}, false, nil, nil},
		{"off, with one workspace allows", false, inv, fakeSignupInvitations{allowed: true}, true, nil, asked},
		{"off, with one workspace refuses", false, inv, fakeSignupInvitations{}, false, nil, asked},
		{"off, the check failed", false, inv, fakeSignupInvitations{err: failure}, false, failure, asked},
	} {
		invitations := tt.workspace
		got, err := signupPolicy{enabled: tt.enabled, invitations: &invitations}.AllowSignup(context.Background(), "carol@corp.com", tt.invitation)
		if got != tt.want || !errors.Is(err, tt.err) {
			t.Errorf("%s: AllowSignup() = %v, %v; want %v, %v", tt.name, got, err, tt.want, tt.err)
		}
		if !slices.Equal(invitations.asked, tt.asked) {
			t.Errorf("%s: workspace was asked %q, want %q", tt.name, invitations.asked, tt.asked)
		}
	}
}
