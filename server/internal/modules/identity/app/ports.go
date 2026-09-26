// Package app holds the identity module's use cases. It declares the ports
// it needs (M2 design 6.3); adapters implement them and module.go wires
// them.
package app

import (
	"context"
	"errors"
	"net/netip"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// ErrNotFound is what a repository returns for a missing row.
var ErrNotFound = errors.New("not found")

// Clock tells the time: every business time comes from it and goes to SQL
// as a parameter (M2 design 3.5, 3.13). platform/clock implements it.
type Clock interface {
	Now() time.Time
}

// NewUser is an account to insert. Its audit columns are Now.
type NewUser struct {
	ID           uuid.UUID
	Email        string // normalized
	PasswordHash string // argon2id PHC string
	DisplayName  string
	Now          time.Time
}

// UserCreator inserts accounts.
type UserCreator interface {
	// CreateUser returns domain.ErrEmailTaken when the address is in use.
	CreateUser(ctx context.Context, u NewUser) error
}

// UserReader reads accounts.
type UserReader interface {
	// GetUser returns ErrNotFound when there is no such account.
	GetUser(ctx context.Context, id uuid.UUID) (domain.User, error)
}

// LoginAccount is what login reads of an account before its transaction:
// the hash is the snapshot it verifies the password against (M2 design 3.5).
type LoginAccount struct {
	ID           uuid.UUID
	PasswordHash string
}

// LoginAccountReader finds the account of an address.
type LoginAccountReader interface {
	// FindLoginAccount returns ErrNotFound when no account has email, a
	// normalized address.
	FindLoginAccount(ctx context.Context, email string) (LoginAccount, error)
}

// LockedAccount is an account's row under the credential lock.
type LockedAccount struct {
	PasswordHash string
	Active       bool
}

// CredentialLocker takes the account row lock that every transaction
// issuing or changing a credential takes first (M2 design 3.5).
type CredentialLocker interface {
	// LockForCredentials locks account id's row until the transaction ends
	// (SELECT … FOR NO KEY UPDATE) and returns it; ErrNotFound when there
	// is none. Call it inside a transaction.
	LockForCredentials(ctx context.Context, id uuid.UUID) (LockedAccount, error)
}

// PasswordHashWriter stores a new hash of an account's password.
type PasswordHashWriter interface {
	UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error
}

// ProfileCreator inserts the preferences of a new account.
type ProfileCreator interface {
	// CreateDefaultProfile inserts profile id of userID with every default.
	CreateDefaultProfile(ctx context.Context, id, userID uuid.UUID, now time.Time) error
}

// NewSession is a login to insert, at generation 0.
type NewSession struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash []byte // SHA-256 of the refresh token's secret
	UserAgent string
	IP        netip.Addr // the zero Addr when unknown
	ExpiresAt time.Time
	Now       time.Time
}

// SessionCreator inserts sessions.
type SessionCreator interface {
	CreateSession(ctx context.Context, s NewSession) error
}

// SessionCredential is what authentication checks of a session.
type SessionCredential struct {
	UserID     uuid.UUID
	ExpiresAt  time.Time
	Revoked    bool
	UserActive bool
}

// SessionReader reads what authentication needs, by primary key.
type SessionReader interface {
	// SessionCredential returns ErrNotFound when there is no such session.
	SessionCredential(ctx context.Context, id uuid.UUID) (SessionCredential, error)
}

// RefreshSession is what a refresh reads of the session its token names.
type RefreshSession struct {
	UserID uuid.UUID
	State  domain.SessionState
}

// SessionGeneration is a session as a refresh token presents it: rotation
// and logout change the session only while it is still at this generation
// with this hash, unrevoked and unexpired at Now (M2 design 3.5).
type SessionGeneration struct {
	ID         uuid.UUID
	Generation uint32
	TokenHash  []byte
	Now        time.Time
}

// SessionRotator is what a refresh reads and writes (M2 design 3.5).
type SessionRotator interface {
	// SessionForRefresh returns ErrNotFound when there is no such session.
	SessionForRefresh(ctx context.Context, id uuid.UUID) (RefreshSession, error)
	// RotateSession moves the session from g to the next generation with
	// newHash; false when the session is no longer at g.
	RotateSession(ctx context.Context, g SessionGeneration, newHash []byte) (bool, error)
	// RevokeForReuse revokes session id with reason reuse_detected, unless
	// it is revoked already.
	RevokeForReuse(ctx context.Context, id uuid.UUID, now time.Time) error
}

// SessionEnder ends sessions at logout.
type SessionEnder interface {
	// EndSession revokes the session with reason logout while it is at g;
	// false when it is not.
	EndSession(ctx context.Context, g SessionGeneration) (bool, error)
}

// PasswordHasher hashes and verifies passwords with argon2id. Both return a
// *shared.Error of 503 server_busy when no slot frees up within the wait
// limit (M2 design 3.8).
type PasswordHasher interface {
	Hash(ctx context.Context, password string) (string, error)
	// Verify reports whether password matches hash, and whether hash has
	// other parameters than the current ones: then login hashes the
	// password again.
	Verify(ctx context.Context, password, hash string) (ok, rehash bool, err error)
}

// AccessClaims are the claims of an access token: nothing about permissions
// (M2 design 3.4).
type AccessClaims struct {
	UserID    uuid.UUID
	SessionID uuid.UUID
	ExpiresAt time.Time
}

// ErrAccessTokenExpired is what AccessTokens.Verify returns for a token
// whose signature is valid but whose exp has passed: the client's cue to
// refresh (M2 design 3.6). Any other failure is a different error.
var ErrAccessTokenExpired = errors.New("access token expired")

// AccessTokens signs and verifies access tokens (JWT, EdDSA).
type AccessTokens interface {
	Issue(c AccessClaims) (string, error)
	// Verify checks the token at now; ErrAccessTokenExpired when only the
	// expiry fails.
	Verify(token string, now time.Time) (AccessClaims, error)
}

// RefreshTokenMAC tags refresh tokens (M2 design 3.4): the first 16 bytes of
// HMAC-SHA256 under a key derived from the signing key.
type RefreshTokenMAC interface {
	Tag(message []byte) [16]byte
	// Verify reports whether tag is message's tag, in constant time.
	Verify(message []byte, tag [16]byte) bool
}

// SignupPolicy decides whether registration is open (M2 design 3.9). From
// bootstrap it is auth.signup_enabled; M3 extends it to invitations.
type SignupPolicy interface {
	AllowSignup(ctx context.Context) (bool, error)
}
