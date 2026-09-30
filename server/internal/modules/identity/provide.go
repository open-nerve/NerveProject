package identity

import (
	"context"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Accounts locks an account row FOR SHARE and returns the account's state;
// found is false when there is no such account (M3 design 6.5). It is the
// first lock of the caller's transaction and never a later one (M3 design
// 3.6 conventions 1 and 6): it conflicts with deactivation's lock of the
// row, so the account cannot be deactivated before the caller commits, and
// the state read is the one committed before the lock. An address is
// matched as given: the caller normalizes it.
type Accounts interface {
	ShareAccount(ctx context.Context, id uuid.UUID) (state app.AccountState, found bool, err error)
	ShareAccountByEmail(ctx context.Context, email string) (state app.AccountState, found bool, err error)
}

// PublicProfiles reads the public profile of each account of ids that
// exists, deactivated ones too, by id, without a lock (M3 design 6.5): the
// workspace module's MemberProfiles. A transaction that holds a lock of a
// later table reads an account's address this way and never through
// Accounts (M3 design 3.6 convention 1).
type PublicProfiles interface {
	PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error)
}

// CredentialLock is M2's credential lock (M2 design 3.5) as another module
// takes it: LockCaller locks the caller's account row FOR NO KEY UPDATE
// until the transaction ends, then checks under the lock that the account
// is active and the caller's session or personal access token valid at now;
// 401 unauthorized otherwise; a failed read is returned as itself. It is
// the first lock of a transaction that issues something with the caller's
// credential: creating invitations (M3 design 3.8), so that a password
// reset committed first leaves none.
type CredentialLock interface {
	LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error
}

// Provided are the adapters identity offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2); CredentialLock is identity's own protocol step, on its
// store alone.
type Provided struct {
	Accounts       Accounts
	PublicProfiles PublicProfiles
	CredentialLock CredentialLock
}

// Provide builds identity's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	store := postgresadapter.New(pool)
	return Provided{Accounts: store, PublicProfiles: store, CredentialLock: callerLock{lock: credentialLock(store)}}
}

// credentialLock is the credential lock over the store, identity's use
// cases' and the other modules' alike.
func credentialLock(store *postgresadapter.Store) app.CredentialLock {
	return app.CredentialLock{Locker: store, Sessions: store, APITokens: store}
}

// callerLock is the credential lock without the locked account: another
// module learns only whether the caller's credential holds, never the
// account's password hash.
type callerLock struct {
	lock app.CredentialLock
}

func (l callerLock) LockCaller(ctx context.Context, actor shared.Actor, now time.Time) error {
	_, err := l.lock.Lock(ctx, actor, now)
	return err
}
