package postgresadapter_test

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

// PublicProfiles answers the accounts of ids that exist, by id: a
// deactivated one too, an unknown id left out, a repeated id once, and
// nothing for no ids.
func TestPublicProfilesReadsTheAccountsAskedFor(t *testing.T) {
	s, pool := newStore(t)
	alice, bob, carol := newUser("alice@corp.com"), newUser("bob@corp.com"), newUser("carol@corp.com")
	for _, u := range []app.NewUser{alice, bob, carol} {
		mustCreate(t, s, u)
	}
	exec(t, pool, "UPDATE users SET first_name = 'Alice', last_name = 'Liddell', display_name = 'al' WHERE id = $1", alice.ID)
	exec(t, pool, "UPDATE users SET is_active = false WHERE id = $1", bob.ID)

	got, err := s.PublicProfiles(context.Background(), []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID, bob.ID})

	want := []app.PublicProfile{
		{ID: alice.ID, Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: "bob@corp.com", DisplayName: bob.DisplayName},
	}
	slices.SortFunc(want, func(a, b app.PublicProfile) int { return slices.Compare(a.ID[:], b.ID[:]) })
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicProfiles() = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.PublicProfiles(context.Background(), nil); err != nil || len(got) != 0 {
		t.Errorf("PublicProfiles(nil) = %+v, %v; want none", got, err)
	}
}

// A read of the profiles that fails answers its error, never no profiles,
// which the member list would take for a member without an account.
func TestAFailedProfilesReadIsAnError(t *testing.T) {
	s, _ := newStore(t)
	alice := newUser("alice@corp.com")
	mustCreate(t, s, alice)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if got, err := s.PublicProfiles(cancelled, []uuid.UUID{alice.ID}); !errors.Is(err, context.Canceled) || got != nil {
		t.Errorf("PublicProfiles() = %+v, %v; want context.Canceled and no profile", got, err)
	}
}

// PublicProfiles takes no lock: it reads an account whose row another
// transaction holds FOR UPDATE, which every row lock waits for (a
// deactivation's FOR NO KEY UPDATE among them), without waiting, as it was
// committed (M3 design 3.6 convention 1).
func TestPublicProfilesDoesNotWaitForTheRowsLock(t *testing.T) {
	s, pool := newStore(t)
	alice := newUser("alice@corp.com")
	mustCreate(t, s, alice)
	end := hold(t, postgres.NewTxManager(pool, 5*time.Second), func(ctx context.Context) error {
		_, err := postgres.DB(ctx, pool).Exec(ctx, "SELECT 1 FROM users WHERE id = $1 FOR UPDATE", alice.ID)
		if err != nil {
			return err
		}
		_, err = postgres.DB(ctx, pool).Exec(ctx, "UPDATE users SET display_name = 'changing' WHERE id = $1", alice.ID)
		return err
	})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	got, err := s.PublicProfiles(ctx, []uuid.UUID{alice.ID})
	if err != nil || len(got) != 1 || got[0].DisplayName != alice.DisplayName {
		t.Errorf("PublicProfiles() under the lock = %+v, %v; want alice as committed, at once", got, err)
	}
	if err := end(); err != nil {
		t.Fatal(err)
	}
}
