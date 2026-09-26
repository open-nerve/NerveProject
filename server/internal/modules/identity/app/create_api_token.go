package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// CreateAPITokenDeps are CreateAPIToken's collaborators.
type CreateAPITokenDeps struct {
	Lock   CredentialLock
	Tokens APITokenCreator
	Tx     shared.TxManager
	Clock  Clock
	Logger *slog.Logger
}

// CreateAPIToken creates a personal access token for the caller:
// POST /api/v0/me/api-tokens. Any credential may, a token too (M2 design
// 4.6); no password is asked for, as in Plane (8.5).
type CreateAPIToken struct {
	d CreateAPITokenDeps
}

// NewCreateAPIToken returns the use case.
func NewCreateAPIToken(d CreateAPITokenDeps) *CreateAPIToken {
	return &CreateAPIToken{d: d}
}

// CreatedAPIToken is a new token and, this once, the token itself.
type CreatedAPIToken struct {
	domain.APIToken
	Token string
}

// Execute checks spec, then, in one transaction, locks the caller's account
// row, checks the caller's credential again and inserts the token (M2
// design 3.5): a token made with a credential that a concurrent reset
// revoked is never inserted. The row and the answer hold the expiry as the
// check returns it, in UTC to the microsecond. A spec without a label gets
// 32 hexadecimal digits, as Plane's uuid4().hex.
func (c *CreateAPIToken) Execute(ctx context.Context, spec domain.APITokenSpec) (CreatedAPIToken, error) {
	actor, err := shared.RequireActor(ctx)
	if err != nil {
		return CreatedAPIToken{}, err
	}
	now := c.d.Clock.Now()
	spec, err = domain.CheckAPIToken(spec, now)
	if err != nil {
		return CreatedAPIToken{}, err
	}
	var label string
	if spec.Label != nil {
		label = *spec.Label
	} else {
		id := uuid.NewV4()
		label = hex.EncodeToString(id[:])
	}
	var pat domain.PAT
	_, _ = rand.Read(pat[:]) // never fails since Go 1.24
	n := NewAPIToken{
		ID: uuid.NewV7(), UserID: actor.UserID, TokenHash: pat.Hash(),
		Label: label, Description: spec.Description, ExpiredAt: spec.ExpiredAt, Now: now,
	}
	err = c.d.Tx.WithinTx(ctx, func(ctx context.Context) error {
		if _, err := c.d.Lock.Lock(ctx, actor, now); err != nil {
			return err
		}
		return c.d.Tokens.CreateAPIToken(ctx, n)
	})
	if err != nil {
		return CreatedAPIToken{}, err
	}
	c.d.Logger.InfoContext(ctx, "API token created", slog.String("user_id", actor.UserID.String()), slog.String("token_id", n.ID.String()))
	return CreatedAPIToken{
		APIToken: domain.APIToken{ID: n.ID, Label: n.Label, Description: n.Description, ExpiredAt: n.ExpiredAt, CreatedAt: now},
		Token:    pat.String(),
	}, nil
}
