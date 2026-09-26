package app_test

import (
	"bytes"
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
	getUserIDs  []uuid.UUID // accounts looked up
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

func (s *fakeStore) GetUser(_ context.Context, id uuid.UUID) (domain.User, error) {
	s.getUserIDs = append(s.getUserIDs, id)
	return s.getUser, s.getUserErr
}

func (s *fakeStore) SessionCredential(_ context.Context, id uuid.UUID) (app.SessionCredential, error) {
	s.credentials = append(s.credentials, id)
	return s.credential, s.credErr
}

// fakeLogins is login's ports over one account, in memory.
type fakeLogins struct {
	account     app.LoginAccount // found by email
	email       string           // the account's normalized address
	active      bool
	hash        string      // the row's hash, as LockForCredentials reads it
	lookedUp    []string    // addresses FindLoginAccount was given
	locks       int         // LockForCredentials calls
	hashUpdates []string    // hashes UpdatePasswordHash wrote
	hashTimes   []time.Time // the times it was given
	sessions    []app.NewSession
	outsideTx   []string
}

func (f *fakeLogins) FindLoginAccount(_ context.Context, email string) (app.LoginAccount, error) {
	f.lookedUp = append(f.lookedUp, email)
	if email != f.email {
		return app.LoginAccount{}, app.ErrNotFound
	}
	return f.account, nil
}

func (f *fakeLogins) LockForCredentials(ctx context.Context, id uuid.UUID) (app.LockedAccount, error) {
	f.locks++
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "lock")
	}
	if id != f.account.ID {
		return app.LockedAccount{}, app.ErrNotFound
	}
	return app.LockedAccount{PasswordHash: f.hash, Active: f.active}, nil
}

func (f *fakeLogins) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	if !inTx(ctx) || id != f.account.ID {
		f.outsideTx = append(f.outsideTx, "password of "+id.String())
	}
	f.hashUpdates = append(f.hashUpdates, hash)
	f.hashTimes = append(f.hashTimes, now)
	f.hash = hash
	return nil
}

func (f *fakeLogins) CreateSession(ctx context.Context, n app.NewSession) error {
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "session")
	}
	f.sessions = append(f.sessions, n)
	return nil
}

// fakeSession is a session row as refresh and logout see it.
type fakeSession struct {
	app.RefreshSession
	reason    string    // revoke_reason
	changedAt time.Time // updated_at, which each write sets
}

// fakeSessions is refresh's and logout's ports, in memory. RotateSession and
// EndSession apply their conditions the way the SQL does.
type fakeSessions struct {
	rows        map[uuid.UUID]*fakeSession
	beforeWrite func() // runs before each conditional write: a transaction that commits first
	afterWrite  func() // runs after each conditional write
	reads       int
	outsideTx   []string
}

func (f *fakeSessions) SessionForRefresh(_ context.Context, id uuid.UUID) (app.RefreshSession, error) {
	f.reads++
	s, ok := f.rows[id]
	if !ok {
		return app.RefreshSession{}, app.ErrNotFound
	}
	return app.RefreshSession{UserID: s.UserID, State: s.State}, nil
}

// at reports whether the session is still at g: the WHERE of rotation and
// logout.
func (f *fakeSessions) at(g app.SessionGeneration) (*fakeSession, bool) {
	if f.beforeWrite != nil {
		f.beforeWrite()
	}
	if f.afterWrite != nil {
		defer f.afterWrite()
	}
	s, ok := f.rows[g.ID]
	return s, ok && s.State.Generation == g.Generation && bytes.Equal(s.State.TokenHash, g.TokenHash) &&
		!s.State.Revoked && g.Now.Before(s.State.ExpiresAt)
}

func (f *fakeSessions) RotateSession(ctx context.Context, g app.SessionGeneration, newHash []byte) (bool, error) {
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "rotate")
	}
	s, ok := f.at(g)
	if ok {
		s.State.Generation++
		s.State.TokenHash = newHash
		s.changedAt = g.Now
	}
	return ok, nil
}

func (f *fakeSessions) RevokeForReuse(ctx context.Context, id uuid.UUID, now time.Time) error {
	if !inTx(ctx) {
		f.outsideTx = append(f.outsideTx, "revoke")
	}
	if s, ok := f.rows[id]; ok && !s.State.Revoked {
		s.State.Revoked, s.reason, s.changedAt = true, "reuse_detected", now
	}
	return nil
}

func (f *fakeSessions) EndSession(_ context.Context, g app.SessionGeneration) (bool, error) {
	s, ok := f.at(g)
	if ok {
		s.State.Revoked, s.reason, s.changedAt = true, "logout", g.Now
	}
	return ok, nil
}

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
// with the id it was given and " outside tx" when it ran outside a
// transaction.
type callLog struct{ calls []string }

func (l *callLog) add(ctx context.Context, call string) {
	if !inTx(ctx) {
		call += " outside tx"
	}
	l.calls = append(l.calls, call)
}

// fakeCredentials is the account row and the session that the credential
// lock reads. A change of password reads the row and writes it and the
// sessions: it logs them too.
type fakeCredentials struct {
	log        *callLog
	email      string
	account    app.LockedAccount
	accountErr error
	session    app.SessionCredential
	sessionErr error
	writtenAt  []time.Time // the times the writes were given
}

func (f *fakeCredentials) PasswordAccount(ctx context.Context, id uuid.UUID) (app.PasswordAccount, error) {
	f.log.add(ctx, "read "+id.String())
	return app.PasswordAccount{Email: f.email, PasswordHash: f.account.PasswordHash}, f.accountErr
}

func (f *fakeCredentials) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	f.log.add(ctx, "password "+id.String()+" "+hash)
	f.writtenAt = append(f.writtenAt, now)
	f.account.PasswordHash = hash
	return nil
}

func (f *fakeCredentials) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error) {
	f.log.add(ctx, "revoke "+string(reason)+" sessions of "+userID.String()+" but "+keep.String())
	f.writtenAt = append(f.writtenAt, now)
	return 1, nil
}

func (f *fakeCredentials) LockForCredentials(ctx context.Context, id uuid.UUID) (app.LockedAccount, error) {
	f.log.add(ctx, "lock "+id.String())
	return f.account, f.accountErr
}

func (f *fakeCredentials) SessionCredential(ctx context.Context, id uuid.UUID) (app.SessionCredential, error) {
	f.log.add(ctx, "session "+id.String())
	return f.session, f.sessionErr
}

// touch is one TouchAPIToken call.
type touch struct {
	id               uuid.UUID
	now, staleBefore time.Time
}

// revocation is one RevokeAPIToken call.
type revocation struct {
	id, userID uuid.UUID
	now        time.Time
}

// fakeAPITokens is the token ports, in memory. It finds a credential by the
// hash and by the id it holds, so a lookup by any other key fails; it lists
// at most limit of its rows.
type fakeAPITokens struct {
	log        *callLog
	credential app.APITokenCredential
	hash       []byte // the hash that finds credential
	readErr    error
	touches    []touch
	touchErr   error
	created    []app.NewAPIToken
	rows       []domain.APIToken
	listed     []listCall
	revoked    []revocation
	revokeOK   bool
	revokeErr  error
}

// listCall is one ListAPITokens call.
type listCall struct {
	userID uuid.UUID
	after  *domain.APITokenCursor
	limit  int
}

func (f *fakeAPITokens) APITokenByHash(ctx context.Context, hash []byte) (app.APITokenCredential, error) {
	f.log.add(ctx, "token by hash")
	if f.readErr != nil {
		return app.APITokenCredential{}, f.readErr
	}
	if !bytes.Equal(hash, f.hash) {
		return app.APITokenCredential{}, app.ErrNotFound
	}
	return f.credential, nil
}

func (f *fakeAPITokens) APITokenByID(ctx context.Context, id uuid.UUID) (app.APITokenCredential, error) {
	f.log.add(ctx, "token "+id.String())
	if f.readErr != nil {
		return app.APITokenCredential{}, f.readErr
	}
	if id != f.credential.ID {
		return app.APITokenCredential{}, app.ErrNotFound
	}
	return f.credential, nil
}

func (f *fakeAPITokens) TouchAPIToken(_ context.Context, id uuid.UUID, now, staleBefore time.Time) error {
	f.touches = append(f.touches, touch{id, now, staleBefore})
	return f.touchErr
}

func (f *fakeAPITokens) CreateAPIToken(ctx context.Context, n app.NewAPIToken) error {
	f.log.add(ctx, "insert "+n.ID.String())
	f.created = append(f.created, n)
	return nil
}

func (f *fakeAPITokens) ListAPITokens(_ context.Context, userID uuid.UUID, after *domain.APITokenCursor, limit int) ([]domain.APIToken, error) {
	f.listed = append(f.listed, listCall{userID, after, limit})
	return f.rows[:min(limit, len(f.rows))], nil
}

func (f *fakeAPITokens) RevokeAPIToken(_ context.Context, id, userID uuid.UUID, now time.Time) (bool, error) {
	f.revoked = append(f.revoked, revocation{id, userID, now})
	return f.revokeOK, f.revokeErr
}

// userUpdate is one UpdateUser call.
type userUpdate struct {
	id    uuid.UUID
	patch domain.UserPatch
	now   time.Time
}

// profileUpdate is one UpdateProfile call.
type profileUpdate struct {
	userID uuid.UUID
	patch  domain.ProfilePatch
	now    time.Time
}

// fakeAccount is one account and its profile, in memory: it finds them by
// the account's id only. err, when set, is every call's error.
type fakeAccount struct {
	user           domain.User
	profile        domain.Profile
	err            error
	userUpdates    []userUpdate
	profileReads   []uuid.UUID
	profileUpdates []profileUpdate
}

func (f *fakeAccount) find(id uuid.UUID) error {
	switch {
	case f.err != nil:
		return f.err
	case id != f.user.ID:
		return app.ErrNotFound
	}
	return nil
}

func (f *fakeAccount) UpdateUser(_ context.Context, id uuid.UUID, p domain.UserPatch, now time.Time) (domain.User, error) {
	f.userUpdates = append(f.userUpdates, userUpdate{id, p, now})
	if err := f.find(id); err != nil {
		return domain.User{}, err
	}
	return f.user, nil
}

func (f *fakeAccount) GetProfile(_ context.Context, userID uuid.UUID) (domain.Profile, error) {
	f.profileReads = append(f.profileReads, userID)
	if err := f.find(userID); err != nil {
		return domain.Profile{}, err
	}
	return f.profile, nil
}

func (f *fakeAccount) UpdateProfile(_ context.Context, userID uuid.UUID, p domain.ProfilePatch, now time.Time) (domain.Profile, error) {
	f.profileUpdates = append(f.profileUpdates, profileUpdate{userID, p, now})
	if err := f.find(userID); err != nil {
		return domain.Profile{}, err
	}
	return f.profile, nil
}

type fixedPolicy struct {
	allow bool
	err   error
}

func (p fixedPolicy) AllowSignup(context.Context) (bool, error) { return p.allow, p.err }
