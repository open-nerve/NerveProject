package postgresadapter_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5/pgxpool"

	postgresadapter "github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/postgres"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/postgres"
)

func ptr[T any](v T) *T { return &v }

// accountWithProfile is a new account and its default profile.
func accountWithProfile(t *testing.T, s *postgresadapter.Store) app.NewUser {
	t.Helper()
	u := newUser("alice@corp.com")
	mustCreate(t, s, u)
	if err := s.CreateDefaultProfile(context.Background(), uuid.NewV7(), u.ID, now); err != nil {
		t.Fatal(err)
	}
	return u
}

// Only the fields set change; updated_at is the use case's time.
func TestUpdateUser(t *testing.T) {
	s, pool := newStore(t)
	u := accountWithProfile(t, s)
	later := now.Add(time.Hour)

	got, err := s.UpdateUser(context.Background(), u.ID, domain.UserPatch{FirstName: ptr("Ada"), Timezone: ptr("Asia/Shanghai")}, later)

	want := domain.User{ID: u.ID, Email: u.Email, FirstName: "Ada", DisplayName: "alice", Timezone: "Asia/Shanghai", CreatedAt: now}
	if err != nil || got != want {
		t.Errorf("UpdateUser() = %+v, %v; want %+v", got, err, want)
	}
	got, err = s.UpdateUser(context.Background(), u.ID, domain.UserPatch{LastName: ptr("Lovelace"), DisplayName: ptr("ada")}, later)
	want.LastName, want.DisplayName = "Lovelace", "ada"
	if err != nil || got != want {
		t.Errorf("second UpdateUser() = %+v, %v; want %+v", got, err, want)
	}
	// An empty patch keeps every field and still records the write.
	evenLater := later.Add(time.Hour)
	got, err = s.UpdateUser(context.Background(), u.ID, domain.UserPatch{}, evenLater)
	if err != nil || got != want {
		t.Errorf("UpdateUser(empty) = %+v, %v; want %+v", got, err, want)
	}
	if read, err := s.GetUser(context.Background(), u.ID); err != nil || read != want {
		t.Errorf("GetUser() = %+v, %v; want %+v", read, err, want)
	}
	assertUpdatedAt(t, pool, "users", "id", u.ID, evenLater)
}

// Another account exists, so a statement that ignored the id would find it.
func TestUpdateUnknownUser(t *testing.T) {
	s, _ := newStore(t)
	accountWithProfile(t, s)
	if _, err := s.UpdateUser(context.Background(), uuid.NewV7(), domain.UserPatch{FirstName: ptr("x")}, now); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateUser() = %v, want app.ErrNotFound", err)
	}
}

// assertUpdatedAt checks the updated_at of the row of table whose column
// is id.
func assertUpdatedAt(t *testing.T, pool *pgxpool.Pool, table, column string, id uuid.UUID, want time.Time) {
	t.Helper()
	var got time.Time
	if err := pool.QueryRow(context.Background(), "SELECT updated_at FROM "+table+" WHERE "+column+" = $1", id).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Errorf("%s.updated_at = %v, want %v", table, got, want)
	}
}

func TestGetProfileReadsTheDefaults(t *testing.T) {
	s, _ := newStore(t)
	u := accountWithProfile(t, s)

	got, err := s.GetProfile(context.Background(), u.ID)

	want := domain.Profile{Theme: "system", Language: "en", UpdatedAt: now}
	if err != nil || got != want {
		t.Errorf("GetProfile() = %+v, %v; want %+v", got, err, want)
	}
	if _, err := s.GetProfile(context.Background(), uuid.NewV7()); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("GetProfile(unknown) = %v, want app.ErrNotFound", err)
	}
}

func TestUpdateProfile(t *testing.T) {
	s, _ := newStore(t)
	u := accountWithProfile(t, s)
	workspace := uuid.NewV7()
	later := now.Add(time.Hour)

	got, err := s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{
		Theme: ptr("dark-contrast"), Language: ptr("zh-CN"), StartOfTheWeek: ptr(6),
		OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true), WorkspaceJoin: ptr(true)},
		IsOnboarded:    ptr(true), IsTourCompleted: ptr(true), LastWorkspaceSet: true, LastWorkspaceID: &workspace,
	}, later)

	want := domain.Profile{
		Theme: "dark-contrast", Language: "zh-CN", StartOfTheWeek: 6,
		OnboardingStep: domain.OnboardingSteps{ProfileComplete: true, WorkspaceJoin: true},
		IsOnboarded:    true, IsTourCompleted: true, LastWorkspaceID: &workspace, UpdatedAt: later,
	}
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile() = %+v, %v; want %+v", got, err, want)
	}
	if read, err := s.GetProfile(context.Background(), u.ID); err != nil || !sameProfile(read, want) {
		t.Errorf("GetProfile() = %+v, %v; want %+v", read, err, want)
	}

	// An empty patch changes no preference and still records the write;
	// clearing the workspace leaves the rest.
	evenLater := later.Add(time.Hour)
	got, err = s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{}, evenLater)
	want.UpdatedAt = evenLater
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile(empty) = %+v, %v; want %+v", got, err, want)
	}
	got, err = s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{LastWorkspaceSet: true}, evenLater)
	want.LastWorkspaceID = nil
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile(clear the workspace) = %+v, %v; want %+v", got, err, want)
	}
	// Each flag is its own column.
	got, err = s.UpdateProfile(context.Background(), u.ID, domain.ProfilePatch{IsOnboarded: ptr(false), IsTourCompleted: ptr(true)}, evenLater)
	want.IsOnboarded = false
	if err != nil || !sameProfile(got, want) {
		t.Errorf("UpdateProfile(the flags apart) = %+v, %v; want %+v", got, err, want)
	}
	if read, err := s.GetProfile(context.Background(), u.ID); err != nil || !sameProfile(read, want) {
		t.Errorf("GetProfile() = %+v, %v; want %+v", read, err, want)
	}
}

// Another profile exists, so a statement that ignored the id would find it.
func TestUpdateUnknownProfile(t *testing.T) {
	s, _ := newStore(t)
	accountWithProfile(t, s)
	if _, err := s.UpdateProfile(context.Background(), uuid.NewV7(), domain.ProfilePatch{Theme: ptr("dark")}, now); !errors.Is(err, app.ErrNotFound) {
		t.Errorf("UpdateProfile() = %v, want app.ErrNotFound", err)
	}
}

// Deactivation sets the account inactive and starts its onboarding over,
// from the defaults of registration; the password, the other preferences
// and other accounts stay (M2 design 3.5, story A12).
func TestDeactivateUserAndResetOnboarding(t *testing.T) {
	s, pool := newStore(t)
	alice, bob := accountWithProfile(t, s), newUser("bob@corp.com")
	mustCreate(t, s, bob)
	if err := s.CreateDefaultProfile(context.Background(), uuid.NewV7(), bob.ID, now); err != nil {
		t.Fatal(err)
	}
	workspace := uuid.NewV7()
	onboarded := domain.ProfilePatch{
		Theme: ptr("dark"), OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true), WorkspaceJoin: ptr(true)},
		IsOnboarded: ptr(true), IsTourCompleted: ptr(true), LastWorkspaceSet: true, LastWorkspaceID: &workspace,
	}
	var bobs domain.Profile
	for _, u := range []app.NewUser{alice, bob} {
		p, err := s.UpdateProfile(context.Background(), u.ID, onboarded, now)
		if err != nil {
			t.Fatal(err)
		}
		bobs = p
	}
	deactivated := now.Add(time.Hour)

	if err := errors.Join(s.DeactivateUser(context.Background(), alice.ID, deactivated),
		s.ResetOnboarding(context.Background(), alice.ID, deactivated)); err != nil {
		t.Fatal(err)
	}

	active := func(u app.NewUser) (bool, string) {
		var active bool
		var password string
		if err := pool.QueryRow(context.Background(), "SELECT is_active, password FROM users WHERE id = $1", u.ID).Scan(&active, &password); err != nil {
			t.Fatal(err)
		}
		return active, password
	}
	if a, password := active(alice); a || password != alice.PasswordHash {
		t.Errorf("alice: active %v, password %q; want inactive with the password kept", a, password)
	}
	if b, _ := active(bob); !b {
		t.Error("bob is inactive, want him untouched")
	}
	assertUpdatedAt(t, pool, "users", "id", alice.ID, deactivated)
	want := domain.Profile{Theme: "dark", Language: "en", UpdatedAt: deactivated}
	if got, err := s.GetProfile(context.Background(), alice.ID); err != nil || !sameProfile(got, want) {
		t.Errorf("alice's profile = %+v, %v; want %+v", got, err, want)
	}
	if got, err := s.GetProfile(context.Background(), bob.ID); err != nil || !sameProfile(got, bobs) {
		t.Errorf("bob's profile = %+v, %v; want it untouched, %+v", got, err, bobs)
	}
}

func sameProfile(a, b domain.Profile) bool {
	sameWorkspace := (a.LastWorkspaceID == nil) == (b.LastWorkspaceID == nil) &&
		(a.LastWorkspaceID == nil || *a.LastWorkspaceID == *b.LastWorkspaceID)
	a.LastWorkspaceID, b.LastWorkspaceID = nil, nil
	return sameWorkspace && a.UpdatedAt.Equal(b.UpdatedAt) && a == b
}

// Two updates of different steps at once both stay (M2 design 3.14): the
// second waits for the first's row lock, then merges into the row the first
// committed. The test holds the first transaction open until the second
// statement is seen waiting for the lock, so the order is not left to
// chance; every wait has a deadline and fails rather than hangs.
func TestUpdateProfileMergesConcurrentSteps(t *testing.T) {
	s, pool := newStore(t)
	u := accountWithProfile(t, s)
	tx := postgres.NewTxManager(pool, 2*time.Second)
	ctx := context.Background()
	updated, release := make(chan struct{}), make(chan struct{})
	first, second := make(chan error, 1), make(chan error, 1)
	go func() {
		first <- tx.WithinTx(ctx, func(ctx context.Context) error {
			if _, err := s.UpdateProfile(ctx, u.ID, domain.ProfilePatch{OnboardingStep: domain.OnboardingStepsPatch{ProfileComplete: ptr(true)}}, now); err != nil {
				return err
			}
			close(updated)
			<-release
			return nil
		})
	}()
	// A failure below must still end the first transaction: the pool's Close
	// in the cleanup waits for its connection.
	var releaseOnce sync.Once
	releaseFirst := func() { releaseOnce.Do(func() { close(release) }) }
	defer releaseFirst()
	select {
	case <-updated:
	case err := <-first:
		t.Fatalf("first update: %v", err)
	case <-time.After(10 * time.Second):
		t.Fatal("the first update did not happen within 10s")
	}
	go func() {
		_, err := s.UpdateProfile(ctx, u.ID, domain.ProfilePatch{OnboardingStep: domain.OnboardingStepsPatch{WorkspaceCreate: ptr(true)}}, now)
		second <- err
	}()
	waitForLockWait(t, pool)
	releaseFirst()
	for name, done := range map[string]chan error{"first": first, "second": second} {
		select {
		case err := <-done:
			if err != nil {
				t.Fatalf("%s update: %v", name, err)
			}
		case <-time.After(10 * time.Second):
			t.Fatalf("the %s update did not finish within 10s", name)
		}
	}

	got, err := s.GetProfile(ctx, u.ID)
	want := domain.OnboardingSteps{ProfileComplete: true, WorkspaceCreate: true}
	if err != nil || got.OnboardingStep != want {
		t.Errorf("steps = %+v, %v; want both updates kept: %+v", got.OnboardingStep, err, want)
	}
}

// waitForLockWait returns once a backend of this test's database waits for
// a lock, and fails the test after 10s.
func waitForLockWait(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(10 * time.Millisecond) {
		var waiting int
		if err := pool.QueryRow(context.Background(),
			"SELECT count(*) FROM pg_stat_activity WHERE datname = current_database() AND wait_event_type = 'Lock'").Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			return
		}
	}
	t.Fatal("no statement waited for a lock within 10s")
}
