package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func (f *invitationsFixture) decline() *app.DeclineWorkspaceInvitation {
	return app.NewDeclineWorkspaceInvitation(f.accounts, f.invitations, f.tx, clockAt{at: clockNow}, f.mac)
}

// Declining records the answer after the account's lock, the workspace's
// FOR SHARE and the invitation's lock, in one transaction (M3 design 3.8,
// 3.6): an active member's invitation as anyone else's; no membership is
// read or written, and the invitation stays.
func TestDeclineWorkspaceInvitation(t *testing.T) {
	for _, tt := range []struct {
		user app.AccountState
		inv  domain.Invitation
	}{{frank, frankToAcme}, {bob, invitationTo(bob, beta, shared.RoleAdmin)}} {
		f := responding(frankToAcme, tt.inv)
		err := f.decline().Execute(as(tt.user), tt.inv.ID, tokenOf(f.mac, tt.inv.ID))
		wantCalls := append(respondedCalls(tt.user, tt.inv, "ShareWorkspace"),
			fmt.Sprintf("DeclineInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if err != nil || !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s: Execute() = %v, calls = %q in %d transactions; want %q in one", tt.inv.Email, err, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// Declining refuses as accepting does (responseRefusals, responseFailures),
// under the workspace's FOR SHARE; a failed write is itself. Without a
// caller it is 401 and nothing is read.
func TestDeclineWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	cases := append(responseRefusals("ShareWorkspace"), responseFailures(failure, "ShareWorkspace")...)
	cases = append(cases, responseRefusal{"the write failed", frank, frankToAcme.ID, "",
		func(f *invitationsFixture) { f.invitations.failing = map[string]error{"DeclineInvitation": failure} }, failure,
		append(respondedCalls(frank, frankToAcme, "ShareWorkspace"),
			fmt.Sprintf("DeclineInvitation %s by %s at %s", frankToAcme.ID, frank.ID, clockNow.Format(time.RFC3339Nano)))})
	for _, tt := range cases {
		f := responding(frankToAcme)
		if tt.set != nil {
			tt.set(f)
		}
		token := tt.token
		if token == "" {
			token = tokenOf(f.mac, tt.id)
		}
		err := f.decline().Execute(as(tt.user), tt.id, token)
		var se *shared.Error
		if !errors.Is(err, tt.want) || (tt.want == failure && errors.As(err, &se)) {
			t.Errorf("%s: Execute() = %v, want %v", tt.name, err, tt.want)
		}
		wantTx := 1
		if len(tt.calls) == 1 {
			wantTx = 0
		}
		if !slices.Equal(f.log.calls, tt.calls) || f.tx.calls != wantTx {
			t.Errorf("%s: calls = %q in %d transactions, want %q in %d", tt.name, f.log.calls, f.tx.calls, tt.calls, wantTx)
		}
	}
	f := responding(frankToAcme)
	if err := f.decline().Execute(context.Background(), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID)); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
