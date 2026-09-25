package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Authenticate turns a bearer token into the request's actor (M2 design
// 3.4–3.6): the access token's signature and expiry, then one primary-key
// query for the session and its account. Personal access tokens join in
// M2/P3; until then a nrv_pat_ token fails as a JWT like any other.
type Authenticate struct {
	tokens   AccessTokens
	sessions SessionReader
	clock    Clock
}

// NewAuthenticate returns the use case.
func NewAuthenticate(tokens AccessTokens, sessions SessionReader, clock Clock) *Authenticate {
	return &Authenticate{tokens: tokens, sessions: sessions, clock: clock}
}

// The reasons a credential fails. They go to the debug log only; the
// caller always sees 401 unauthorized.
var (
	errSessionUnknown  = errors.New("session does not exist")
	errSessionMismatch = errors.New("session belongs to another account")
	errSessionRevoked  = errors.New("session is revoked")
	errSessionExpired  = errors.New("session has expired")
	errUserDeactivated = errors.New("account is deactivated")
)

// Execute returns the actor of token. An invalid credential is a
// *shared.Error of 401 that wraps the reason; an expired access token also
// matches ErrAccessTokenExpired. Any other error is an internal fault.
func (a *Authenticate) Execute(ctx context.Context, token string) (shared.Actor, error) {
	now := a.clock.Now()
	claims, err := a.tokens.Verify(token, now)
	if err != nil {
		return shared.Actor{}, unauthenticated(err)
	}
	cred, err := a.sessions.SessionCredential(ctx, claims.SessionID)
	switch {
	case errors.Is(err, ErrNotFound):
		return shared.Actor{}, unauthenticated(errSessionUnknown)
	case err != nil:
		return shared.Actor{}, err
	case cred.UserID != claims.UserID:
		return shared.Actor{}, unauthenticated(errSessionMismatch)
	case cred.Revoked:
		return shared.Actor{}, unauthenticated(errSessionRevoked)
	case !now.Before(cred.ExpiresAt):
		return shared.Actor{}, unauthenticated(errSessionExpired)
	case !cred.UserActive:
		return shared.Actor{}, unauthenticated(errUserDeactivated)
	}
	return shared.Actor{UserID: claims.UserID, SessionID: claims.SessionID}, nil
}

func unauthenticated(reason error) error {
	return fmt.Errorf("%w: %w", shared.Unauthenticated(), reason)
}
