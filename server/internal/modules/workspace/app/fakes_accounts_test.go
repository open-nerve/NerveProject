package app_test

import (
	"context"
	"fmt"
	"slices"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/workspace/app"
)

// fakeAccounts answers the state of the accounts it holds, by id and by
// address, and logs each lock.
type fakeAccounts struct {
	log      *callLog
	accounts []app.AccountState
	err      error
}

func (f *fakeAccounts) ShareAccount(ctx context.Context, id uuid.UUID) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccount %s", id)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.ID == id })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

func (f *fakeAccounts) ShareAccountByEmail(ctx context.Context, email string) (app.AccountState, bool, error) {
	f.log.add(ctx, "ShareAccountByEmail %s", email)
	i := slices.IndexFunc(f.accounts, func(a app.AccountState) bool { return a.Email == email })
	if f.err != nil || i < 0 {
		return app.AccountState{}, false, f.err
	}
	return f.accounts[i], true, nil
}

// fakeProfiles answers the profiles it holds of the ids asked for, in the
// order it holds them, and logs each call with its ids.
type fakeProfiles struct {
	log      *callLog
	profiles []app.PublicProfile
	err      error
}

func (f *fakeProfiles) PublicProfiles(ctx context.Context, ids []uuid.UUID) ([]app.PublicProfile, error) {
	f.log.add(ctx, "PublicProfiles %v", ids)
	if f.err != nil {
		return nil, fmt.Errorf("read public profiles: %w", f.err)
	}
	var out []app.PublicProfile
	for _, p := range f.profiles {
		if slices.Contains(ids, p.ID) {
			out = append(out, p)
		}
	}
	return out, nil
}
