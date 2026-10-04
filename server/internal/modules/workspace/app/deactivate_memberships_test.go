package app_test

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/domain"
)

// deactivating runs the Deactivator over f's fakes, with the clock logging
// its reads among the calls, in tx, as identity's deactivation does: it is
// the last step of that transaction (M3 design 3.9). user is the account
// deactivated, at his address.
func deactivating(f *membersFixture, tx *fakeTx, user app.AccountState) error {
	d := app.NewDeactivator(f.workspaces, f.projects, clockAt{clockNow, f.log})
	return tx.WithinTx(context.Background(), func(ctx context.Context) error { return d.DeactivateMemberships(ctx, user.ID, user.Email) })
}

// deactivationCalls are the calls of the ending of user's memberships of
// workspaces, in id order: those workspaces locked, the only admin asked,
// the invitations the two deletes write locked, the clock read, every
// invitation to his address deleted, the pending ones of those he leaves
// with no active member, his memberships of the workspaces ended, then the
// project cascade's step, called once across them; each write by him at
// the clock's one time.
func deactivationCalls(user app.AccountState, workspaces ...uuid.UUID) []string {
	at := clockNow.Format(time.RFC3339Nano)
	return []string{
		"LockMemberWorkspaces " + user.ID.String(),
		fmt.Sprintf("SoleAdmin %v %s", workspaces, user.ID),
		fmt.Sprintf("LockInvitationsToDelete %v %s %s", workspaces, user.ID, user.Email),
		"Now",
		fmt.Sprintf("DeleteInvitationsTo %s by %s at %s", user.Email, user.ID, at),
		fmt.Sprintf("DeleteInvitationsOfWorkspacesLeftEmpty %v %s by %s at %s", workspaces, user.ID, user.ID, at),
		fmt.Sprintf("EndWorkspaceMemberships %v %s by %s at %s", workspaces, user.ID, user.ID, at),
		fmt.Sprintf("EndMemberships %v %s by %s at %s", workspaces, user.ID, user.ID, at),
	}
}

// The Deactivator ends bob's memberships of acme, a member's, and of beta,
// a guest's, the two workspaces whose active member he is, in their id
// order; the invitations to his address and his project memberships with
// them, by him at the clock's one time, read after the workspaces' locks,
// the check of the only admin and the invitations' lock (M3 design 3.3,
// 3.6 convention 6, 3.9), all in the transaction it runs in. carol, whose
// membership of acme ended, has none: her workspaces are none, and the
// calls come all the same, over none, so that the invitations to her
// address go too.
func TestDeactivateMembershipsEndsEveryMembership(t *testing.T) {
	for _, tt := range []struct {
		user       app.AccountState
		workspaces []uuid.UUID
	}{{bob, []uuid.UUID{acme.ID, beta.ID}}, {carol, nil}} {
		f, tx := newMembers(), &fakeTx{}

		err := deactivating(f, tx, tt.user)

		if want := deactivationCalls(tt.user, tt.workspaces...); err != nil || !slices.Equal(f.log.calls, want) {
			t.Errorf("%s: DeactivateMemberships() = %v, calls\n%q\nwant nil,\n%q", tt.user.Email, err, f.log.calls, want)
		}
		for _, w := range tt.workspaces {
			if m, found, err := f.workspaces.MemberOf(context.Background(), w, tt.user.ID); err != nil || !found || m.IsActive {
				t.Errorf("%s's membership of %s after the deactivation: %+v, %v, %v; want it ended", tt.user.Email, w, m, found, err)
			}
		}
	}
}

// The only active admin of a workspace that has another active member
// is refused: alice, acme's, with bob its member, gets
// workspace.sole_admin itself, the 409 of the contract (M3 design 3.7 rule
// 2), before the invitations' lock, before the clock is read and before
// any write; her membership stays.
func TestDeactivateMembershipsRefusesTheOnlyAdmin(t *testing.T) {
	f, tx := newMembers(), &fakeTx{}

	err := deactivating(f, tx, alice)

	if !errors.Is(err, domain.ErrSoleAdmin) {
		t.Errorf("DeactivateMemberships() = %v; want %v", err, domain.ErrSoleAdmin)
	}
	answeredAs(t, "the only admin", err, domain.ErrSoleAdmin)
	if want := deactivationCalls(alice, acme.ID)[:2]; !slices.Equal(f.log.calls, want) {
		t.Errorf("calls\n%q\nwant\n%q", f.log.calls, want)
	}
	if m, _, _ := f.workspaces.MemberOf(context.Background(), acme.ID, alice.ID); !m.IsActive {
		t.Error("alice's membership of acme after the refusal: ended; want it active")
	}
}

// Each failure of a step, and the project cascade's refusal, is the
// answer as it came, the first *shared.Error in its chain the one
// injected or none (a 500); no step runs after it, none is tried again.
func TestDeactivateMembershipsFailsAtEachStep(t *testing.T) {
	failure := errors.New("connection reset")
	calls := deactivationCalls(bob, acme.ID, beta.ID)
	for _, tt := range []struct {
		name  string
		set   func(f *membersFixture)
		want  error
		calls []string
	}{
		{"the lock", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"LockMemberWorkspaces": failure} }, failure, calls[:1]},
		{"the check", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"SoleAdmin": failure} }, failure, calls[:2]},
		{"the invitations' lock", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"LockInvitationsToDelete": failure} },
			failure, calls[:3]},
		{"the invitations", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"DeleteInvitationsTo": failure} }, failure,
			calls[:5]},
		{"the invitations of the workspaces left empty", func(f *membersFixture) {
			f.workspaces.endErrs = map[string]error{"DeleteInvitationsOfWorkspacesLeftEmpty": failure}
		}, failure, calls[:6]},
		{"the memberships", func(f *membersFixture) { f.workspaces.endErrs = map[string]error{"EndWorkspaceMemberships": failure} }, failure,
			calls[:7]},
		{"the projects' step", func(f *membersFixture) { f.projects.errs = map[string]error{"EndMemberships": failure} }, failure, calls},
		{"the only admin of a project", func(f *membersFixture) { f.projects.errs = map[string]error{"EndMemberships": projectSoleAdmin} },
			projectSoleAdmin, calls},
	} {
		f, tx := newMembers(), &fakeTx{}
		tt.set(f)

		err := deactivating(f, tx, bob)

		if !errors.Is(err, tt.want) {
			t.Errorf("%s failing: DeactivateMemberships() = %v; want %v", tt.name, err, tt.want)
		}
		answeredAs(t, tt.name, err, tt.want)
		if !slices.Equal(f.log.calls, tt.calls) {
			t.Errorf("%s failing: calls\n%q\nwant\n%q", tt.name, f.log.calls, tt.calls)
		}
	}
}
