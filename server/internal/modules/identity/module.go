// Package identity is the accounts module (M2 design 3.3, 6.2): accounts,
// profiles, sessions and personal access tokens. It brings registration,
// login, refresh, logout, the caller's account, preferences and tokens, and
// the authentication every other operation goes through.
package identity

import (
	"context"
	"crypto/rand"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	argon2adapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/argon2"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	httpadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/http"
	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/signing"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/platform/ratelimit"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// Deps are what bootstrap builds for the module.
type Deps struct {
	Pool   *pgxpool.Pool
	Tx     shared.TxManager
	Clock  app.Clock
	Logger *slog.Logger
	// SignupPolicy is auth.signup_enabled (M2 decision 2).
	SignupPolicy app.SignupPolicy
	// SigningKeyPEM is the content of auth.jwt.private_key_file; nil for
	// none, then the key is ephemeral (dev and test only, M2 design 3.7).
	SigningKeyPEM   []byte
	AccessTokenTTL  time.Duration
	SessionTTL      time.Duration
	RefreshDeadline time.Duration // auth.refresh_deadline
	Password        PasswordHashing
	RateLimits      RateLimits
}

// PasswordHashing is auth.password: argon2id's parameters and the limits on
// concurrent hashes (M2 design 3.8).
type PasswordHashing struct {
	MemoryKiB     uint32
	Iterations    uint32
	Parallelism   uint8
	MaxConcurrent int
	MaxWait       time.Duration
}

// RateLimits are the module's own buckets, on the limiter they belong to
// (M2 design 3.10).
type RateLimits struct {
	Limiter      httpadapter.RateLimiter
	LoginIP      *ratelimit.Bucket
	LoginIPEmail *ratelimit.Bucket
	RegisterIP   *ratelimit.Bucket
	PasswordUser *ratelimit.Bucket
}

// Module is the wired identity module.
type Module struct {
	uc            httpadapter.UseCases
	settings      httpadapter.Settings
	authenticator *authn.Authenticator
}

// New wires the module. A signing key that cannot be parsed is an error
// that never quotes the key.
func New(d Deps) (*Module, error) {
	keys, err := signingKeys(d)
	if err != nil {
		return nil, err
	}
	hasher := argon2adapter.New(argon2adapter.Params(d.Password), d.Logger)
	// Login verifies an unknown address against this, so that it takes as
	// long as a known one (M2 design 3.9).
	dummy, err := hasher.Hash(context.Background(), rand.Text())
	if err != nil {
		return nil, fmt.Errorf("hash the dummy password: %w", err)
	}
	rules := domain.NewPasswordRules()
	store := postgresadapter.New(d.Pool)
	lock := app.CredentialLock{Locker: store, Sessions: store, APITokens: store}
	tokens := signing.NewAccessTokens(keys)
	issuance := app.Issuance{
		Tokens:     tokens,
		MAC:        signing.NewRefreshTokenMAC(keys),
		AccessTTL:  d.AccessTokenTTL,
		SessionTTL: d.SessionTTL,
	}
	return &Module{
		uc: httpadapter.UseCases{
			Register: app.NewRegister(app.RegisterDeps{
				Policy: d.SignupPolicy, Rules: rules, Hasher: hasher, Tx: d.Tx,
				Users: store, Profiles: store, Sessions: store, Issuance: issuance, Clock: d.Clock, Logger: d.Logger,
			}),
			Login: app.NewLogin(app.LoginDeps{
				Accounts: store, Locker: store, Passwords: store, Sessions: store, Hasher: hasher, Tx: d.Tx,
				Issuance: issuance, Clock: d.Clock, Logger: d.Logger, DummyHash: dummy,
			}),
			Refresh:  app.NewRefresh(app.RefreshDeps{Sessions: store, Tx: d.Tx, Issuance: issuance, Clock: d.Clock, Logger: d.Logger}),
			Logout:   app.NewLogout(store, d.Clock, d.Logger),
			GetMe:    app.NewGetMe(store),
			UpdateMe: app.NewUpdateMe(store, d.Clock),
			ChangePassword: app.NewChangePassword(app.ChangePasswordDeps{
				Accounts: store, Lock: lock, Passwords: store, Sessions: store, Hasher: hasher,
				Rules: rules, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
			}),
			GetProfile:    app.NewGetProfile(store),
			UpdateProfile: app.NewUpdateProfile(store, d.Clock),
			ListAPITokens: app.NewListAPITokens(store),
			CreateAPIToken: app.NewCreateAPIToken(app.CreateAPITokenDeps{
				Lock: lock, Tokens: store, Tx: d.Tx, Clock: d.Clock, Logger: d.Logger,
			}),
			RevokeAPIToken: app.NewRevokeAPIToken(store, d.Clock, d.Logger),
		},
		settings: httpadapter.Settings{
			Limits:          httpadapter.Limits(d.RateLimits),
			RefreshDeadline: d.RefreshDeadline,
			Logger:          d.Logger,
		},
		authenticator: authn.New(app.NewAuthenticate(app.AuthenticateDeps{
			AccessTokens: tokens, Sessions: store, APITokens: store, Touch: store, Clock: d.Clock,
		})),
	}, nil
}

func signingKeys(d Deps) (*signing.Keys, error) {
	if d.SigningKeyPEM == nil {
		d.Logger.Warn("auth.jwt.private_key_file is not set: signing with an ephemeral key; " +
			"access tokens stop verifying at restart (dev and test only)")
		return signing.EphemeralKeys(), nil
	}
	keys, err := signing.ParseKeys(d.SigningKeyPEM)
	if err != nil {
		return nil, fmt.Errorf("auth.jwt.private_key_file: %w", err)
	}
	return keys, nil
}

// PublicOperations are the module's routes that need no token.
func (m *Module) PublicOperations() []string {
	return httpadapter.PublicOperations()
}

// Authenticator checks the bearer token of every non-public operation.
func (m *Module) Authenticator() httpserver.Authenticator {
	return m.authenticator
}

// Register mounts the module's API on router behind api's middlewares.
func (m *Module) Register(router *httpserver.Router, api *httpserver.API) {
	httpadapter.Register(router, api, m.uc, m.settings)
}
