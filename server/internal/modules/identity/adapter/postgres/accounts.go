package postgresadapter

import (
	"context"
	"errors"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
)

// ShareAccount locks account id's row FOR SHARE until the transaction ends
// and returns its state; found is false when there is none (M3 design 6.5).
// Call it inside a transaction, as the transaction's first lock (M3 design
// 3.6 convention 1).
func (s *Store) ShareAccount(ctx context.Context, id uuid.UUID) (app.AccountState, bool, error) {
	row, err := s.queries(ctx).ShareAccount(ctx, id)
	return accountState(row.ID, row.Email, row.IsActive, err)
}

// ShareAccountByEmail is ShareAccount for the account with email, a
// normalized address.
func (s *Store) ShareAccountByEmail(ctx context.Context, email string) (app.AccountState, bool, error) {
	row, err := s.queries(ctx).ShareAccountByEmail(ctx, email)
	return accountState(row.ID, row.Email, row.IsActive, err)
}

func accountState(id uuid.UUID, email string, active bool, err error) (app.AccountState, bool, error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return app.AccountState{}, false, nil
	case err != nil:
		return app.AccountState{}, false, fmt.Errorf("lock the account row: %w", err)
	}
	return app.AccountState{ID: id, Email: email, Active: active}, true, nil
}
