package app

import (
	"context"
	"errors"
	"log/slog"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// ChangePasswordDeps are ChangePassword's collaborators.
type ChangePasswordDeps struct {
	Accounts  PasswordAccountReader
	Lock      CredentialLock
	Passwords PasswordHashWriter
	Sessions  SessionRevoker
	Hasher    PasswordHasher
	Rules     *domain.PasswordRules
	Tx        shared.TxManager
	Clock     Clock
	Logger    *slog.Logger
}

// ChangePassword changes the caller's password:
// POST /api/v0/me/change-password.
type ChangePassword struct {
	d ChangePasswordDeps
}

// NewChangePassword returns the use case.
func NewChangePassword(d ChangePasswordDeps) *ChangePassword {
	return &ChangePassword{d: d}
}

// ChangePasswordInput is the current password and the new one.
type ChangePasswordInput struct {
	Current string
	New     string
}

// Execute changes the caller's password (M2 design 3.5, 3.8):
//
//  1. the new password is checked by the password rules, with the
//     account's address for the stem rule: 422 validation_failed on
//     new_password;
//  2. outside the transaction, the current password is verified against
//     the account's hash, the snapshot, and the new one is hashed; a wrong
//     current password is 422 identity.current_password_incorrect;
//  3. one transaction takes the credential lock, checks that the hash still
//     equals the snapshot, writes the new hash and revokes the account's
//     other sessions with reason password_changed: all of them when the
//     caller is a personal access token, which has no session. Tokens stay.
//
// When the hash changed in between, a concurrent login hashed the password
// again, or the password changed: the current password is verified against
// the new hash, and step 3 is done once more. A second change fails.
func (c *ChangePassword) Execute(ctx context.Context, in ChangePasswordInput) error {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return err
	}
	account, err := c.d.Accounts.PasswordAccount(ctx, actor.UserID)
	switch {
	case errors.Is(err, ErrNotFound):
		// Authentication found the account a moment ago; it is gone now.
		return shared.Unauthenticated()
	case err != nil:
		return err
	}
	if f := c.d.Rules.Check("new_password", in.New, account.Email); f != nil {
		return shared.Invalid(*f)
	}

	snapshot, hash := account.PasswordHash, ""
	for range 2 {
		ok, _, err := c.d.Hasher.Verify(ctx, in.Current, snapshot)
		if err != nil {
			return err
		}
		if !ok {
			break
		}
		if hash == "" {
			if hash, err = c.d.Hasher.Hash(ctx, in.New); err != nil {
				return err
			}
		}
		found, err := c.change(ctx, actor, snapshot, hash)
		if err != nil {
			return err
		}
		if found == snapshot {
			c.d.Logger.InfoContext(ctx, "password changed", slog.String("user_id", actor.UserID.String()))
			return nil
		}
		snapshot = found
	}
	return domain.ErrCurrentPasswordIncorrect
}

// change is step 3. It returns the hash it found under the lock: when that
// is not snapshot, it wrote nothing.
func (c *ChangePassword) change(ctx context.Context, actor shared.Actor, snapshot, hash string) (string, error) {
	now := c.d.Clock.Now()
	found := snapshot
	err := c.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		locked, err := c.d.Lock.Lock(ctx, actor, now)
		if err != nil {
			return err
		}
		if found = locked.PasswordHash; found != snapshot {
			return nil
		}
		if err := c.d.Passwords.UpdatePasswordHash(ctx, actor.UserID, hash, now); err != nil {
			return err
		}
		_, err = c.d.Sessions.RevokeSessions(ctx, actor.UserID, actor.SessionID, domain.RevokePasswordChanged, now)
		return err
	})
	return found, err
}
