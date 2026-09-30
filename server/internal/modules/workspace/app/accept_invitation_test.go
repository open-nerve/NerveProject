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
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	// The invitees: dave (declined in acme, daveToAcme), erin, whose
	// membership of acme, an admin's, has ended, and frank, never a member.
	dave       = app.AccountState{ID: uuid.NewV7(), Email: "dave@corp.com", Active: true}
	erin       = app.AccountState{ID: uuid.NewV7(), Email: "erin@corp.com", Active: true}
	frank      = app.AccountState{ID: uuid.NewV7(), Email: "frank@corp.com", Active: true}
	erinInAcme = domain.Membership{ID: uuid.NewV7(), WorkspaceID: acme.ID, MemberID: erin.ID, Role: shared.RoleAdmin, CreatedAt: now}
	// frankToAcme is the invitation the refusals answer, unless a case
	// names another.
	frankToAcme = invitationTo(frank, acme, shared.RoleMember)
)

// invitationTo is a pending invitation of user's address to w as role.
func invitationTo(user app.AccountState, w domain.Workspace, role shared.Role) domain.Invitation {
	return domain.Invitation{ID: uuid.NewV7(), WorkspaceID: w.ID, Email: user.Email, Role: role, CreatedAt: now, CreatedByID: &alice.ID}
}

// responding is invitationsFixture holding inv too, erin's ended membership
// of acme, and the accounts: alice, bob, carol (deactivated), dave, erin
// and frank.
func responding(inv ...domain.Invitation) *invitationsFixture {
	f := newInvitations()
	f.invitations.invitations = append(f.invitations.invitations, inv...)
	f.invitations.memberships[acme.ID] = append(f.invitations.memberships[acme.ID], erinInAcme)
	f.accounts = &fakeAccounts{log: f.log, accounts: []app.AccountState{alice, bob, carol, dave, erin, frank}}
	return f
}

func (f *invitationsFixture) accept() *app.AcceptWorkspaceInvitation {
	return app.NewAcceptWorkspaceInvitation(app.AcceptInvitationDeps{Accounts: f.accounts, Invitations: f.invitations, Tx: f.tx,
		Clock: clockAt{at: clockNow}, MAC: f.mac})
}

// respondedCalls are the calls up to the answer to inv by user: the token,
// the account's lock, the invitation's read, the workspace's lock by
// lockName, the invitation's lock.
func respondedCalls(user app.AccountState, inv domain.Invitation, lockName string) []string {
	return []string{"Verify " + inv.ID.String(), "ShareAccount " + user.ID.String(), "InvitationByID " + inv.ID.String(),
		lockName + " " + inv.WorkspaceID.String(), "LockInvitation " + inv.ID.String()}
}

// Accepting never changes an active membership (M3 design 3.8, 9.1): bob,
// acme's member and beta's guest, keeps his role whatever the invitation's,
// higher, lower or the same, and only the invitation is consumed; erin's
// ended membership is restored with the invitation's role; frank is
// inserted with it. Each in one transaction, after the account's, the
// workspace's and the invitation's locks; the answer is the workspace with
// the caller's role as it now is, and the invitation is gone.
func TestAcceptWorkspaceInvitation(t *testing.T) {
	at := clockNow.Format(time.RFC3339Nano)
	tests := []struct {
		name  string
		user  app.AccountState
		inv   domain.Invitation
		w     domain.Workspace
		role  shared.Role // the answer's
		write string      // the membership's, "" for none
	}{
		{"an active member, invited higher", bob, invitationTo(bob, acme, shared.RoleAdmin), acme, shared.RoleMember, ""},
		{"an active member, invited lower", bob, invitationTo(bob, acme, shared.RoleGuest), acme, shared.RoleMember, ""},
		{"an active member, invited the same", bob, invitationTo(bob, acme, shared.RoleMember), acme, shared.RoleMember, ""},
		{"an active guest of another workspace, invited higher", bob, invitationTo(bob, beta, shared.RoleAdmin), beta, shared.RoleGuest, ""},
		{"a former admin, invited as a guest", erin, invitationTo(erin, acme, shared.RoleGuest), acme, shared.RoleGuest,
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at)},
		{"a former member elsewhere, new here", erin, invitationTo(erin, beta, shared.RoleMember), beta, shared.RoleMember,
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", erin.ID, beta.ID, shared.RoleMember, erin.ID, at)},
		{"never a member", frank, invitationTo(frank, acme, shared.RoleAdmin), acme, shared.RoleAdmin,
			fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleAdmin, frank.ID, at)},
	}
	for _, tt := range tests {
		f := responding(tt.inv)
		got, err := f.accept().Execute(as(tt.user), tt.inv.ID, tokenOf(f.mac, tt.inv.ID))
		want := tt.w
		want.Role = tt.role
		if err != nil || got != want {
			t.Errorf("%s: Execute() = %+v, %v; want %+v", tt.name, got, err, want)
		}
		wantCalls := append(respondedCalls(tt.user, tt.inv, "LockWorkspace"), "MemberOf "+tt.w.ID.String()+" "+tt.user.ID.String())
		if tt.write != "" {
			wantCalls = append(wantCalls, tt.write)
		}
		wantCalls = append(wantCalls, fmt.Sprintf("AcceptInvitation %s by %s at %s", tt.inv.ID, tt.user.ID, at), "WorkspaceByID "+tt.w.ID.String())
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, f.tx.calls, wantCalls)
		}
		if slices.ContainsFunc(f.invitations.invitations, func(inv domain.Invitation) bool { return inv.ID == tt.inv.ID }) {
			t.Errorf("%s: the invitation is still there", tt.name)
		}
	}
}

// responseRefusals are the refusals accepting and declining share, each
// with the calls it makes (M3 design 3.8, 8.2): a wrong token reads nothing
// and opens no transaction; an account deactivated or gone, read under its
// lock, is 401 before the invitation is read; an invitation not there,
// deleted meanwhile, of a workspace deleted meanwhile, or of another
// workspace when read again under the lock (M3 design 3.6 convention 2) is
// workspace.invitation_not_found; another address is
// workspace.invitation_email_mismatch, also for a declined invitation; a
// declined one, also declined meanwhile, workspace.invitation_responded.
// None writes. lockName is the workspace's lock.
func responseRefusals(lockName string) []responseRefusal {
	stranger := app.AccountState{ID: uuid.NewV7(), Email: "stranger@corp.com", Active: true}
	decided := respondedCalls(frank, frankToAcme, lockName)
	nobodys := uuid.NewV7()
	return []responseRefusal{
		{"another invitation's token", frank, frankToAcme.ID, tokenOf(fakeMAC{key: "the instance's key"}, carolToAcme.ID), nil,
			domain.ErrInvitationNotFound, decided[:1]},
		{"a deactivated account", carol, carolToAcme.ID, "", nil, shared.Unauthenticated(), respondedCalls(carol, carolToAcme, lockName)[:2]},
		{"an account gone", stranger, frankToAcme.ID, "", nil, shared.Unauthenticated(), respondedCalls(stranger, frankToAcme, lockName)[:2]},
		{"no such invitation", frank, nobodys, "", nil, domain.ErrInvitationNotFound,
			[]string{"Verify " + nobodys.String(), "ShareAccount " + frank.ID.String(), "InvitationByID " + nobodys.String()}},
		{"acme deleted while the lock waited", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": app.ErrNotFound} }, domain.ErrInvitationNotFound, decided[:4]},
		{"deleted while the lock waited", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.invitations = nil } }, domain.ErrInvitationNotFound, decided},
		{"of beta when read again under the lock", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations[len(f.invitations.invitations)-1].WorkspaceID = beta.ID }
			},
			domain.ErrInvitationNotFound, decided},
		{"another address", alice, frankToAcme.ID, "", nil, domain.ErrInvitationEmailMismatch, respondedCalls(alice, frankToAcme, lockName)},
		{"another address, declined", alice, daveToAcme.ID, "", nil, domain.ErrInvitationEmailMismatch, respondedCalls(alice, daveToAcme, lockName)},
		{"declined", dave, daveToAcme.ID, "", nil, domain.ErrInvitationResponded, respondedCalls(dave, daveToAcme, lockName)},
		{"declined while the lock waited", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) {
				f.invitations.onLock = func() { f.invitations.invitations[len(f.invitations.invitations)-1].RespondedAt = &declined }
			},
			domain.ErrInvitationResponded, decided},
	}
}

type responseRefusal struct {
	name  string
	user  app.AccountState
	id    uuid.UUID
	token string // "" for the invitation's
	set   func(f *invitationsFixture)
	want  error
	calls []string
}

// responseFailures are the failures accepting and declining share before
// they write, each the error injected, never a problem of the contract.
func responseFailures(failure error, lockName string) []responseRefusal {
	decided := respondedCalls(frank, frankToAcme, lockName)
	return []responseRefusal{
		{"the account's lock failed", frank, frankToAcme.ID, "", func(f *invitationsFixture) { f.accounts.err = failure }, failure, decided[:2]},
		{"the read failed", frank, frankToAcme.ID, "", func(f *invitationsFixture) { f.invitations.readErr = failure }, failure, decided[:3]},
		{"the workspace's lock failed", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} }, failure, decided[:4]},
		{"the invitation's lock failed", frank, frankToAcme.ID, "",
			func(f *invitationsFixture) { f.invitations.onLock = func() { f.invitations.readErr = failure } }, failure, decided},
	}
}

// Each refusal and failure is the answer, and nothing is written: see
// responseRefusals and responseFailures; a failure of a write, erin's
// restore among them, of the answer's read or of the commit is itself too,
// and no workspace is answered. Without a caller it is 401 and nothing is
// read.
func TestAcceptWorkspaceInvitationRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	cases := append(responseRefusals("LockWorkspace"), responseFailures(failure, "LockWorkspace")...)
	decided := append(respondedCalls(frank, frankToAcme, "LockWorkspace"), "MemberOf "+acme.ID.String()+" "+frank.ID.String())
	at := clockNow.Format(time.RFC3339Nano)
	created := fmt.Sprintf("CreateMember %s in %s as %d by %s at %s", frank.ID, acme.ID, shared.RoleMember, frank.ID, at)
	accepted := fmt.Sprintf("AcceptInvitation %s by %s at %s", frankToAcme.ID, frank.ID, at)
	for _, step := range []struct {
		name  string
		set   func(f *invitationsFixture)
		calls []string
	}{
		{"MemberOf", func(f *invitationsFixture) { f.invitations.failing = map[string]error{"MemberOf": failure} }, decided},
		{"CreateMember", func(f *invitationsFixture) { f.invitations.memberErr = failure }, append(slices.Clone(decided), created)},
		{"AcceptInvitation", func(f *invitationsFixture) { f.invitations.failing = map[string]error{"AcceptInvitation": failure} },
			append(slices.Clone(decided), created, accepted)},
		{"WorkspaceByID", func(f *invitationsFixture) { f.invitations.failing = map[string]error{"WorkspaceByID": failure} },
			append(slices.Clone(decided), created, accepted, "WorkspaceByID "+acme.ID.String())},
	} {
		cases = append(cases, responseRefusal{step.name + " failed", frank, frankToAcme.ID, "", step.set, failure, step.calls})
	}
	erinToAcme := invitationTo(erin, acme, shared.RoleGuest)
	cases = append(cases,
		responseRefusal{"RestoreMember failed", erin, erinToAcme.ID, "", func(f *invitationsFixture) {
			f.invitations.invitations = append(f.invitations.invitations, erinToAcme)
			f.invitations.failing = map[string]error{"RestoreMember": failure}
		}, failure, append(respondedCalls(erin, erinToAcme, "LockWorkspace"), "MemberOf "+acme.ID.String()+" "+erin.ID.String(),
			fmt.Sprintf("RestoreMember %s as %d by %s at %s", erinInAcme.ID, shared.RoleGuest, erin.ID, at))},
		responseRefusal{"the commit failed", frank, frankToAcme.ID, "", func(f *invitationsFixture) { f.tx.commitErr = failure }, failure,
			append(slices.Clone(decided), created, accepted, "WorkspaceByID "+acme.ID.String())})
	for _, tt := range cases {
		f := responding(frankToAcme)
		if tt.set != nil {
			tt.set(f)
		}
		token := tt.token
		if token == "" {
			token = tokenOf(f.mac, tt.id)
		}
		got, err := f.accept().Execute(as(tt.user), tt.id, token)
		if !errors.Is(err, tt.want) || got != (domain.Workspace{}) {
			t.Errorf("%s: Execute() = %+v, %v; want no workspace and %v", tt.name, got, err, tt.want)
		}
		var se *shared.Error
		if tt.want == failure && errors.As(err, &se) {
			t.Errorf("%s: Execute() = %v, which is also a problem of the contract", tt.name, err)
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
	if _, err := f.accept().Execute(context.Background(), frankToAcme.ID, tokenOf(f.mac, frankToAcme.ID)); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
