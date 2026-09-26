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

func (f *credentialFixture) deactivate() *app.Deactivate {
	return app.NewDeactivate(app.DeactivateDeps{
		Lock: f.lock(), Users: f.creds, Profiles: f.creds, Sessions: f.creds, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

// The administrator's deactivation takes no credential lock: it has no
// caller's credential to check again.
func (f *adminFixture) deactivate() *app.Deactivate {
	return app.NewDeactivate(app.DeactivateDeps{
		Accounts: f.store, Users: f.store, Profiles: f.store, Sessions: f.store, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

// One transaction takes the lock and checks the caller's credential again,
// then writes in the global lock order: the account, the profile, the
// sessions, all of them, whatever the credential (M2 design 3.5, 6.4).
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

			err := f.deactivate().Execute(shared.WithActor(context.Background(), tt.actor))

			want := []string{
				"lock " + userID.String(), tt.credential, "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
				"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
			}
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

	err := f.deactivate().Execute(shared.WithActor(context.Background(), tokenActor))

	if !errors.Is(err, shared.Unauthenticated()) || len(f.creds.writtenAt) != 0 {
		t.Errorf("Execute() = %v, %d writes; want 401 and none", err, len(f.creds.writtenAt))
	}
}

// A failed write fails the deactivation, which is then not logged.
func TestDeactivateWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	tests := []struct {
		name string
		fail func(*fakeCredentials)
	}{
		{"the account", func(c *fakeCredentials) { c.deactivateErr = boom }},
		{"the onboarding", func(c *fakeCredentials) { c.resetErr = boom }},
		{"the sessions", func(c *fakeCredentials) { c.revokeErr = boom }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()
			tt.fail(f.creds)

			if err := f.deactivate().Execute(shared.WithActor(context.Background(), sessionActor)); !errors.Is(err, boom) || strings.Contains(f.logs.String(), "account deactivated") {
				t.Errorf("Execute() = %v, logs %s; want %v and no deactivation logged", err, f.logs.String(), boom)
			}
		})
	}
}

func TestDeactivateWithoutAnActor(t *testing.T) {
	f := newCredentialFixture()

	if err := f.deactivate().Execute(context.Background()); !errors.Is(err, shared.Unauthenticated()) || len(f.log.calls) != 0 {
		t.Errorf("Execute() = %v after calls %q, want 401 before any", err, f.log.calls)
	}
}

// `nerve users deactivate` locks the account by its normalized address and
// writes what the self-service deactivation writes, in the same order
// (M2 decision 3, design 3.17).
func TestDeactivateByEmail(t *testing.T) {
	f := newAdminFixture()

	got, err := f.deactivate().ExecuteByEmail(context.Background(), " Alice@Corp.com ")

	if want := (app.DeactivateResult{Email: "alice@corp.com", Sessions: 1}); err != nil || got != want {
		t.Fatalf("ExecuteByEmail() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{
		"lock alice@corp.com", "deactivate " + userID.String(), "reset onboarding of " + userID.String(),
		"revoke deactivated sessions of " + userID.String() + " but " + uuid.Nil().String(),
	}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	if logs := f.logs.String(); !strings.Contains(logs, `"msg":"account deactivated","user_id":"`+userID.String()+`","revoked_sessions":1,"by":"cli"`) {
		t.Errorf("logs = %s, want the deactivation with user_id and the sessions revoked, by cli", logs)
	}
}

func TestDeactivateByEmailOfAnUnknownAccount(t *testing.T) {
	f := newAdminFixture()

	_, err := f.deactivate().ExecuteByEmail(context.Background(), "carol@corp.com")

	if !errors.Is(err, domain.ErrAccountNotFound) || !slices.Equal(f.log.calls, []string{"lock carol@corp.com"}) || f.logs.Len() != 0 {
		t.Errorf("ExecuteByEmail() = %v after calls %q, logs %s; want identity.account_not_found after the lock alone", err, f.log.calls, f.logs.String())
	}
}

// A failed write fails the administrator's deactivation too, which is then
// not logged.
func TestDeactivateByEmailWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	f := newAdminFixture()
	f.store.revokeErr = boom

	if _, err := f.deactivate().ExecuteByEmail(context.Background(), "alice@corp.com"); !errors.Is(err, boom) || f.logs.Len() != 0 {
		t.Errorf("ExecuteByEmail() = %v, logs %s; want %v and nothing logged", err, f.logs.String(), boom)
	}
}
