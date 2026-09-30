package app_test

import (
	"fmt"
	"log/slog"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Each write on an existing workspace reads the clock once, in its
// transaction, after its lock and its decision, just before it writes
// (P2 spec 2.6): a write
// that queued behind another on the lock never stamps an earlier time than
// the one it waited for. The deletion's cascade uses that one read for
// every step. The clock logs its read among the fakes' calls.
func TestEachWriteReadsTheClockUnderItsLock(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	tests := []struct {
		name string
		run  func() (calls []string, err error)
		want []string
	}{
		{"updateWorkspace", func() ([]string, error) {
			_, f := newUpdate()
			_, err := app.NewUpdateWorkspace(f.workspaces, f.auth, f.tx, clockAt{clockNow, f.log}).
				Execute(as(alice), "acme", domain.WorkspacePatch{Name: ptr("Renamed")})
			return f.log.calls, err
		}, append(lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionUpdate), "Now",
			"UpdateWorkspace "+acme.ID.String()+` name="Renamed" size=<nil> timezone=<nil> by `+alice.ID.String()+" at "+at)},
		{"updateWorkspacePreferences", func() ([]string, error) {
			f := newPrefs()
			_, err := app.NewUpdateWorkspacePreferences(f.workspaces, f.auth, f.tx, clockAt{clockNow, f.log}).
				Execute(as(bob), "beta", domain.PreferencesPatch{})
			return f.log.calls, err
		}, append(lockedDecision(bob, beta, "ShareWorkspaceBySlug", domain.ActionPreferencesUpdate), "Now",
			"UpsertPreferences "+beta.ID.String()+" "+bob.ID.String()+" mode=<nil> limit=<nil> at "+at)},
		{"deleteWorkspace", func() ([]string, error) {
			_, f := newDelete()
			err := app.NewDeleteWorkspace(f.workspaces, f.auth, f.tx, clockAt{now, f.log}, slog.New(slog.DiscardHandler)).
				Execute(as(alice), "acme")
			return f.log.calls, err
		}, slices.Concat(lockedDecision(alice, acme, "LockWorkspaceBySlug", domain.ActionDelete), []string{"Now"}, cascadeCalls(alice, acme))},
		{"updateWorkspaceMember", func() ([]string, error) {
			f := newMembers()
			_, err := app.NewUpdateWorkspaceMember(f.workspaces, f.profiles, f.auth, &fakeTx{}, clockAt{clockNow, f.log}).
				Execute(as(alice), bobInAcme.ID, shared.RoleGuest)
			return f.log.calls, err
		}, append(lockedMemberCalls(alice, bobInAcme), "Now",
			fmt.Sprintf("UpdateMemberRole %s to %d by %s at %s", bobInAcme.ID, shared.RoleGuest, alice.ID, at),
			fmt.Sprintf("PublicProfiles %v", []uuid.UUID{bob.ID}))},
		{"updateWorkspaceInvitation", func() ([]string, error) {
			f := newInvitations()
			_, err := app.NewUpdateWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{clockNow, f.log}, f.mac).
				Execute(as(alice), carolToAcme.ID, shared.RoleGuest)
			return f.log.calls, err
		}, append(lockedInvitationCalls(alice, carolToAcme, domain.ActionInvitationUpdate), "Now",
			fmt.Sprintf("UpdateInvitationRole %s to %d by %s at %s", carolToAcme.ID, shared.RoleGuest, alice.ID, at))},
		{"deleteWorkspaceInvitation", func() ([]string, error) {
			f := newInvitations()
			err := app.NewDeleteWorkspaceInvitation(f.invitations, f.auth, f.tx, clockAt{clockNow, f.log}).Execute(as(alice), daveToAcme.ID)
			return f.log.calls, err
		}, append(lockedInvitationCalls(alice, daveToAcme, domain.ActionInvitationDelete), "Now",
			fmt.Sprintf("DeleteInvitation %s by %s at %s", daveToAcme.ID, alice.ID, at))},
		{"acceptWorkspaceInvitation", func() ([]string, error) {
			f := responding(frankToAcme)
			_, err := app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations, Tx: f.tx,
				Clock: clockAt{clockNow, f.log}, MAC: f.mac}).Execute(as(frank), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID))
			return f.log.calls, err
		}, append(respondedCalls(frank, frankToAcme, "LockWorkspace"), "Now", "MemberOf "+acme.ID.String()+" "+frank.ID.String(),
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleMember, frank.ID, at),
			fmt.Sprintf("AcceptInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at), "WorkspaceByID "+acme.ID.String())},
		{"declineWorkspaceInvitation", func() ([]string, error) {
			f := responding(frankToAcme)
			err := app.NewDeclineWorkspaceInvitation(f.accounts, f.invitations, f.tx, clockAt{clockNow, f.log}, f.mac).
				Execute(as(frank), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID))
			return f.log.calls, err
		}, append(respondedCalls(frank, frankToAcme, "ShareWorkspace"), "Now",
			fmt.Sprintf("DeclineInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at))},
	}
	for _, tt := range tests {
		if calls, err := tt.run(); err != nil || !slices.Equal(calls, tt.want) {
			t.Errorf("%s: calls = %q, %v; want %q", tt.name, calls, err, tt.want)
		}
	}
}
