package app_test

import (
	"context"
	"errors"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

var tokenID = uuid.MustParse("0199a2b4-0000-7000-8000-000000000003")

// samplePAT is a token whose bytes all differ.
func samplePAT() domain.PAT {
	var p domain.PAT
	for i := range p {
		p[i] = byte(i*5 + 1)
	}
	return p
}

// newPATAuthenticate authenticates samplePAT as cred; the access tokens and
// sessions it holds are there to show that a PAT never reaches them.
func newPATAuthenticate(cred app.APITokenCredential) (*app.Authenticate, *fakeAPITokens, *fakeTokens, *fakeStore) {
	tokens := &fakeAPITokens{log: &callLog{}, credential: cred, hash: samplePAT().Hash()}
	access, sessions := newFakeTokens(), &fakeStore{}
	uc := app.NewAuthenticate(app.AuthenticateDeps{AccessTokens: access, Sessions: sessions, APITokens: tokens, Touch: tokens, Clock: clocktest.At(now)})
	return uc, tokens, access, sessions
}

func validToken() app.APITokenCredential {
	expires := now.Add(time.Hour)
	return app.APITokenCredential{ID: tokenID, UserID: userID, ExpiredAt: &expires, UserActive: true}
}

// A token without expired_at never expires, as Plane creates it by default;
// the tests below use one that expires in an hour.
func TestAuthenticateAPAT(t *testing.T) {
	cred := validToken()
	cred.ExpiredAt = nil
	uc, tokens, access, sessions := newPATAuthenticate(cred)

	actor, err := uc.Execute(context.Background(), samplePAT().String())

	if err != nil || actor != (shared.Actor{UserID: userID, APITokenID: tokenID}) {
		t.Errorf("Execute() = %+v, %v; want the token's account and id", actor, err)
	}
	if len(tokens.log.calls) != 1 || len(access.verifiedAt) != 0 || len(sessions.credentials) != 0 {
		t.Errorf("lookups = %q, access tokens verified %d, sessions read %d; want one lookup by hash, no JWT, no session",
			tokens.log.calls, len(access.verifiedAt), len(sessions.credentials))
	}
	// Never used before: last_used is written, at the clock's time.
	if want := []touch{{tokenID, now, now.Add(-time.Minute)}}; len(tokens.touches) != 1 || tokens.touches[0] != want[0] {
		t.Errorf("touches = %+v, want %+v", tokens.touches, want)
	}
}

// last_used is written at most once a minute (M2 design 3.5): not when it
// is less than a minute old.
func TestAuthenticateAPATTouchesAtMostOnceAMinute(t *testing.T) {
	for _, tt := range []struct {
		name    string
		used    time.Duration // before now
		touched bool
	}{
		{"used 59 seconds ago", 59 * time.Second, false},
		{"used a minute ago", time.Minute, false},
		{"used 61 seconds ago", 61 * time.Second, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cred := validToken()
			used := now.Add(-tt.used)
			cred.LastUsed = &used
			uc, tokens, _, _ := newPATAuthenticate(cred)

			if _, err := uc.Execute(context.Background(), samplePAT().String()); err != nil {
				t.Fatal(err)
			}
			if got := len(tokens.touches) == 1; got != tt.touched {
				t.Errorf("touched = %v, want %v", got, tt.touched)
			}
		})
	}
}

func TestAuthenticateRejectsAPAT(t *testing.T) {
	other := uuid.MustParse("0199a2b4-0000-7000-8000-000000000009")
	tests := []struct {
		name   string
		token  string
		cred   func(*app.APITokenCredential)
		reason string
		looked bool // whether the token was looked up
	}{
		{"malformed", domain.PATPrefix + "AAAA", nil, "personal access token is malformed", false},
		{"unknown", func() string { p := samplePAT(); p[0]++; return p.String() }(), nil, "personal access token does not exist", true},
		{"revoked", "", func(c *app.APITokenCredential) { c.Revoked = true }, "personal access token is revoked", true},
		{"expiring now", "", func(c *app.APITokenCredential) { c.ExpiredAt = &now }, "personal access token has expired", true},
		{"of a deactivated account", "", func(c *app.APITokenCredential) { c.UserActive = false }, "account is deactivated", true},
		{"of another account", "", func(c *app.APITokenCredential) { c.UserID = other }, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cred := validToken()
			if tt.cred != nil {
				tt.cred(&cred)
			}
			uc, tokens, _, _ := newPATAuthenticate(cred)
			token := tt.token
			if token == "" {
				token = samplePAT().String()
			}

			actor, err := uc.Execute(context.Background(), token)

			if tt.reason == "" {
				// A token authenticates its own account, whichever that is.
				if err != nil || actor.UserID != other {
					t.Errorf("Execute() = %+v, %v; want the token's own account", actor, err)
				}
				return
			}
			var se *shared.Error
			if !errors.As(err, &se) || se.ProblemStatus() != 401 || err.Error() != "Authentication is required.: "+tt.reason {
				t.Errorf("Execute() = %v, want 401 because %s", err, tt.reason)
			}
			if looked := len(tokens.log.calls) > 0; looked != tt.looked || len(tokens.touches) != 0 {
				t.Errorf("looked up %v, touched %d; want looked up %v, never touched", looked, len(tokens.touches), tt.looked)
			}
		})
	}
}

// A database failure is an internal fault, not a 401: whether reading the
// token or writing last_used.
func TestAuthenticateAPATDatabaseFailureIsNot401(t *testing.T) {
	boom := errors.New("connection refused")
	for _, failing := range []string{"read", "touch"} {
		uc, tokens, _, _ := newPATAuthenticate(validToken())
		if failing == "read" {
			tokens.readErr = boom
		} else {
			tokens.touchErr = boom
		}

		_, err := uc.Execute(context.Background(), samplePAT().String())

		var se *shared.Error
		if !errors.Is(err, boom) || errors.As(err, &se) {
			t.Errorf("the %s fails: Execute() = %v, want the database error, not a problem", failing, err)
		}
	}
}
