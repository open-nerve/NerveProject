package bootstrap

import (
	"context"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// The values that cross from one module's port to another's: modules do not
// import each other, so where an implementation answers its own type, a few
// lines here convert it (M3 design 6.5).

// workspaceAccounts is identity's Accounts as workspace's port: the same
// locks, identity's AccountState converted into workspace's.
type workspaceAccounts struct {
	accounts identity.Accounts
}

func (a workspaceAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (workspace.AccountState, bool, error) {
	state, found, err := a.accounts.ShareAccount(ctx, id)
	return workspace.AccountState(state), found, err
}

func (a workspaceAccounts) ShareAccountByEmail(ctx context.Context, email string) (workspace.AccountState, bool, error) {
	state, found, err := a.accounts.ShareAccountByEmail(ctx, email)
	return workspace.AccountState(state), found, err
}
