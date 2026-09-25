package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/netip"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// dummyHash is the fixture's DummyHash: it verifies no password a test sends.
const dummyHash = "hashed:the dummy password"

type loginFixture struct {
	logins *fakeLogins
	hasher *fakeHasher
	tx     *fakeTx
	tokens *fakeTokens
	logs   *bytes.Buffer
	uc     *app.Login
}

// newLogin has one account, alice@corp.com, whose row holds hash.
func newLogin(hash string, active bool) *loginFixture {
	f := &loginFixture{
		logins: &fakeLogins{account: app.LoginAccount{ID: userID, PasswordHash: hash}, email: "alice@corp.com", active: active, hash: hash},
		hasher: &fakeHasher{},
		tx:     &fakeTx{},
		tokens: newFakeTokens(),
		logs:   &bytes.Buffer{},
	}
	f.uc = app.NewLogin(app.LoginDeps{
		Accounts:  f.logins,
		Locker:    f.logins,
		Passwords: f.logins,
		Sessions:  f.logins,
		Hasher:    f.hasher,
		Tx:        f.tx,
		Issuance:  issuance(f.tokens, fakeMAC{}),
		Clock:     clocktest.At(now),
		Logger:    slog.New(slog.NewJSONHandler(f.logs, nil)),
		DummyHash: dummyHash,
	})
	return f
}

var loginInput = app.LoginInput{
	Email:     "  Alice@Corp.com ",
	Password:  "Tr0ub4dor&3",
	UserAgent: "agent\x00/1",
	IP:        netip.MustParseAddr("203.0.113.7"),
}

func TestLoginSignsIn(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)

	tokens, err := f.uc.Execute(context.Background(), loginInput)
	if err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(f.logins.lookedUp, []string{"alice@corp.com"}) || !slices.Equal(f.hasher.verified, []string{"hashed:Tr0ub4dor&3"}) || f.hasher.calls != 0 {
		t.Errorf("looked up %q, verified against %q, hashed %d times; want the normalized address, the row's hash, no hash",
			f.logins.lookedUp, f.hasher.verified, f.hasher.calls)
	}
	if f.tx.calls != 1 || f.logins.locks != 1 || len(f.logins.outsideTx) != 0 || len(f.logins.hashUpdates) != 0 {
		t.Errorf("transactions %d, locks %d, outside one %q, hash updates %q; want the lock and the insert in one", f.tx.calls, f.logins.locks, f.logins.outsideTx, f.logins.hashUpdates)
	}
	if len(f.logins.sessions) != 1 {
		t.Fatalf("sessions = %d, want 1", len(f.logins.sessions))
	}
	s := f.logins.sessions[0]
	if s.UserID != userID || s.UserAgent != "agent/1" || s.IP != loginInput.IP || !s.ExpiresAt.Equal(now.Add(720*time.Hour)) || !s.Now.Equal(now) || !isV7(s.ID) {
		t.Errorf("session = %+v", s)
	}
	refresh, ok := domain.ParseRefreshToken(tokens.RefreshToken)
	if !ok || refresh.SessionID != s.ID || refresh.Generation != 0 || !bytes.Equal(refresh.SecretHash(), s.TokenHash) || !(fakeMAC{}).Verify(refresh.MACMessage(), refresh.Tag) {
		t.Errorf("refresh token %+v of session %+v; want generation 0, its secret's hash stored, tagged", refresh, s)
	}
	want := app.AccessClaims{UserID: userID, SessionID: s.ID, ExpiresAt: now.Add(15 * time.Minute)}
	if !slices.Equal(f.tokens.issued, []app.AccessClaims{want}) || tokens.AccessExpiresIn != 15*time.Minute || !tokens.RefreshExpiresAt.Equal(s.ExpiresAt) {
		t.Errorf("tokens = %+v, claims %+v; want claims %+v", tokens, f.tokens.issued, want)
	}
}

// An unknown address costs what a wrong password costs: one verification,
// with the current parameters, of the dummy hash (M2 design 3.9). An
// address that cannot be valid is not even looked up.
func TestLoginFailsAlikeForAnUnknownAddressAndAWrongPassword(t *testing.T) {
	tests := []struct {
		name, email, password string
		lookedUp              []string
		verified              string
	}{
		{"wrong password", "alice@corp.com", "Tr0ub4dor&4", []string{"alice@corp.com"}, "hashed:Tr0ub4dor&3"},
		{"unknown address", "bob@corp.com", "Tr0ub4dor&3", []string{"bob@corp.com"}, dummyHash},
		{"empty address", "  ", "Tr0ub4dor&3", nil, dummyHash},
		{"a NUL in the address", "alice\x00@corp.com", "Tr0ub4dor&3", nil, dummyHash},
		{"an address that is not UTF-8", "alice\xff@corp.com", "Tr0ub4dor&3", nil, dummyHash},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newLogin("hashed:Tr0ub4dor&3", true)

			_, err := f.uc.Execute(context.Background(), app.LoginInput{Email: tt.email, Password: tt.password})

			if !errors.Is(err, domain.ErrInvalidCredentials) {
				t.Errorf("Execute() = %v, want identity.invalid_credentials", err)
			}
			if !slices.Equal(f.logins.lookedUp, tt.lookedUp) || !slices.Equal(f.hasher.verified, []string{tt.verified}) || f.hasher.calls != 0 {
				t.Errorf("looked up %q, verified against %q, hashed %d times; want %q, [%q], 0", f.logins.lookedUp, f.hasher.verified, f.hasher.calls, tt.lookedUp, tt.verified)
			}
			if f.tx.calls != 0 || len(f.logins.sessions) != 0 {
				t.Errorf("transactions %d, sessions %d; want none", f.tx.calls, len(f.logins.sessions))
			}
		})
	}
}

// A deactivated account is told only to whoever knows its password, and
// only inside the transaction (M2 design 3.9).
func TestLoginRevealsDeactivationOnlyWithTheRightPassword(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", false)

	_, wrong := f.uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Tr0ub4dor&4"})
	locksAfterWrong := f.logins.locks
	_, right := f.uc.Execute(context.Background(), loginInput)

	if !errors.Is(wrong, domain.ErrInvalidCredentials) || locksAfterWrong != 0 {
		t.Errorf("wrong password: %v after %d locks; want identity.invalid_credentials before any lock", wrong, locksAfterWrong)
	}
	if !errors.Is(right, domain.ErrAccountDeactivated) || f.logins.locks != 1 || len(f.logins.sessions) != 0 {
		t.Errorf("right password: %v, %d locks, %d sessions; want identity.account_deactivated under the lock, no session", right, f.logins.locks, len(f.logins.sessions))
	}
}

// Other parameters: the password is hashed again outside the transaction
// and the new hash written under the lock (M2 design 3.8).
func TestLoginRehashesWhenTheParametersChanged(t *testing.T) {
	f := newLogin("old:Tr0ub4dor&3", true)

	if _, err := f.uc.Execute(context.Background(), loginInput); err != nil {
		t.Fatal(err)
	}

	if len(f.logins.hashTimes) != 1 || !f.logins.hashTimes[0].Equal(now) {
		t.Errorf("the hash was written at %v, want once at the clock's now", f.logins.hashTimes)
	}
	if f.hasher.calls != 1 || !slices.Equal(f.logins.hashUpdates, []string{"hashed:Tr0ub4dor&3"}) || len(f.logins.outsideTx) != 0 || len(f.logins.sessions) != 1 {
		t.Errorf("hashed %d times, wrote %q, outside the transaction %q, sessions %d; want one new hash written in the transaction",
			f.hasher.calls, f.logins.hashUpdates, f.logins.outsideTx, len(f.logins.sessions))
	}
}

// A concurrent login rehashed the same password between the snapshot and
// the lock: the new hash is verified once more, and the login goes through
// without writing a hash of its own (M2 design 3.5, interleaving 5).
func TestLoginVerifiesAgainWhenAConcurrentLoginRehashed(t *testing.T) {
	f := newLogin("old:Tr0ub4dor&3", true)
	f.hasher.onVerify = func() { f.logins.hash = "hashed:Tr0ub4dor&3" }

	_, err := f.uc.Execute(context.Background(), loginInput)

	if err != nil || len(f.logins.sessions) != 1 {
		t.Fatalf("Execute() = %v with %d sessions, want a session", err, len(f.logins.sessions))
	}
	if !slices.Equal(f.hasher.verified, []string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}) || f.tx.calls != 2 || len(f.logins.hashUpdates) != 0 {
		t.Errorf("verified against %q in %d transactions, wrote %q; want the snapshot, then the new hash, and no write",
			f.hasher.verified, f.tx.calls, f.logins.hashUpdates)
	}
}

func TestLoginFailsWhenThePasswordChangedMeanwhile(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)
	f.hasher.onVerify = func() { f.logins.hash = "hashed:N3w-password" }

	_, err := f.uc.Execute(context.Background(), loginInput)

	if !errors.Is(err, domain.ErrInvalidCredentials) || len(f.logins.sessions) != 0 {
		t.Errorf("Execute() = %v with %d sessions; want identity.invalid_credentials and none", err, len(f.logins.sessions))
	}
}

// The hash changes again under the second try: the login gives up.
func TestLoginFailsWhenTheHashChangesTwice(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)
	next := []string{"old:Tr0ub4dor&3", "hashed:Tr0ub4dor&3"}
	f.hasher.onVerify = func() { f.logins.hash, next = next[0], next[1:] }

	_, err := f.uc.Execute(context.Background(), loginInput)

	if !errors.Is(err, domain.ErrInvalidCredentials) || len(f.logins.sessions) != 0 || f.tx.calls != 2 {
		t.Errorf("Execute() = %v with %d sessions in %d transactions; want identity.invalid_credentials after two", err, len(f.logins.sessions), f.tx.calls)
	}
}

func TestLoginWhenTheHasherIsBusy(t *testing.T) {
	for _, email := range []string{"alice@corp.com", "bob@corp.com"} {
		f := newLogin("hashed:Tr0ub4dor&3", true)
		f.hasher.verifyErr = shared.ServerBusy(time.Second)

		_, err := f.uc.Execute(context.Background(), app.LoginInput{Email: email, Password: "Tr0ub4dor&3"})

		var se *shared.Error
		if !errors.As(err, &se) || se.Code != shared.CodeServerBusy || f.tx.calls != 0 {
			t.Errorf("%s: Execute() = %v after %d transactions; want server_busy before any", email, err, f.tx.calls)
		}
	}
}

// The log names the account and the session, never the address, the
// password or a token (M2 design 8.4).
func TestLoginLogsNoSecret(t *testing.T) {
	f := newLogin("hashed:Tr0ub4dor&3", true)
	tokens, err := f.uc.Execute(context.Background(), loginInput)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.uc.Execute(context.Background(), app.LoginInput{Email: "alice@corp.com", Password: "Wr0ng-password", IP: loginInput.IP})
	_, _ = f.uc.Execute(context.Background(), app.LoginInput{Email: "bob@corp.com", Password: "Tr0ub4dor&3", IP: loginInput.IP})

	logs := f.logs.String()
	session := f.logins.sessions[0]
	for _, want := range []string{
		`"msg":"signed in","user_id":"` + userID.String() + `","session_id":"` + session.ID.String() + `","ip":"203.0.113.7"`,
		`"msg":"sign-in failed","reason":"invalid_credentials","ip":"203.0.113.7","user_id":"` + userID.String() + `"`,
		`"msg":"sign-in failed","reason":"invalid_credentials","ip":"203.0.113.7"}`,
	} {
		if !strings.Contains(logs, want) {
			t.Errorf("logs lack %s:\n%s", want, logs)
		}
	}
	for _, secret := range []string{"Tr0ub4dor", "Wr0ng", "alice", "bob", tokens.AccessToken, tokens.RefreshToken, string(session.TokenHash)} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs contain %q:\n%s", secret, logs)
		}
	}
}
