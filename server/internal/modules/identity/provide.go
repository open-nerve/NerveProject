package identity

import (
	"context"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
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

// Provided are the adapters identity offers the other modules. They depend
// on the pool alone, so bootstrap builds them before any module (M3 design
// 6.6, step 2).
type Provided struct {
	Accounts Accounts
}

// Provide builds identity's adapters for the other modules.
func Provide(pool *pgxpool.Pool) Provided {
	return Provided{Accounts: postgresadapter.New(pool)}
}
