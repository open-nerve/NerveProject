package authn_test

import (
	"context"
	"errors"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The platform's port, satisfied by structure.
var _ httpserver.Authenticator = (*authn.Authenticator)(nil)

type fakeUseCase struct {
	actor shared.Actor
	err   error
}

func (f fakeUseCase) Execute(context.Context, string) (shared.Actor, error) { return f.actor, f.err }

func TestAuthenticatePutsTheActorInTheContext(t *testing.T) {
	actor := shared.Actor{UserID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"), SessionID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")}

	ctx, key, err := authn.New(fakeUseCase{actor: actor}).Authenticate(context.Background(), "token")

	got, actorErr := shared.RequireActor(ctx)
	if err != nil || actorErr != nil || got != actor || key != "session:0199a2b4-0000-7000-8000-000000000002" {
		t.Errorf("Authenticate() = actor %+v (%v), key %q, %v", got, actorErr, key, err)
	}
}

func TestAuthenticatePassesErrorsThrough(t *testing.T) {
	invalid := shared.Unauthenticated()
	boom := errors.New("database is down")
	for _, want := range []error{invalid, boom} {
		ctx, key, err := authn.New(fakeUseCase{err: want}).Authenticate(context.Background(), "token")
		if !errors.Is(err, want) || ctx != nil || key != "" {
			t.Errorf("Authenticate() = %v, %q, %v; want nil, \"\", %v", ctx, key, err, want)
		}
	}
}
