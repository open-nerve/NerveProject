package bootstrap

import (
	"context"
	"fmt"
	"io"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/logging"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// UserCommand is one `nerve users` command on the administrator's use
// cases: it runs and returns the line it prints.
type UserCommand func(ctx context.Context, admin *identity.Admin) (string, error)

// Users runs cmd on the minimal composition of M2 design 3.17: a pool and
// identity's administrator use cases; no HTTP server, no jobs client. The
// command's line goes to out, the logs to logOut. An error is one line for
// the administrator, and the database is unchanged.
func Users(ctx context.Context, cfg config.Config, logOut, out io.Writer, cmd UserCommand) error {
	logger, err := logging.New(logOut, cfg.Log)
	if err != nil {
		return err
	}
	pool, err := postgres.NewPool(ctx, cfg.Database)
	if err != nil {
		return err
	}
	defer pool.Close()
	admin := identity.NewAdmin(identity.AdminDeps{
		Pool:     pool,
		Tx:       postgres.NewTxManager(pool, cfg.Database.CommitTimeout),
		Clock:    clock.System{},
		Logger:   logger,
		Password: passwordHashing(cfg.Auth.Password),
	})
	line, err := cmd(ctx, admin)
	if err != nil {
		return commandError(err)
	}
	return writeLine(out, line)
}

// CreateUser is `nerve users create` (M2 decision 2).
func CreateUser(email, password string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		created, err := admin.CreateUser.Execute(ctx, email, password)
		return "created user " + created, err
	}
}

// ResetPassword is `nerve users reset-password` (M2 design 3.5).
func ResetPassword(email, password string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.ResetPassword.Execute(ctx, email, password)
		return fmt.Sprintf("password reset for %s: revoked %d sessions, %d API tokens", r.Email, r.Sessions, r.APITokens), err
	}
}

// SetEmail is `nerve users set-email` (M2 decision 1).
func SetEmail(email, newEmail string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.SetEmail.Execute(ctx, email, newEmail)
		return fmt.Sprintf("email changed from %s to %s: revoked %d sessions", r.From, r.To, r.Sessions), err
	}
}

// DeactivateUser is `nerve users deactivate` (M2 decision 3).
func DeactivateUser(email string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.Deactivate.ExecuteByEmail(ctx, email)
		return fmt.Sprintf("deactivated %s: revoked %d sessions", r.Email, r.Sessions), err
	}
}

// ActivateUser is `nerve users activate` (M2 decision 3).
func ActivateUser(email string) UserCommand {
	return func(ctx context.Context, admin *identity.Admin) (string, error) {
		r, err := admin.Activate.Execute(ctx, email)
		return fmt.Sprintf("activated %s: %d API tokens are usable again", r.Email, r.APITokens), err
	}
}
