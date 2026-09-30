package bootstrap

import (
	"context"
	"errors"
	"slices"
	"testing"
	"uuid"

	identityapp "github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/workspace"
)

// fakeIdentityAccounts answers the states it holds, by id and by address, and
// records what it was asked.
type fakeIdentityAccounts struct {
	states []identityapp.AccountState
	err    error
	asked  []string
}

func (f *fakeIdentityAccounts) ShareAccount(_ context.Context, id uuid.UUID) (identityapp.AccountState, bool, error) {
	f.asked = append(f.asked, "id "+id.String())
	i := slices.IndexFunc(f.states, func(s identityapp.AccountState) bool { return s.ID == id })
	if i < 0 {
		return identityapp.AccountState{}, false, f.err
	}
	return f.states[i], true, f.err
}

func (f *fakeIdentityAccounts) ShareAccountByEmail(_ context.Context, email string) (identityapp.AccountState, bool, error) {
	f.asked = append(f.asked, "email "+email)
	i := slices.IndexFunc(f.states, func(s identityapp.AccountState) bool { return s.Email == email })
	if i < 0 {
		return identityapp.AccountState{}, false, f.err
	}
	return f.states[i], true, f.err
}

// workspaceAccounts hands workspace identity's answer to the same question:
// the account asked for, its state converted, found and the error as they
// came.
func TestWorkspaceAccountsConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.AccountState{ID: uuid.NewV7(), Email: "alice@corp.com", Active: true}
	bob := identityapp.AccountState{ID: uuid.NewV7(), Email: "bob@corp.com", Active: false}
	fake := &fakeIdentityAccounts{states: []identityapp.AccountState{alice, bob}}
	a := workspaceAccounts{accounts: fake}
	ctx := context.Background()

	type answer struct {
		state workspace.AccountState
		found bool
	}
	var got []answer
	for _, id := range []uuid.UUID{alice.ID, bob.ID, uuid.NewV7()} {
		s, found, err := a.ShareAccount(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, answer{s, found})
	}
	for _, email := range []string{"bob@corp.com", "carol@corp.com"} {
		s, found, err := a.ShareAccountByEmail(ctx, email)
		if err != nil {
			t.Fatal(err)
		}
		got = append(got, answer{s, found})
	}
	want := []answer{
		{workspace.AccountState{ID: alice.ID, Email: "alice@corp.com", Active: true}, true},
		{workspace.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true},
		{},
		{workspace.AccountState{ID: bob.ID, Email: "bob@corp.com", Active: false}, true},
		{},
	}
	if !slices.Equal(got, want) {
		t.Errorf("answers = %+v, want %+v", got, want)
	}
	if len(fake.asked) != 5 || fake.asked[0] != "id "+alice.ID.String() || fake.asked[3] != "email bob@corp.com" {
		t.Errorf("identity was asked %q", fake.asked)
	}

	failure := errors.New("connection reset")
	fake.err = failure
	if _, _, err := a.ShareAccount(ctx, alice.ID); !errors.Is(err, failure) {
		t.Errorf("ShareAccount() = %v, want %v", err, failure)
	}
	if _, _, err := a.ShareAccountByEmail(ctx, "alice@corp.com"); !errors.Is(err, failure) {
		t.Errorf("ShareAccountByEmail() = %v, want %v", err, failure)
	}
}

// fakeIdentityProfiles answers the profiles it holds of the ids asked for,
// in the order it holds them, and records the ids.
type fakeIdentityProfiles struct {
	profiles []identityapp.PublicProfile
	err      error
	asked    [][]uuid.UUID
}

func (f *fakeIdentityProfiles) PublicProfiles(_ context.Context, ids []uuid.UUID) ([]identityapp.PublicProfile, error) {
	f.asked = append(f.asked, ids)
	var out []identityapp.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, f.err
}

// workspaceProfiles asks identity for the ids workspace asks for and hands
// over each profile, every field converted, in identity's order; an error
// as it came, without profiles.
func TestWorkspaceProfilesConvertsIdentitysAnswer(t *testing.T) {
	alice := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"}
	bob := identityapp.PublicProfile{ID: uuid.NewV7(), Email: "bob@corp.com", DisplayName: "bob"}
	fake := &fakeIdentityProfiles{profiles: []identityapp.PublicProfile{alice, bob}}
	p := workspaceProfiles{profiles: fake}
	ids := []uuid.UUID{bob.ID, uuid.NewV7(), alice.ID}

	got, err := p.PublicProfiles(context.Background(), ids)

	want := []workspace.PublicProfile{
		{ID: alice.ID, Email: "alice@corp.com", FirstName: "Alice", LastName: "Liddell", DisplayName: "al"},
		{ID: bob.ID, Email: "bob@corp.com", DisplayName: "bob"},
	}
	if err != nil || !slices.Equal(got, want) {
		t.Errorf("PublicProfiles() = %+v, %v; want %+v", got, err, want)
	}
	if len(fake.asked) != 1 || !slices.Equal(fake.asked[0], ids) {
		t.Errorf("identity was asked %v, want %v", fake.asked, ids)
	}
	failure := errors.New("connection reset")
	fake.err = failure
	if got, err := p.PublicProfiles(context.Background(), ids); !errors.Is(err, failure) || got != nil {
		t.Errorf("PublicProfiles() = %+v, %v; want no profiles and %v", got, err, failure)
	}
}
