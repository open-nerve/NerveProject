package app_test

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/open-nerve/NerveProject/server/internal/modules/identity/app"
	"github.com/open-nerve/NerveProject/server/internal/modules/identity/domain"
	"github.com/open-nerve/NerveProject/server/internal/platform/clock/clocktest"
	"github.com/open-nerve/NerveProject/server/internal/shared"
)

func revoke(tokens *fakeAPITokens, logs *bytes.Buffer) error {
	uc := app.NewRevokeAPIToken(tokens, clocktest.At(now), slog.New(slog.NewJSONHandler(logs, nil)))
	return uc.Execute(shared.WithActor(context.Background(), sessionActor), tokenID)
}

func TestRevokeAPIToken(t *testing.T) {
	tokens, logs := &fakeAPITokens{revokeOK: true}, &bytes.Buffer{}

	err := revoke(tokens, logs)

	if want := (revocation{tokenID, userID, now}); err != nil || len(tokens.revoked) != 1 || tokens.revoked[0] != want {
		t.Errorf("Execute() = %v, revoked %+v; want %+v", err, tokens.revoked, want)
	}
	if s := logs.String(); !strings.Contains(s, `"msg":"API token revoked"`) || !strings.Contains(s, `"token_id":"`+tokenID.String()) ||
		!strings.Contains(s, `"user_id":"`+userID.String()) {
		t.Errorf("logs = %s, want the revocation with user_id and token_id", s)
	}
}

// A token that does not exist, is revoked already or is another account's
// is not found (M2 design 5.4).
func TestRevokeAnAPITokenThatIsNotTheCallers(t *testing.T) {
	err := revoke(&fakeAPITokens{revokeOK: false}, &bytes.Buffer{})

	if !errors.Is(err, domain.ErrAPITokenNotFound) {
		t.Errorf("Execute() = %v, want identity.api_token_not_found", err)
	}
}

func TestRevokeAPITokenDatabaseFailure(t *testing.T) {
	boom := errors.New("connection refused")

	if err := revoke(&fakeAPITokens{revokeErr: boom}, &bytes.Buffer{}); !errors.Is(err, boom) {
		t.Errorf("Execute() = %v, want the database error", err)
	}
}
