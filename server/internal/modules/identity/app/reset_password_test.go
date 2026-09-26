package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
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

// adminFixture is the administrator's use cases over fakeAdmin, sharing
// one call log.
type adminFixture struct {
	log    *callLog
	store  *fakeAdmin
	tx     *fakeTx
	hasher *fakeHasher
	logs   *bytes.Buffer
}

func newAdminFixture() *adminFixture {
	log := &callLog{}
	return &adminFixture{log: log, store: newFakeAdmin(log), tx: &fakeTx{}, hasher: &fakeHasher{}, logs: &bytes.Buffer{}}
}

func (f *adminFixture) logger() *slog.Logger { return slog.New(slog.NewJSONHandler(f.logs, nil)) }

func (f *adminFixture) resetPassword() *app.ResetPassword {
	return app.NewResetPassword(app.ResetPasswordDeps{
		Accounts: f.store, Passwords: f.store, Sessions: f.store, APITokens: f.store, Hasher: f.hasher,
		Rules: domain.NewPasswordRules(), Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger(),
	})
}

const newPassword = "N3w-Passw0rd!"

// One transaction locks the account by its normalized address, writes the
// hash, then revokes every session and every token, in the global lock
// order (M2 design 3.5).
func TestResetPassword(t *testing.T) {
	f := newAdminFixture()

	got, err := f.resetPassword().Execute(context.Background(), "  Alice@Corp.COM ", newPassword)

	if want := (app.ResetPasswordResult{Email: "alice@corp.com", Sessions: 1, APITokens: 3}); err != nil || got != want {
		t.Fatalf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	want := []string{
		"lock alice@corp.com", "password " + userID.String() + " hashed:" + newPassword,
		"revoke password_reset sessions of " + userID.String() + " but " + uuid.Nil().String(),
		"revoke the tokens of " + userID.String(),
	}
	if !slices.Equal(f.log.calls, want) || f.tx.calls != 1 || !slices.Equal(f.store.writtenAt, []time.Time{now, now, now}) {
		t.Errorf("calls %q in %d transactions at %v; want %q in one at %v", f.log.calls, f.tx.calls, f.store.writtenAt, want, now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"password reset","user_id":"`+userID.String()+`","revoked_sessions":1,"revoked_api_tokens":3,"by":"cli"`) {
		t.Errorf("logs = %s, want the reset with user_id and what it revoked, by cli", logs)
	}
	assertNoSecret(t, logs, "password", []byte(newPassword))
	assertNoSecret(t, logs, "hash", []byte("hashed:"+newPassword))
}

// The password rules apply, with the account's address for the stem rule;
// a refused password is neither hashed nor written.
func TestResetPasswordChecksTheNewPassword(t *testing.T) {
	tests := []struct {
		name, email, password, code string
	}{
		{"weak", "alice@corp.com", "short", shared.FieldWeakPassword},
		{"the address's stem", "zebracorn@corp.com", "Zebracorn1!", shared.FieldCommonPassword},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newAdminFixture()

			_, err := f.resetPassword().Execute(context.Background(), tt.email, tt.password)

			var se *shared.Error
			if !errors.As(err, &se) || len(se.Fields) != 1 || se.Fields[0].Field != "password" || se.Fields[0].Code != tt.code ||
				f.hasher.calls != 0 || len(f.log.calls) != 0 {
				t.Errorf("Execute() = %+v after %d hashes and calls %q; want 422 %s on password, nothing else", err, f.hasher.calls, f.log.calls, tt.code)
			}
		})
	}
}

// An address no account has is identity.account_not_found: nothing is
// written, nothing logged.
func TestResetPasswordOfAnUnknownAccount(t *testing.T) {
	f := newAdminFixture()

	_, err := f.resetPassword().Execute(context.Background(), "carol@corp.com", newPassword)

	if !errors.Is(err, domain.ErrAccountNotFound) || !slices.Equal(f.log.calls, []string{"lock carol@corp.com"}) || f.logs.Len() != 0 {
		t.Errorf("Execute() = %v after calls %q, logs %s; want identity.account_not_found after the lock alone", err, f.log.calls, f.logs.String())
	}
}

func TestResetPasswordWhenAWriteFails(t *testing.T) {
	boom := errors.New("connection reset")
	for _, fail := range []func(*fakeAdmin){
		func(s *fakeAdmin) { s.hashErr = boom },
		func(s *fakeAdmin) { s.revokeErr = boom },
	} {
		f := newAdminFixture()
		fail(f.store)

		if _, err := f.resetPassword().Execute(context.Background(), "alice@corp.com", newPassword); !errors.Is(err, boom) || f.logs.Len() != 0 {
			t.Errorf("Execute() = %v, logs %s; want %v and nothing logged", err, f.logs.String(), boom)
		}
	}
}
