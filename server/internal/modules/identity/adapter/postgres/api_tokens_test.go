package postgresadapter_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
)

// newToken is a token of userID made at when, whose hash comes from name.
func newToken(userID uuid.UUID, name string, when time.Time) app.NewAPIToken {
	hash := sha256.Sum256([]byte(name))
	return app.NewAPIToken{ID: uuid.NewV7(), UserID: userID, TokenHash: hash[:], Label: name, Now: when}
}

func mustCreateToken(t *testing.T, s *postgresadapter.Store, n app.NewAPIToken) {
	t.Helper()
	if err := s.CreateAPIToken(context.Background(), n); err != nil {
		t.Fatal(err)
	}
}

func TestCreateAPIToken(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	expires := now.Add(7 * 24 * time.Hour)
	n := newToken(u.ID, "deploy", now)
	n.Description, n.ExpiredAt = "ci", &expires

	mustCreateToken(t, s, n)

	var hash []byte
	var label, description string
	var expired, lastUsed, deleted *time.Time
	var createdBy, updatedBy *uuid.UUID
	var created, updated time.Time
	err := pool.QueryRow(context.Background(), `SELECT token_hash, label, description, expired_at, last_used, deleted_at,
		created_by_id, updated_by_id, created_at, updated_at FROM api_tokens WHERE id = $1 AND user_id = $2`, n.ID, u.ID).
		Scan(&hash, &label, &description, &expired, &lastUsed, &deleted, &createdBy, &updatedBy, &created, &updated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(hash, n.TokenHash) || label != "deploy" || description != "ci" || expired == nil || !expired.Equal(expires) ||
		lastUsed != nil || deleted != nil {
		t.Errorf("row = % x %q %q expired %v last used %v deleted %v", hash, label, description, expired, lastUsed, deleted)
	}
	// The account created the token for itself; the audit columns are the
	// use case's time (M2 design 3.13).
	if createdBy == nil || *createdBy != u.ID || updatedBy == nil || *updatedBy != u.ID || !created.Equal(now) || !updated.Equal(now) {
		t.Errorf("audit = created by %v at %v, updated by %v at %v; want the account at %v", createdBy, created, updatedBy, updated, now)
	}
}

// Pages follow created_at, newest first, then id: rows made at the same
// instant are neither repeated nor skipped across a page boundary. Revoked
// tokens and other accounts' tokens are not listed.
func TestListAPITokensPageByPage(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	// Three tokens share an instant, so a page of two ends inside them.
	for i, at := range []time.Time{now, now.Add(time.Second), now.Add(time.Second), now.Add(time.Second), now.Add(2 * time.Second)} {
		mustCreateToken(t, s, newToken(alice.ID, "t"+string(rune('a'+i)), at))
	}
	mustCreateToken(t, s, newToken(bob.ID, "bob's", now.Add(time.Hour)))
	revoked := newToken(alice.ID, "revoked", now.Add(time.Minute))
	mustCreateToken(t, s, revoked)
	if _, err := s.RevokeAPIToken(context.Background(), revoked.ID, alice.ID, now); err != nil {
		t.Fatal(err)
	}

	var got []uuid.UUID
	var after *domain.APITokenCursor
	for range 4 {
		page, err := s.ListAPITokens(context.Background(), alice.ID, after, 2)
		if err != nil {
			t.Fatal(err)
		}
		if len(page) == 0 {
			break
		}
		for _, tok := range page {
			got = append(got, tok.ID)
		}
		last := page[len(page)-1]
		after = &domain.APITokenCursor{CreatedAt: last.CreatedAt, ID: last.ID}
	}
	if want := expectedOrder(t, pool, alice.ID); !slices.Equal(got, want) || len(got) != 5 {
		t.Errorf("pages = %v, want %v", got, want)
	}
}

// expectedOrder is the order the list promises, computed apart from the
// query under test: created_at descending, then id descending.
func expectedOrder(t *testing.T, pool *pgxpool.Pool, userID uuid.UUID) []uuid.UUID {
	t.Helper()
	rows, err := pool.Query(context.Background(), "SELECT id, created_at FROM api_tokens WHERE user_id = $1 AND deleted_at IS NULL", userID)
	if err != nil {
		t.Fatal(err)
	}
	type row struct {
		id uuid.UUID
		at time.Time
	}
	var all []row
	for rows.Next() {
		var r row
		if err := rows.Scan(&r.id, &r.at); err != nil {
			t.Fatal(err)
		}
		all = append(all, r)
	}
	slices.SortFunc(all, func(a, b row) int {
		if c := b.at.Compare(a.at); c != 0 {
			return c
		}
		return bytes.Compare(b.id[:], a.id[:])
	})
	ids := make([]uuid.UUID, len(all))
	for i, r := range all {
		ids[i] = r.id
	}
	return ids
}

func TestListAPITokensReadsEveryField(t *testing.T) {
	s, _ := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	expires := now.Add(time.Hour)
	n := newToken(u.ID, "deploy", now)
	n.Description, n.ExpiredAt = "ci", &expires
	mustCreateToken(t, s, n)
	used := now.Add(time.Minute)
	if err := s.TouchAPIToken(context.Background(), n.ID, used, used.Add(-time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, err := s.ListAPITokens(context.Background(), u.ID, nil, 10)

	if err != nil || len(got) != 1 {
		t.Fatalf("ListAPITokens() = %+v, %v; want one token", got, err)
	}
	tok := got[0]
	if tok.ID != n.ID || tok.Label != "deploy" || tok.Description != "ci" || tok.ExpiredAt == nil || !tok.ExpiredAt.Equal(expires) ||
		tok.LastUsed == nil || !tok.LastUsed.Equal(used) || !tok.CreatedAt.Equal(now) {
		t.Errorf("ListAPITokens() = %+v", tok)
	}
}

func TestRevokeAPIToken(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := newUser("alice@corp.com"), newUser("bob@corp.com")
	mustCreate(t, s, alice)
	mustCreate(t, s, bob)
	n := newToken(alice.ID, "deploy", now)
	mustCreateToken(t, s, n)
	later := now.Add(time.Hour)

	byBob, err1 := s.RevokeAPIToken(context.Background(), n.ID, bob.ID, later)
	unknown, err2 := s.RevokeAPIToken(context.Background(), uuid.NewV7(), alice.ID, later)
	first, err3 := s.RevokeAPIToken(context.Background(), n.ID, alice.ID, later)
	again, err4 := s.RevokeAPIToken(context.Background(), n.ID, alice.ID, later.Add(time.Hour))

	if err := errors.Join(err1, err2, err3, err4); err != nil || byBob || unknown || !first || again {
		t.Errorf("revoke by another account %v, unknown %v, first %v, again %v (%v); want only the first", byBob, unknown, first, again, err)
	}
	var deleted, updated time.Time
	var updatedBy uuid.UUID
	if err := pool.QueryRow(context.Background(), "SELECT deleted_at, updated_at, updated_by_id FROM api_tokens WHERE id = $1", n.ID).
		Scan(&deleted, &updated, &updatedBy); err != nil {
		t.Fatal(err)
	}
	if !deleted.Equal(later) || !updated.Equal(later) || updatedBy != alice.ID {
		t.Errorf("row = deleted %v updated %v by %v; want both at %v by the account", deleted, updated, updatedBy, later)
	}
}

func TestAPITokenCredentials(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	expires := now.Add(time.Hour)
	n := newToken(u.ID, "deploy", now)
	n.ExpiredAt = &expires
	mustCreateToken(t, s, n)

	byHash, err1 := s.APITokenByHash(context.Background(), n.TokenHash)
	byID, err2 := s.APITokenByID(context.Background(), n.ID)

	want := app.APITokenCredential{ID: n.ID, UserID: u.ID, ExpiredAt: &expires, UserActive: true}
	for _, got := range []app.APITokenCredential{byHash, byID} {
		if got.ID != want.ID || got.UserID != want.UserID || got.ExpiredAt == nil || !got.ExpiredAt.Equal(expires) ||
			got.LastUsed != nil || got.Revoked || !got.UserActive {
			t.Errorf("credential = %+v, want %+v", got, want)
		}
	}
	if err := errors.Join(err1, err2); err != nil {
		t.Fatal(err)
	}
	other := sha256.Sum256([]byte("other"))
	if _, err := s.APITokenByHash(context.Background(), other[:]); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("APITokenByHash(unknown) = %v, want app.ErrNotFound", err)
	}
	if _, err := s.APITokenByID(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("APITokenByID(unknown) = %v, want app.ErrNotFound", err)
	}

	if _, err := s.RevokeAPIToken(context.Background(), n.ID, u.ID, now); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(context.Background(), "UPDATE users SET is_active = false"); err != nil {
		t.Fatal(err)
	}
	byHash, err1 = s.APITokenByHash(context.Background(), n.TokenHash)
	byID, err2 = s.APITokenByID(context.Background(), n.ID)
	if err := errors.Join(err1, err2); err != nil || !byHash.Revoked || byHash.UserActive || !byID.Revoked || byID.UserActive {
		t.Errorf("after revoke and deactivate: by hash %+v, by id %+v, %v", byHash, byID, err)
	}
}

// last_used is written at most once a minute, and using a token changes
// nothing else of it (M2 design 3.5).
func TestTouchAPIToken(t *testing.T) {
	s, pool := newStore(t)
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	n := newToken(u.ID, "deploy", now)
	mustCreateToken(t, s, n)
	lastUsed := func() time.Time {
		t.Helper()
		var used *time.Time
		var updated time.Time
		if err := pool.QueryRow(context.Background(), "SELECT last_used, updated_at FROM api_tokens WHERE id = $1", n.ID).Scan(&used, &updated); err != nil {
			t.Fatal(err)
		}
		if !updated.Equal(now) {
			t.Errorf("updated_at = %v, want it unchanged at %v", updated, now)
		}
		if used == nil {
			return time.Time{}
		}
		return *used
	}
	first := now.Add(time.Minute)
	touch := func(at time.Time) {
		t.Helper()
		if err := s.TouchAPIToken(context.Background(), n.ID, at, at.Add(-time.Minute)); err != nil {
			t.Fatal(err)
		}
	}

	touch(first)
	if got := lastUsed(); !got.Equal(first) {
		t.Errorf("last_used after the first use = %v, want %v", got, first)
	}
	touch(first.Add(59 * time.Second))
	if got := lastUsed(); !got.Equal(first) {
		t.Errorf("last_used after a use within the minute = %v, want it still %v", got, first)
	}
	touch(first.Add(61 * time.Second))
	if got := lastUsed(); !got.Equal(first.Add(61 * time.Second)) {
		t.Errorf("last_used after the minute = %v, want %v", got, first.Add(61*time.Second))
	}
}
