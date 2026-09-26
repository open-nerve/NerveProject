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

// newPasswordChange is ChangePassword over the credential fixture, whose
// account zqxwv@corp.com has the password Tr0ub4dor&3.
func newPasswordChange() (*credentialFixture, *fakeHasher, *app.ChangePassword) {
	f, h := newCredentialFixture(), &fakeHasher{}
	f.creds.email = "zqxwv@corp.com"
	return f, h, app.NewChangePassword(app.ChangePasswordDeps{
		Accounts: f.creds, Lock: f.lock(), Passwords: f.creds, Sessions: f.creds, Hasher: h,
		Rules: domain.NewPasswordRules(), Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

var change = app.ChangePasswordInput{Current: "Tr0ub4dor&3", New: "N3w-Passw0rd!"}

func TestChangePassword(t *testing.T) {
	f, h, uc := newPasswordChange()

	err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change)

	// The snapshot is read and argon2 runs outside the transaction; the
	// lock, the recheck of the session and the writes are one transaction
	// (M2 design 3.5). The session the request came with stays.
	want := []string{
		"read " + userID.String() + " outside tx", "lock " + userID.String(), "session " + sessionID.String(),
		"password " + userID.String() + " hashed:N3w-Passw0rd!",
		"revoke password_changed sessions of " + userID.String() + " but " + sessionID.String(),
	}
	if err != nil || !slices.Equal(f.log.calls, want) || f.tx.calls != 1 {
		t.Errorf("Execute() = %v, calls %q in %d transactions; want %q in one", err, f.log.calls, f.tx.calls, want)
	}
	if !slices.Equal(h.verified, []string{"hashed:Tr0ub4dor&3"}) || h.calls != 1 || !slices.Equal(f.creds.writtenAt, []time.Time{now, now}) {
		t.Errorf("verified %q, %d hashes, writes at %v; want the snapshot verified, one hash, both writes at %v", h.verified, h.calls, f.creds.writtenAt, now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"password changed"`) || !strings.Contains(logs, `"user_id":"`+userID.String()) {
		t.Errorf("logs = %s, want the change with user_id", logs)
	}
	assertNoSecret(t, logs, "current password", []byte(change.Current))
	assertNoSecret(t, logs, "new password", []byte(change.New))
	assertNoSecret(t, logs, "snapshot hash", []byte("hashed:"+change.Current))
	assertNoSecret(t, logs, "new hash", []byte("hashed:"+change.New))
}

// A personal access token has no session: every session is revoked, and
// the lock checks the token again (M2 design 3.5).
func TestChangePasswordWithATokenRevokesEverySession(t *testing.T) {
	f, _, uc := newPasswordChange()

	err := uc.Execute(shared.WithActor(context.Background(), tokenActor), change)

	want := []string{
		"lock " + userID.String(), "token " + tokenID.String(), "password " + userID.String() + " hashed:N3w-Passw0rd!",
		"revoke password_changed sessions of " + userID.String() + " but " + uuid.Nil().String(),
	}
	if err != nil || !slices.Equal(f.log.calls[1:], want) {
		t.Errorf("Execute() = %v, calls %q; want %q after the read", err, f.log.calls, want)
	}
}

// The new password is checked, with the account's address, before any
// argon2 (M2 design 3.8).
func TestChangePasswordChecksTheNewPasswordFirst(t *testing.T) {
	tests := []struct{ name, password, code string }{
		{"empty", "", shared.FieldRequired},
		{"weak", "password", shared.FieldWeakPassword},
		{"the stem of the address", "Zqxwv123!", shared.FieldCommonPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, h, uc := newPasswordChange()

			err := uc.Execute(shared.WithActor(context.Background(), sessionActor), app.ChangePasswordInput{Current: change.Current, New: tt.password})

			var se *shared.Error
			if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 1 ||
				se.Fields[0].Field != "new_password" || se.Fields[0].Code != tt.code {
				t.Errorf("Execute() = %v, want validation_failed with new_password %s", err, tt.code)
			}
			if len(h.verified)+h.calls != 0 || len(f.log.calls) != 1 {
				t.Errorf("argon2 ran %d times, calls %q; want only the read", len(h.verified)+h.calls, f.log.calls)
			}
		})
	}
}

func TestChangePasswordWithAWrongCurrentPassword(t *testing.T) {
	f, h, uc := newPasswordChange()

	err := uc.Execute(shared.WithActor(context.Background(), sessionActor), app.ChangePasswordInput{Current: "Tr0ub4dor&4", New: change.New})

	if !errors.Is(err, domain.ErrCurrentPasswordIncorrect) || h.calls != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() = %v after %d hashes and %d transactions, want identity.current_password_incorrect before either", err, h.calls, f.tx.calls)
	}
}

// When the hash under the lock is not the snapshot, the current password
// is verified against it and the transaction runs once more, with the new
// password hashed once (M2 design 3.5).
func TestChangePasswordAfterAConcurrentChange(t *testing.T) {
	tests := []struct {
		name     string
		hashes   []string // what the row holds after the first, second, … verification
		err      error
		verified []string
		changed  bool
	}{
		{"a login hashed the password again", []string{"hashed:Tr0ub4dor&3"}, nil,
			[]string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}, true},
		{"the password changed", []string{"hashed:Other-Passw0rd!"}, domain.ErrCurrentPasswordIncorrect,
			[]string{"old:Tr0ub4dor&3", "hashed:Other-Passw0rd!"}, false},
		{"it changed twice", []string{"hashed:Tr0ub4dor&3", "old:Tr0ub4dor&3"}, domain.ErrCurrentPasswordIncorrect,
			[]string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, h, uc := newPasswordChange()
			f.creds.account.PasswordHash = "old:Tr0ub4dor&3"
			h.onVerify = func() {
				if i := len(h.verified) - 1; i < len(tt.hashes) {
					f.creds.account.PasswordHash = tt.hashes[i] // a transaction that commits meanwhile
				}
			}

			err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change)

			if !errors.Is(err, tt.err) || !slices.Equal(h.verified, tt.verified) || h.calls != 1 {
				t.Errorf("Execute() = %v, verified %q, %d hashes; want %v, %q, one hash", err, h.verified, h.calls, tt.err, tt.verified)
			}
			if f.log.calls[0] != "read "+userID.String()+" outside tx" || slices.Contains(f.log.calls[1:], f.log.calls[0]) {
				t.Errorf("calls %q, want the account read once, first: the retry redoes only the check and the transaction", f.log.calls)
			}
			if changed := f.creds.account.PasswordHash == "hashed:N3w-Passw0rd!"; changed != tt.changed {
				t.Errorf("the row holds %q, want the new password: %v", f.creds.account.PasswordHash, tt.changed)
			}
		})
	}
}

// Under the lock the caller's credential is checked again: a session that
// a concurrent reset revoked changes nothing (M2 design 3.5).
func TestChangePasswordRechecksTheCredentialUnderTheLock(t *testing.T) {
	f, _, uc := newPasswordChange()
	f.creds.session.Revoked = true

	err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change)

	if !errors.Is(err, shared.Unauthenticated()) || len(f.creds.writtenAt) != 0 {
		t.Errorf("Execute() = %v, %d writes; want 401 and none", err, len(f.creds.writtenAt))
	}
}

func TestChangePasswordErrors(t *testing.T) {
	boom, busy := errors.New("connection refused"), shared.ServerBusy(time.Second)
	authed := shared.WithActor(context.Background(), sessionActor)
	tests := []struct {
		name   string
		ctx    context.Context
		change func(*credentialFixture, *fakeHasher)
		want   error
	}{
		{"without an actor", context.Background(), func(*credentialFixture, *fakeHasher) {}, shared.Unauthenticated()},
		{"the account gone", authed, func(f *credentialFixture, _ *fakeHasher) { f.creds.accountErr = app.ErrNotFound }, shared.Unauthenticated()},
		{"the database down", authed, func(f *credentialFixture, _ *fakeHasher) { f.creds.accountErr = boom }, boom},
		{"no argon2 slot to verify", authed, func(_ *credentialFixture, h *fakeHasher) { h.verifyErr = busy }, busy},
		{"no argon2 slot to hash", authed, func(_ *credentialFixture, h *fakeHasher) { h.err = busy }, busy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, h, uc := newPasswordChange()
			tt.change(f, h)

			if err := uc.Execute(tt.ctx, change); !errors.Is(err, tt.want) || len(f.creds.writtenAt) != 0 {
				t.Errorf("Execute() = %v, %d writes; want %v and none", err, len(f.creds.writtenAt), tt.want)
			}
		})
	}
}

// A write that fails fails the change, and its transaction with it: the
// change is not logged as done.
func TestChangePasswordWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	tests := []struct {
		name string
		fail func(*fakeCredentials)
	}{
		{"the hash", func(c *fakeCredentials) { c.hashErr = boom }},
		{"the sessions", func(c *fakeCredentials) { c.revokeErr = boom }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, _, uc := newPasswordChange()
			tt.fail(f.creds)

			if err := uc.Execute(shared.WithActor(context.Background(), sessionActor), change); !errors.Is(err, boom) || strings.Contains(f.logs.String(), "password changed") {
				t.Errorf("Execute() = %v, logs %s; want %v and no change logged", err, f.logs.String(), boom)
			}
		})
	}
}
