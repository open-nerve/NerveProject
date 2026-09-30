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

// session is the caller's session: the credential CallerLock checks.
var session = uuid.NewV7()

// withSession is user as the caller, signed in with session.
func withSession(user app.AccountState) context.Context {
	return shared.WithActor(context.Background(), shared.Actor{UserID: user.ID, SessionID: session})
}

// create is CreateWorkspaceInvitations over f's fakes and clock.
func (f *invitationsFixture) create(clock app.Clock) *app.CreateWorkspaceInvitations {
	return app.NewCreateWorkspaceInvitations(app.CreateInvitationsDeps{Caller: f.caller, Invitations: f.invitations, Profiles: f.profiles,
		Auth: f.auth, Tx: f.tx, Clock: clock, MAC: f.mac})
}

// createCalls are the calls of a creation by user in w up to the inserts:
// the credential, the workspace's lock, the decision, the members, their
// addresses, the invitations.
func createCalls(user app.AccountState, w domain.Workspace, active ...uuid.UUID) []string {
	return []string{
		fmt.Sprintf("LockCaller %s session %s token %s at %s", user.ID, session, uuid.Nil(), clockNow.Format(time.RFC3339Nano)),
		"ShareWorkspaceBySlug " + w.Slug,
		"Authorize " + user.ID.String() + " workspace_invitation.create on " + w.ID.String() + "/" + uuid.Nil().String(),
		"ListMembers " + w.ID.String(),
		fmt.Sprintf("PublicProfiles %v", active),
		"ListInvitations " + w.ID.String(),
	}
}

// fieldsOf are err's field problems as "field code".
func fieldsOf(err error) []string {
	var e *shared.Error
	if !errors.As(err, &e) {
		return nil
	}
	var out []string
	for _, f := range e.Fields {
		out = append(out, f.Field+" "+f.Code)
	}
	return out
}

// A creation checks the batch, then in one transaction takes the caller's
// credential lock, the workspace's FOR SHARE and the decision, reads the
// active members' addresses and the invitations, and inserts the rows in
// the order of their addresses, by the caller at the clock's time. The
// answer is the batch in the request's order, as stored, each with its
// token. An address invited to another workspace is free. Two admins, two
// workspaces.
func TestCreateWorkspaceInvitationsLocksDecidesChecksThenInserts(t *testing.T) {
	for _, tt := range []struct {
		user   app.AccountState
		w      domain.Workspace
		batch  []domain.NewInvitation
		active []uuid.UUID
		insert string
		want   []domain.NewInvitation
	}{
		{alice, acme, []domain.NewInvitation{{Email: " Zoe@corp.com ", Role: shared.RoleMember}, {Email: "erin@corp.com", Role: shared.RoleGuest}}, []uuid.UUID{alice.ID, bob.ID},
			"erin@corp.com as 5, zoe@corp.com as 15", []domain.NewInvitation{{Email: "zoe@corp.com", Role: shared.RoleMember}, {Email: "erin@corp.com", Role: shared.RoleGuest}}},
		{bob, beta, []domain.NewInvitation{{Email: "carol@corp.com", Role: shared.RoleAdmin}}, []uuid.UUID{bob.ID},
			"carol@corp.com as 20", []domain.NewInvitation{{Email: "carol@corp.com", Role: shared.RoleAdmin}}},
	} {
		f := newInvitations()
		got, err := f.create(clockAt{at: clockNow}).Execute(withSession(tt.user), tt.w.Slug, tt.batch)
		if err != nil || len(got) != len(tt.want) {
			t.Fatalf("%s invites to %s: %+v, %v; want %d invitations", tt.user.Email, tt.w.Slug, got, err, len(tt.want))
		}
		for i, inv := range got {
			want := domain.InvitationWithToken{Invitation: domain.Invitation{ID: inv.ID, WorkspaceID: tt.w.ID, Email: tt.want[i].Email,
				Role: tt.want[i].Role, CreatedAt: now, CreatedByID: &tt.user.ID}, Token: tokenOf(f.mac, inv.ID)}
			if !sameInvitations([]domain.InvitationWithToken{inv}, []domain.InvitationWithToken{want}) || inv.ID == uuid.Nil() {
				t.Errorf("%s invites to %s: invitation %d = %+v, want %+v", tt.user.Email, tt.w.Slug, i, inv, want)
			}
		}
		wantCalls := append(createCalls(tt.user, tt.w, tt.active...),
			fmt.Sprintf("CreateInvitations %s in %s by %s at %s", tt.insert, tt.w.ID, tt.user.ID, clockNow.Format(time.RFC3339Nano)))
		if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
			t.Errorf("%s invites to %s: calls = %q in %d transactions, want %q in one", tt.user.Email, tt.w.Slug, f.log.calls, f.tx.calls, wantCalls)
		}
	}
}

// txClock is clockNow; at each read it notes how many transactions tx had
// begun.
type txClock struct {
	tx    *fakeTx
	reads []int
}

func (c *txClock) Now() time.Time {
	c.reads = append(c.reads, c.tx.calls)
	return clockNow
}

// The clock is read once, before the transaction, whose first call is the
// credential lock: the rows are new, and the credential is checked at the
// request's time (spec P3 2.8).
func TestCreatingInvitationsReadsTheClockBeforeItsTransaction(t *testing.T) {
	f := newInvitations()
	clock := &txClock{tx: f.tx}
	if _, err := f.create(clock).Execute(withSession(alice), "acme", []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}}); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(clock.reads, []int{0}) || f.tx.calls != 1 {
		t.Errorf("the clock was read with %v transactions begun, of %d; want once, before the one", clock.reads, f.tx.calls)
	}
}

// A caller with a personal access token is locked as that credential:
// CallerLock checks his token, as it would his session (M2 design 3.5).
func TestCreatingInvitationsLocksACallersPersonalAccessToken(t *testing.T) {
	f := newInvitations()
	token := uuid.NewV7()
	ctx := shared.WithActor(context.Background(), shared.Actor{UserID: alice.ID, APITokenID: token})
	if _, err := f.create(clockAt{at: clockNow}).Execute(ctx, "acme", []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}}); err != nil {
		t.Fatal(err)
	}
	want := fmt.Sprintf("LockCaller %s session %s token %s at %s", alice.ID, uuid.Nil(), token, clockNow.Format(time.RFC3339Nano))
	if f.log.calls[0] != want {
		t.Errorf("the first call = %q, want %q", f.log.calls[0], want)
	}
}

// A batch the request alone refuses is refused before anything is read or
// locked: no transaction.
func TestCreateWorkspaceInvitationsChecksTheBatchFirst(t *testing.T) {
	f := newInvitations()
	_, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme",
		[]domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}, {Email: "ZOE@corp.com", Role: 15}, {Email: "not an address", Role: 20}, {Email: "yan@corp.com", Role: 10}})
	want := []string{"invitations[1].email duplicate", "invitations[2].email invalid_format", "invitations[3].role invalid_format"}
	if !errors.Is(err, shared.Invalid()) || !slices.Equal(fieldsOf(err), want) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() = %v (%q), calls %q in %d transactions; want 422 %q and nothing", err, fieldsOf(err), f.log.calls, f.tx.calls, want)
	}
}

// Under the lock, the addresses of active members are not_allowed and
// those of undeleted invitations, pending or declined, duplicate, each on
// its index in the request, in one 422; nothing is inserted. An ended
// membership refuses nothing, nor does another workspace's invitation.
func TestCreateWorkspaceInvitationsRefusesMembersAndInvitedAddresses(t *testing.T) {
	f := newInvitations()
	batch := []domain.NewInvitation{{Email: "bob@corp.com", Role: 5}, {Email: "frank@corp.com", Role: 5}, {Email: "dave@corp.com", Role: 5}, {Email: "CAROL@corp.com", Role: 5}, {Email: "alice@corp.com", Role: 20},
		{Email: "erin@corp.com", Role: 5}}

	_, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme", batch)

	want := []string{"invitations[0].email not_allowed", "invitations[2].email duplicate", "invitations[3].email duplicate",
		"invitations[4].email not_allowed"}
	if !errors.Is(err, shared.Invalid()) || !slices.Equal(fieldsOf(err), want) {
		t.Errorf("Execute() = %v (%q), want 422 %q", err, fieldsOf(err), want)
	}
	if wantCalls := createCalls(alice, acme, alice.ID, bob.ID); !slices.Equal(f.log.calls, wantCalls) {
		t.Errorf("calls = %q, want %q: no insert", f.log.calls, wantCalls)
	}
	// carol's membership of acme ended: without her invitation, she can be
	// invited again.
	f = newInvitations()
	f.invitations.invitations = []domain.Invitation{daveToAcme, erinToBeta}
	if got, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme", []domain.NewInvitation{{Email: "carol@corp.com", Role: 5}}); err != nil ||
		len(got) != 1 {
		t.Errorf("inviting carol, whose membership ended: %+v, %v; want her invitation", got, err)
	}
}

// When a concurrent batch committed an address after the check, the
// unique key refuses it: the answer is the check's, on the address's index
// in the request as sent, not in the order the rows were inserted.
func TestADuplicateFromTheUniqueKeyNamesTheRequestsIndex(t *testing.T) {
	f := newInvitations()
	f.invitations.taken = "zoe@corp.com"
	_, err := f.create(clockAt{at: clockNow}).Execute(withSession(alice), "acme",
		[]domain.NewInvitation{{Email: "yan@corp.com", Role: 5}, {Email: "zoe@corp.com", Role: 5}, {Email: "abe@corp.com", Role: 5}})
	if want := []string{"invitations[1].email duplicate"}; !errors.Is(err, shared.Invalid()) || !slices.Equal(fieldsOf(err), want) {
		t.Errorf("Execute() = %v (%q), want 422 %q", err, fieldsOf(err), want)
	}
	if last := f.log.calls[len(f.log.calls)-1]; last != fmt.Sprintf("CreateInvitations abe@corp.com as 5, yan@corp.com as 5, zoe@corp.com as 5 in %s by %s at %s",
		acme.ID, alice.ID, clockNow.Format(time.RFC3339Nano)) {
		t.Errorf("the insert = %q, want the rows in the order of their addresses", last)
	}
}

// Each refusal and failure is the answer, and nothing after it runs: a
// revoked credential is identity's 401; a workspace not there or not
// visible, workspace.not_found; the Authorizer's forbidden; a failure of
// any read or of the insert is itself, never a 404 or a 422. Without a
// caller it is 401 and nothing runs.
func TestCreateWorkspaceInvitationsRefusals(t *testing.T) {
	failure := errors.New("connection reset")
	all := append(createCalls(alice, acme, alice.ID, bob.ID),
		fmt.Sprintf("CreateInvitations zoe@corp.com as 5 in %s by %s at %s", acme.ID, alice.ID, clockNow.Format(time.RFC3339Nano)))
	tests := []struct {
		name  string
		user  app.AccountState
		slug  string
		set   func(f *invitationsFixture)
		want  error
		calls int // how many of the calls of all, or of the calls up to the decision
	}{
		{"the credential revoked", alice, "acme", func(f *invitationsFixture) { f.caller.err = shared.Unauthenticated() }, shared.Unauthenticated(), 1},
		{"the credential lock failed", alice, "acme", func(f *invitationsFixture) { f.caller.err = failure }, failure, 1},
		{"no such workspace", alice, "nothing", nil, domain.ErrNotFound, 2},
		{"the workspace lock failed", alice, "acme", func(f *invitationsFixture) { f.invitations.lockErrs = map[string]error{"acme": failure} },
			failure, 2},
		{"not visible", carol, "acme", nil, domain.ErrNotFound, 3},
		{"forbidden", bob, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{bob.ID, acme.ID}: shared.Forbidden()} },
			shared.Forbidden(), 3},
		{"the Authorizer failed", alice, "acme", func(f *invitationsFixture) { f.auth.errs = map[grantKey]error{{alice.ID, acme.ID}: failure} },
			failure, 3},
		{"the members failed", alice, "acme", func(f *invitationsFixture) { f.invitations.membersErr = failure }, failure, 4},
		{"their profiles failed", alice, "acme", func(f *invitationsFixture) { f.profiles.err = failure }, failure, 5},
		{"the invitations failed", alice, "acme", func(f *invitationsFixture) { f.invitations.listErr = failure }, failure, 6},
		{"the insert failed", alice, "acme", func(f *invitationsFixture) { f.invitations.createErr = failure }, failure, 7},
		{"the commit failed", alice, "acme", func(f *invitationsFixture) { f.tx.commitErr = failure }, failure, 7},
	}
	for _, tt := range tests {
		f := newInvitations()
		if tt.set != nil {
			tt.set(f)
		}
		got, err := f.create(clockAt{at: clockNow}).Execute(withSession(tt.user), tt.slug, []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}})
		if !errors.Is(err, tt.want) || got != nil || (tt.want == failure && (errors.Is(err, domain.ErrNotFound) || errors.Is(err, shared.Invalid()))) {
			t.Errorf("%s: Execute() = %+v, %v; want %v", tt.name, got, err, tt.want)
		}
		want := slices.Clone(all[:tt.calls])
		if tt.user != alice {
			want = createCalls(tt.user, acme)[:tt.calls]
		}
		if tt.slug != "acme" {
			want[1] = "ShareWorkspaceBySlug " + tt.slug
		}
		if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
			t.Errorf("%s: calls = %q in %d transactions, want %q in one", tt.name, f.log.calls, f.tx.calls, want)
		}
	}
	f := newInvitations()
	if _, err := f.create(clockAt{at: clockNow}).Execute(context.Background(), "acme", []domain.NewInvitation{{Email: "zoe@corp.com", Role: 5}}); !errors.Is(err,
		shared.Unauthenticated()) || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() without an actor = %v, calls %q; want 401 unauthorized and nothing", err, f.log.calls)
	}
}
