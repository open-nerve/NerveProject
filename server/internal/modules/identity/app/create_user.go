package app

import (
	"context"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// accounts is the step that registration and `nerve users create` share
// (M2 design 3.17, 6.2): check the address and the password, hash the
// password, insert the account and its default profile.
type accounts struct {
	rules    *domain.PasswordRules
	hasher   PasswordHasher
	clock    Clock
	users    UserCreator
	profiles ProfileCreator
}

// prepare checks email and password, every problem at once (422), and
// hashes the password outside any transaction (M2 design 3.5). The account
// it returns has the normalized address, and the clock's time once the
// hash is done for its audit columns.
func (a accounts) prepare(ctx context.Context, email, password string) (NewUser, error) {
	email, err := domain.NewAccount(a.rules, email, password)
	if err != nil {
		return NewUser{}, err
	}
	hash, err := a.hasher.Hash(ctx, password)
	if err != nil {
		return NewUser{}, err
	}
	return NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: hash, DisplayName: domain.DisplayNameFromEmail(email), Now: a.clock.Now()}, nil
}

// create inserts u and its default profile; an address in use is
// identity.email_taken. Run it inside the caller's transaction.
func (a accounts) create(ctx context.Context, u NewUser) error {
	if err := a.users.CreateUser(ctx, u); err != nil {
		return err
	}
	return a.profiles.CreateDefaultProfile(ctx, uuid.NewV7(), u.ID, u.Now)
}

// CreateUserDeps are CreateUser's collaborators.
type CreateUserDeps struct {
	Rules    *domain.PasswordRules
	Hasher   PasswordHasher
	Tx       shared.TxManager
	Users    UserCreator
	Profiles ProfileCreator
	Clock    Clock
	Logger   *slog.Logger
}

// CreateUser creates an account for the server's administrator:
// `nerve users create` (M2 decision 2, design 3.17).
type CreateUser struct {
	d        CreateUserDeps
	accounts accounts
}

// NewCreateUser returns the use case.
func NewCreateUser(d CreateUserDeps) *CreateUser {
	return &CreateUser{d: d, accounts: accounts{rules: d.Rules, hasher: d.Hasher, clock: d.Clock, users: d.Users, profiles: d.Profiles}}
}

// Execute creates the account of email with password, and its profile, in
// one transaction; it opens no session. It is registration's first step
// without the sign-up policy: the administrator creates accounts whether
// sign-up is open or not. It returns the normalized address.
func (u *CreateUser) Execute(ctx context.Context, email, password string) (string, error) {
	user, err := u.accounts.prepare(ctx, email, password)
	if err != nil {
		return "", err
	}
	if err := u.d.Tx.WithinTx(ctx, func(ctx context.Context) error { return u.accounts.create(ctx, user) }); err != nil {
		return "", err
	}
	u.d.Logger.InfoContext(ctx, "account created", slog.String("user_id", user.ID.String()), slog.String("by", byCLI))
	return user.Email, nil
}

// byCLI is the value of "by" in the logs of the administrator's commands
// (M2 design 8.4): the command line, not the account itself.
const byCLI = "cli"
