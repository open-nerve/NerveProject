package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"regexp"
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

// credentialFixture is the credential lock over fakes that share one call
// log: a live session and a live token of userID, an active account.
type credentialFixture struct {
	log    *callLog
	creds  *fakeCredentials
	tokens *fakeAPITokens
	tx     *fakeTx
	logs   *bytes.Buffer
}

func newCredentialFixture() *credentialFixture {
	log := &callLog{}
	return &credentialFixture{
		log:    log,
		creds:  &fakeCredentials{log: log, account: app.LockedAccount{PasswordHash: "hashed:Tr0ub4dor&3", Active: true}, session: validCredential()},
		tokens: &fakeAPITokens{log: log, credential: validToken()},
		tx:     &fakeTx{},
		logs:   &bytes.Buffer{},
	}
}

func (f *credentialFixture) lock() app.CredentialLock {
	return app.CredentialLock{Locker: f.creds, Sessions: f.creds, APITokens: f.tokens}
}

func (f *credentialFixture) logger() *slog.Logger { return slog.New(slog.NewJSONHandler(f.logs, nil)) }

func (f *credentialFixture) createAPIToken() *app.CreateAPIToken {
	return app.NewCreateAPIToken(app.CreateAPITokenDeps{Lock: f.lock(), Tokens: f.tokens, Tx: f.tx, Clock: clocktest.At(now), Logger: f.logger()})
}

var (
	sessionActor = shared.Actor{UserID: userID, SessionID: sessionID}
	tokenActor   = shared.Actor{UserID: userID, APITokenID: tokenID}
)

func TestCreateAPIToken(t *testing.T) {
	f := newCredentialFixture()
	label, expires := "deploy", now.Add(7*24*time.Hour)

	got, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor),
		domain.APITokenSpec{Label: &label, Description: "ci", ExpiredAt: &expires})

	if err != nil {
		t.Fatal(err)
	}
	pat, ok := domain.ParsePAT(got.Token)
	if !ok || len(f.tokens.created) != 1 {
		t.Fatalf("Execute() token %q (parses %v), %d rows; want a token and one row", got.Token, ok, len(f.tokens.created))
	}
	n := f.tokens.created[0]
	if n.UserID != userID || !bytes.Equal(n.TokenHash, pat.Hash()) || n.Label != "deploy" || n.Description != "ci" ||
		n.ExpiredAt == nil || !n.ExpiredAt.Equal(expires) || !n.Now.Equal(now) || !isV7(n.ID) {
		t.Errorf("row = %+v, want the caller's token with the hash of %q", n, got.Token)
	}
	want := domain.APIToken{ID: n.ID, Label: "deploy", Description: "ci", ExpiredAt: &expires, CreatedAt: now}
	if got.ID != want.ID || got.Label != want.Label || got.Description != want.Description || !got.ExpiredAt.Equal(expires) ||
		got.LastUsed != nil || !got.CreatedAt.Equal(now) {
		t.Errorf("Execute() = %+v, want %+v", got.APIToken, want)
	}
	// Locked, the credential checked again under the lock, then inserted:
	// one transaction (M2 design 3.5).
	wantCalls := []string{"lock " + userID.String(), "session " + sessionID.String(), "insert " + n.ID.String()}
	if !slices.Equal(f.log.calls, wantCalls) || f.tx.calls != 1 {
		t.Errorf("calls = %q in %d transactions, want %q in one", f.log.calls, f.tx.calls, wantCalls)
	}
	logs := f.logs.String()
	if !strings.Contains(logs, `"msg":"API token created"`) || !strings.Contains(logs, `"user_id":"`+userID.String()) ||
		!strings.Contains(logs, `"token_id":"`+n.ID.String()) {
		t.Errorf("logs = %s, want the creation with user_id and token_id", logs)
	}
	assertNoSecret(t, logs, "token", []byte(got.Token))
	assertNoSecret(t, logs, "token bytes", pat[:])
	assertNoSecret(t, logs, "token hash", n.TokenHash)
}

// Whatever offset and precision the caller sends, the row and the answer
// hold the expiry the database stores, in UTC to the microsecond, so the
// answer reads back unchanged (M2 design 3.13).
func TestCreateAPITokenStoresAndAnswersTheExpiryInUTC(t *testing.T) {
	f := newCredentialFixture()
	sent := time.Date(2030, 1, 1, 12, 0, 0, 123456789, time.FixedZone("", 2*3600))

	got, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor), domain.APITokenSpec{ExpiredAt: &sent})

	if err != nil || len(f.tokens.created) != 1 {
		t.Fatalf("Execute() = %v with %d rows, want one row", err, len(f.tokens.created))
	}
	const want = "2030-01-01T10:00:00.123456Z"
	for _, held := range []struct {
		name string
		at   *time.Time
	}{{"row", f.tokens.created[0].ExpiredAt}, {"answer", got.ExpiredAt}} {
		if held.at == nil || held.at.Location() != time.UTC || held.at.Format(time.RFC3339Nano) != want {
			t.Errorf("the %s's expiry = %v, want %s in UTC", held.name, held.at, want)
		}
	}
}

// Without a label the token gets 32 hexadecimal digits, as Plane's
// uuid4().hex, different each time.
func TestCreateAPITokenGeneratesALabel(t *testing.T) {
	f := newCredentialFixture()
	ctx := shared.WithActor(context.Background(), sessionActor)

	first, err1 := f.createAPIToken().Execute(ctx, domain.APITokenSpec{})
	second, err2 := f.createAPIToken().Execute(ctx, domain.APITokenSpec{})

	hex32 := regexp.MustCompile(`^[0-9a-f]{32}$`)
	if err := errors.Join(err1, err2); err != nil || !hex32.MatchString(first.Label) || first.Label == second.Label ||
		f.tokens.created[0].Label != first.Label {
		t.Errorf("labels %q and %q (%v), want two different ones of 32 hexadecimal digits", first.Label, second.Label, err)
	}
	if first.Token == second.Token {
		t.Errorf("two tokens are both %q", first.Token)
	}
}

// A token may create tokens (M2 design 4.6): the lock checks that token
// again, not a session.
func TestCreateAPITokenWithAToken(t *testing.T) {
	f := newCredentialFixture()

	got, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), tokenActor), domain.APITokenSpec{})

	wantCalls := []string{"lock " + userID.String(), "token " + tokenID.String(), "insert " + got.ID.String()}
	if err != nil || !slices.Equal(f.log.calls, wantCalls) {
		t.Errorf("calls = %q, %v; want %q", f.log.calls, err, wantCalls)
	}
}

// Under the lock the caller's credential is checked again: one that a
// concurrent reset or logout ended since the request was authenticated
// creates nothing (M2 design 3.5, interleaving 3).
func TestCreateAPITokenRechecksTheCredentialUnderTheLock(t *testing.T) {
	other := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	tests := []struct {
		name   string
		actor  shared.Actor
		change func(*credentialFixture)
		reason string
	}{
		{"account gone", sessionActor, func(f *credentialFixture) { f.creds.accountErr = app.ErrNotFound }, "account does not exist"},
		{"account deactivated", sessionActor, func(f *credentialFixture) { f.creds.account.Active = false }, "account is deactivated"},
		{"session revoked", sessionActor, func(f *credentialFixture) { f.creds.session.Revoked = true }, "session is revoked"},
		{"session expired", sessionActor, func(f *credentialFixture) { f.creds.session.ExpiresAt = now }, "session has expired"},
		{"session gone", sessionActor, func(f *credentialFixture) { f.creds.sessionErr = app.ErrNotFound }, "session does not exist"},
		{"session of another account", sessionActor, func(f *credentialFixture) { f.creds.session.UserID = other }, "session belongs to another account"},
		{"token revoked", tokenActor, func(f *credentialFixture) { f.tokens.credential.Revoked = true }, "personal access token is revoked"},
		{"token expired", tokenActor, func(f *credentialFixture) { f.tokens.credential.ExpiredAt = &now }, "personal access token has expired"},
		{"token of another account", tokenActor, func(f *credentialFixture) { f.tokens.credential.UserID = other }, "personal access token belongs to another account"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newCredentialFixture()
			tt.change(f)

			_, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), tt.actor), domain.APITokenSpec{})

			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != 401 || err.Error() != "Authentication is required.: "+tt.reason {
				t.Errorf("Execute() = %v, want 401 because %s", err, tt.reason)
			}
			if len(f.tokens.created) != 0 {
				t.Errorf("inserted %+v, want nothing", f.tokens.created)
			}
		})
	}
}

func TestCreateAPITokenLockFailureIsNot401(t *testing.T) {
	f := newCredentialFixture()
	boom := errors.New("connection refused")
	f.creds.accountErr = boom

	_, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor), domain.APITokenSpec{})

	var se *shared.Error
	if !errors.Is(err, boom) || errors.As(err, &se) || len(f.tokens.created) != 0 {
		t.Errorf("Execute() = %v, inserted %d; want the database error and nothing inserted", err, len(f.tokens.created))
	}
}

// The spec is checked before anything is locked, before the transaction
// opens.
func TestCreateAPITokenChecksTheSpecFirst(t *testing.T) {
	f := newCredentialFixture()
	empty := ""

	_, err := f.createAPIToken().Execute(shared.WithActor(context.Background(), sessionActor), domain.APITokenSpec{Label: &empty})

	var se *shared.Error
	if !errors.As(err, &se) || se.Code != shared.CodeValidationFailed || len(f.log.calls) != 0 || f.tx.calls != 0 {
		t.Errorf("Execute() = %v after calls %q in %d transactions, want 422 before any, in none", err, f.log.calls, f.tx.calls)
	}
}
