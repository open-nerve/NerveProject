package app_test

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// fakeMemberships is the workspace module's Deactivator
// (app.MembershipDeactivator): it logs its call, its account and address,
// to the log it shares, and answers err.
type fakeMemberships struct {
	log *callLog
	err error
}

func (f *fakeMemberships) DeactivateMemberships(ctx context.Context, userID uuid.UUID, email string) error {
	f.log.add(ctx, "deactivate the memberships of "+userID.String()+" at "+email)
	return f.err
}

// lockedEmail is the address the credential lock reads under the lock: not
// one the use case could have from anywhere else.
const lockedEmail = "alice@locked.example"

func (f *credentialFixture) deactivate(memberships *fakeMemberships) *app.Deactivate {
	f.creds.account.Email = lockedEmail
	return app.NewDeactivate(app.DeactivateDeps{
		Lock: f.lock(), Users: f.creds, Profiles: f.creds, Sessions: f.creds, Memberships: memberships, Tx: f.tx, Clock: clocktest.At(now),
		Logger: f.logger(),
	})
}

// The administrator's deactivation takes no credential lock: it has no
// caller's credential to check again.
func (f *adminFixture) deactivate(memberships *fakeMemberships) *app.Deactivate {
	return app.NewDeactivate(app.DeactivateDeps{
		Accounts: f.store, Users: f.store, Profiles: f.store, Sessions: f.store, Memberships: memberships, Tx: f.tx, Clock: clocktest.At(now),
		Logger: f.logger(),
	})
}

// deactivateCalls are a deactivation's calls by the caller with credential,
// in order: the lock, the credential checked again, the account, its
// onboarding, its sessions, and last its memberships, of the account at the
// address read under the lock.
func deactivateCalls(credential string) []string {
	return []string{
		"lock " + userID.String(), credential, "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
		"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
		"deactivate the memberships of " + userID.String() + " at " + lockedEmail,
	}
}

// One transaction takes the lock and checks the caller's credential again,
// then writes in the global lock order: the account, the profile, the
// sessions, all of them, whatever the credential (M2 design 3.5, 6.4); and
// last the memberships, of the account at the address read under the lock
// (M3 design 3.6, 3.9).
func TestDeactivate(t *testing.T) {
	tests := []struct {
		name       string
		actor      shared.Actor
		credential string
	}{
		{"with a session", sessionActor, "session " + sessionID.String()},
		{"with a token", tokenActor, "token " + tokenID.String()},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()

			err := f.deactivate(&fakeMemberships{log: f.log}).Execute(shared.WithActor(context.Background(), tt.actor))

			want := deactivateCalls(tt.credential)
			if err != nil || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.creds.writtenAt, []time.Time{now, now, now}) {
				t.Errorf("Execute() = %v, calls %q in %d transactions at %v; want %q in one at %v", err, f.log.calls, f.tx.calls, f.creds.writtenAt, want, now)
			}
			logs := f.logs.String()
			if !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"self"`) {
				t.Errorf("logs = %s, want the deactivation with user_id, the sessions revoked and by self", logs)
			}
		})
	}
}

// A credential that a concurrent reset revoked since authentication
// deactivates nothing (M2 design 3.5).
func TestDeactivateRechecksTheCredentialUnderTheLock(t *testing.T) {
	f := newCredentialFixture()
	f.tokens.credential.Revoked = true

	err := f.deactivate(&fakeMemberships{log: f.log}).Execute(shared.WithActor(context.Background(), tokenActor))

	if !errors.Is(err, shared.Unauthenticated()) || len(f.creds.writtenAt) != 0 || slices.ContainsFunc(f.log.calls, isMemberships) {
		t.Errorf("Execute() = %v, %d writes, calls %q; want 401 and none", err, len(f.creds.writtenAt), f.log.calls)
	}
}

// isMemberships reports whether call is the memberships' deactivation.
func isMemberships(call string) bool { return strings.HasPrefix(call, "deactivate the memberships") }

// The memberships' refusals and failures, as they come: a 409 of the
// workspace or project module, which identity does not import (stand-ins
// with their kind, code and detail), or a failure (M3 design 3.9, 9.4).
var membershipsErrors = []error{
	shared.NewError(shared.KindConflict, "workspace.sole_admin",
		"The workspace would be left without an admin: its only active admin cannot leave it, nor can his membership end while it "+
			"has other active members. It must first be given another admin, or be deleted."),
	shared.NewError(shared.KindConflict, "project.sole_admin",
		"The project would be left without an admin: its only active admin cannot leave it, nor can his membership end while it has "+
			"other active members. It must first be given another admin, or be deleted."),
	errors.New("connection reset"),
}

// A failed write fails the deactivation, which is then not logged; so does
// the memberships' refusal or failure, which comes back as itself, the
// first problem in its chain the one it was, out of the one transaction:
// what its function returned, which it rolls back on. The calls are the
// deactivation's up to the failing one, which a failing fake does not log,
// or up to the memberships' one call; none after it, none tried again.
func TestDeactivateWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	type failing struct {
		name  string
		fail  func(*fakeCredentials, *fakeMemberships)
		want  error
		calls int
	}
	tests := []failing{
		{"the account", func(c *fakeCredentials, _ *fakeMemberships) { c.deactivateErr = boom }, boom, 2},
		{"the onboarding", func(c *fakeCredentials, _ *fakeMemberships) { c.resetErr = boom }, boom, 3},
		{"the sessions", func(c *fakeCredentials, _ *fakeMemberships) { c.revokeErr = boom }, boom, 4},
	}
	for _, err := range membershipsErrors {
		tests = append(tests, failing{"the memberships: " + err.Error(), func(_ *fakeCredentials, m *fakeMemberships) { m.err = err }, err, 6})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()
			m := &fakeMemberships{log: f.log}
			tt.fail(f.creds, m)

			err := f.deactivate(m).Execute(shared.WithActor(context.Background(), sessionActor))

			if !errors.Is(err, tt.want) || strings.Contains(f.logs.String(), "account deactivated") || f.tx.calls != 1 {
				t.Errorf("Execute() = %v in %d transactions, logs %s; want %v in one and no deactivation logged", err, f.tx.calls, f.logs.String(),
					tt.want)
			}
			if first := firstProblem(err); first != firstProblem(tt.want) {
				t.Errorf("Execute() = %v, answered as %v; want %v", err, first, firstProblem(tt.want))
			}
			if f.tx.returned != err {
				t.Errorf("Execute() = %v, its transaction's function returned %v; want the same: the error it rolled back on", err, f.tx.returned)
			}
			if want := deactivateCalls("session " + sessionID.String())[:tt.calls]; !slices.Equal(f.log.calls, want) {
				t.Errorf("calls %q, want %q: nothing after the failing step, none tried again", f.log.calls, want)
			}
		})
	}
}

// firstProblem is the first *shared.Error in err's chain, the one the API
// answers, or nil.
func firstProblem(err error) error {
	var se *shared.Error
	if errors.As(err, &se) {
		return se
	}
	return nil
}

func TestDeactivateWithoutAnActor(t *testing.T) {
	f := newCredentialFixture()

	if err := f.deactivate(&fakeMemberships{log: f.log}).Execute(context.Background()); !errors.Is(err, shared.Unauthenticated()) ||
		len(f.log.calls) != 0 {
		t.Errorf("Execute() = %v after calls %q, want 401 before any", err, f.log.calls)
	}
}

// deactivateByEmailCalls are the calls of `nerve users deactivate` of
// alice@corp.com, in order: the lock by the address, the account, its
// onboarding, its sessions, and last its memberships, at that address.
var deactivateByEmailCalls = []string{
	"lock alice@corp.com", "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
	"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
	"deactivate the memberships of " + userID.String() + " at alice@corp.com",
}

// `nerve users deactivate` locks the account by its normalized address and
// writes what the self-service deactivation writes, in the same order, the
// memberships of the account at that address, the one it locked (M2
// decision 3, design 3.17; M3 design 3.9).
func TestDeactivateByEmail(t *testing.T) {
	f := newAdminFixture()

	got, err := f.deactivate(&fakeMemberships{log: f.log}).ExecuteByEmail(context.Background(), " Alice@Corp.com ")

	if want := (app.DeactivateResult{Email: "alice@corp.com", Sessions: 1}); err != nil || got != want {
		t.Fatalf("ExecuteByEmail() = %+v, %v; want %+v", got, err, want)
	}
	if want := deactivateByEmailCalls; !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	if logs := f.logs.String(); !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"cli"`) {
		t.Errorf("logs = %s, want the deactivation with user_id and the sessions revoked, by cli", logs)
	}
}

func TestDeactivateByEmailOfAnUnknownAccount(t *testing.T) {
	f := newAdminFixture()

	_, err := f.deactivate(&fakeMemberships{log: f.log}).ExecuteByEmail(context.Background(), "carol@corp.com")

	if !errors.Is(err, domain.ErrAccountNotFound) || !slices.Equal(f.log.calls, []string{"lock carol@corp.com"}) || f.logs.Len() != 0 {
		t.Errorf("ExecuteByEmail() = %v after calls %q, logs %s; want identity.account_not_found after the lock alone", err, f.log.calls, f.logs.String())
	}
}

// A failed write fails the administrator's deactivation too, which is then
// not logged; so does the memberships' refusal or failure, as itself, out
// of the one transaction, as Execute's: the command prints its detail. The
// calls are the command's up to the failing one, none after it, none tried
// again.
func TestDeactivateByEmailWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	f := newAdminFixture()
	f.store.revokeErr = boom

	if _, err := f.deactivate(&fakeMemberships{log: f.log}).ExecuteByEmail(context.Background(), "alice@corp.com"); !errors.Is(err, boom) ||
		f.tx.returned != err || f.logs.Len() != 0 || !slices.Equal(f.log.calls, deactivateByEmailCalls[:3]) {
		t.Errorf("ExecuteByEmail() = %v, its transaction's function returned %v, calls %q, logs %s; want %v, the same, after %q and "+
			"nothing logged", err, f.tx.returned, f.log.calls, f.logs.String(), boom, deactivateByEmailCalls[:3])
	}
	for _, want := range membershipsErrors {
		f := newAdminFixture()
		_, err := f.deactivate(&fakeMemberships{log: f.log, err: want}).ExecuteByEmail(context.Background(), "alice@corp.com")
		if !errors.Is(err, want) || firstProblem(err) != firstProblem(want) || f.tx.returned != err || f.logs.Len() != 0 ||
			!slices.Equal(f.log.calls, deactivateByEmailCalls) {
			t.Errorf("ExecuteByEmail() with the memberships failing = %v, its transaction's function returned %v, calls %q, logs %s; "+
				"want %v as itself, the same, after %q, and nothing logged", err, f.tx.returned, f.log.calls, f.logs.String(), want,
				deactivateByEmailCalls)
		}
	}
}
