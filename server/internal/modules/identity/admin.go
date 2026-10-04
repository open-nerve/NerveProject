package identity

import (
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// AdminDeps are what the server administrator's commands need: the pool,
// the password hashing and the deactivation's MembershipDeactivator, and
// no signing key, rate limit or sign-up policy (M2 design 3.17, M3 design
// 6.6).
type AdminDeps struct {
	Pool     *pgxpool.Pool
	Tx       shared.TxManager
	Clock    app.Clock
	Logger   *slog.Logger
	Password PasswordHashing
	// Memberships is workspace's Deactivator, which `nerve users
	// deactivate` ends the account's memberships through (M3 design 3.9).
	Memberships app.MembershipDeactivator
}

// Admin is the server administrator's use cases, behind `nerve users`
// (M2 design 3.17). The API shares them where it has the operation:
// deactivation is one use case with two entries.
type Admin struct {
	CreateUser    *app.CreateUser
	ResetPassword *app.ResetPassword
	SetEmail      *app.SetEmail
	Deactivate    *app.Deactivate
	Activate      *app.Activate
}

// NewAdmin wires the administrator's use cases on the pool alone: the
// command line builds no HTTP server and no jobs client, so it is the
// module's own minimal composition, not New's.
func NewAdmin(d AdminDeps) *Admin {
	hasher := argon2adapter.New(argon2adapter.Params(d.Password), d.Logger)
	rules := domain.NewPasswordRules()
	store := postgresadapter.New(d.Pool)
	return &Admin{
		CreateUser: app.NewCreateUser(app.CreateUserDeps{
			Rules: rules, Hasher: hasher, Tx: d.Tx, Users: store, Profiles: store, Clock: d.Clock, Logger: d.Logger,
		}),
		ResetPassword: app.NewResetPassword(app.ResetPasswordDeps{
			Accounts: store, Passwords: store, Sessions: store, APITokens: store, Hasher: hasher,
			Rules: rules, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		SetEmail: app.NewSetEmail(app.SetEmailDeps{
			Accounts: store, Users: store, Sessions: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
		Deactivate: app.NewDeactivate(app.DeactivateDeps{
			Accounts: store, Users: store, Profiles: store, Sessions: store, Memberships: d.Memberships, Tx: d.Tx, Clock: d.Clock,
			Logger: d.Logger,
		}),
		Activate: app.NewActivate(app.ActivateDeps{
			Accounts: store, Users: store, APITokens: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
		}),
	}
}
