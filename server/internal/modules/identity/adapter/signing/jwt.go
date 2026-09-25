package signing

import (
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/golang-jwt/jwt/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// claims are the access token's only claims: sub, sid and exp (M2 design 3.4).
type claims struct {
	jwt.RegisteredClaims
	SessionID string `json:"sid"`
}

// AccessTokens implements app.AccessTokens: JWTs signed with EdDSA
// (Ed25519), header {"alg":"EdDSA","typ":"JWT"}.
type AccessTokens struct {
	keys *Keys
}

// NewAccessTokens returns access tokens signed with keys.
func NewAccessTokens(keys *Keys) *AccessTokens {
	return &AccessTokens{keys: keys}
}

// Issue signs c.
func (a *AccessTokens) Issue(c app.AccessClaims) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims{
		RegisteredClaims: jwt.RegisteredClaims{Subject: c.UserID.String(), ExpiresAt: jwt.NewNumericDate(c.ExpiresAt)},
		SessionID:        c.SessionID.String(),
	})
	return token.SignedString(a.keys.private)
}

// Verify checks token at now: only EdDSA, strict base64url, exp required.
// A valid signature whose exp has passed is app.ErrAccessTokenExpired: the
// signature is checked before the claims (jwt/v5 parser.go).
func (a *AccessTokens) Verify(token string, now time.Time) (app.AccessClaims, error) {
	parser := jwt.NewParser(
		jwt.WithValidMethods([]string{jwt.SigningMethodEdDSA.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithStrictDecoding(),
		jwt.WithTimeFunc(func() time.Time { return now }),
	)
	var c claims
	_, err := parser.ParseWithClaims(token, &c, func(*jwt.Token) (any, error) { return a.keys.public, nil })
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return app.AccessClaims{}, app.ErrAccessTokenExpired
	case err != nil:
		return app.AccessClaims{}, err
	}
	userID, err := uuid.Parse(c.Subject)
	if err != nil {
		return app.AccessClaims{}, fmt.Errorf("sub: %w", err)
	}
	sessionID, err := uuid.Parse(c.SessionID)
	if err != nil {
		return app.AccessClaims{}, fmt.Errorf("sid: %w", err)
	}
	return app.AccessClaims{UserID: userID, SessionID: sessionID, ExpiresAt: c.ExpiresAt.Time}, nil
}
