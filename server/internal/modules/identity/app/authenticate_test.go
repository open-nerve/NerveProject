package app_test

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var (
	userID    = uuid.MustParse("0199a2b4-0000-7000-8000-000000000001")
	sessionID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")
)

func newAuthenticate(cred app.SessionCredential, credErr error) (*app.Authenticate, *fakeTokens, *fakeStore) {
	tokens := newFakeTokens()
	store := &fakeStore{credential: cred, credErr: credErr}
	uc := app.NewAuthenticate(app.AuthenticateDeps{
		AccessTokens: tokens, Sessions: store, APITokens: &fakeAPITokens{log: &callLog{}}, Touch: &fakeAPITokens{}, Clock: clocktest.At(now),
		Logger: slog.New(slog.DiscardHandler),
	})
	return uc, tokens, store
}

func validCredential() app.SessionCredential {
	return app.SessionCredential{UserID: userID, ExpiresAt: now.Add(time.Hour), UserActive: true}
}

func TestAuthenticateAValidToken(t *testing.T) {
	uc, tokens, store := newAuthenticate(validCredential(), nil)
	token, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(time.Minute)})

	actor, err := uc.Execute(context.Background(), token)

	if err != nil || actor != (shared.Actor{UserID: userID, SessionID: sessionID}) {
		t.Errorf("Execute() = %+v, %v", actor, err)
	}
	if len(store.credentials) != 1 || store.credentials[0] != sessionID {
		t.Errorf("sessions looked up = %v, want one lookup of %v", store.credentials, sessionID)
	}
	if len(tokens.verifiedAt) != 1 || !tokens.verifiedAt[0].Equal(now) {
		t.Errorf("token verified at %v, want once at the clock's %v", tokens.verifiedAt, now)
	}
}

func TestAuthenticateRejects(t *testing.T) {
	other := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	tests := []struct {
		name    string
		cred    func(*app.SessionCredential)
		credErr error
		token   string
		reason  string
	}{
		{"a token that is not ours", nil, nil, "garbage", "signature is invalid"},
		{"a refresh token as bearer", nil, nil, domain.RefreshTokenPrefix + "AAAA", "signature is invalid"},
		{"an expired access token", nil, nil, "expired", "access token expired"},
		{"an unknown session", nil, app.ErrNotFound, "", "session does not exist"},
		{"another account's session", func(c *app.SessionCredential) { c.UserID = other }, nil, "", "session belongs to another account"},
		{"a revoked session", func(c *app.SessionCredential) { c.Revoked = true }, nil, "", "session is revoked"},
		{"a session expiring now", func(c *app.SessionCredential) { c.ExpiresAt = now }, nil, "", "session has expired"},
		{"a deactivated account", func(c *app.SessionCredential) { c.UserActive = false }, nil, "", "account is deactivated"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := validCredential()
			if tt.cred != nil {
				tt.cred(&cred)
			}
			uc, tokens, _ := newAuthenticate(cred, tt.credErr)
			token := tt.token
			switch token {
			case "":
				token, _ = tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(time.Minute)})
			case "expired":
				token, _ = tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now})
			}

			_, err := uc.Execute(context.Background(), token)

			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != 401 || se.Code != shared.CodeUnauthorized {
				t.Fatalf("Execute() = %v, want 401 unauthorized", err)
			}
			if want := "Authentication is required.: " + tt.reason; err.Error() != want {
				t.Errorf("error = %q, want %q", err, want)
			}
		})
	}
}

// Only a valid signature whose exp has come, at that instant or past it, is
// "expired": the client's cue to refresh, which the failure gate does not
// count (M2 design 3.6).
func TestAuthenticateTellsAnExpiredAccessToken(t *testing.T) {
	for _, exp := range []time.Time{now, now.Add(-time.Second)} {
		uc, tokens, _ := newAuthenticate(validCredential(), nil)
		old, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: exp})

		_, expired := uc.Execute(context.Background(), old)
		_, forged := uc.Execute(context.Background(), "forged")

		if !errors.Is(expired, app.ErrAccessTokenExpired) || errors.Is(forged, app.ErrAccessTokenExpired) {
			t.Errorf("exp %v: expired = %v, forged = %v; want only the first to be ErrAccessTokenExpired", exp, expired, forged)
		}
	}
}

func TestAuthenticateDatabaseFailureIsNot401(t *testing.T) {
	boom := errors.New("connection refused")
	uc, tokens, _ := newAuthenticate(app.SessionCredential{}, boom)
	token, _ := tokens.Issue(app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: now.Add(time.Minute)})

	_, err := uc.Execute(context.Background(), token)

	var se *shared.Error
	if !errors.Is(err, boom) || errors.As(err, &se) {
		t.Errorf("Execute() = %v, want the database error, not a problem", err)
	}
}
