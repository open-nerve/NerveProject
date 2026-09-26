package app_test

import (
	"bytes"
	"context"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

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

// fakeCredentials is the account row and the session that the credential
// lock reads. A change of password and a deactivation read and write the
// account's rows: it logs them too.
type fakeCredentials struct {
	log        *callLog
	email      string
	account    app.LockedAccount
	accountErr error
	session    app.SessionCredential
	sessionErr error
	hashErr    error       // UpdatePasswordHash fails with it
	revokeErr  error       // RevokeSessions fails with it
	writtenAt  []time.Time // the times the writes were given
}

func (f *fakeCredentials) PasswordAccount(ctx context.Context, id uuid.UUID) (app.PasswordAccount, error) {
	f.log.add(ctx, "read "+id.String())
	return app.PasswordAccount{Email: f.email, PasswordHash: f.account.PasswordHash}, f.accountErr
}

func (f *fakeCredentials) UpdatePasswordHash(ctx context.Context, id uuid.UUID, hash string, now time.Time) error {
	if f.hashErr != nil {
		return f.hashErr
	}
	f.log.add(ctx, "password "+id.String()+" "+hash)
	f.writtenAt = append(f.writtenAt, now)
	f.account.PasswordHash = hash
	return nil
}

func (f *fakeCredentials) DeactivateUser(ctx context.Context, id uuid.UUID, now time.Time) error {
	f.log.add(ctx, "deactivate "+id.String())
	f.writtenAt = append(f.writtenAt, now)
	return nil
}

func (f *fakeCredentials) ResetOnboarding(ctx context.Context, userID uuid.UUID, now time.Time) error {
	f.log.add(ctx, "reset onboarding of "+userID.String())
	f.writtenAt = append(f.writtenAt, now)
	return nil
}

func (f *fakeCredentials) RevokeSessions(ctx context.Context, userID, keep uuid.UUID, reason domain.RevokeReason, now time.Time) (int, error) {
	if f.revokeErr != nil {
		return 0, f.revokeErr
	}
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
