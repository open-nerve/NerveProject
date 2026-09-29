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
