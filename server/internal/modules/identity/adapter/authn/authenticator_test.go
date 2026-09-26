package authn_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"uuid"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/adapter/authn"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/platform/httpserver"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

// The platform's port, satisfied by structure.
var _ httpserver.Authenticator = (*authn.Authenticator)(nil)

// fakeUseCase answers its one token; any other token is a test error.
type fakeUseCase struct {
	token string
	actor shared.Actor
	err   error
}

func (f fakeUseCase) Execute(_ context.Context, token string) (shared.Actor, error) {
	if token != f.token {
		return shared.Actor{}, fmt.Errorf("fakeUseCase got token %q, want %q", token, f.token)
	}
	return f.actor, f.err
}

func TestAuthenticatePutsTheActorInTheContext(t *testing.T) {
	actor := shared.Actor{UserID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"), SessionID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000002")}

	ctx, key, err := authn.New(fakeUseCase{token: "token", actor: actor}).Authenticate(context.Background(), "token")

	got, actorErr := shared.RequireActor(ctx)
	if err != nil || actorErr != nil || got != actor || key != "session:0199a2b4-0000-7000-8000-000000000002" {
		t.Errorf("Authenticate() = actor %+v (%v), key %q, %v", got, actorErr, key, err)
	}
}

// A personal access token is limited by its own id (M2 design 3.10).
func TestAuthenticateKeysAPATByItsID(t *testing.T) {
	actor := shared.Actor{UserID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000001"), APITokenID: uuid.MustParse("0199a2b4-0000-7000-8000-000000000003")}

	ctx, key, err := authn.New(fakeUseCase{token: "nrv_pat_x", actor: actor}).Authenticate(context.Background(), "nrv_pat_x")

	got, actorErr := shared.RequireActor(ctx)
	if err != nil || actorErr != nil || got != actor || key != "pat:0199a2b4-0000-7000-8000-000000000003" {
		t.Errorf("Authenticate() = actor %+v (%v), key %q, %v", got, actorErr, key, err)
	}
}

// expiredCredential is the platform's optional interface on a 401.
type expiredCredential interface{ ExpiredCredential() bool }

func TestAuthenticatePassesErrorsThrough(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		unauthorized bool // whether the error is a 401 ProblemError
		expired      bool // whether it reports ExpiredCredential() true
	}{
		{"invalid credential", fmt.Errorf("%w: %w", shared.Unauthenticated(), errors.New("session is revoked")), true, false},
		{"expired access token", fmt.Errorf("%w: %w", shared.Unauthenticated(), app.ErrAccessTokenExpired), true, true},
		{"internal fault", errors.New("database is down"), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, key, err := authn.New(fakeUseCase{token: "token", err: tt.err}).Authenticate(context.Background(), "token")

			var ec expiredCredential
			isExpired := errors.As(err, &ec) && ec.ExpiredCredential()
			if !errors.Is(err, tt.err) || err.Error() != tt.err.Error() || ctx != nil || key != "" || isExpired != tt.expired {
				t.Errorf("Authenticate() = %v, %q, %v (expired %v); want nil, \"\", %v (expired %v)", ctx, key, err, isExpired, tt.err, tt.expired)
			}
			var pe httpserver.ProblemError
			if is401 := errors.As(err, &pe) && pe.ProblemStatus() == 401; is401 != tt.unauthorized {
				t.Errorf("Authenticate() error %v is a 401: %v, want %v", err, is401, tt.unauthorized)
			}
		})
	}
}
