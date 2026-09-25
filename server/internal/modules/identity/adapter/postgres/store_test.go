package postgresadapter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"net/netip"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/platform/config"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres/pgtest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// now is the fixed clock's time: whole microseconds, as timestamptz stores
// them, so the audit columns read back equal to it (M2 design 3.13).
var now = clocktest.At(time.Date(2026, 9, 25, 10, 0, 0, 123456789, time.UTC)).Now()

func newStore(t *testing.T) (*postgresadapter.Store, *pgxpool.Pool) {
	t.Helper()
	pool, err := postgres.NewPool(context.Background(), config.DatabaseConfig{URL: pgtest.NewDatabase(t), MaxConns: 4})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return postgresadapter.New(pool), pool
}

func newUser(email string) app.NewUser {
	return app.NewUser{ID: uuid.NewV7(), Email: email, PasswordHash: "$argon2id$v=19$m=64,t=1,p=1$c2FsdA$a2V5", DisplayName: domain.DisplayNameFromEmail(email), Now: now}
}

func mustCreate(t *testing.T, s *postgresadapter.Store, u app.NewUser) {
	t.Helper()
	if err := s.CreateUser(context.Background(), u); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAndGetUser(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)

	got, err := s.GetUser(context.Background(), u.ID)

	want := domain.User{ID: u.ID, Email: u.Email, DisplayName: "alice", Timezone: "UTC", CreatedAt: now}
	if err != nil || got != want {
		t.Errorf("GetUser() = %+v, %v; want %+v", got, err, want)
	}
	if got.CreatedAt.Location() != time.UTC {
		t.Errorf("created_at in %v, want UTC", got.CreatedAt.Location())
	}
	var password string
	var active bool
	var created, updated time.Time
	if err := pool.QueryRow(context.Background(), "SELECT password, is_active, created_at, updated_at FROM users WHERE id = $1", u.ID).
		Scan(&password, &active, &created, &updated); err != nil {
		t.Fatal(err)
	}
	if password != u.PasswordHash || !active || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("row = %q active=%v created %v updated %v; want the hash, active, and both audit columns at the clock's %v", password, active, created, updated, now)
	}
}

func TestGetUnknownUser(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.GetUser(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetUser() = %v, want app.ErrNotFound", err)
	}
}

// The unique constraint turns a race between two registrations into 409.
func TestCreateUserWithATakenAddress(t *testing.T) {
	s, pool := newStore(t)
	mustCreate(t, s, newUser("alice@corp.com"))
	tx := postgres.NewTxManager(pool, 2*time.Second)

	err := tx.WithinTx(context.Background(), func(ctx context.Context) error {
		return s.CreateUser(ctx, newUser("alice@corp.com"))
	})

	if !errors.Is(err, domain.ErrEmailTaken) {
		t.Errorf("CreateUser() = %v, want identity.email_taken", err)
	}
}

// The domain validates every value first, so a CHECK violation is a bug
// that got past it: an internal error, never a domain error.
func TestCreateUserBreakingACheckIsInternal(t *testing.T) {
	s, _ := newStore(t)

	err := s.CreateUser(context.Background(), newUser("Alice@corp.com"))

	var se *shared.Error
	var pgErr *pgconn.PgError
	if errors.As(err, &se) || !errors.As(err, &pgErr) || pgErr.ConstraintName != "users_email_check" {
		t.Errorf("CreateUser() = %v, want the check_violation of users_email_check, not a domain error", err)
	}
}

func TestCreateDefaultProfile(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	profileID := uuid.NewV7()

	if err := s.CreateDefaultProfile(context.Background(), profileID, u.ID, now); err != nil {
		t.Fatal(err)
	}

	var userID uuid.UUID
	var theme, language string
	var defaultSteps bool
	var week int16
	var onboarded bool
	var created, updated time.Time
	err := pool.QueryRow(context.Background(), `SELECT user_id, theme, language, start_of_the_week, is_onboarded,
		onboarding_step = '{"profile_complete": false, "workspace_create": false, "workspace_invite": false, "workspace_join": false}'::jsonb, created_at, updated_at
		FROM profiles WHERE id = $1`, profileID).Scan(&userID, &theme, &language, &week, &onboarded, &defaultSteps, &created, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if userID != u.ID || theme != "system" || language != "en" || week != 0 || onboarded || !defaultSteps ||
		!created.Equal(now) || !updated.Equal(now) {
		t.Errorf("profile = %v %s %s %d onboarded=%v default steps=%v %v %v", userID, theme, language, week, onboarded, defaultSteps, created, updated)
	}
}

func TestCreateSessionAndReadItsCredential(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	hash := sha256.Sum256([]byte("secret"))
	n := app.NewSession{
		ID: uuid.NewV7(), UserID: u.ID, TokenHash: hash[:], UserAgent: "agent/1",
		IP: netip.MustParseAddr("2001:db8::7"), ExpiresAt: now.Add(720 * time.Hour), Now: now,
	}

	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}

	var tokenHash []byte
	var generation int32
	var ua string
	var ip *netip.Addr
	var expires, created, updated time.Time
	err := pool.QueryRow(context.Background(), `SELECT token_hash, generation, user_agent, ip, expires_at, created_at, updated_at
		FROM auth_sessions WHERE id = $1`, n.ID).Scan(&tokenHash, &generation, &ua, &ip, &expires, &created, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(tokenHash, hash[:]) || generation != 0 || ua != "agent/1" || ip == nil || *ip != n.IP ||
		!expires.Equal(n.ExpiresAt) || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("session row = % x g=%d %q %v %v %v %v", tokenHash, generation, ua, ip, expires, created, updated)
	}

	cred, err := s.SessionCredential(context.Background(), n.ID)
	want := app.SessionCredential{UserID: u.ID, ExpiresAt: n.ExpiresAt, Revoked: false, UserActive: true}
	if err != nil || cred != want {
		t.Errorf("SessionCredential() = %+v, %v; want %+v", cred, err, want)
	}

	if _, err := pool.Exec(context.Background(), "UPDATE auth_sessions SET revoked_at = now(), revoke_reason = 'logout'"); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE users SET is_active = false"); err != nil {
		t.Fatal(err)
	}
	if cred, err := s.SessionCredential(context.Background(), n.ID); err != nil || !cred.Revoked || cred.UserActive {
		t.Errorf("SessionCredential() after revoke and deactivate = %+v, %v", cred, err)
	}
}

func TestCreateSessionWithoutAnIP(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	hash := sha256.Sum256([]byte("secret"))
	n := app.NewSession{ID: uuid.NewV7(), UserID: u.ID, TokenHash: hash[:], ExpiresAt: now.Add(time.Hour), Now: now}

	if err := s.CreateSession(context.Background(), n); err != nil {
		t.Fatal(err)
	}

	var ip *netip.Addr
	if err := pool.QueryRow(context.Background(), "SELECT ip FROM auth_sessions WHERE id = $1", n.ID).Scan(&ip); err != nil || ip != nil {
		t.Errorf("ip = %v, %v; want NULL", ip, err)
	}
}

func TestUnknownSessionCredential(t *testing.T) {
	s, _ := newStore(t)
	if _, err := s.SessionCredential(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("SessionCredential() = %v, want app.ErrNotFound", err)
	}
}
