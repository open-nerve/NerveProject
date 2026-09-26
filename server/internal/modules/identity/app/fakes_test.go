package app_test

import (
	"context"
	"crypto/sha256"
	"errors"
	"time"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
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

// fakeHasher "hashes" by prefixing "hashed:"; a hash prefixed "old:" has
// other parameters, so Verify asks for a rehash. It counts its calls.
type fakeHasher struct {
	calls     int // Hash calls
	err       error
	verified  []string // the hashes Verify was given
	verifyErr error
	onVerify  func() // runs inside every Verify: a transaction that commits meanwhile
}

func (h *fakeHasher) Hash(_ context.Context, password string) (string, error) {
	h.calls++
	if h.err != nil {
		return "", h.err
	}
	return "hashed:" + password, nil
}

func (h *fakeHasher) Verify(_ context.Context, password, hash string) (bool, bool, error) {
	h.verified = append(h.verified, hash)
	if h.onVerify != nil {
		h.onVerify()
	}
	if h.verifyErr != nil {
		return false, false, h.verifyErr
	}
	switch hash {
	case "hashed:" + password:
		return true, false, nil
	case "old:" + password:
		return true, true, nil
	}
	return false, false, nil
}

// fakeTokens issues "access:<sid>" and verifies what it issued at the instant
// it is given: like a JWT's exp, a token is expired from its ExpiresAt on. It
// records those instants.
type fakeTokens struct {
	issued     []app.AccessClaims
	claims     map[string]app.AccessClaims
	verifiedAt []time.Time
}

func newFakeTokens() *fakeTokens {
	return &fakeTokens{claims: map[string]app.AccessClaims{}}
}

func (f *fakeTokens) Issue(c app.AccessClaims) (string, error) {
	f.issued = append(f.issued, c)
	token := "access:" + c.SessionID.String()
	f.claims[token] = c
	return token, nil
}

var errBadSignature = errors.New("signature is invalid")

func (f *fakeTokens) Verify(token string, now time.Time) (app.AccessClaims, error) {
	f.verifiedAt = append(f.verifiedAt, now)
	c, ok := f.claims[token]
	if !ok {
		return app.AccessClaims{}, errBadSignature
	}
	if !now.Before(c.ExpiresAt) {
		return app.AccessClaims{}, app.ErrAccessTokenExpired
	}
	return c, nil
}

// fakeMAC tags with the first 16 bytes of SHA-256 over its key and the
// message: deterministic, and different for every message and key. Another
// key stands for a changed signing key.
type fakeMAC struct{ key string }

func (m fakeMAC) Tag(message []byte) [16]byte {
	sum := sha256.Sum256(append([]byte(m.key), message...))
	return [16]byte(sum[:16])
}

func (m fakeMAC) Verify(message []byte, tag [16]byte) bool { return m.Tag(message) == tag }

// callLog records the calls of the fakes that share it, in order, each
// with an identifying detail, an id when there is one, and " outside tx"
// when it ran outside a transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, call string) {
	if !inTx(ctx) {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

type fixedPolicy struct {
	allow bool
	err   error
}

func (p fixedPolicy) AllowSignup(context.Context) (bool, error) { return p.allow, p.err }
