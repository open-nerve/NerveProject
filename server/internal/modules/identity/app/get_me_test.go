package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func TestGetMe(t *testing.T) {
	want := domain.User{ID: userID, Email: "alice@corp.com", DisplayName: "alice", Timezone: "UTC", CreatedAt: now}
	store := &fakeStore{getUser: want}
	uc := app.NewGetMe(store)
	ctx := shared.WithActor(context.Background(), shared.Actor{UserID: userID, SessionID: sessionID})

	got, err := uc.Execute(ctx)

	if err != nil || got != want {
		t.Errorf("Execute() = %+v, %v; want %+v", got, err, want)
	}
	if len(store.getUserIDs) != 1 || store.getUserIDs[0] != userID {
		t.Errorf("accounts looked up = %v, want one lookup of the actor's %v", store.getUserIDs, userID)
	}
}

func TestGetMeIsUnauthenticated(t *testing.T) {
	authed := shared.WithActor(context.Background(), shared.Actor{UserID: userID})
	tests := []struct {
		name  string
		ctx   context.Context
		store *fakeStore
	}{
		{"without an actor", context.Background(), &fakeStore{}},
		{"when the account is gone", authed, &fakeStore{getUserErr: app.ErrNotFound}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := app.NewGetMe(tt.store).Execute(tt.ctx)
			if !errors.Is(err, shared.Unauthenticated()) {
				t.Errorf("Execute() = %v, want 401 unauthorized", err)
			}
		})
	}
}
