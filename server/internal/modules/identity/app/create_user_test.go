package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

type createUserFixture struct {
	store  *fakeStore
	tx     *fakeTx
	hasher *fakeHasher
	logs   *bytes.Buffer
	uc     *app.CreateUser
}

func newCreateUser() *createUserFixture {
	f := &createUserFixture{store: &fakeStore{}, tx: &fakeTx{}, hasher: &fakeHasher{}, logs: &bytes.Buffer{}}
	f.uc = app.NewCreateUser(app.CreateUserDeps{
		Rules: domain.NewPasswordRules(), Hasher: f.hasher, Tx: f.tx, Users: f.store, Profiles: f.store,
		Clock: clocktest.At(now), Logger: slog.New(slog.NewJSONHandler(f.logs, nil)),
	})
	return f
}

// The account and its profile are inserted in one transaction, with the
// normalized address and the hash; no session is opened. There is no
// sign-up policy to ask: the administrator creates accounts either way
// (M2 decision 2).
func TestCreateUser(t *testing.T) {
	f := newCreateUser()

	email, err := f.uc.Execute(context.Background(), "  Carol@Corp.COM ", "Tr0ub4dor&3")

	if err != nil || email != "carol@corp.com" {
		t.Fatalf("Execute() = %q, %v; want carol@corp.com", email, err)
	}
	if len(f.store.users) != 1 || len(f.store.profiles) != 1 || len(f.store.sessions) != 0 || f.tx.calls != 1 || len(f.store.outsideTx) != 0 ||
		f.hasher.calls != 1 {
		t.Fatalf("%d accounts, %d profiles, %d sessions in %d transactions (outside: %v) after %d hashes; want one account and profile in one, no session, one hash",
			len(f.store.users), len(f.store.profiles), len(f.store.sessions), f.tx.calls, f.store.outsideTx, f.hasher.calls)
	}
	u := f.store.users[0]
	if u.Email != "carol@corp.com" || u.PasswordHash != "hashed:Tr0ub4dor&3" || u.DisplayName != "carol" || !u.Now.Equal(now) ||
		!isV7(u.ID) || f.store.profiles[0] != u.ID {
		t.Errorf("account = %+v with profile of %v, want carol's with the hash, at %v", u, f.store.profiles[0], now)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"account created","user_id":"`+u.ID.String()+`","by":"cli"`) {
		t.Errorf("logs = %s, want the creation with user_id, by cli", logs)
	}
	assertNoSecret(t, logs, "password", []byte("Tr0ub4dor&3"))
	assertNoSecret(t, logs, "hash", []byte(u.PasswordHash))
}

// A bad address and a weak password are reported at once, before hashing;
// nothing is inserted.
func TestCreateUserChecksTheAddressAndThePassword(t *testing.T) {
	f := newCreateUser()

	_, err := f.uc.Execute(context.Background(), "not-an-address", "short")

	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(se.Fields) != 2 || f.hasher.calls != 0 || len(f.store.users) != 0 {
		t.Errorf("Execute() = %+v after %d hashes, %d accounts; want 422 on both fields, before hashing, nothing inserted", err, f.hasher.calls, len(f.store.users))
	}
}

// An address in use is identity.email_taken, and nothing is logged.
func TestCreateUserWithATakenAddress(t *testing.T) {
	f := newCreateUser()
	f.store.createErr = domain.ErrEmailTaken

	if _, err := f.uc.Execute(context.Background(), "carol@corp.com", "Tr0ub4dor&3"); !errors.Is(err, domain.ErrEmailTaken) || f.logs.Len() != 0 ||
		len(f.store.profiles) != 0 {
		t.Errorf("Execute() = %v, logs %s, %d profiles; want identity.email_taken, no log, no profile", err, f.logs.String(), len(f.store.profiles))
	}
}
