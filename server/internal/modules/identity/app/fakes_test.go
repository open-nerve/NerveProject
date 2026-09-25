package app_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// fakeTx runs fn in a context marked as inside the transaction; fakeStore
// records whether each write happened there.
type fakeTx struct{ calls int }

type inTxKey struct{}

func (f *fakeTx) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	f.calls++
	return fn(context.WithValue(ctx, inTxKey{}, true))
}

func inTx(ctx context.Context) bool { return ctx.Value(inTxKey{}) == true }

// fakeStore is every repository port, in memory.
type fakeStore struct {
	users       []app.NewUser
	profiles    []uuid.UUID // user ids
	sessions    []app.NewSession
	outsideTx   []string // writes made outside a transaction
	createErr   error    // CreateUser's error
	getUser     domain.User
	getUserErr  error
	credential  app.SessionCredential
	credErr     error
	credentials []uuid.UUID // sessions looked up
}

func (s *fakeStore) CreateUser(ctx context.Context, u app.NewUser) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "user")
	}
	if s.createErr != nil {
		return s.createErr
	}
	s.users = append(s.users, u)
	return nil
}

func (s *fakeStore) CreateDefaultProfile(ctx context.Context, _, userID uuid.UUID, _ time.Time) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "profile")
	}
	s.profiles = append(s.profiles, userID)
	return nil
}

func (s *fakeStore) CreateSession(ctx context.Context, n app.NewSession) error {
	if !inTx(ctx) {
		s.outsideTx = append(s.outsideTx, "session")
	}
	s.sessions = append(s.sessions, n)
	return nil
}

func (s *fakeStore) GetUser(context.Context, uuid.UUID) (domain.User, error) {
	return s.getUser, s.getUserErr
}

func (s *fakeStore) SessionCredential(_ context.Context, id uuid.UUID) (app.SessionCredential, error) {
	s.credentials = append(s.credentials, id)
	return s.credential, s.credErr
}

// fakeHasher "hashes" by prefixing, and counts its calls.
type fakeHasher struct {
	calls int
	err   error
}

func (h *fakeHasher) Hash(_ context.Context, password string) (string, error) {
	h.calls++
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + password, nil
}

// fakeTokens issues "access:<sid>" and verifies what it issued; tokens in
// expired are expired.
type fakeTokens struct {
	issued  []app.AccessClaims
	claims  map[string]app.AccessClaims
	expired map[string]bool
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{claims: map[string]app.AccessClaims{}, expired: map[string]bool{}}
}

func (f *fakeTokens) Issue(c app.AccessClaims) (string, error) {
	f.issued = append(f.issued, c)
	token := "access:" + c.SessionID.String()
	f.claims[token] = c
	return token, nil
}

var errBadSignature = errors.New("signature is invalid")

func (f *fakeTokens) Verify(token string, _ time.Time) (app.AccessClaims, error) {
	if f.expired[token] {
		return app.AccessClaims{}, app.ErrAccessTokenExpired
	}
	c, ok := f.claims[token]
	if !ok {
		return app.AccessClaims{}, errBadSignature
	}
	return c, nil
}

// fakeMAC tags with the first 16 bytes of SHA-256: deterministic, and
// different for every message.
type fakeMAC struct{}

func (fakeMAC) Tag(message []byte) [16]byte {
	sum := sha256.Sum256(message)
	return [16]byte(sum[:16])
}

type fixedPolicy struct {
	allow bool
	err   error
}

func (p fixedPolicy) AllowSignup(context.Context) (bool, error) { return p.allow, p.err }
